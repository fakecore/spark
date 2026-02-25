package system

import (
	"context"

	pb "spark/api/system/v1"
	"spark/internal/biz/system"
	"spark/internal/common/constants"
	"spark/internal/conf"
	"spark/internal/data/dal/model"
)

type FileService struct {
	pb.UnimplementedFileServer
	bc     *conf.Bootstrap
	fileUc *system.FileUsecase
}

var _ pb.FileServer = (*FileService)(nil)

func NewFileService(bc *conf.Bootstrap, fileUc *system.FileUsecase) *FileService {
	return &FileService{
		bc:     bc,
		fileUc: fileUc,
	}
}

func (s *FileService) GenFileUploadUrl(ctx context.Context, req *pb.GenFileUploadUrlRequest) (*pb.GenFileUploadUrlReply, error) {
	// 1. 生成上传URL
	result, err := s.fileUc.GenUploadUrl(ctx, req.BizType, req.FileName, req.MimeType, req.FileSize, req.ForceSts)
	if err != nil {
		return nil, err
	}

	// 2. 创建"待确认"状态的文件记录
	// 根据返回的信息获取相关数据
	ossBucket := ""
	ossPath := ""

	if result.Sts != nil {
		ossBucket = result.Sts.BucketName
		ossPath = result.Sts.ObjectName
	}

	fileStatus := constants.FileStatus_Uploaded
	if s.bc.Oss.Callback.Enable {
		fileStatus = constants.FileStatus_Pending
	}
	// 创建文件记录
	fileRecord := &model.SysFile{
		FileName:  req.FileName,
		FileSize:  req.FileSize,
		MimeType:  &req.MimeType,
		BizType:   req.BizType,
		Status:    fileStatus, // 2=待上传状态
		OssBucket: ossBucket,
		OssPath:   ossPath,
	}

	// 如果提供了业务用途，设置它
	if req.UsageType != "" {
		fileRecord.UsageType = &req.UsageType
	}

	// 将文件记录保存到数据库
	err = s.fileUc.Create(ctx, fileRecord)
	if err != nil {
		return nil, err
	}

	// 3. 构建返回结果
	reply := &pb.GenFileUploadUrlReply{}
	if result.Sts != nil {
		reply.Sts = &pb.GenFileUploadUrlReply_Sts{
			AccessKeyId:     result.Sts.AccessKeyId,
			AccessKeySecret: result.Sts.AccessKeySecret,
			SecurityToken:   result.Sts.SecurityToken,
			Expiration:      result.Sts.Expiration,
			ObjectName:      result.Sts.ObjectName,
			Endpoint:        result.Sts.Endpoint,
			Region:          result.Sts.Region,
			BucketName:      result.Sts.BucketName,
		}
	}
	if result.Url != nil {
		reply.Url = &pb.GenFileUploadUrlReply_Url{
			Url:           result.Url.Url,
			Method:        result.Url.Method,
			Expiration:    result.Url.Expiration,
			SignedHeaders: result.Url.SignedHeaders,
		}
	}
	return reply, nil
}

// FileUploadCallback 处理文件上传完成回调
func (s *FileService) FileUploadCallback(ctx context.Context, req *pb.FileUploadCallbackRequest) (*pb.FileUploadCallbackReply, error) {
	// 1. 根据oss_bucket和oss_path查找待确认的文件记录
	fileRecords, err := s.fileUc.FindByOssPath(ctx, req.Bucket, req.Object)
	if err != nil {
		return &pb.FileUploadCallbackReply{
			Code:    500,
			Message: "查询文件记录失败:" + err.Error(),
		}, nil
	}

	if len(fileRecords) == 0 {
		// 文件记录不存在，这是不正常情况
		// 正常流程应该是先创建记录再上传，不可能先触发回调
		// 记录错误并返回失败
		return &pb.FileUploadCallbackReply{
			Code:    404,
			Message: "未找到对应的文件记录，可能存在异常",
		}, nil
	}

	// 更新现有记录为"已确认"状态
	fileRecord := fileRecords[0]
	fileRecord.Status = 0          // 0=正常状态
	fileRecord.FileSize = req.Size // 更新实际大小

	// 如果有MD5，设置它
	if etag := req.Etag; etag != "" {
		fileRecord.Md5 = &etag
	}

	err = s.fileUc.Update(ctx, fileRecord)
	if err != nil {
		return &pb.FileUploadCallbackReply{
			Code:    500,
			Message: "更新文件记录失败:" + err.Error(),
		}, nil
	}

	// 返回成功响应
	return &pb.FileUploadCallbackReply{
		Code:    0,
		Message: "success",
		FileUrl: generateFileUrl(req.Bucket, req.Object),
	}, nil
}

// 生成文件访问URL
func generateFileUrl(bucket, objectKey string) *string {
	// 根据存储桶和对象键生成文件访问URL
	// 实际实现应该基于OSS配置生成正确的访问URL
	fileUrl := "https://" + bucket + ".oss-cn-hangzhou.aliyuncs.com/" + objectKey
	return &fileUrl
}

// GetFileUsecase 获取FileUsecase实例，用于后台作业
func (s *FileService) GetFileUsecase() *system.FileUsecase {
	return s.fileUc
}

// 从OSS路径中提取文件名
func getFileNameFromOssPath(ossPath string) string {
	// 简单实现，实际应考虑路径复杂性
	// 例如从路径 "user/file_20230604_12345.jpg" 提取 "file.jpg"
	// 这里省略实现细节
	return ossPath
}

// 从存储桶名称确定业务类型
func getBizTypeFromBucket(bucket string) int32 {
	// 根据存储桶名称确定业务类型
	// 例如 "public-bucket" 对应 BizType=1
	// 这里简单处理，实际应根据配置或规则确定
	if bucket == "public-bucket" {
		return 1 // 公开桶
	}
	return 2 // 默认为私有桶
}
