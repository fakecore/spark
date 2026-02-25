package system

import (
	"context"

	pb "spark/api/system/v1"
)

type UserOnlineService struct {
	pb.UnimplementedUserOnlineServer
}

func NewUserOnlineService() *UserOnlineService {
	return &UserOnlineService{}
}

func (s *UserOnlineService) ListUserOnline(ctx context.Context, req *pb.ListUserOnlineRequest) (*pb.ListUserOnlineReply, error) {
	return &pb.ListUserOnlineReply{}, nil
}
func (s *UserOnlineService) ForceLogout(ctx context.Context, req *pb.ForceLogoutRequest) (*pb.ForceLogoutReply, error) {
	return &pb.ForceLogoutReply{}, nil
}
