package oss

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"spark/internal/conf"
	clog "spark/pkg/clog"
	"spark/pkg/viewer"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	sts20150401 "github.com/alibabacloud-go/sts-20150401/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
)

type AliyunOssService struct {
	ossConfig    *conf.OSS
	partSize     int64
	client       *oss.Client
	stsClient    *sts20150401.Client
	lastValidate time.Time
	logger       clog.Logger
	buckets      map[int32]string
}

// NewAliyunOssService 创建阿里云OSS服务
func NewAliyunOssService(ossConfig *conf.OSS, logger clog.Logger) (*AliyunOssService, error) {
	config := ossConfig.Aliyun
	partSize := ossConfig.FilePart.PartSize
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if config.AccessKeyId == "" || config.AccessKeySecret == "" || config.Region == "" {
		return nil, fmt.Errorf("config is invalid, config should not be empty")
	}
	if partSize <= 0 {
		partSize = 1024 * 1024 * 1024 // 默认1GB
	}
	// 创建OSS客户端
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider(config.AccessKeyId, config.AccessKeySecret)).
		WithRegion(config.Region)

	client := oss.NewClient(cfg)

	aliyunStsConfig := ossConfig.Aliyun.Sts
	// 创建权限策略客户端。
	stsConfig := &openapi.Config{
		// 必填，步骤1.1获取到的 AccessKey ID。
		AccessKeyId: tea.String(aliyunStsConfig.AccessKeyId),
		// 必填，步骤1.1获取到的 AccessKey Secret。
		AccessKeySecret: tea.String(aliyunStsConfig.AccessKeySecret),
		Endpoint:        tea.String(aliyunStsConfig.Endpoint),
		RegionId:        tea.String(aliyunStsConfig.Region),
	}
	stsClient, err := sts20150401.NewClient(stsConfig)
	if err != nil {
		return nil, fmt.Errorf("创建STS客户端失败: %w", err)
	}

	buckets := make(map[int32]string)
	for _, bucket := range config.Buckets {
		if bucket.Name == "" {
			return nil, fmt.Errorf("bucket name is empty")
		}
		buckets[bucket.BizType] = bucket.Name
	}

	return &AliyunOssService{
		ossConfig:    ossConfig,
		client:       client,
		stsClient:    stsClient,
		lastValidate: time.Time{},
		partSize:     partSize,
		logger:       logger,
		buckets:      buckets,
	}, nil
}

// GenUploadUrl 生成上传URL
func (s *AliyunOssService) GenUploadUrl(ctx context.Context, fileName string, fileSize int64, mimeType string, bizType int32, forceSts bool) (reply *GenUploadUrlReply, err error) {
	bucketName, ok := s.buckets[bizType]
	if !ok {
		return nil, fmt.Errorf("bizType %s 不存在", BizTypeMap[bizType])
	}
	userView := viewer.MustGetUserViewFromContext(ctx)

	filename, err := GenFileName(userView.GetUser().ID, fileName)
	if err != nil {
		return nil, fmt.Errorf("生成前缀失败: %w", err)
	}

	if fileSize > s.partSize || forceSts {
		return s.preocessSts(ctx, bucketName, filename, mimeType, fileSize)
	} else {
		return s.preocessSingle(ctx, bucketName, filename, mimeType)
	}
}

func (s *AliyunOssService) GetCallback(ctx context.Context) (map[string]string, error) {
	if !s.ossConfig.Callback.Enable {
		return map[string]string{}, nil
	}
	callback := s.ossConfig.Callback
	// 定义回调参数
	callbackMap := map[string]string{
		"callbackUrl":      callback.Url,      // 设置回调服务器的URL，例如https://example.com:23450。
		"callbackBody":     callback.Body,     // 设置回调请求体。
		"callbackBodyType": callback.BodyType, //设置回调请求体类型。
	}

	// 将回调参数转换为JSON并进行Base64编码，以便将其作为回调参数传递
	callbackStr, err := json.Marshal(callbackMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal callback map: %w", err)
	}
	callbackBase64 := base64.StdEncoding.EncodeToString(callbackStr)
	return map[string]string{
		"x-oss-callback": callbackBase64,
	}, nil
}

func (s *AliyunOssService) preocessSts(ctx context.Context, bucketName, objectName, mimeType string, fileSize int64) (*GenUploadUrlReply, error) {
	s.logger.Info(ctx, "sts mode")
	userView := viewer.MustGetUserViewFromContext(ctx)
	roleArn := s.ossConfig.Aliyun.Sts.RoleArn

	// 确保DurationSeconds在有效范围内 (15分钟到1小时)
	duration := s.ossConfig.Aliyun.Sts.Expiration
	if duration < 900 { // 小于15分钟
		duration = 900
	} else if duration > 3600 { // 大于1小时
		duration = 3600
	}

	// 使用RAM用户的AccessKey ID和AccessKey Secret向STS申请临时访问凭证。
	request := &sts20150401.AssumeRoleRequest{
		// 指定STS临时访问凭证过期时间，范围：15分钟(900秒)到1小时(3600秒)
		DurationSeconds: tea.Int64(duration),
		// 从环境变量中获取步骤1.3生成的RAM角色的RamRoleArn。
		RoleArn: tea.String(roleArn),
		// 指定自定义角色会话名称，这里使用和第一段代码一致的 examplename
		RoleSessionName: tea.String(fmt.Sprintf("oss-sts-%d-%d", userView.GetUser().ID, time.Now().UnixNano())),
		Policy:          tea.String(s.generatePolicy(ctx, bucketName, objectName, fileSize)),
	}

	response, err := s.stsClient.AssumeRoleWithOptions(request, &util.RuntimeOptions{})
	if err != nil {
		fmt.Printf("Failed to assume role: %v\n", err)
		return nil, fmt.Errorf("生成上传URL失败: %w", err)
	}

	credentials := response.Body.Credentials
	expiration, err := time.Parse(time.RFC3339, tea.StringValue(credentials.Expiration))
	if err != nil {
		return nil, fmt.Errorf("解析过期时间失败: %w", err)
	}
	callbackParams, err := s.GetCallback(ctx)
	return &GenUploadUrlReply{
		Sts: &StsInfo{
			AccessKeyId:     tea.StringValue(credentials.AccessKeyId),
			AccessKeySecret: tea.StringValue(credentials.AccessKeySecret),
			SecurityToken:   tea.StringValue(credentials.SecurityToken),
			Expiration:      expiration.Unix(),
			BucketName:      bucketName,
			Region:          s.ossConfig.Aliyun.Region,
			ObjectName:      objectName,
			Endpoint:        s.ossConfig.Aliyun.Endpoint,
		},
		Callback: &CallbackInfo{
			Headers: callbackParams,
		},
	}, nil
}

func (s *AliyunOssService) preocessSingle(ctx context.Context, bucketName, objectName, mimeType string) (*GenUploadUrlReply, error) {
	s.logger.Info(ctx, "url mode")
	//您可以在生成预签名URL时，通过指定Header参数定义上传策略。例如，您可以设置文件存储类型x-oss-storage-class和文件类型Content-Type（如下代码示例）。
	// 重要生成预签名URL时指定了Header参数，实际使用预签名URL上传文件时，也必须传递相同的Header。否则，OSS将因为签名校验失败返回403错误。
	callbackParams, err := s.GetCallback(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取回调失败: %w", err)
	}
	putObjectRequest := &oss.PutObjectRequest{
		Bucket:      oss.Ptr(bucketName),
		ContentType: oss.Ptr(mimeType),
		Key:         oss.Ptr(objectName),
	}
	if callbackParams != nil && callbackParams["x-oss-callback"] != "" {
		putObjectRequest.Callback = oss.Ptr(callbackParams["x-oss-callback"])
	}

	presignResult, err := s.client.Presign(ctx, putObjectRequest, oss.PresignExpires(30*time.Minute))
	if err != nil {
		return nil, fmt.Errorf("生成上传URL失败: %w", err)
	}
	return &GenUploadUrlReply{
		Url: &UrlInfo{
			Url:           presignResult.URL,
			Method:        presignResult.Method,
			Expiration:    presignResult.Expiration.Unix(),
			SignedHeaders: presignResult.SignedHeaders,
		},
		Callback: &CallbackInfo{
			Headers: callbackParams,
		},
	}, nil
}

func (s *AliyunOssService) generatePolicy(ctx context.Context, bucketName string, objectName string, maxSizeMB int64) string {
	// 构建资源ARN
	targetRes := fmt.Sprintf("acs:oss:*:*:%s/%s", bucketName, objectName)

	// 构建RAM权限策略
	type Statement struct {
		Action   []string `json:"Action"`
		Effect   string   `json:"Effect"`
		Resource []string `json:"Resource"`
		Version  string   `json:"Version"`
	}

	type Policy struct {
		Statement []Statement `json:"Statement"`
	}

	policy := Policy{
		Statement: []Statement{
			{
				Action:   []string{"*"},
				Effect:   "Allow",
				Resource: []string{targetRes},
				Version:  "1",
			},
		},
	}

	// JSON序列化
	policyJSON, _ := json.Marshal(policy)
	s.logger.Info(ctx, "policy", clog.String("policy", string(policyJSON)))

	return string(policyJSON)
}

// DeleteObject 从阿里云OSS中删除对象
func (s *AliyunOssService) DeleteObject(ctx context.Context, bucket, objectPath string) error {
	// 验证参数
	if bucket == "" || objectPath == "" {
		return fmt.Errorf("bucket or objectPath is empty")
	}

	// 创建删除请求
	deleteReq := &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(bucket),
		Key:    oss.Ptr(objectPath),
	}

	// 执行删除操作
	_, err := s.client.DeleteObject(ctx, deleteReq)
	if err != nil {
		return fmt.Errorf("删除OSS对象失败: %w", err)
	}

	return nil
}
