package system

import (
	"context"

	v1 "spark/api/common/v1"
	"spark/internal/biz/system"
	"spark/internal/data/dal/model"
	"spark/internal/data/dal/query"

	"spark/internal/data/do"
	clog "spark/pkg/clog"
	"spark/pkg/utils"
	"spark/pkg/viewer"
)

type systemPostRepo struct {
	logger clog.Logger
	query  *query.Query
}

// NewSystemPostRepo .
func NewSystemPostRepo(q *query.Query, logger clog.Logger) system.SystemPostRepo {
	return &systemPostRepo{
		logger: logger,
		query:  q,
	}
}

func (r *systemPostRepo) Get(ctx context.Context, id int64) (*model.SysPost, error) {
	postQuery := r.query.SysPost
	post, err := postQuery.WithContext(ctx).Where(postQuery.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return post, nil
}

func (r *systemPostRepo) List(ctx context.Context, pageSize, current int32, postCode, postName *string, status *int32) ([]*model.SysPost, int64, error) {
	postQuery := r.query.SysPost

	postDo := postQuery.WithContext(ctx)
	if postCode != nil && *postCode != "" {
		postDo = postDo.Where(postQuery.PostCode.Like("%" + *postCode + "%"))
	}
	if postName != nil && *postName != "" {
		postDo = postDo.Where(postQuery.PostName.Like("%" + *postName + "%"))
	}
	if status != nil {
		postDo = postDo.Where(postQuery.Status.Eq(*status))
	}

	var total int64
	posts, err := postDo.Scopes(utils.Paginate(current, pageSize)).Find()
	if err != nil {
		return nil, 0, v1.ErrorSystemPostError("list post failed: %v", err)
	}
	total, err = postDo.Count()
	if err != nil {
		return nil, 0, v1.ErrorSystemPostError("count posts failed: %v", err)
	}
	return posts, total, nil
}

func (r *systemPostRepo) Create(ctx context.Context, post *model.SysPost) error {
	err := r.query.SysPost.WithContext(ctx).Create(post)
	if err != nil {
		return v1.ErrorSystemPostError("create post failed: %v", err)
	}
	return nil
}

func (r *systemPostRepo) Update(ctx context.Context, post *do.SysPost) error {
	userView := viewer.MustGetUserViewFromContext(ctx)
	postQuery := r.query.SysPost
	updateData := utils.StructToMap(post)
	updateData["updated_by"] = userView.GetUser().ID

	result, err := postQuery.WithContext(ctx).Where(postQuery.ID.Eq(post.ID)).Updates(updateData)
	if err != nil {
		return v1.ErrorSystemPostError("update post failed: %v", err)
	}
	if result.RowsAffected == 0 {
		return v1.ErrorSystemPostError("post not found")
	}
	return nil
}

func (r *systemPostRepo) Delete(ctx context.Context, id int64) error {
	postQuery := r.query.SysPost
	_, err := postQuery.WithContext(ctx).Where(postQuery.ID.Eq(id)).Delete()
	if err != nil {
		return v1.ErrorSystemPostError("delete post failed: %v", err)
	}
	return nil
}
