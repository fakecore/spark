package system

import (
	"context"

	"spark/internal/biz/system"
	"spark/internal/data/dal/model"
	"spark/internal/data/dal/query"

	clog "spark/pkg/clog"
	"spark/pkg/utils"
)

var _ system.FileRepo = &fileRepo{}

func NewSystemFileRepo(q *query.Query, logger clog.Logger) system.FileRepo {
	return &fileRepo{
		logger: logger,
		query:  q,
	}
}

type fileRepo struct {
	logger clog.Logger
	query  *query.Query
}

// Create implements system.FileRepo.
func (r *fileRepo) Create(ctx context.Context, dept *model.SysFile) error {
	err := r.query.SysFile.WithContext(ctx).Create(dept)
	if err != nil {
		return err
	}
	return nil
}

// Delete implements system.FileRepo.
func (r *fileRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.query.SysFile.WithContext(ctx).Where(r.query.SysFile.ID.Eq(id)).Delete()
	if err != nil {
		return err
	}
	return nil
}

// Get implements system.FileRepo.
func (f *fileRepo) Get(ctx context.Context, id int64) (*model.SysFile, error) {
	fileQuery := f.query.SysFile
	file, err := fileQuery.WithContext(ctx).Where(fileQuery.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return file, nil
}

// List implements system.FileRepo.
func (f *fileRepo) List(ctx context.Context, current int32, pageSize int32) ([]*model.SysFile, int64, error) {
	fileQuery := f.query.SysFile
	total, err := fileQuery.WithContext(ctx).Count()
	if err != nil {
		return nil, 0, err
	}
	files, err := fileQuery.WithContext(ctx).Scopes(utils.Paginate(current, pageSize)).Find()
	if err != nil {
		return nil, 0, err
	}
	return files, total, nil
}

// FindByOssPath implements system.FileRepo.
func (f *fileRepo) FindByOssPath(ctx context.Context, bucket string, objectPath string) ([]*model.SysFile, error) {
	fileQuery := f.query.SysFile
	files, err := fileQuery.WithContext(ctx).
		Where(fileQuery.OssBucket.Eq(bucket)).
		Where(fileQuery.OssPath.Eq(objectPath)).
		Find()
	if err != nil {
		return nil, err
	}
	return files, nil
}

// Update implements system.FileRepo.
func (f *fileRepo) Update(ctx context.Context, file *model.SysFile) error {
	// 使用updates方法需要文件记录有ID
	if file.ID <= 0 {
		return f.query.SysFile.WithContext(ctx).Create(file)
	}
	_, err := f.query.SysFile.WithContext(ctx).
		Where(f.query.SysFile.ID.Eq(file.ID)).
		Updates(map[string]interface{}{
			"file_name":   file.FileName,
			"file_size":   file.FileSize,
			"mime_type":   file.MimeType,
			"biz_type":    file.BizType,
			"usage_type":  file.UsageType,
			"md5":         file.Md5,
			"oss_bucket":  file.OssBucket,
			"oss_path":    file.OssPath,
			"status":      file.Status,
			"create_time": file.CreateTime,
		})
	return err
}

// DeleteExpiredPendingFiles implements system.FileRepo.
func (f *fileRepo) DeleteExpiredPendingFiles(ctx context.Context, cutoffTime int64) (int64, error) {
	// 删除截止时间之前创建的、状态为"待上传"的文件记录
	fileQuery := f.query.SysFile
	result, err := fileQuery.WithContext(ctx).
		Where(fileQuery.Status.Eq(2)). // 2=待上传状态
		Where(fileQuery.CreateTime.Lt(cutoffTime)).
		Delete()
	if err != nil {
		return 0, err
	}
	return result.RowsAffected, nil
}

// FindDeletedFiles 查找已软删除且超过指定时间的文件记录
func (f *fileRepo) FindDeletedFiles(ctx context.Context, cutoffTime int64) ([]*model.SysFile, error) {
	fileQuery := f.query.SysFile
	// 获取原始DB访问
	db := fileQuery.WithContext(ctx).UnderlyingDB().Unscoped()

	var files []*model.SysFile
	err := db.Where("deleted_at > 0 AND deleted_at < ?", cutoffTime).Find(&files).Error

	if err != nil {
		return nil, err
	}
	return files, nil
}

// PermanentlyDelete 永久删除文件记录(硬删除)
func (f *fileRepo) PermanentlyDelete(ctx context.Context, id int64) error {
	fileQuery := f.query.SysFile
	// 使用ForceDelete或直接执行SQL语句来绕过软删除机制
	_, err := fileQuery.WithContext(ctx).
		Unscoped(). // 绕过软删除机制
		Where(fileQuery.ID.Eq(id)).
		Delete()

	return err
}
