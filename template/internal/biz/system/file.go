package system

import (
	"context"
	"time"

	"spark/internal/data/dal/model"
	"spark/pkg/utils/oss"
)

type FileRepo interface {
	Create(ctx context.Context, file *model.SysFile) error
	Delete(ctx context.Context, id int64) error
	Get(ctx context.Context, id int64) (*model.SysFile, error)
	List(ctx context.Context, current int32, pageSize int32) ([]*model.SysFile, int64, error)
	// 根据OSS路径和存储桶名称查找文件
	FindByOssPath(ctx context.Context, bucket string, objectPath string) ([]*model.SysFile, error)
	// 更新文件记录
	Update(ctx context.Context, file *model.SysFile) error
	// 删除过期的待上传状态的文件记录
	DeleteExpiredPendingFiles(ctx context.Context, cutoffTime int64) (int64, error)
	// 查找已软删除且超过指定时间的文件记录
	FindDeletedFiles(ctx context.Context, cutoffTime int64) ([]*model.SysFile, error)
	// 永久删除文件记录(硬删除)
	PermanentlyDelete(ctx context.Context, id int64) error
}

type FileUsecase struct {
	repo       FileRepo
	ossService oss.OssService
}

func NewSystemFileUsecase(repo FileRepo, ossService oss.OssService) *FileUsecase {
	return &FileUsecase{
		repo:       repo,
		ossService: ossService,
	}
}

func (uc *FileUsecase) Get(ctx context.Context, id int64) (*model.SysFile, error) {
	return uc.repo.Get(ctx, id)
}

func (uc *FileUsecase) List(ctx context.Context, current int32, pageSize int32) ([]*model.SysFile, int64, error) {
	return uc.repo.List(ctx, current, pageSize)
}

func (uc *FileUsecase) Create(ctx context.Context, file *model.SysFile) error {
	return uc.repo.Create(ctx, file)
}

func (uc *FileUsecase) Update(ctx context.Context, file *model.SysFile) error {
	return uc.repo.Update(ctx, file)
}

func (uc *FileUsecase) Delete(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *FileUsecase) FindByOssPath(ctx context.Context, bucket string, objectPath string) ([]*model.SysFile, error) {
	return uc.repo.FindByOssPath(ctx, bucket, objectPath)
}

// FindDeletedFiles 查找已删除且超过指定时间的文件记录
func (uc *FileUsecase) FindDeletedFiles(ctx context.Context, cutoffTime int64) ([]*model.SysFile, error) {
	return uc.repo.FindDeletedFiles(ctx, cutoffTime)
}

// DeleteOssObject 从OSS中删除文件对象
func (uc *FileUsecase) DeleteOssObject(ctx context.Context, bucket, objectPath string) error {
	return uc.ossService.DeleteObject(ctx, bucket, objectPath)
}

// PermanentlyDelete 永久删除文件记录(硬删除)
func (uc *FileUsecase) PermanentlyDelete(ctx context.Context, id int64) error {
	return uc.repo.PermanentlyDelete(ctx, id)
}

// CleanupPendingFiles 清理过期的待上传文件记录
// maxAge 指定最大过期时间（毫秒），例如 24小时 = 24*60*60*1000
func (uc *FileUsecase) CleanupPendingFiles(ctx context.Context, maxAge int64) (int64, error) {
	// 使用当前时间减去最大过期时间，得到截止时间
	cutoffTime := time.Now().UnixMilli() - maxAge

	// 删除截止时间之前创建的、状态为"待上传"的文件记录
	// 这里需要实现FileRepo的新方法
	count, err := uc.repo.DeleteExpiredPendingFiles(ctx, cutoffTime)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (uc *FileUsecase) GenUploadUrl(ctx context.Context,
	bizType int32,
	fileName string,
	mimeType string,
	fileSize int64,
	forceSts bool) (*oss.GenUploadUrlReply, error) {
	return uc.ossService.GenUploadUrl(ctx, fileName, fileSize, mimeType, bizType, forceSts)
}
