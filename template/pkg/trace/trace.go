package trace

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const TraceName = "spark"

// Context keys for trace information
type contextKey string

const (
	TraceIDContextKey contextKey = "trace_id"
	SpanIDContextKey  contextKey = "span_id"
)

var tracer oteltrace.Tracer

type Config struct {
	ServiceName        string            `yaml:"service_name" json:"service_name"`
	ServiceVersion     string            `yaml:"service_version" json:"service_version"`
	Endpoint           string            `yaml:"endpoint" json:"endpoint"` // OTLP HTTP endpoint (URL or host:port)
	ResourceAttributes map[string]string `yaml:"resource_attributes" json:"resource_attributes"`
	SamplingRate       float64           `yaml:"sampling_rate" json:"sampling_rate"`
	Enabled            bool              `yaml:"enabled" json:"enabled"`
}

func DefaultConfig() *Config {
	return &Config{
		ServiceName:    "spark",
		ServiceVersion: "1.0.0",
		Endpoint:       "",
		SamplingRate:   1.0,
		Enabled:        true,
	}
}

func InitTracer(config *Config) (func(), error) {
	if config == nil {
		config = DefaultConfig()
	}

	if !config.Enabled {
		otel.SetTracerProvider(noop.NewTracerProvider())
		tracer = otel.Tracer(TraceName)
		return func() {}, nil
	}

	attrs := []attribute.KeyValue{
		attribute.String("service.name", config.ServiceName),
		attribute.String("service.version", config.ServiceVersion),
	}
	for k, v := range config.ResourceAttributes {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		attrs = append(attrs, attribute.String(k, v))
	}

	res, err := resource.New(context.Background(), resource.WithAttributes(attrs...))
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	var exporter trace.SpanExporter
	if strings.TrimSpace(config.Endpoint) != "" {
		ep, err := parseOTLPEndpoint(config.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("parse otlp endpoint: %w", err)
		}

		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(ep.hostport),
		}
		if ep.path != "" {
			opts = append(opts, otlptracehttp.WithURLPath(ep.path))
		}
		if ep.insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}

		exporter, err = otlptracehttp.New(context.Background(), opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
		}
	} else {
		// Use a no-op exporter instead of stdout to avoid JSON pollution in logs
		exporter = &noopExporter{}
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		trace.WithSampler(trace.TraceIDRatioBased(config.SamplingRate)),
	)

	// Set global providers for Kratos tracing middleware
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer = otel.Tracer(TraceName)

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		tp.Shutdown(ctx)
	}, nil
}

type otlpEndpoint struct {
	hostport string
	path     string
	insecure bool
}

func parseOTLPEndpoint(raw string) (otlpEndpoint, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return otlpEndpoint{}, fmt.Errorf("empty endpoint")
	}

	u, err := url.Parse(raw)
	if err == nil && u.Scheme != "" {
		switch u.Scheme {
		case "http", "https":
		default:
			return otlpEndpoint{}, fmt.Errorf("unsupported scheme %q (expected http/https)", u.Scheme)
		}
		host := ensureHostPort(u.Host)
		if host == "" {
			return otlpEndpoint{}, fmt.Errorf("missing host in %q", raw)
		}
		path := strings.TrimSpace(u.EscapedPath())
		if path == "/" {
			path = ""
		}
		return otlpEndpoint{
			hostport: host,
			path:     path,
			insecure: u.Scheme == "http",
		}, nil
	}

	// Allow "host:port/path" without scheme.
	if strings.Contains(raw, "/") {
		u2, err := url.Parse("http://" + raw)
		if err != nil {
			return otlpEndpoint{}, fmt.Errorf("parse host/path: %w", err)
		}
		host := ensureHostPort(u2.Host)
		if host == "" {
			return otlpEndpoint{}, fmt.Errorf("missing host in %q", raw)
		}
		path := strings.TrimSpace(u2.EscapedPath())
		if path == "/" {
			path = ""
		}
		return otlpEndpoint{
			hostport: host,
			path:     path,
			insecure: true,
		}, nil
	}

	host := ensureHostPort(raw)
	if host == "" {
		return otlpEndpoint{}, fmt.Errorf("missing host in %q", raw)
	}
	return otlpEndpoint{
		hostport: host,
		path:     "",
		insecure: true,
	}, nil
}

func ensureHostPort(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if _, _, err := net.SplitHostPort(host); err == nil {
		return host
	}
	// If host is already bracketed IPv6 without port: "[::1]"
	if strings.HasPrefix(host, "[") && strings.Contains(host, "]") {
		return host + ":4318"
	}
	// Unbracketed IPv6 without port.
	if ip := net.ParseIP(host); ip != nil && strings.Contains(host, ":") {
		return net.JoinHostPort(host, "4318")
	}
	return net.JoinHostPort(host, "4318")
}

// GetTracerProvider returns the configured tracer provider
// This can be used with Kratos tracing middleware options
func GetTracerProvider() oteltrace.TracerProvider {
	return otel.GetTracerProvider()
}

// GetPropagator returns the configured text map propagator
// This can be used with Kratos tracing middleware options
func GetPropagator() propagation.TextMapPropagator {
	return otel.GetTextMapPropagator()
}

func StartSpan(ctx context.Context, name string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	return tracer.Start(ctx, name, opts...)
}

func SpanFromContext(ctx context.Context) oteltrace.Span {
	return oteltrace.SpanFromContext(ctx)
}

func SpanContextFromContext(ctx context.Context) oteltrace.SpanContext {
	return oteltrace.SpanContextFromContext(ctx)
}

func TraceIDFromContext(ctx context.Context) string {
	if sc := SpanContextFromContext(ctx); sc.IsValid() {
		return sc.TraceID().String()
	}
	// Fallback to custom trace ID if no OpenTelemetry trace context
	if customTraceID, ok := ctx.Value(TraceIDContextKey).(string); ok {
		return customTraceID
	}
	return ""
}

func SpanIDFromContext(ctx context.Context) string {
	if sc := SpanContextFromContext(ctx); sc.IsValid() {
		return sc.SpanID().String()
	}
	return ""
}

func InjectHTTPHeaders(ctx context.Context, req *http.Request) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
}

func ExtractHTTPHeaders(req *http.Request) context.Context {
	return otel.GetTextMapPropagator().Extract(context.Background(), propagation.HeaderCarrier(req.Header))
}

// ExtractTraceContext extracts trace context from HTTP request headers
// It supports both OpenTelemetry standard headers and custom X-Trace-ID header
func ExtractTraceContext(ctx context.Context, req *http.Request) context.Context {
	// First try to extract using OpenTelemetry propagation
	extractedCtx := otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(req.Header))

	// If no trace context was extracted, try custom X-Trace-ID header
	if !oteltrace.SpanContextFromContext(extractedCtx).IsValid() {
		if customTraceID := req.Header.Get("X-Trace-ID"); customTraceID != "" {
			// Store custom trace ID in context for logging purposes
			extractedCtx = context.WithValue(extractedCtx, TraceIDContextKey, customTraceID)
		}
	}

	return extractedCtx
}

func RecordError(span oteltrace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// noopExporter implements trace.SpanExporter but does nothing
// This prevents JSON trace output pollution in logs
type noopExporter struct{}

func (e *noopExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	return nil
}

func (e *noopExporter) Shutdown(ctx context.Context) error {
	return nil
}
