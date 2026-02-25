package middleware

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	pb "spark/api/system/v1"
	"spark/internal/service/system"
	"spark/pkg/operlog"
	"spark/pkg/viewer"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// OperationLog Middleware for recording operation logs
func OperationLog(logSvc *system.LogService) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			startTime := time.Now()

			// Initialize LogMeta and inject into context
			ctx, meta := operlog.NewContext(ctx)

			var (
				operName     string
				deptName     string
				operatorType int32 = 0 // 0: Other
			)

			// Extract user info from context
			if v, ok := viewer.UserViewFromContext(ctx); ok {
				if user := v.GetUser(); user != nil {
					operName = user.Name
					// deptName = user.DeptName // Assuming User has DeptName
					operatorType = 1 // 1: Background User
				}
			}

			// Execute handler
			reply, err = handler(ctx, req)

			// Async record log
			go func(ctx context.Context, req interface{}, reply interface{}, err error, startTime time.Time, meta *operlog.LogMeta) {
				tr, ok := transport.FromServerContext(ctx)
				if !ok {
					return
				}

				ht, ok := tr.(*http.Transport)
				if !ok {
					return // Only support HTTP for now
				}

				// Skip GET requests or specific paths if needed
				if ht.Request().Method == "GET" {
					return
				}

				// Determine Title
				title := "System Operation"
				if meta.Title != "" {
					title = meta.Title
				}

				// Determine BusinessType
				businessType := int32(0)
				if meta.BusinessType != 0 {
					businessType = meta.BusinessType
				}

				// Determine OperParam
				operParam := getReqParam(req)
				if meta.OperParam != "" {
					operParam = meta.OperParam
				}

				// Prepare error message
				errorMsg := ""
				if err != nil {
					errorMsg = err.Error()
				}

				// Create operation log request
				createReq := &pb.CreateOperLogRequest{
					Title:         title,
					BusinessType:  businessType,
					Method:        ht.Operation(),
					RequestMethod: ht.Request().Method,
					OperatorType:  strconv.Itoa(int(operatorType)),
					OperName:      operName,
					DeptName:      deptName,
					OperUrl:       ht.Request().URL.String(),
					OperIp:        ht.Request().RemoteAddr, // Might need real IP extraction logic
					OperLocation:  "",
					OperParam:     operParam,
					JsonResult:    "",
					Status:        0,
					ErrorMsg:      errorMsg,
					CostTime:      time.Since(startTime).Milliseconds(),
				}

				_, _ = logSvc.CreateOperLog(context.Background(), createReq)

			}(ctx, req, reply, err, startTime, meta)

			return reply, err
		}
	}
}

func getReqParam(req interface{}) string {
	if req == nil {
		return ""
	}
	b, err := json.Marshal(req)
	if err != nil {
		return ""
	}
	return string(b)
}
