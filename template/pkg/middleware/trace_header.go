package middleware

import (
	"context"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"

	"spark/pkg/trace"
)

// TraceHeader is a middleware that sets trace ID in response headers
func TraceHeader() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			// 执行处理器
			reply, err = handler(ctx, req)

			// 从 context 中获取 trace ID 并设置到 transport 的 header 中
			traceID := trace.TraceIDFromContext(ctx)
			spanID := trace.SpanIDFromContext(ctx)

			if info, ok := transport.FromServerContext(ctx); ok {
				if header := info.ReplyHeader(); header != nil {
					if traceID != "" {
						header.Set("X-Trace-ID", traceID)
					}
					if spanID != "" {
						header.Set("X-Span-ID", spanID)
					}
				}
			}

			return reply, err
		}
	}
}
