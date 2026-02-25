package system

import (
	"context"

	pb "spark/api/system/v1"
)

type PackageService struct {
	pb.UnimplementedPackageServer
}

func NewPackageService() *PackageService {
	return &PackageService{}
}

func (s *PackageService) CreatePackage(ctx context.Context, req *pb.CreatePackageRequest) (*pb.CreatePackageReply, error) {
	return &pb.CreatePackageReply{}, nil
}
func (s *PackageService) UpdatePackage(ctx context.Context, req *pb.UpdatePackageRequest) (*pb.UpdatePackageReply, error) {
	return &pb.UpdatePackageReply{}, nil
}
func (s *PackageService) DeletePackage(ctx context.Context, req *pb.DeletePackageRequest) (*pb.DeletePackageReply, error) {
	return &pb.DeletePackageReply{}, nil
}
func (s *PackageService) GetPackage(ctx context.Context, req *pb.GetPackageRequest) (*pb.GetPackageReply, error) {
	return &pb.GetPackageReply{}, nil
}
func (s *PackageService) ListPackage(ctx context.Context, req *pb.ListPackageRequest) (*pb.ListPackageReply, error) {
	return &pb.ListPackageReply{}, nil
}
