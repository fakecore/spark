package oss

import (
	"context"

	"spark/internal/conf"
)

var _ OssService = &TencentOssService{}

func NewTencentOssService(conf *conf.OSS_Tencent) (*TencentOssService, error) {
	return &TencentOssService{conf: conf}, nil
}

type TencentOssService struct {
	conf *conf.OSS_Tencent
}

// DeleteObject implements OssService.
func (t *TencentOssService) DeleteObject(ctx context.Context, bucket string, objectPath string) error {
	panic("unimplemented")
}

// GenUploadUrl implements OssService.
func (t *TencentOssService) GenUploadUrl(ctx context.Context, fileName string, fileSize int64, mimeType string, bizType int32, forceSts bool) (reply *GenUploadUrlReply, err error) {
	panic("unimplemented")
}
