package system

import (
	"context"
	"fmt"

	"spark/internal/biz/system"
	"spark/internal/data/dal/model"
	"spark/internal/data/dal/query"
	clog "spark/pkg/clog"
	"spark/pkg/utils"

	"gorm.io/gorm"
)

type systemUserPostRepo struct {
	db     *gorm.DB
	logger clog.Logger
	query  *query.Query
}

var _ system.SystemUserPostRepo = &systemUserPostRepo{}

func NewSystemUserPostRepo(db *gorm.DB, q *query.Query, logger clog.Logger) system.SystemUserPostRepo {
	return &systemUserPostRepo{
		db:     db,
		logger: logger,
		query:  q,
	}
}

// ListUserPosts 获取用户的岗位列表
func (r *systemUserPostRepo) ListUserPosts(ctx context.Context, userID int64) ([]*model.SysPost, error) {
	// 1. 查询用户岗位关联
	userPostRelations, err := r.query.SysUserPost.WithContext(ctx).
		Where(r.query.SysUserPost.UserID.Eq(userID)).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to get user posts: %w", err)
	}

	if len(userPostRelations) == 0 {
		return []*model.SysPost{}, nil
	}

	// 2. 提取岗位ID
	postIDs := make([]int64, 0, len(userPostRelations))
	for _, up := range userPostRelations {
		postIDs = append(postIDs, up.PostID)
	}

	// 3. 查询岗位详情
	posts, err := r.query.SysPost.WithContext(ctx).
		Where(r.query.SysPost.ID.In(postIDs...)).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to get post details: %w", err)
	}

	return posts, nil
}

// ReplaceUserPosts 替换用户的岗位关联
func (r *systemUserPostRepo) ReplaceUserPosts(ctx context.Context, userID int64, postIDs []int64) error {
	tx, ok := utils.GetTx(ctx)
	ownsTx := false
	if !ok || tx == nil {
		tx = r.db.WithContext(ctx).Begin()
		if tx.Error != nil {
			return fmt.Errorf("failed to begin transaction: %w", tx.Error)
		}
		ownsTx = true
		ctx = utils.WithTx(ctx, tx)
	}

	defer func() {
		if rec := recover(); rec != nil && ownsTx {
			tx.Rollback()
		}
	}()

	// 1. 删除现有的用户岗位关联
	if err := tx.Where("user_id = ?", userID).Delete(&model.SysUserPost{}).Error; err != nil {
		if ownsTx {
			tx.Rollback()
		}
		return fmt.Errorf("failed to delete existing user-post relations: %w", err)
	}

	// 2. 创建新的岗位关联
	if len(postIDs) > 0 {
		userPosts := make([]*model.SysUserPost, 0, len(postIDs))
		for _, postID := range postIDs {
			userPosts = append(userPosts, &model.SysUserPost{
				UserID: userID,
				PostID: postID,
			})
		}

		if err := tx.CreateInBatches(userPosts, 100).Error; err != nil {
			if ownsTx {
				tx.Rollback()
			}
			return fmt.Errorf("failed to create user-post relations: %w", err)
		}
	}

	if ownsTx {
		if err := tx.Commit().Error; err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	return nil
}
