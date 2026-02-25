package oss

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

var BizTypeMap = map[int32]string{
	1: "public",
	2: "private",
}

// OssProviderType OSS服务提供商类型
type OssProviderType int32

// OSS服务提供商类型常量
const (
	OssProviderUnspecified OssProviderType = iota
	OssProviderAliyun
	OssProviderTencent
	OssProviderAWS
	OssProviderMinio
	OssProviderHuawei
)

// String 返回提供商类型的字符串表示
func (p OssProviderType) String() string {
	switch p {
	case OssProviderAliyun:
		return "阿里云OSS"
	case OssProviderTencent:
		return "腾讯云COS"
	case OssProviderAWS:
		return "AWS S3"
	case OssProviderMinio:
		return "MinIO"
	case OssProviderHuawei:
		return "华为云OBS"
	default:
		return "未指定"
	}
}

// OssConfigInfo OSS配置信息
type OssConfigInfo struct {
	// OSS服务提供商
	ProviderType OssProviderType
	// 是否是默认服务
	IsDefault bool
	// 服务状态(1=启用/0=禁用)
	Status int32
	// 配置名称
	Name string
	// 公开桶名称
	PublicBucket string
	// 私有桶名称
	PrivateBucket string
	// 内部桶名称
	InternalBucket string
	// 访问域名(endpoint)
	Endpoint string
	// 区域(region)
	Region string
}

func GenFileName(userId int64, fileName string) (string, error) {
	// 使用SHA-256加密
	userHash := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d", userId))))[:16]

	// 处理文件名以确保唯一性
	ext := filepath.Ext(fileName)
	baseName := strings.TrimSuffix(fileName, ext)
	timestamp := time.Now().UnixNano() / 1e6 // 毫秒级时间戳
	fileHash := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%s%d", fileName, timestamp))))[:8]
	uniqueFileName := fmt.Sprintf("%s_%d_%s%s", baseName, timestamp, fileHash, ext)

	return fmt.Sprintf("%s/%s", userHash, uniqueFileName), nil
}
