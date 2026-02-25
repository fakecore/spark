package middleware

import (
	"context"
	"spark/pkg/clog"
	"spark/pkg/trace"
	"time"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

type contextKey string

const RequestIDContextKey contextKey = "request_id"

type TracingConfig struct {
	TraceIDHeader   string
	RequestIDHeader string
	LogRequests     bool
	LogResponses    bool
}

func DefaultTracingConfig() *TracingConfig {
	return &TracingConfig{
		TraceIDHeader:   "X-Trace-ID",
		RequestIDHeader: "X-Request-ID",
		LogRequests:     true,
		LogResponses:    true,
	}
}

// KratosHTTPTracing is now a simplified middleware that adds extra logging with trace info
// Note: Use this together with the standard tracing.Server() middleware from Kratos
func KratosHTTPTracing(logger clog.Logger, config *TracingConfig) middleware.Middleware {
	if config == nil {
		config = DefaultTracingConfig()
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			start := time.Now()

			// Get trace info from context (set by Kratos tracing middleware)
			traceID := trace.TraceIDFromContext(ctx)
			spanID := trace.SpanIDFromContext(ctx)

			// Get operation name from transport
			var operation string
			if tr, ok := transport.FromServerContext(ctx); ok {
				operation = tr.Operation()
			}

			// Log request with trace info
			if config.LogRequests && traceID != "" {
				logger.Info(ctx, "HTTP request with trace",
					clog.String("operation", operation),
					clog.String("span_id", spanID),
					clog.String("trace_id", traceID),
				)
			}

			reply, err := handler(ctx, req)

			// Log response with trace info
			if config.LogResponses {
				dur := time.Since(start)
				fields := []clog.Field{
					clog.Duration("duration", dur),
					clog.String("operation", operation),
				}
				if traceID != "" {
					fields = append(fields,
						clog.String("trace_id", traceID),
						clog.String("span_id", spanID),
					)
				}

				if err != nil {
					fields = append(fields, clog.Err(err))
					logger.Error(ctx, "HTTP response error with trace", fields...)
				} else {
					logger.Info(ctx, "HTTP response with trace", fields...)
				}
			}

			return reply, err
		}
	}
}

// RequestIDFromContext extracts the request ID from context
func RequestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDContextKey).(string); ok {
		return requestID
	}
	return ""
}
