package system

import (
	"context"
	"time"

	pb "spark/api/system/v1"
	"spark/internal/data/dal/model"
	"spark/internal/data/dal/query"
	"spark/pkg/utils"
)

type LogService struct {
	pb.UnimplementedLogServer
	query *query.Query
}

func NewLogService(q *query.Query) *LogService {
	return &LogService{
		query: q,
	}
}

func (s *LogService) CreateOperLog(ctx context.Context, req *pb.CreateOperLogRequest) (*pb.CreateOperLogReply, error) {
	// Convert operator_type from string to int32
	var operatorType int32
	switch req.OperatorType {
	case "1":
		operatorType = 1
	case "2":
		operatorType = 2
	default:
		operatorType = 0
	}

	// Create operation log model
	operLog := &model.SysOperLog{
		Title:         utils.StringPtr(req.Title),
		BusinessType:  req.BusinessType,
		Method:        utils.StringPtr(req.Method),
		RequestMethod: utils.StringPtr(req.RequestMethod),
		OperatorType:  operatorType,
		OperName:      utils.StringPtr(req.OperName),
		DeptName:      utils.StringPtr(req.DeptName),
		OperURL:       utils.StringPtr(req.OperUrl),
		OperIP:        utils.StringPtr(req.OperIp),
		OperLocation:  utils.StringPtr(req.OperLocation),
		OperParam:     utils.StringPtr(req.OperParam),
		ErrorMsg:      utils.StringPtr(req.ErrorMsg),
	}

	// Save to database
	if err := s.query.SysOperLog.WithContext(ctx).Create(operLog); err != nil {
		return nil, err
	}

	return &pb.CreateOperLogReply{}, nil
}
func (s *LogService) GetOperLog(ctx context.Context, req *pb.GetOperLogRequest) (*pb.GetOperLogReply, error) {
	return &pb.GetOperLogReply{}, nil
}
func (s *LogService) ListOperLog(ctx context.Context, req *pb.ListOperLogRequest) (*pb.ListOperLogReply, error) {
	q := s.query.SysOperLog
	logQuery := q.WithContext(ctx)

	if req.Title != "" {
		logQuery = logQuery.Where(q.Title.Like("%" + req.Title + "%"))
	}
	if req.BusinessType != 0 {
		logQuery = logQuery.Where(q.BusinessType.Eq(req.BusinessType))
	}
	if req.OperName != "" {
		logQuery = logQuery.Where(q.OperName.Like("%" + req.OperName + "%"))
	}
	// Status field is missing in model, skipping filter
	// if req.Status != 0 {
	// 	logQuery = logQuery.Where(q.Status.Eq(req.Status))
	// }
	if req.BeginTime != 0 {
		logQuery = logQuery.Where(q.CreatedAt.Gte(req.BeginTime))
	}
	if req.EndTime != 0 {
		logQuery = logQuery.Where(q.CreatedAt.Lte(req.EndTime))
	}

	total, err := logQuery.Count()
	if err != nil {
		return nil, err
	}

	logs, err := logQuery.Scopes(utils.Paginate(req.Current, req.PageSize)).Order(q.ID.Desc()).Find()
	if err != nil {
		return nil, err
	}

	items := make([]*pb.OperLogInfo, 0, len(logs))
	// Use the project's standard timezone utility
	loc := utils.GetChinaLocation()

	for _, log := range logs {
		// Convert millisecond timestamp to Asia/Shanghai timezone
		localTime := time.UnixMilli(log.CreatedAt).In(loc)
		// Convert back to Unix timestamp in seconds for the response
		operTime := localTime.Unix()

		items = append(items, &pb.OperLogInfo{
			Id:            log.ID,
			Title:         safeStr(log.Title),
			BusinessType:  log.BusinessType,
			Method:        safeStr(log.Method),
			RequestMethod: safeStr(log.RequestMethod),
			OperatorType:  string(rune(log.OperatorType)), // Assuming mapping
			OperName:      safeStr(log.OperName),
			DeptName:      safeStr(log.DeptName),
			OperUrl:       safeStr(log.OperURL),
			OperIp:        safeStr(log.OperIP),
			OperLocation:  safeStr(log.OperLocation),
			OperParam:     safeStr(log.OperParam),
			JsonResult:    "", // Not in model
			Status:        0,  // Not in model
			ErrorMsg:      safeStr(log.ErrorMsg),
			OperTime:      operTime,
			CostTime:      0, // Not in model
		})
	}

	return &pb.ListOperLogReply{
		Items: items,
		Total: int32(total),
	}, nil
}

func safeStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (s *LogService) DeleteOperLog(ctx context.Context, req *pb.DeleteOperLogRequest) (*pb.DeleteOperLogReply, error) {
	return &pb.DeleteOperLogReply{}, nil
}
func (s *LogService) ClearOperLog(ctx context.Context, req *pb.ClearOperLogRequest) (*pb.ClearOperLogReply, error) {
	return &pb.ClearOperLogReply{}, nil
}
func (s *LogService) CreateLoginLog(ctx context.Context, req *pb.CreateLoginLogRequest) (*pb.CreateLoginLogReply, error) {
	return &pb.CreateLoginLogReply{}, nil
}
func (s *LogService) GetLoginLog(ctx context.Context, req *pb.GetLoginLogRequest) (*pb.GetLoginLogReply, error) {
	return &pb.GetLoginLogReply{}, nil
}
func (s *LogService) ListLoginLog(ctx context.Context, req *pb.ListLoginLogRequest) (*pb.ListLoginLogReply, error) {
	return &pb.ListLoginLogReply{}, nil
}
func (s *LogService) DeleteLoginLog(ctx context.Context, req *pb.DeleteLoginLogRequest) (*pb.DeleteLoginLogReply, error) {
	return &pb.DeleteLoginLogReply{}, nil
}
func (s *LogService) ClearLoginLog(ctx context.Context, req *pb.ClearLoginLogRequest) (*pb.ClearLoginLogReply, error) {
	return &pb.ClearLoginLogReply{}, nil
}
