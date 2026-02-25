package system

import (
	"context"

	pb "spark/api/system/v1"
	"spark/internal/biz/system"
	"spark/internal/data/dal/model"
	"spark/internal/data/do"
	"spark/pkg/utils"
	"spark/pkg/viewer"
)

type PostService struct {
	pb.UnimplementedPostServer
	postUc *system.SystemPostUsecase
}

func NewPostService(postUc *system.SystemPostUsecase) *PostService {
	return &PostService{postUc: postUc}
}

func (s *PostService) CreatePost(ctx context.Context, req *pb.CreatePostRequest) (*pb.CreatePostReply, error) {
	userView := viewer.MustGetUserViewFromContext(ctx)
	user := userView.GetUser()

	post := &model.SysPost{
		Status:   req.Status,
		Sort:     req.Sort,
		PostCode: req.PostCode,
		PostName: req.PostName,
		Remark:   req.Remark,
		BaseModel: model.BaseModel{
			CreatedBy: &user.ID,
			UpdatedBy: &user.ID,
		},
	}

	err := s.postUc.Create(ctx, post)
	if err != nil {
		return nil, err
	}

	return &pb.CreatePostReply{}, nil
}

func (s *PostService) UpdatePost(ctx context.Context, req *pb.UpdatePostRequest) (*pb.UpdatePostReply, error) {
	post := &do.SysPost{
		ID:       req.Id,
		Status:   &req.Status,
		PostSort: &req.Sort,
		PostCode: &req.PostCode,
		PostName: &req.PostName,
		Remark:   req.Remark,
	}

	err := s.postUc.Update(ctx, post)
	if err != nil {
		return nil, err
	}
	return &pb.UpdatePostReply{}, nil
}

func (s *PostService) DeletePost(ctx context.Context, req *pb.DeletePostRequest) (*pb.DeletePostReply, error) {
	err := s.postUc.Delete(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.DeletePostReply{}, nil
}

func (s *PostService) GetPost(ctx context.Context, req *pb.GetPostRequest) (*pb.GetPostReply, error) {
	post, err := s.postUc.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	pbPost := &pb.PostInfo{}
	utils.ObjConvert(post, pbPost)
	return &pb.GetPostReply{Post: pbPost}, nil
}

func (s *PostService) ListPost(ctx context.Context, req *pb.ListPostRequest) (*pb.ListPostReply, error) {
	posts, total, err := s.postUc.List(ctx, req.PageSize, req.Current, req.PostCode, req.PostName, req.Status)
	if err != nil {
		return nil, err
	}

	pbPosts := make([]*pb.PostInfo, 0, len(posts))
	for _, post := range posts {
		pbPost := &pb.PostInfo{}
		utils.ObjConvert(post, pbPost)
		pbPosts = append(pbPosts, pbPost)
	}

	return &pb.ListPostReply{
		Items: pbPosts,
		Total: int32(total),
	}, nil
}
