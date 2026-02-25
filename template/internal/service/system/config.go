package system

import (
	"context"

	pb "spark/api/system/v1"
)

type ConfigService struct {
	pb.UnimplementedConfigServer
}

func NewConfigService() *ConfigService {
	return &ConfigService{}
}

func (s *ConfigService) CreateConfig(ctx context.Context, req *pb.CreateConfigRequest) (*pb.CreateConfigReply, error) {
	return &pb.CreateConfigReply{}, nil
}
func (s *ConfigService) UpdateConfig(ctx context.Context, req *pb.UpdateConfigRequest) (*pb.UpdateConfigReply, error) {
	return &pb.UpdateConfigReply{}, nil
}
func (s *ConfigService) DeleteConfig(ctx context.Context, req *pb.DeleteConfigRequest) (*pb.DeleteConfigReply, error) {
	return &pb.DeleteConfigReply{}, nil
}
func (s *ConfigService) GetConfig(ctx context.Context, req *pb.GetConfigRequest) (*pb.GetConfigReply, error) {
	return &pb.GetConfigReply{}, nil
}
func (s *ConfigService) ListConfig(ctx context.Context, req *pb.ListConfigRequest) (*pb.ListConfigReply, error) {
	return &pb.ListConfigReply{}, nil
}
