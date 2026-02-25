package middleware

import (
	"context"
	"errors"
	"strings"
	"time"

	commonv1 "spark/api/common/v1"
	systemV1 "spark/api/system/v1"
	"spark/internal/common/constants"
	"spark/internal/data/dal/model"
	"spark/internal/service/system"
	"spark/pkg/clog"
	"spark/pkg/principal"
	"spark/pkg/utils"
	"spark/pkg/viewer"

	"github.com/casbin/casbin/v2"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"spark/pkg/kvstore"
)

type AuthOption struct {
	AccessSecret string
	AccessExpire time.Duration
	UserService  *system.UserService
	RoleService  *system.RoleService
	KV           kvstore.Store
	Casbin       *casbin.Enforcer
	Logger       clog.Logger
}

// AuthInterceptorMiddleware creates an authentication middleware
func AuthInterceptorMiddleware(opts AuthOption) middleware.Middleware {
	logger := opts.Logger

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if opts.KV == nil {
				logger.Error(ctx, "kv store is nil (auth middleware misconfigured)")
				return nil, commonv1.ErrorSystemAuthError("系统配置错误,请联系管理员")
			}

			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return nil, commonv1.ErrorSystemAuthError("missing auth info")
			}

			// 1. Get and validate token format
			token := tr.RequestHeader().Get("Authorization")
			token = strings.TrimPrefix(token, "Bearer ")
			if token == "" {
				return nil, commonv1.ErrorSystemAuthError("授权验证失败,请重新登录")
			}

			// 2. Parse and validate token
			claims, err := parseValidJwt(ctx, token, opts)
			if err != nil {
				return nil, err
			}
			if logger.Enabled(clog.DebugLevel) {
				logger.Debug(ctx, "claims parsed", clog.Any("claims", claims))
			}

			// 3. Get user information
			userReply, err := opts.UserService.GetUser(ctx, &systemV1.GetUserRequest{Id: claims.UserID})
			if err != nil {
				logger.Error(ctx, "查询用户异常", clog.Int64("user_id", claims.UserID), clog.Err(err))
				return nil, commonv1.ErrorSystemAuthError("查询用户异常")
			}
			user := userReply.User

			// Convert to model and add to viewer context
			modelUser := &model.SysUser{}
			utils.ObjConvert(user, modelUser)
			ctx = viewer.NewContext(ctx, viewer.NewNormalViewer(modelUser))

			// 4. Validate user status
			if err := validateUserStatus(user, logger); err != nil {
				return nil, err
			}

			// Add user to context for backward compatibility
			ctx = context.WithValue(ctx, "user", user)

			ctx, err = bindPrincipalContext(ctx, claims, user, opts.RoleService)
			if err != nil {
				logger.Error(ctx, "构建权限上下文失败", clog.Int64("user_id", claims.UserID), clog.Int64("role_id", claims.RoleID), clog.Err(err))
				return nil, commonv1.ErrorSystemAuthError("用户权限异常,请重新登录")
			}

			// 5. Verify token in Redis (支持多设备登录)
			tokenHash := constants.GenerateTokenHash(token)
			redisKey := constants.GenTokenHashRedisKey(user.Id, tokenHash)
			_, err = opts.KV.Get(ctx, redisKey)
			if err != nil {
				if errors.Is(err, kvstore.ErrNotFound) {
					logger.Warn(ctx, "token not found in kv store", clog.String("user_name", user.Name))
					return nil, commonv1.ErrorSystemAuthError("登录已失效,请重新登录")
				}
				logger.Error(ctx, "获取用户缓存失败", clog.String("user_name", user.Name), clog.Err(err))
				return nil, commonv1.ErrorSystemAuthError("用户校验失败,请重新登录")
			}

			// 6. Check permissions
			// 使用 Casbin 权限检查
			if !handleCasbin(ctx, tr, opts.RoleService, opts.Casbin, claims, logger) {
				return nil, commonv1.ErrorSystemAuthError("权限不足")
			}

			// 7. Handle token renewal if needed
			if err := handleToken(ctx, claims, redisKey, tr, token, logger, opts); err != nil {
				return nil, err
			}

			// Process the request with authenticated context
			return handler(ctx, req)
		}
	}
}

// validateUserStatus checks if the user account is active
func validateUserStatus(u *systemV1.UserInfo, logger clog.Logger) error {
	userStatus := constants.UserStatus(u.Status)
	if userStatus.IsNotActive() {
		logger.Error(context.Background(), "用户状态异常", clog.String("user_name", u.Name), clog.Int32("status", u.Status))
		if userStatus == constants.UserStatus_Ban {
			return commonv1.ErrorSystemAuthError("用户账号已禁用")
		}
		return commonv1.ErrorSystemAuthError("用户账号未激活")
	}
	return nil
}

func bindPrincipalContext(ctx context.Context, claims *utils.CustomClaims, user *systemV1.UserInfo, roleService *system.RoleService) (context.Context, error) {
	if claims == nil || user == nil || claims.RoleID <= 0 || roleService == nil {
		return ctx, nil
	}

	roleCode := ""
	for _, role := range user.Roles {
		if role != nil && role.Id == claims.RoleID {
			roleCode = role.Code
			break
		}
	}

	roleReply, err := roleService.GetRole(ctx, &systemV1.GetRoleRequest{Id: claims.RoleID})
	if err != nil {
		return nil, err
	}

	p := &principal.Principal{
		UserID:       claims.UserID,
		RoleID:       claims.RoleID,
		RoleCode:     roleCode,
		DeptID:       user.DeptId,
		Scope:        constants.DataScopeFromDB(roleReply.Role.DataScope),
		IsSuperAdmin: roleCode == constants.RoleCodeSuperAdmin || user.IsAdmin == 1,
		ClientType:   claims.ClientType,
	}
	return principal.NewContext(ctx, p), nil
}

// handleCasbin verifies if the user has permission to access the requested resource
func handleCasbin(ctx context.Context, r transport.Transporter, roleService *system.RoleService, casbin *casbin.Enforcer, claims *utils.CustomClaims, logger clog.Logger) bool {
	path := r.Operation()
	method := r.RequestHeader().Get("X-Method")

	logger.Info(ctx, "handleCasbin", clog.String("path", path), clog.String("method", method))

	// Super admin bypass
	if p, ok := principal.FromContext(ctx); ok && p.IsSuperAdmin {
		return true
	}

	hasPermission, err := roleService.CheckAPIPermission(ctx, &systemV1.CheckAPIPermissionRequest{
		RoleId: claims.RoleID,
		Path:   path,
		Method: method,
	})
	if err != nil {
		logger.Error(ctx, "权限用例检查失败", clog.Int64("role_id", claims.RoleID), clog.String("path", path), clog.Err(err))
		return false
	} else {
		if !hasPermission.HasPermission {
			logger.Info(ctx, "用户权限不足", clog.Int64("user_id", claims.UserID), clog.String("path", path))
		}
		return hasPermission.HasPermission
	}
}

// parseValidJwt parses and validates JWT token
func parseValidJwt(ctx context.Context, token string, opts AuthOption) (*utils.CustomClaims, error) {
	jwt := utils.NewJWTUtils(opts.AccessSecret, opts.AccessExpire)
	claims, err := jwt.ParseToken(token)
	if err != nil {
		if err == utils.ErrExpiredToken {
			return nil, commonv1.ErrorSystemAuthError("授权已到期，请重新登录")
		}
		opts.Logger.Error(ctx, "解析token失败", clog.String("token", token), clog.Err(err))
		return nil, commonv1.ErrorSystemAuthError("授权验证失败,请重新登录")
	}

	return claims, nil
}

// isClientAPIPath 检查是否为客户端专用 API 路径
// TODO: 根据业务需求配置客户端允许访问的接口列表
func isClientAPIPath(path string) bool {
	// 桌面客户端允许访问的接口列表（按业务需求配置）
	allowedAPIs := []string{
		// 示例："/api.v1.Client/",
	}

	for _, allowed := range allowedAPIs {
		// 如果是前缀匹配（以 / 结尾），检查是否以该前缀开头
		if len(allowed) > 0 && allowed[len(allowed)-1] == '/' {
			if len(path) >= len(allowed) && path[:len(allowed)] == allowed {
				return true
			}
		} else {
			// 精确匹配
			if path == allowed {
				return true
			}
		}
	}
	return false
}

// handleToken handles token renewal if expiring soon
func handleToken(ctx context.Context, claims *utils.CustomClaims, redisKey string, tr transport.Transporter, token string, logger clog.Logger, opts AuthOption) error {
	userViewer, ok := viewer.UserViewFromContext(ctx)
	if !ok {
		return commonv1.ErrorSystemAuthError("获取用户信息失败")
	}
	user := userViewer.GetUser()
	jwt := utils.NewJWTUtils(opts.AccessSecret, opts.AccessExpire)

	// Renew token if expiring within 15 minutes
	if jwt.IsTokenExpiredWithin(claims, 15*time.Minute) {
		newToken, err := jwt.RenewToken(token)
		if err != nil {
			logger.Error(ctx, "刷新token失败", clog.String("token", token), clog.Err(err))
			return commonv1.ErrorSystemAuthError("刷新token失败")
		}

		// Delete old token from Redis
		err = opts.KV.Del(ctx, redisKey)
		if err != nil {
			logger.Error(ctx, "删除当前token失败", clog.String("user_name", user.Name), clog.Err(err))
		}

		// Calculate new token hash and key
		newTokenHash := constants.GenerateTokenHash(newToken)
		newRedisKey := constants.GenTokenHashRedisKey(user.ID, newTokenHash)

		// Store new token in Redis
		if err = opts.KV.Set(ctx, newRedisKey, newToken, opts.AccessExpire); err != nil {
			logger.Error(ctx, "更新Redis token失败", clog.String("user_name", user.Name), clog.Err(err))
			return commonv1.ErrorSystemAuthError("更新Redis token失败")
		}

		tr.ReplyHeader().Add("Authorization", "Bearer "+newToken)
	}
	return nil
}
