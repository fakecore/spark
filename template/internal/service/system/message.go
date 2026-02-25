package system

import (
	"context"

	pb "spark/api/system/v1"
)

type MessageService struct {
	pb.UnimplementedMessageServer
}

func NewMessageService() *MessageService {
	return &MessageService{}
}

func (s *MessageService) CreateMessage(ctx context.Context, req *pb.CreateMessageRequest) (*pb.CreateMessageReply, error) {
	return &pb.CreateMessageReply{}, nil
}
func (s *MessageService) UpdateMessage(ctx context.Context, req *pb.UpdateMessageRequest) (*pb.UpdateMessageReply, error) {
	return &pb.UpdateMessageReply{}, nil
}
func (s *MessageService) DeleteMessage(ctx context.Context, req *pb.DeleteMessageRequest) (*pb.DeleteMessageReply, error) {
	return &pb.DeleteMessageReply{}, nil
}
func (s *MessageService) GetMessage(ctx context.Context, req *pb.GetMessageRequest) (*pb.GetMessageReply, error) {
	return &pb.GetMessageReply{}, nil
}
func (s *MessageService) ListMessage(ctx context.Context, req *pb.ListMessageRequest) (*pb.ListMessageReply, error) {
	return &pb.ListMessageReply{}, nil
}
func (s *MessageService) ReadMessage(ctx context.Context, req *pb.ReadMessageRequest) (*pb.ReadMessageReply, error) {
	return &pb.ReadMessageReply{}, nil
}
