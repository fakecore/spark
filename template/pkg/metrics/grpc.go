package metrics

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor instruments gRPC unary handlers with Prometheus metrics.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	Init()
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)

		method := ""
		if info != nil {
			method = info.FullMethod
		}
		code := status.Code(err).String()
		GRPCRequestsTotal.WithLabelValues(method, code).Inc()
		GRPCRequestDurationSeconds.WithLabelValues(method).Observe(time.Since(start).Seconds())

		return resp, err
	}
}
