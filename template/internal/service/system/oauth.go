package system

import (
	"context"

	pb "spark/api/system/v1"
)

type OAuthService struct {
	pb.UnimplementedOAuthServer
}

func NewOAuthService() *OAuthService {
	return &OAuthService{}
}

func (s *OAuthService) Login(ctx context.Context, req *pb.OAuthLoginRequest) (*pb.OAuthLoginReply, error) {
	return &pb.OAuthLoginReply{}, nil
}
func (s *OAuthService) RefreshToken(ctx context.Context, req *pb.OAuthRefreshTokenRequest) (*pb.OAuthRefreshTokenReply, error) {
	return &pb.OAuthRefreshTokenReply{}, nil
}
func (s *OAuthService) Logout(ctx context.Context, req *pb.OAuthLogoutRequest) (*pb.OAuthLogoutReply, error) {
	return &pb.OAuthLogoutReply{}, nil
}
func (s *OAuthService) GetUserInfo(ctx context.Context, req *pb.OAuthGetUserInfoRequest) (*pb.OAuthGetUserInfoReply, error) {
	return &pb.OAuthGetUserInfoReply{}, nil
}
func (s *OAuthService) GetUserPerms(ctx context.Context, req *pb.OAuthGetUserPermsRequest) (*pb.OAuthGetUserPermsReply, error) {
	return &pb.OAuthGetUserPermsReply{}, nil
}
func (s *OAuthService) GetUserRoutes(ctx context.Context, req *pb.OAuthGetUserRoutesRequest) (*pb.OAuthGetUserRoutesReply, error) {
	return &pb.OAuthGetUserRoutesReply{}, nil
}
func (s *OAuthService) BindOAuth(ctx context.Context, req *pb.OAuthBindRequest) (*pb.OAuthBindReply, error) {
	return &pb.OAuthBindReply{}, nil
}
func (s *OAuthService) UnbindOAuth(ctx context.Context, req *pb.OAuthUnbindRequest) (*pb.OAuthUnbindReply, error) {
	return &pb.OAuthUnbindReply{}, nil
}
