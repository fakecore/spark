package system

import (
	"context"
	clog "spark/pkg/clog"
	utils2 "spark/pkg/utils"
	"time"

	pb "spark/api/system/v1"
	"spark/internal/biz/system"
	"spark/internal/common/constants"
	"spark/internal/conf"
	"spark/internal/data/do"

	"spark/pkg/kvstore"
	"spark/pkg/viewer"

	"github.com/go-kratos/kratos/v2/errors"
	"gorm.io/gorm"
)

type AuthService struct {
	userUc *system.SystemUserUsecase
	logger clog.Logger
	config *conf.Auth
	kv     kvstore.Store
}

func NewAuthService(userUc *system.SystemUserUsecase, logger clog.Logger, config *conf.Bootstrap, kv kvstore.Store) *AuthService {
	return &AuthService{
		userUc: userUc,
		logger: logger,
		config: config.Auth,
		kv:     kv,
	}
}

func (s *AuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginReply, error) {
	if s.kv == nil {
		s.logger.Error(ctx, "kv store is nil (auth service misconfigured)")
		return nil, errors.New(500, "SYSTEM_CONFIG_ERROR", "系统配置错误,请联系管理员")
	}

	loginUser, err := s.userUc.GetUserByName(ctx, req.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(401, "USER_NOT_FOUND", "用户不存在")
		}
		s.logger.Error(ctx, "查询用户失败",
			clog.Err(err),
			clog.String("username", req.Username),
		)
		return nil, errors.New(500, "LOGIN_ERROR", "登录失败,请稍后重试")
	}

	// 2. 验证密码
	match, err := utils2.VerifyPassword(req.Password, loginUser.Password)
	if err != nil || !match {
		s.logger.Debug(ctx, "密码验证失败",
			clog.Err(err),
			clog.String("username", req.Username),
		)
		return nil, errors.New(401, "INVALID_CREDENTIALS", "输入账号/密码错误")
	}

	// 3. 检查用户状态
	userStatus := constants.UserStatus(loginUser.Status)
	if userStatus.IsNotActive() {
		if userStatus == constants.UserStatus_Ban {
			return nil, errors.New(403, "USER_BANNED", "用户账号已禁用")
		}
		return nil, errors.New(403, "USER_INACTIVE", "用户账号未激活")
	}

	// 4. 检查用户角色
	roles, err := s.userUc.GetUserRoles(ctx, int64(loginUser.ID))
	if err != nil {
		s.logger.Error(ctx, "获取用户角色失败", clog.Err(err))
		return nil, errors.New(500, "LOGIN_ERROR", "登录失败,请稍后重试")
	}

	if len(roles) == 0 {
		s.logger.Error(ctx, "用户没有分配角色",
			clog.Int64("userID", loginUser.ID),
			clog.String("userName", loginUser.Name))
		return nil, errors.New(403, "NO_ROLES", "用户没有权限,请联系管理员")
	}

	anyRoleEnable := false
	var activeRoleID int64
	for _, role := range roles {
		if role.Status == int32(constants.CommonStatus_ACTIVE) {
			anyRoleEnable = true
			activeRoleID = role.ID
			break
		}
	}
	if !anyRoleEnable {
		s.logger.Error(ctx, "用户所有角色都已禁用",
			clog.Int64("userID", loginUser.ID),
			clog.String("userName", loginUser.Name))
		return nil, errors.New(403, "ROLES_DISABLED", "用户所属角色都已禁用")
	}

	// 5. 生成token
	jwtTool := utils2.NewJWTUtils(s.config.AccessSecret, time.Duration(s.config.AccessExpire.Seconds)*time.Second)
	token, expire, err := jwtTool.GenerateToken(loginUser.ID, activeRoleID)
	if err != nil {
		s.logger.Error(ctx, "创建token出错",
			clog.Int64("userID", loginUser.ID),
			clog.Err(err))
		return nil, errors.New(500, "TOKEN_ERROR", "登录失败,请稍后重试")
	}

	// 6. 更新用户最后登录时间
	currentTime := time.Now().UnixMilli()
	_, err = s.userUc.Update(ctx, &do.SysUser{
		ID:            loginUser.ID,
		LastLoginTime: &currentTime,
	})
	if err != nil {
		s.logger.Error(ctx, "更新用户最后登录时间失败",
			clog.Int64("userID", loginUser.ID),
			clog.Err(err))
		// 不返回错误，避免影响登录流程
	}

	// 7. 生成token的hash值并保存到Redis
	tokenHash := constants.GenerateTokenHash(token)
	redisKey := constants.GenTokenHashRedisKey(loginUser.ID, tokenHash)
	err = s.kv.Set(ctx, redisKey, token, time.Duration(s.config.AccessExpire.Seconds)*time.Second)
	if err != nil {
		s.logger.Error(ctx, "保存token到Redis失败",
			clog.Err(err),
		)
		return nil, errors.New(500, "REDIS_ERROR", "登录失败,请稍后重试")
	}

	return &pb.LoginReply{
		Token:   token,
		Expires: int64(expire.Unix()),
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutReply, error) {
	if s.kv == nil {
		s.logger.Error(ctx, "kv store is nil (auth service misconfigured)")
		return nil, errors.New(500, "SYSTEM_CONFIG_ERROR", "系统配置错误,请联系管理员")
	}

	userInfo, ok := viewer.UserViewFromContext(ctx)
	if !ok {
		return nil, errors.New(401, "USER_NOT_FOUND", "用户信息不存在")
	}
	user := userInfo.GetUser()

	// 删除该用户的所有token（所有设备登出）
	pattern := constants.GenUserTokenPattern(user.ID)
	keys, err := s.kv.Keys(ctx, pattern)
	if err != nil {
		s.logger.Error(ctx, "获取用户token keys失败", clog.Err(err))
		return nil, errors.New(500, "LOGOUT_ERROR", "登出失败,请稍后重试")
	}

	if len(keys) > 0 {
		err = s.kv.Del(ctx, keys...)
		if err != nil {
			s.logger.Error(ctx, "删除用户tokens失败", clog.Err(err))
			return nil, errors.New(500, "LOGOUT_ERROR", "登出失败,请稍后重试")
		}
	}

	return &pb.LogoutReply{}, nil
}

// LogoutCurrentDevice 登出当前设备（单设备登出）
// 需要在调用时传入当前的token
func (s *AuthService) LogoutCurrentDevice(ctx context.Context, token string) error {
	if s.kv == nil {
		s.logger.Error(ctx, "kv store is nil (auth service misconfigured)")
		return errors.New(500, "SYSTEM_CONFIG_ERROR", "系统配置错误,请联系管理员")
	}

	userInfo, ok := viewer.UserViewFromContext(ctx)
	if !ok {
		return errors.New(401, "USER_NOT_FOUND", "用户信息不存在")
	}
	user := userInfo.GetUser()

	// 计算当前token的hash并删除
	tokenHash := constants.GenerateTokenHash(token)
	redisKey := constants.GenTokenHashRedisKey(user.ID, tokenHash)

	err := s.kv.Del(ctx, redisKey)
	if err != nil {
		s.logger.Error(ctx, "删除当前设备token失败", clog.Err(err))
		return errors.New(500, "LOGOUT_ERROR", "登出失败,请稍后重试")
	}

	return nil
}
