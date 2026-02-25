package oss

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"spark/internal/conf"
	clog "spark/pkg/clog"
)

var (
	ErrOSSNotConfigured = errors.New("file upload service is not configured")
	ErrOSSUnavailable   = errors.New("file upload service is temporarily unavailable")
)

type OssProvider interface {
	OssService
	GetStatus() OssStatus
	Update(context.Context, *conf.OSS) error
}

type OssStatus int

const (
	OssStatusUninitialized OssStatus = iota
	OssStatusReady
	OssStatusFailed
)

type ossManager struct {
	logger    clog.Logger
	mu        sync.Mutex
	ossConfig atomic.Value
	oss       atomic.Pointer[ossServiceRef]
	status    atomic.Int32
}

type ossServiceRef struct {
	service OssService
}

func NewOssProvider(logger clog.Logger) OssProvider {
	m := &ossManager{
		logger: logger,
	}
	m.oss.Store(&ossServiceRef{service: &noOpOssService{logger: logger, reason: "not configured"}})
	m.status.Store(int32(OssStatusUninitialized))
	return m
}

func (m *ossManager) GenUploadUrl(ctx context.Context, fileName string, fileSize int64, mimeType string, bizType int32, forceSts bool) (*GenUploadUrlReply, error) {
	return m.currentService().GenUploadUrl(ctx, fileName, fileSize, mimeType, bizType, forceSts)
}

func (m *ossManager) DeleteObject(ctx context.Context, bucket string, objectPath string) error {
	return m.currentService().DeleteObject(ctx, bucket, objectPath)
}

func (m *ossManager) GetStatus() OssStatus {
	return OssStatus(m.status.Load())
}

func (m *ossManager) Update(ctx context.Context, cfg *conf.OSS) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	oldService := m.currentService()

	if cfg == nil || cfg.Service == "" {
		m.oss.Store(&ossServiceRef{service: &noOpOssService{logger: m.logger, reason: "not configured"}})
		m.ossConfig.Store((*conf.OSS)(nil))
		m.status.Store(int32(OssStatusReady))
		m.logger.Info(ctx, "oss: not configured")
		m.closeServiceLater(oldService)
		return nil
	}

	if err := m.validateConfig(cfg); err != nil {
		m.status.Store(int32(OssStatusFailed))
		m.logger.Error(ctx, "oss: config invalid", clog.Err(err))
		return err
	}

	ossService, err := m.createService(ctx, cfg)
	if err != nil {
		m.status.Store(int32(OssStatusFailed))
		m.logger.Error(ctx, "oss: create failed", clog.Err(err))
		return err
	}

	if ossService == nil {
		m.status.Store(int32(OssStatusFailed))
		m.logger.Error(ctx, "oss: create returned nil")
		return errors.New("create returned nil")
	}

	m.oss.Store(&ossServiceRef{service: ossService})
	m.ossConfig.Store(cfg)
	m.status.Store(int32(OssStatusReady))
	m.logger.Info(ctx, "oss: updated", clog.String("service", cfg.Service))
	m.closeServiceLater(oldService)
	return nil
}

func (m *ossManager) currentService() OssService {
	ref := m.oss.Load()
	if ref != nil && ref.service != nil {
		return ref.service
	}
	return &noOpOssService{logger: m.logger, reason: "not configured"}
}

func (m *ossManager) closeServiceLater(oldService OssService) {
	closer, ok := oldService.(interface{ Close() error })
	if !ok {
		return
	}

	go func() {
		time.Sleep(30 * time.Second)
		if err := closer.Close(); err != nil {
			m.logger.Warn(context.Background(), "oss: close previous service failed", clog.Err(err))
		}
	}()
}

func (m *ossManager) validateConfig(cfg *conf.OSS) error {
	switch cfg.Service {
	case "aliyun":
		aliyun := cfg.GetAliyun()
		if aliyun == nil || aliyun.AccessKeyId == "" || aliyun.AccessKeySecret == "" {
			return errors.New("aliyun access_key_id and access_key_secret are required")
		}
	case "tencent":
		tencent := cfg.GetTencent()
		if tencent == nil || tencent.SecretId == "" || tencent.SecretKey == "" {
			return errors.New("tencent secret_id and secret_key are required")
		}
	case "s3":
		s3 := cfg.GetS3()
		if s3 == nil || s3.Bucket == "" || s3.Region == "" {
			return errors.New("s3 bucket and region are required")
		}
	case "local":
		local := cfg.GetLocal()
		if local == nil || local.Path == "" {
			return errors.New("local path is required")
		}
	case "memory":
	default:
		return fmt.Errorf("unsupported service: %s", cfg.Service)
	}
	return nil
}

func (m *ossManager) createService(ctx context.Context, cfg *conf.OSS) (OssService, error) {
	switch cfg.Service {
	case "aliyun":
		return m.createAliyunService(ctx, cfg)
	case "tencent":
		return m.createTencentService(ctx, cfg)
	case "local", "memory", "s3":
		return m.createBlobStorageService(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported service: %s", cfg.Service)
	}
}

func (m *ossManager) createAliyunService(ctx context.Context, cfg *conf.OSS) (OssService, error) {
	aliyun := cfg.GetAliyun()
	s3Cfg := &s3Config{
		Region:          aliyun.Region,
		Bucket:          getAliyunBucket(aliyun.Buckets),
		Endpoint:        getAliyunS3Endpoint(aliyun.Region),
		AccessKeyId:     aliyun.AccessKeyId,
		SecretAccessKey: aliyun.AccessKeySecret,
	}

	storage, err := newBlobStorageWithS3(ctx, s3Cfg, m.logger)
	if err != nil {
		return nil, err
	}

	if storage == nil {
		return nil, errors.New("failed to create aliyun service")
	}

	return &BlobStorageAdapter{storage: storage}, nil
}

func (m *ossManager) createTencentService(ctx context.Context, cfg *conf.OSS) (OssService, error) {
	tencent := cfg.GetTencent()
	s3Cfg := &s3Config{
		Region:          tencent.Region,
		Bucket:          tencent.Bucket,
		Endpoint:        getTencentS3Endpoint(tencent.Region),
		AccessKeyId:     tencent.SecretId,
		SecretAccessKey: tencent.SecretKey,
	}

	storage, err := newBlobStorageWithS3(ctx, s3Cfg, m.logger)
	if err != nil {
		return nil, err
	}

	if storage == nil {
		return nil, errors.New("failed to create tencent service")
	}

	return &BlobStorageAdapter{storage: storage}, nil
}

func (m *ossManager) createBlobStorageService(ctx context.Context, cfg *conf.OSS) (OssService, error) {
	storage, err := NewBlobStorage(ctx, cfg, m.logger)
	if err != nil {
		return nil, err
	}

	if storage == nil {
		return nil, errors.New("failed to create blob storage")
	}

	return &BlobStorageAdapter{storage: storage}, nil
}

type noOpOssService struct {
	logger clog.Logger
	reason string
}

func (n *noOpOssService) GenUploadUrl(ctx context.Context, fileName string, fileSize int64, mimeType string, bizType int32, forceSts bool) (*GenUploadUrlReply, error) {
	n.logger.Debug(ctx, "oss: called but unavailable", clog.String("reason", n.reason))
	return nil, ErrOSSUnavailable
}

func (n *noOpOssService) DeleteObject(ctx context.Context, bucket string, objectPath string) error {
	n.logger.Debug(ctx, "oss: called but unavailable", clog.String("reason", n.reason))
	return ErrOSSUnavailable
}

type s3Config struct {
	Region          string
	Bucket          string
	Endpoint        string
	AccessKeyId     string
	SecretAccessKey string
	SessionToken    string
	ForcePathStyle  bool
}

func getAliyunBucket(buckets []*conf.OSS_Aliyun_Bucket) string {
	for _, b := range buckets {
		if b != nil {
			return b.Name
		}
	}
	return ""
}

func getAliyunS3Endpoint(region string) string {
	if region == "" {
		return "https://s3.cn-hangzhou.aliyuncs.com"
	}
	return fmt.Sprintf("https://s3.%s.aliyuncs.com", region)
}

func getTencentS3Endpoint(region string) string {
	if region == "" {
		return "https://cos.ap-guangzhou.myqcloud.com"
	}
	return fmt.Sprintf("https://cos.%s.myqcloud.com", region)
}
