package trace

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// HTTPMiddleware creates an HTTP middleware that traces incoming requests
func HTTPMiddleware(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := ExtractHTTPHeaders(r)
			ctx, span := StartSpan(ctx, r.Method+" "+r.URL.Path,
				oteltrace.WithAttributes(
					attribute.String("http.method", r.Method),
					attribute.String("http.url", r.URL.String()),
					attribute.String("http.scheme", r.URL.Scheme),
					attribute.String("http.host", r.Host),
					attribute.String("http.user_agent", r.UserAgent()),
					attribute.String("service.name", serviceName),
				),
				oteltrace.WithSpanKind(oteltrace.SpanKindServer),
			)
			defer span.End()

			// Add trace context to response headers
			InjectHTTPHeaders(ctx, r)

			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)

			// Record response status
			if span.IsRecording() {
				span.SetAttributes(attribute.Int("http.status_code", getStatusCode(w)))
			}
		})
	}
}

// getStatusCode extracts status code from response writer
func getStatusCode(w http.ResponseWriter) int {
	if rw, ok := w.(interface{ Status() int }); ok {
		return rw.Status()
	}
	return 200 // default to OK
}

// TraceHandler wraps a handler function with tracing
func TraceHandler(name string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := StartSpan(r.Context(), name)
		defer span.End()

		r = r.WithContext(ctx)
		handler(w, r)
	}
}

// WithSpan executes a function within a traced span
func WithSpan(ctx context.Context, name string, fn func(context.Context) error, opts ...oteltrace.SpanStartOption) error {
	ctx, span := StartSpan(ctx, name, opts...)
	defer span.End()

	if err := fn(ctx); err != nil {
		RecordError(span, err)
		return err
	}

	return nil
}

// TraceFunction is a helper to trace any function execution
func TraceFunction(ctx context.Context, functionName string, fn func() error) error {
	return WithSpan(ctx, functionName, func(ctx context.Context) error {
		return fn()
	})
}
