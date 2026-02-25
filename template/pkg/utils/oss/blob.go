package oss

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"time"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/memblob"
	_ "gocloud.dev/blob/s3blob"

	"spark/internal/conf"
	clog "spark/pkg/clog"
)

// BlobStorageAdapter wraps BlobStorage to implement OssService interface
type BlobStorageAdapter struct {
	storage *BlobStorage
}

func (a *BlobStorageAdapter) Close() error {
	if a == nil || a.storage == nil {
		return nil
	}
	return a.storage.Close()
}

func (a *BlobStorageAdapter) GenUploadUrl(ctx context.Context, fileName string, fileSize int64, mimeType string, bizType int32, forceSts bool) (*GenUploadUrlReply, error) {
	key := a.storage.GenerateKey(bizType, fileName)
	return &GenUploadUrlReply{
		Url: &UrlInfo{
			Url:        key,
			Method:     "POST",
			Expiration: time.Now().Add(30 * time.Minute).Unix(),
		},
	}, nil
}

func (a *BlobStorageAdapter) DeleteObject(ctx context.Context, bucket string, objectPath string) error {
	return a.storage.Delete(ctx, objectPath)
}

type BlobStorage struct {
	bucket *blob.Bucket
	logger clog.Logger
}

func NewBlobStorage(ctx context.Context, cfg *conf.OSS, logger clog.Logger) (*BlobStorage, error) {
	if cfg == nil {
		return nil, nil
	}

	var bucket *blob.Bucket
	var err error

	switch cfg.Service {
	case "local":
		if cfg.Local == nil || cfg.Local.Path == "" {
			return nil, nil
		}
		bucket, err = blob.OpenBucket(ctx, "file://"+cfg.Local.Path)
	case "memory":
		bucket, err = blob.OpenBucket(ctx, "mem://")
	case "s3":
		s3Cfg := cfg.S3
		if s3Cfg == nil || s3Cfg.Bucket == "" || s3Cfg.Region == "" {
			return nil, nil
		}
		url := fmt.Sprintf("s3://%s?region=%s", s3Cfg.Bucket, s3Cfg.Region)
		if s3Cfg.Endpoint != "" {
			url += "&endpoint=" + s3Cfg.Endpoint
		}
		bucket, err = blob.OpenBucket(ctx, url)
	default:
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open bucket: %w", err)
	}

	return &BlobStorage{
		bucket: bucket,
		logger: logger,
	}, nil
}

func (s *BlobStorage) Close() error {
	return s.bucket.Close()
}

func (s *BlobStorage) WriteAll(ctx context.Context, key string, data []byte, opts *blob.WriterOptions) error {
	return s.bucket.WriteAll(ctx, key, data, opts)
}

func (s *BlobStorage) ReadAll(ctx context.Context, key string) ([]byte, error) {
	return s.bucket.ReadAll(ctx, key)
}

func (s *BlobStorage) Delete(ctx context.Context, key string) error {
	return s.bucket.Delete(ctx, key)
}

func (s *BlobStorage) Exists(ctx context.Context, key string) (bool, error) {
	return s.bucket.Exists(ctx, key)
}

func (s *BlobStorage) List(ctx context.Context, opts *blob.ListOptions) *blob.ListIterator {
	return s.bucket.List(opts)
}

func (s *BlobStorage) Attributes(ctx context.Context, key string) (*blob.Attributes, error) {
	return s.bucket.Attributes(ctx, key)
}

func (s *BlobStorage) NewWriter(ctx context.Context, key string, opts *blob.WriterOptions) (*blob.Writer, error) {
	return s.bucket.NewWriter(ctx, key, opts)
}

func (s *BlobStorage) NewReader(ctx context.Context, key string, opts *blob.ReaderOptions) (io.ReadCloser, error) {
	return s.bucket.NewReader(ctx, key, opts)
}

func (s *BlobStorage) Copy(ctx context.Context, destKey, srcKey string) error {
	data, err := s.bucket.ReadAll(ctx, srcKey)
	if err != nil {
		return err
	}
	return s.bucket.WriteAll(ctx, destKey, data, nil)
}

type UploadedFile struct {
	Key      string
	Size     int64
	Metadata map[string]string
}

func (s *BlobStorage) SaveUploadedFile(ctx context.Context, key string, file *multipart.FileHeader) (*UploadedFile, error) {
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer src.Close()

	writer, err := s.bucket.NewWriter(ctx, key, nil)
	if err != nil {
		return nil, fmt.Errorf("create writer: %w", err)
	}

	size, err := io.Copy(writer, src)
	if err != nil {
		writer.Close()
		return nil, fmt.Errorf("copy file: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close writer: %w", err)
	}

	return &UploadedFile{
		Key:  key,
		Size: size,
	}, nil
}

func (s *BlobStorage) GenerateKey(bizType int32, fileName string) string {
	timestamp := time.Now().UnixNano()
	ext := getExt(fileName)
	return fmt.Sprintf("%s/%d_%d%s", getBizTypeDir(bizType), bizType, timestamp, ext)
}

func getExt(filename string) string {
	if i := len(filename) - 1; i >= 0 {
		if filename[i] == '.' {
			return ""
		}
		for j := i; j >= 0; j-- {
			if filename[j] == '.' {
				return filename[j:]
			}
		}
	}
	return ""
}

func getBizTypeDir(bizType int32) string {
	dirs := map[int32]string{
		1: "avatar",
		2: "image",
		3: "video",
		4: "audio",
		5: "document",
		6: "other",
	}
	if dir, ok := dirs[bizType]; ok {
		return dir
	}
	return "other"
}

func (s *BlobStorage) DeleteExpiredFiles(ctx context.Context, maxAge time.Duration) (int64, error) {
	cutoff := time.Now().Add(-maxAge)
	var deleted int64

	iter := s.bucket.List(&blob.ListOptions{Prefix: ""})
	for {
		obj, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return deleted, err
		}
		if obj.ModTime.Before(cutoff) {
			if err := s.bucket.Delete(ctx, obj.Key); err != nil {
				s.logger.Error(ctx, "delete expired file failed", clog.String("key", obj.Key), clog.Err(err))
				continue
			}
			deleted++
		}
	}

	return deleted, nil
}

func newBlobStorageWithS3(ctx context.Context, cfg *s3Config, logger clog.Logger) (*BlobStorage, error) {
	if cfg == nil || cfg.Bucket == "" || cfg.Region == "" {
		return nil, nil
	}

	url := fmt.Sprintf("s3://%s?region=%s", cfg.Bucket, cfg.Region)
	if cfg.Endpoint != "" {
		url += "&endpoint=" + cfg.Endpoint
	}

	bucket, err := blob.OpenBucket(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to open S3 bucket: %w", err)
	}

	return &BlobStorage{
		bucket: bucket,
		logger: logger,
	}, nil
}
