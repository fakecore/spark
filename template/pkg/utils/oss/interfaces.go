package oss

import (
	"context"
	"time"
)

// OssConfig OSS配置接口
type OssConfig interface {
	// 获取提供商类型
	GetProviderType() OssProviderType
	// 验证配置是否有效
	Validate(ctx context.Context) error
	// 转换为配置信息
	ToConfigInfo() *OssConfigInfo
}

type StsInfo struct {
	AccessKeyId     string
	AccessKeySecret string
	SecurityToken   string
	Expiration      int64
	BucketName      string
	Region          string
	ObjectName      string
	Endpoint        string
}

type UrlInfo struct {
	Url           string
	Method        string
	Expiration    int64
	SignedHeaders map[string]string
}

type CallbackInfo struct {
	Headers map[string]string
}

type GenUploadUrlReply struct {
	Sts      *StsInfo      `json:"sts,omitempty"`
	Url      *UrlInfo      `json:"url,omitempty"`
	Callback *CallbackInfo `json:"callback,omitempty"`
}

// OssService 统一的OSS服务接口
type OssService interface {
	// 生成上传URL
	GenUploadUrl(ctx context.Context, fileName string, fileSize int64, mimeType string, bizType int32, forceSts bool) (reply *GenUploadUrlReply, err error)
	// 删除对象
	DeleteObject(ctx context.Context, bucket string, objectPath string) error
}

// OssEvent 事件类型
type OssEvent struct {
	Type      string
	Provider  OssProviderType
	Message   string
	Timestamp time.Time
}
