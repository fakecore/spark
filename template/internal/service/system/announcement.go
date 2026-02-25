package system

import (
	"context"

	pb "spark/api/system/v1"
)

type AnnouncementService struct {
	pb.UnimplementedAnnouncementServer
}

func NewAnnouncementService() *AnnouncementService {
	return &AnnouncementService{}
}

func (s *AnnouncementService) CreateAnnouncement(ctx context.Context, req *pb.CreateAnnouncementRequest) (*pb.CreateAnnouncementReply, error) {
	return &pb.CreateAnnouncementReply{}, nil
}
func (s *AnnouncementService) UpdateAnnouncement(ctx context.Context, req *pb.UpdateAnnouncementRequest) (*pb.UpdateAnnouncementReply, error) {
	return &pb.UpdateAnnouncementReply{}, nil
}
func (s *AnnouncementService) DeleteAnnouncement(ctx context.Context, req *pb.DeleteAnnouncementRequest) (*pb.DeleteAnnouncementReply, error) {
	return &pb.DeleteAnnouncementReply{}, nil
}
func (s *AnnouncementService) GetAnnouncement(ctx context.Context, req *pb.GetAnnouncementRequest) (*pb.GetAnnouncementReply, error) {
	return &pb.GetAnnouncementReply{}, nil
}
func (s *AnnouncementService) ListAnnouncement(ctx context.Context, req *pb.ListAnnouncementRequest) (*pb.ListAnnouncementReply, error) {
	return &pb.ListAnnouncementReply{}, nil
}
