package middleware

import (
	"context"
	"fmt"
	"time"

	"spark/pkg/metrics"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

// MetricsServer records basic request metrics for Kratos HTTP handlers.
//
// Note: this does not instrument raw gRPC servers. Use pkg/metrics.UnaryServerInterceptor for gRPC.
func MetricsServer() middleware.Middleware {
	metrics.Init()
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			start := time.Now()
			operation := ""
			if info, ok := transport.FromServerContext(ctx); ok {
				operation = info.Operation()
			}

			reply, err = handler(ctx, req)

			code := int32(200)
			if se := errors.FromError(err); se != nil {
				code = se.Code
			}
			metrics.HTTPRequestsTotal.WithLabelValues(operation, fmt.Sprintf("%d", code)).Inc()
			metrics.HTTPRequestDurationSeconds.WithLabelValues(operation).Observe(time.Since(start).Seconds())
			return reply, err
		}
	}
}
