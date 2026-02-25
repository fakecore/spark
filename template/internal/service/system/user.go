package system

import (
	"context"
	"fmt"
	"math/rand"

	clog "spark/pkg/clog"
	"spark/pkg/utils"
	"spark/pkg/viewer"

	commonv1 "spark/api/common/v1"
	v1 "spark/api/system/v1"
	"spark/internal/biz/system"
	"spark/internal/common/constants"
	"spark/internal/data/dal/model"
	"spark/internal/data/do"
)

type UserService struct {
	v1.UnimplementedUserServer
	userUc *system.SystemUserUsecase
	logger clog.Logger
}

func NewUserService(u *system.SystemUserUsecase, logger clog.Logger) *UserService {
	return &UserService{
		userUc: u,
		logger: logger,
	}
}

func (s *UserService) CreateUser(ctx context.Context, req *v1.CreateUserRequest) (*v1.CreateUserReply, error) {
	s.logger.Info(ctx, "Creating user")

	// 参数验证
	if req.Name == nil || *req.Name == "" {
		return nil, commonv1.ErrorSystemUserError("用户名不能为空")
	}

	// 用户名格式验证
	name := *req.Name
	if len(name) < 2 || len(name) > 50 {
		return nil, commonv1.ErrorSystemUserError("用户名长度必须在2-50个字符之间")
	}

	user := &model.SysUser{
		Nickname: utils.GetDataOrDefault(req.Nickname, name),
		Mobile:   utils.GetDataOrDefault(req.Mobile, ""),
		Email:    utils.GetDataOrDefault(req.Email, ""),
		Name:     name,
		DeptID:   utils.GetDataOrDefault(req.DeptId, 0),
		Status:   utils.GetDataOrDefault(req.Status, 1),
	}
	hashedPassword, err := utils.HashPassword(generatePassword())
	if err != nil {
		s.logger.Error(ctx, "Failed to hash password:", clog.Err(err))
		return nil, commonv1.ErrorSystemUserError("内部错误, 请稍后重试")
	}
	user.Password = hashedPassword

	_, err = s.userUc.CreateUser(ctx, user, req.RoleIds)
	if err != nil {
		s.logger.Error(ctx, "Failed to create user:", clog.Err(err))
		return nil, err
	}

	return &v1.CreateUserReply{}, nil
}

func (s *UserService) UpdateUser(ctx context.Context, req *v1.UpdateUserRequest) (*v1.UpdateUserReply, error) {
	// 更新用户信息
	user := &do.SysUser{
		ID:       req.Id,
		Nickname: req.Nickname,
		Mobile:   req.Mobile,
		Email:    req.Email,
		Avatar:   req.Avatar,
		DeptID:   req.DeptId,
		RoleIds:  req.RoleIds,
		PostIds:  req.PostIds,
		Status:   req.Status,
	}

	s.logger.Info(ctx, "Updating user",
		clog.Int64("user_id", req.Id),
		clog.Int64s("role_ids", req.RoleIds),
		clog.Int64s("post_ids", req.PostIds))

	// 使用单个事务更新用户信息和绑定关系
	_, err := s.userUc.UpdateUserWithBindings(ctx, user, req.RoleIds, req.PostIds)
	if err != nil {
		s.logger.Error(ctx, "Failed to update user with bindings:", clog.Err(err))
		return nil, commonv1.ErrorSystemUserError("更新用户信息失败")
	}

	return &v1.UpdateUserReply{}, nil
}

func (s *UserService) DeleteUser(ctx context.Context, req *v1.DeleteUserRequest) (*v1.DeleteUserReply, error) {
	s.logger.Info(ctx, "Deleting user")

	err := s.userUc.DeleteUser(ctx, req.Id)
	if err != nil {
		s.logger.Error(ctx, "Failed to delete user:", clog.Err(err))
		return nil, err
	}

	return &v1.DeleteUserReply{}, nil
}

func (s *UserService) GetUser(ctx context.Context, req *v1.GetUserRequest) (*v1.GetUserReply, error) {
	s.logger.Info(ctx, "Getting user")

	user, err := s.userUc.GetUserProfile(ctx, req.Id)
	if err != nil {
		s.logger.Error(ctx, "Failed to get user:", clog.Err(err))
		return nil, err
	}

	// 创建API响应
	userInfo := &v1.UserInfo{
		Id:        user.ID,
		Name:      utils.GetDataOrDefault(user.Name, ""),
		Nickname:  utils.GetDataOrDefault(user.Nickname, ""),
		Mobile:    utils.GetDataOrDefault(user.Mobile, ""),
		Email:     utils.GetDataOrDefault(user.Email, ""),
		Avatar:    utils.GetDataOrDefault(user.Avatar, ""),
		Status:    utils.GetDataOrDefault(user.Status, 0),
		DeptId:    utils.GetDataOrDefault(user.DeptID, 0),
		Birthday:  utils.GetDataOrDefault(user.Birthday, 0),
		CreatedAt: utils.GetDataOrDefault(user.CreatedAt, 0),
		UpdatedAt: utils.GetDataOrDefault(user.UpdatedAt, 0),
		DeptName:  utils.GetDataOrDefault(user.DeptName, ""),
	}

	// 设置角色相关字段
	userInfo.Roles = make([]*v1.SysRole, 0, len(user.Roles))
	for _, role := range user.Roles {
		roleInfo := &v1.SysRole{}
		utils.ObjConvert(role, roleInfo)
		userInfo.Roles = append(userInfo.Roles, roleInfo)
	}

	// 设置岗位相关字段
	userInfo.PostNames = make(map[int64]string, len(user.Posts))
	for _, post := range user.Posts {
		userInfo.PostNames[post.ID] = post.PostName
	}

	return &v1.GetUserReply{
		User: userInfo,
	}, nil
}

func (s *UserService) ListUser(ctx context.Context, req *v1.ListUserRequest) (*v1.ListUserReply, error) {
	// 根据租户权限获取用户列表
	users, total, err := s.userUc.GetUserList(ctx, req.Current, req.PageSize, req.Name, req.Nickname, req.Status, req.DeptId) // 使用过滤条件
	if err != nil {
		s.logger.Error(ctx, "Failed to list users:", clog.Err(err))
		return nil, err
	}

	userList := make([]*v1.UserInfo, 0, len(users))
	for _, user := range users {
		// 创建API响应
		userInfo := &v1.UserInfo{
			Id:        user.ID,
			Name:      user.Name,
			Nickname:  user.Nickname,
			Mobile:    user.Mobile,
			Email:     user.Email,
			Avatar:    user.Avatar,
			Status:    user.Status,
			DeptId:    user.DeptID,
			Birthday:  user.Birthday,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			IsAdmin:   user.IsAdmin,
			// 注意：Roles、Posts、DeptName 需要通过独立查询获取
			// 这些关联关系已从 model 中移除，改为业务层处理
		}

		userList = append(userList, userInfo)
	}

	return &v1.ListUserReply{
		Items: userList,
		Total: total,
	}, nil
}

func (s *UserService) GetUserProfile(ctx context.Context, req *v1.GetUserProfileRequest) (*v1.GetUserProfileReply, error) {

	userId := viewer.MustGetUserViewFromContext(ctx).GetUser().ID

	userProfile, err := s.userUc.GetUserProfile(ctx, userId)
	if err != nil {
		s.logger.Error(ctx, "Failed to get user profile:", clog.Err(err))
		return nil, err
	}

	profile := &v1.UserProfile{}
	utils.ObjConvert(userProfile, profile)

	return &v1.GetUserProfileReply{
		Profile: profile,
	}, nil
}

func (s *UserService) UpdateUserStatus(ctx context.Context, req *v1.UpdateUserStatusRequest) (*v1.UpdateUserStatusReply, error) {

	_, err := s.userUc.Update(ctx, &do.SysUser{
		ID:     req.Id,
		Status: &req.Status,
	})

	if err != nil {
		return nil, commonv1.ErrorSystemUserError("无法更新用户状态,请稍后再试")
	}
	return &v1.UpdateUserStatusReply{}, nil
}

func (s *UserService) ResetPassword(ctx context.Context, req *v1.ResetPasswordRequest) (*v1.ResetPasswordReply, error) {

	pwd, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, commonv1.ErrorSystemUserError("系统错误,请稍后再试")
	}

	_, err = s.userUc.Update(ctx, &do.SysUser{
		ID:       req.Id,
		Password: &pwd,
	})

	if err != nil {
		return nil, commonv1.ErrorSystemUserError("系统错误,请稍后再试")
	}

	return &v1.ResetPasswordReply{}, nil
}

func (s *UserService) ChangePassword(ctx context.Context, req *v1.ChangePasswordRequest) (*v1.ChangePasswordReply, error) {
	s.logger.Info(ctx, "Changing password")
	userView := viewer.MustGetUserViewFromContext(ctx)
	err := verifySmsCode(userView.GetUser().ID, req.VerifyCode)
	if err != nil {
		s.logger.Error(ctx, "Failed to verify sms code", clog.Err(err))
		return nil, commonv1.ErrorCommonError("验证码验证失败")
	}
	u, err := s.userUc.GetUserByID(ctx, userView.GetUser().ID)
	if err != nil {
		return nil, commonv1.ErrorCommonError("用户不存在")
	}
	matched, err := utils.VerifyPassword(req.OldPassword, u.Password)
	if err != nil || !matched {
		return nil, commonv1.ErrorCommonError("旧密码输入错误,请重试")
	}

	pwd, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return nil, commonv1.ErrorSystemUserError("系统错误,请稍后再试")
	}

	_, err = s.userUc.Update(ctx, &do.SysUser{
		ID:       userView.GetUser().ID,
		Password: &pwd,
	})

	if err != nil {
		return nil, commonv1.ErrorSystemUserError("系统错误,请稍后再试")
	}
	return &v1.ChangePasswordReply{}, nil
}

// AdminChangePassword 允许管理员重置任意用户密码
func (s *UserService) AdminChangePassword(ctx context.Context, req *v1.AdminChangePasswordRequest) (*v1.AdminChangePasswordReply, error) {
	// 权限校验（可选：如需判断当前用户是否为管理员，可补充）
	if req.UserId == 0 || req.NewPassword == "" {
		return &v1.AdminChangePasswordReply{Code: 1, Msg: "参数错误"}, nil
	}
	hashed, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		s.logger.Error(ctx, "Hash password failed", clog.Err(err))
		return &v1.AdminChangePasswordReply{Code: 2, Msg: "密码加密失败"}, nil
	}
	err = s.userUc.UpdatePasswordByID(ctx, req.UserId, hashed)
	if err != nil {
		s.logger.Error(ctx, "Update password failed", clog.Err(err))
		return &v1.AdminChangePasswordReply{Code: 3, Msg: "修改失败"}, nil
	}
	return &v1.AdminChangePasswordReply{Code: 0, Msg: "修改成功"}, nil
}

func (s *UserService) ValidUser(ctx context.Context, userID int64) (*model.SysUser, error) {
	user, err := s.userUc.SelectUserProfile(ctx, userID)
	if err != nil {
		s.logger.Error(ctx, "Failed to get user profile", clog.Err(err))
		return nil, err
	}

	userStatus := constants.UserStatus(user.Status)
	if userStatus.IsNotActive() {
		s.logger.Error(ctx, "用户状态异常", clog.String("name", user.Name), clog.Int32("status", user.Status))
		if userStatus == constants.UserStatus_Ban {
			return nil, commonv1.ErrorSystemAuthError("用户账号已禁用")
		}
		return nil, commonv1.ErrorSystemAuthError("用户账号未激活")
	}

	// if len(user.Roles) == 0 {
	// 	return nil, commonv1.ErrorSystemAuthError("用户没有分配角色")
	// }

	// available := false
	// for _, role := range user.Roles {
	// 	if role.Status == int32(constants.CommonStatus_ACTIVE) {
	// 		available = true
	// 		break
	// 	}
	// }

	// if !available {
	// 	s.logger.Error(ctx,"用户 Id:%d %s 所有角色都已禁用", user.ID, user.Name)
	// 	return nil, commonv1.ErrorSystemAuthError("用户所属角色都已禁用")
	// }

	return user, nil
}

func generatePassword() string {
	return "123456aA!"
}

func (s *UserService) generateUserName() string {
	randomNum := fmt.Sprintf("%06d", rand.Intn(1000000))
	return randomNum
}

func verifySmsCode(userID int64, code string) error {
	return nil
}
