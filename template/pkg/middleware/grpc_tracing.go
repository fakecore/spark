package middleware

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"spark/pkg/clog"
	log "spark/pkg/clog"
	"spark/pkg/trace"
)

// GRPCTracingConfig holds configuration for gRPC tracing middleware
type GRPCTracingConfig struct {
	LogRequests  bool // Whether to log incoming requests
	LogResponses bool // Whether to log outgoing responses
}

// DefaultGRPCTracingConfig returns default gRPC tracing configuration
func DefaultGRPCTracingConfig() *GRPCTracingConfig {
	return &GRPCTracingConfig{
		LogRequests:  true,
		LogResponses: true,
	}
}

// GrpcTracing returns a Kratos gRPC middleware for tracing
func GrpcTracing(logger clog.Logger, config *GRPCTracingConfig) middleware.Middleware {
	if config == nil {
		config = DefaultGRPCTracingConfig()
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			startTime := time.Now()

			// Start tracing span
			ctx, span := trace.StartSpan(ctx, "gRPC request")
			defer span.End()
			traceID := trace.TraceIDFromContext(ctx)

			// Get transport info if available
			var operation string
			if tr, ok := transport.FromServerContext(ctx); ok {
				operation = tr.Operation()
			}

			// Log incoming request
			if config.LogRequests {
				logger.Info(ctx, "gRPC request started",
					log.String("operation", operation),
					log.String("trace_id", traceID),
					log.Any("request", req),
				)
			}

			// Call next handler
			reply, err := handler(ctx, req)

			duration := time.Since(startTime)

			// Log response
			if config.LogResponses {
				fields := []log.Field{
					log.String("operation", operation),
					log.String("trace_id", traceID),
					log.Duration("duration", duration),
				}

				if err != nil {
					logger.Error(ctx, "gRPC request completed with error", append(fields, log.Err(err))...)
				} else {
					logger.Info(ctx, "gRPC request completed successfully", fields...)
				}
			}

			return reply, err
		}
	}
}

// GRPCUnaryServerInterceptor returns a gRPC unary server interceptor for tracing
func GRPCUnaryServerInterceptor(logger clog.Logger, config *GRPCTracingConfig) grpc.UnaryServerInterceptor {
	if config == nil {
		config = DefaultGRPCTracingConfig()
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()

		// Start new span for client interceptor
		ctx, span := trace.StartSpan(ctx, "gRPC client call")
		defer span.End()
		traceID := trace.TraceIDFromContext(ctx)

		// Add trace ID to outgoing metadata
		ctx = addTraceIDToMetadata(ctx, traceID)

		// Log incoming request
		if config.LogRequests {
			logger.Info(ctx, "gRPC unary request started",
				log.String("method", info.FullMethod),
				log.String("trace_id", traceID),
			)
		}

		// Call handler
		resp, err := handler(ctx, req)

		duration := time.Since(startTime)

		// Log response
		if config.LogResponses {
			fields := []log.Field{
				log.String("method", info.FullMethod),
				log.String("trace_id", traceID),
				log.Duration("duration", duration),
			}

			if err != nil {
				logger.Error(ctx, "gRPC unary request completed with error", append(fields, log.Err(err))...)
			} else {
				logger.Info(ctx, "gRPC unary request completed successfully", fields...)
			}
		}

		return resp, err
	}
}

// GRPCStreamServerInterceptor returns a gRPC stream server interceptor for tracing
func GRPCStreamServerInterceptor(logger clog.Logger, config *GRPCTracingConfig) grpc.StreamServerInterceptor {
	if config == nil {
		config = DefaultGRPCTracingConfig()
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		startTime := time.Now()
		ctx := ss.Context()

		// Start new span for client interceptor
		ctx, span := trace.StartSpan(ctx, "gRPC client call")
		defer span.End()
		traceID := trace.TraceIDFromContext(ctx)

		// Create wrapped stream with new context
		wrapped := &wrappedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		// Add trace ID to outgoing metadata
		wrapped.ctx = addTraceIDToMetadata(wrapped.ctx, traceID)

		// Log stream start
		if config.LogRequests {
			logger.Info(ctx, "gRPC stream request started",
				log.String("method", info.FullMethod),
				log.String("trace_id", traceID),
				log.Bool("client_stream", info.IsClientStream),
				log.Bool("server_stream", info.IsServerStream),
			)
		}

		// Call handler
		err := handler(srv, wrapped)

		duration := time.Since(startTime)

		// Log stream completion
		if config.LogResponses {
			fields := []log.Field{
				log.String("method", info.FullMethod),
				log.String("trace_id", traceID),
				log.Duration("duration", duration),
			}

			if err != nil {
				logger.Error(ctx, "gRPC stream request completed with error", append(fields, log.Err(err))...)
			} else {
				logger.Info(ctx, "gRPC stream request completed successfully", fields...)
			}
		}

		return err
	}
}

// GRPCUnaryClientInterceptor returns a gRPC unary client interceptor for tracing
func GRPCUnaryClientInterceptor(logger clog.Logger) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		startTime := time.Now()

		// Start tracing span
		ctx, span := trace.StartSpan(ctx, "gRPC client "+method)
		defer span.End()
		traceID := trace.TraceIDFromContext(ctx)

		// Add trace ID to outgoing metadata
		ctx = addTraceIDToMetadata(ctx, traceID)

		// Log outgoing request
		logger.Info(ctx, "gRPC client request started",
			log.String("method", method),
			log.String("trace_id", traceID),
			log.String("target", cc.Target()),
		)

		// Call method
		err := invoker(ctx, method, req, reply, cc, opts...)

		duration := time.Since(startTime)

		// Log response
		fields := []log.Field{
			log.String("method", method),
			log.String("trace_id", traceID),
			log.String("target", cc.Target()),
			log.Duration("duration", duration),
		}

		if err != nil {
			logger.Error(ctx, "gRPC client request completed with error", append(fields, log.Err(err))...)
		} else {
			logger.Info(ctx, "gRPC client request completed successfully", fields...)
		}

		return err
	}
}

// extractTraceIDFromMetadata extracts trace ID from gRPC metadata
func extractTraceIDFromMetadata(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	traceIDs := md.Get("x-trace-id")
	if len(traceIDs) == 0 {
		return ""
	}

	// Simply return the first trace ID without validation
	return traceIDs[0]
}

// addTraceIDToMetadata adds trace ID to outgoing metadata
func addTraceIDToMetadata(ctx context.Context, traceID string) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	} else {
		md = md.Copy()
	}

	md.Set("x-trace-id", traceID)
	return metadata.NewOutgoingContext(ctx, md)
}

// wrappedServerStream wraps grpc.ServerStream with a new context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
