package clog

import (
	"context"
	"sync"

	"spark/pkg/trace"

	"go.uber.org/zap"
)

var (
	globalLogger Logger
	globalMutex  sync.RWMutex
)

// InitGlobalLogger initializes the global logger instance
func InitGlobalLogger(config *Config) error {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	logger, err := NewLogger(config)
	if err != nil {
		return err
	}

	globalLogger = logger
	return nil
}

// InitGlobalLoggerWithName initializes the global logger instance with a name
func InitGlobalLoggerWithName(name string, config *Config) error {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	logger, err := NewLoggerWithName(name, config)
	if err != nil {
		return err
	}

	globalLogger = logger
	return nil
}

// SetGlobalLogger sets the global logger instance directly
func SetGlobalLogger(logger Logger) {
	globalMutex.Lock()
	defer globalMutex.Unlock()
	globalLogger = logger
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() Logger {
	globalMutex.RLock()
	defer globalMutex.RUnlock()
	return globalLogger
}

// Global logger convenience functions using structured logging interface

// Debug logs a debug message using the global logger
func Debug(ctx context.Context, msg string, fields ...Field) {
	if globalLogger != nil {
		globalLogger.Debug(ctx, msg, fields...)
	}
}

// Info logs an info message using the global logger
func Info(ctx context.Context, msg string, fields ...Field) {
	if globalLogger != nil {
		globalLogger.Info(ctx, msg, fields...)
	}
}

// Warn logs a warning message using the global logger
func Warn(ctx context.Context, msg string, fields ...Field) {
	if globalLogger != nil {
		globalLogger.Warn(ctx, msg, fields...)
	}
}

// Error logs an error message using the global logger
func Error(ctx context.Context, msg string, fields ...Field) {
	if globalLogger != nil {
		globalLogger.Error(ctx, msg, fields...)
	}
}

// DPanic logs a debug panic message using the global logger
func DPanic(ctx context.Context, msg string, fields ...Field) {
	if globalLogger != nil {
		globalLogger.DPanic(ctx, msg, fields...)
	}
}

// Panic logs a panic message using the global logger
func Panic(ctx context.Context, msg string, fields ...Field) {
	if globalLogger != nil {
		globalLogger.Panic(ctx, msg, fields...)
	}
}

// Fatal logs a fatal message and exits using the global logger
func Fatal(ctx context.Context, msg string, fields ...Field) {
	if globalLogger != nil {
		globalLogger.Fatal(ctx, msg, fields...)
	}
}

// WithContext returns a logger with trace information from context
func WithContext(ctx context.Context) Logger {
	if globalLogger != nil {
		return globalLogger.WithContext(ctx)
	}
	return &NoOpLogger{}
}

// WithTraceID returns a logger with trace ID
func WithTraceID(traceID string) Logger {
	if globalLogger != nil {
		return globalLogger.WithTraceID(traceID)
	}
	return &NoOpLogger{}
}

// WithService returns a logger with service information
func WithService(service, module string) Logger {
	if globalLogger != nil {
		return globalLogger.WithService(service, module)
	}
	return &NoOpLogger{}
}

// WithFields returns a logger with additional fields
func WithFields(fields map[string]interface{}) Logger {
	if globalLogger != nil {
		return globalLogger.WithFields(fields)
	}
	return &NoOpLogger{}
}

// WithField returns a logger with an additional field
func WithField(key string, value interface{}) Logger {
	if globalLogger != nil {
		return globalLogger.WithField(key, value)
	}
	return &NoOpLogger{}
}

// Sugar returns a sugared logger
func Sugar() SugarLogger {
	if globalLogger != nil {
		return globalLogger.Sugar()
	}
	return &NoOpSugarLogger{}
}

// Sync flushes any buffered log entries
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}

// Enabled checks if the given level is enabled
func Enabled(level Level) bool {
	if globalLogger != nil {
		return globalLogger.Enabled(level)
	}
	return false
}

// Legacy compatibility functions that return zap logger directly
// These are for backward compatibility with existing code

// LegacyWithContext returns a zap logger with trace information from context
func LegacyWithContext(ctx context.Context) *zap.Logger {
	if globalLogger != nil {
		if zapImpl, ok := globalLogger.(*ZapLoggerImpl); ok {
			traceID := trace.TraceIDFromContext(ctx)
			spanID := trace.SpanIDFromContext(ctx)

			fields := []zap.Field{}
			if traceID != "" {
				fields = append(fields, zap.String("trace_id", traceID))
			}
			if spanID != "" {
				fields = append(fields, zap.String("span_id", spanID))
			}

			if len(fields) > 0 {
				return zapImpl.GetZapLogger().With(fields...)
			}
			return zapImpl.GetZapLogger()
		}
	}
	return zap.NewNop()
}

// LegacyWithService returns a zap logger with service information
func LegacyWithService(service, module string) *zap.Logger {
	if globalLogger != nil {
		if zapImpl, ok := globalLogger.(*ZapLoggerImpl); ok {
			return zapImpl.GetZapLogger().With(
				zap.String("service", service),
				zap.String("module", module),
			)
		}
	}
	return zap.NewNop()
}

// LegacyWithFields returns a zap logger with additional fields
func LegacyWithFields(fields map[string]interface{}) *zap.Logger {
	if globalLogger != nil {
		if zapImpl, ok := globalLogger.(*ZapLoggerImpl); ok {
			zapFields := make([]zap.Field, 0, len(fields))
			for k, v := range fields {
				zapFields = append(zapFields, zap.Any(k, v))
			}
			return zapImpl.GetZapLogger().With(zapFields...)
		}
	}
	return zap.NewNop()
}

// NoOpLogger implements Logger interface with no-op operations
type NoOpLogger struct{}

func (l *NoOpLogger) Debug(ctx context.Context, msg string, fields ...Field)  {}
func (l *NoOpLogger) Info(ctx context.Context, msg string, fields ...Field)   {}
func (l *NoOpLogger) Warn(ctx context.Context, msg string, fields ...Field)   {}
func (l *NoOpLogger) Error(ctx context.Context, msg string, fields ...Field)  {}
func (l *NoOpLogger) DPanic(ctx context.Context, msg string, fields ...Field) {}
func (l *NoOpLogger) Panic(ctx context.Context, msg string, fields ...Field)  {}
func (l *NoOpLogger) Fatal(ctx context.Context, msg string, fields ...Field)  {}
func (l *NoOpLogger) Enabled(level Level) bool                                { return false }
func (l *NoOpLogger) WithContext(ctx context.Context) Logger                  { return l }
func (l *NoOpLogger) WithTraceID(traceID string) Logger                       { return l }
func (l *NoOpLogger) WithService(service, module string) Logger               { return l }
func (l *NoOpLogger) WithFields(fields map[string]interface{}) Logger         { return l }
func (l *NoOpLogger) WithField(key string, value interface{}) Logger          { return l }
func (l *NoOpLogger) Sugar() SugarLogger                                      { return &NoOpSugarLogger{} }
func (l *NoOpLogger) Sync() error                                             { return nil }
func (l *NoOpLogger) Clone() Logger                                           { return l }

// NoOpSugarLogger implements SugarLogger interface with no-op operations
type NoOpSugarLogger struct{}

func (s *NoOpSugarLogger) Debugf(template string, args ...interface{})  {}
func (s *NoOpSugarLogger) Infof(template string, args ...interface{})   {}
func (s *NoOpSugarLogger) Warnf(template string, args ...interface{})   {}
func (s *NoOpSugarLogger) Errorf(template string, args ...interface{})  {}
func (s *NoOpSugarLogger) DPanicf(template string, args ...interface{}) {}
func (s *NoOpSugarLogger) Panicf(template string, args ...interface{})  {}
func (s *NoOpSugarLogger) Fatalf(template string, args ...interface{})  {}
func (s *NoOpSugarLogger) Debug(args ...interface{})                    {}
func (s *NoOpSugarLogger) Info(args ...interface{})                     {}
func (s *NoOpSugarLogger) Warn(args ...interface{})                     {}
func (s *NoOpSugarLogger) Error(args ...interface{})                    {}
func (s *NoOpSugarLogger) DPanic(args ...interface{})                   {}
func (s *NoOpSugarLogger) Panic(args ...interface{})                    {}
func (s *NoOpSugarLogger) Fatal(args ...interface{})                    {}
func (s *NoOpSugarLogger) With(args ...interface{}) SugarLogger         { return s }
func (s *NoOpSugarLogger) WithOptions(opts ...interface{}) SugarLogger  { return s }
func (s *NoOpSugarLogger) Enabled(level Level) bool                     { return false }
func (s *NoOpSugarLogger) Sync() error                                  { return nil }
func (s *NoOpSugarLogger) Desugar() Logger                              { return &NoOpLogger{} }
