package system

import (
	"context"
	"fmt"
	"strings"

	v1 "spark/api/common/v1"
	"spark/internal/data/dal/model"
	"spark/internal/data/do"
	clog "spark/pkg/clog"
)

type SystemUserRepo interface {
	FindByID(context.Context, int64) (*model.SysUser, error)
	FindByName(context.Context, string) (*model.SysUser, error)
	FindByMobile(context.Context, string) (*model.SysUser, error)

	Create(ctx context.Context, user *model.SysUser) (*model.SysUser, error)
	Update(context.Context, *do.SysUser) (*do.SysUser, error)
	Delete(context.Context, int64) error

	ListUser(ctx context.Context, current int32, pageSize int32, name *string, nickname *string, status *int32, deptId *int64) ([]*model.SysUser, int32, error)
}

type SystemUserPostRepo interface {
	// ListUserPosts 获取用户的岗位列表
	ListUserPosts(ctx context.Context, userID int64) ([]*model.SysPost, error)
	// ReplaceUserPosts 替换用户的岗位关联
	ReplaceUserPosts(ctx context.Context, userID int64, postIDs []int64) error
}

// DeptRepoGetter 定义获取部门信息的接口
type DeptRepoGetter interface {
	// Get 获取部门信息
	Get(ctx context.Context, deptID int64) (*model.SysDept, error)
}

type SystemUserUsecase struct {
	repo         SystemUserRepo
	roleRepo     SystemRoleRepo
	casbinRepo   CasbinRuleRepo
	userPostRepo SystemUserPostRepo
	deptRepo     DeptRepoGetter
	logger       clog.Logger
}

func NewSystemUserUsecase(
	repo SystemUserRepo,
	roleRepo SystemRoleRepo,
	casbinRepo CasbinRuleRepo,
	userPostRepo SystemUserPostRepo,
	deptRepo DeptRepoGetter,
	logger clog.Logger,
) *SystemUserUsecase {
	return &SystemUserUsecase{
		repo:         repo,
		roleRepo:     roleRepo,
		casbinRepo:   casbinRepo,
		userPostRepo: userPostRepo,
		deptRepo:     deptRepo,
		logger:       logger,
	}
}

// Update 更新用户基本信息
func (uc *SystemUserUsecase) Update(ctx context.Context, u *do.SysUser) (*do.SysUser, error) {
	return uc.repo.Update(ctx, u)
}

// UpdateUserWithBindings 在 usecase 层编排用户信息和绑定关系的更新
func (uc *SystemUserUsecase) UpdateUserWithBindings(ctx context.Context, user *do.SysUser, roleIDs []int64, postIDs []int64) (*do.SysUser, error) {
	if user.ID <= 0 {
		return nil, fmt.Errorf("invalid user ID: %d", user.ID)
	}

	// 1. 验证用户存在
	_, err := uc.repo.FindByID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// 2. 验证角色存在
	if len(roleIDs) > 0 {
		for _, roleID := range roleIDs {
			_, err := uc.roleRepo.Get(ctx, roleID)
			if err != nil {
				return nil, fmt.Errorf("role ID %d not found: %w", roleID, err)
			}
		}
	}

	// 3. 更新用户基本信息
	_, err = uc.repo.Update(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// 4. 更新角色绑定
	if len(roleIDs) > 0 {
		err = uc.UpdateUserRoles(ctx, user.ID, roleIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to update user roles: %w", err)
		}
	}

	// 5. 更新岗位绑定
	if len(postIDs) > 0 {
		err = uc.userPostRepo.ReplaceUserPosts(ctx, user.ID, postIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to update user posts: %w", err)
		}
	}

	return user, nil
}

// CreateUser 创建用户并分配角色
func (uc *SystemUserUsecase) CreateUser(ctx context.Context, u *model.SysUser, roleIDs []int64) (*model.SysUser, error) {
	// 1. 验证用户名是否已存在
	if u.Name != "" {
		existingUser, err := uc.repo.FindByName(ctx, u.Name)
		if err == nil && existingUser != nil {
			return nil, v1.ErrorCommonError("用户名已存在")
		}
	}

	// 2. 验证角色存在
	if len(roleIDs) > 0 {
		for _, roleID := range roleIDs {
			_, err := uc.roleRepo.Get(ctx, roleID)
			if err != nil {
				return nil, fmt.Errorf("role ID %d not found: %w", roleID, err)
			}
		}
	}

	// 3. 创建用户
	user, err := uc.repo.Create(ctx, u)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "1062") {
			return nil, v1.ErrorCommonError("用户名已存在")
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// 4. 分配角色
	if len(roleIDs) > 0 {
		for _, roleID := range roleIDs {
			err = uc.casbinRepo.AssignUserRole(ctx, user.ID, roleID)
			if err != nil {
				uc.logger.Warn(ctx, "failed to assign role to user", clog.Int64("user_id", user.ID), clog.Int64("role_id", roleID), clog.Err(err))
			}
		}
	}

	return user, nil
}

func (uc *SystemUserUsecase) SelectUserProfile(ctx context.Context, id int64) (*model.SysUser, error) {
	return uc.repo.FindByID(ctx, id)
}

func (uc *SystemUserUsecase) GetUserByName(ctx context.Context, username string) (*model.SysUser, error) {
	return uc.repo.FindByName(ctx, username)
}

func (uc *SystemUserUsecase) GetUserList(ctx context.Context, current int32, pageSize int32, name *string, nickname *string, status *int32, deptId *int64) ([]*model.SysUser, int32, error) {
	return uc.repo.ListUser(ctx, current, pageSize, name, nickname, status, deptId)
}

func (uc *SystemUserUsecase) GetUserByID(ctx context.Context, id int64) (*model.SysUser, error) {
	return uc.repo.FindByID(ctx, id)
}

func (uc *SystemUserUsecase) GetUserByMobile(ctx context.Context, mobile string) (*model.SysUser, error) {
	return uc.repo.FindByMobile(ctx, mobile)
}

// GetUserRoles 获取用户的角色列表
func (uc *SystemUserUsecase) GetUserRoles(ctx context.Context, id int64) ([]*model.SysRole, error) {
	// 1. 获取用户的角色ID列表
	roleIDs, err := uc.casbinRepo.ListUserRoleIDs(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user role IDs: %w", err)
	}

	if len(roleIDs) == 0 {
		return []*model.SysRole{}, nil
	}

	return uc.roleRepo.ListByIDs(ctx, roleIDs)
}

func (uc *SystemUserUsecase) UpdateUserRoles(ctx context.Context, userID int64, roleIDs []int64) error {
	if len(roleIDs) > 0 {
		for _, roleID := range roleIDs {
			_, err := uc.roleRepo.Get(ctx, roleID)
			if err != nil {
				return fmt.Errorf("role ID %d not found: %w", roleID, err)
			}
		}
	}

	currentRoleIDs, err := uc.casbinRepo.ListUserRoleIDs(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get current user roles: %w", err)
	}

	currentMap := make(map[int64]bool, len(currentRoleIDs))
	for _, roleID := range currentRoleIDs {
		currentMap[roleID] = true
	}

	newMap := make(map[int64]bool, len(roleIDs))
	for _, roleID := range roleIDs {
		newMap[roleID] = true
	}

	for _, roleID := range currentRoleIDs {
		if !newMap[roleID] {
			if err := uc.casbinRepo.RevokeUserRole(ctx, userID, roleID); err != nil {
				uc.logger.Warn(ctx, "failed to revoke user role", clog.Int64("user_id", userID), clog.Int64("role_id", roleID), clog.Err(err))
			}
		}
	}

	for _, roleID := range roleIDs {
		if !currentMap[roleID] {
			if err := uc.casbinRepo.AssignUserRole(ctx, userID, roleID); err != nil {
				return fmt.Errorf("failed to assign user role: %w", err)
			}
		}
	}

	return nil
}

func (uc *SystemUserUsecase) DeleteUser(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}

// GetUserProfile 获取用户完整档案（聚合查询）
func (uc *SystemUserUsecase) GetUserProfile(ctx context.Context, userID int64) (*do.SysUserProfile, error) {
	// 1. 查询用户基本信息
	user, err := uc.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	profile := &do.SysUserProfile{}
	// 将 model.SysUser 转换为 do.SysUserProfile
	profile.ID = user.ID
	profile.Name = &user.Name
	profile.Nickname = &user.Nickname
	profile.Mobile = &user.Mobile
	profile.Birthday = &user.Birthday
	profile.Password = &user.Password
	profile.Status = &user.Status
	profile.Email = &user.Email
	profile.Sex = &user.Sex
	profile.Avatar = &user.Avatar
	profile.DeptID = &user.DeptID
	profile.IsAdmin = &user.IsAdmin
	profile.Address = user.Address
	profile.Remark = user.Remark
	profile.LastLoginIP = user.LastLoginIP
	profile.LastLoginTime = user.LastLoginTime
	profile.CreatedBy = user.CreatedBy
	profile.UpdatedBy = user.UpdatedBy
	profile.CreatedAt = &user.CreatedAt
	profile.UpdatedAt = &user.UpdatedAt

	// 2. 查询用户角色
	roles, err := uc.GetUserRoles(ctx, userID)
	if err != nil {
		uc.logger.Warn(ctx, "failed to get user roles", clog.Int64("user_id", userID), clog.Err(err))
	} else {
		profile.Roles = roles
	}

	// 3. 查询用户岗位
	posts, err := uc.userPostRepo.ListUserPosts(ctx, userID)
	if err != nil {
		uc.logger.Warn(ctx, "failed to get user posts", clog.Int64("user_id", userID), clog.Err(err))
	} else {
		profile.Posts = posts
	}

	// 4. 查询部门信息
	if user.DeptID > 0 {
		dept, err := uc.deptRepo.Get(ctx, user.DeptID)
		if err != nil {
			uc.logger.Warn(ctx, "failed to get user dept", clog.Int64("user_id", userID), clog.Int64("dept_id", user.DeptID), clog.Err(err))
		} else {
			profile.DeptName = &dept.DeptName
		}
	}

	return profile, nil
}

// UpdatePasswordByID 通过用户ID更新密码
func (uc *SystemUserUsecase) UpdatePasswordByID(ctx context.Context, userID int64, hashedPassword string) error {
	user := &do.SysUser{
		ID:       userID,
		Password: &hashedPassword,
	}
	_, err := uc.repo.Update(ctx, user)
	if err != nil {
		uc.logger.Error(ctx, "Update password by ID failed", clog.Err(err))
		return err
	}
	return nil
}
