package clog

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"spark/pkg/trace"
)

// Level represents the logging level
type Level int8

const (
	DebugLevel Level = iota - 1
	InfoLevel
	WarnLevel
	ErrorLevel
	DPanicLevel
	PanicLevel
	FatalLevel
)

// Field represents a key-value pair for structured logging
type Field struct {
	Key   string
	Value interface{}
}

// Logger defines the core logging interface that supports structured logging
type Logger interface {
	Debug(ctx context.Context, msg string, fields ...Field)
	Info(ctx context.Context, msg string, fields ...Field)
	Warn(ctx context.Context, msg string, fields ...Field)
	Error(ctx context.Context, msg string, fields ...Field)
	DPanic(ctx context.Context, msg string, fields ...Field)
	Panic(ctx context.Context, msg string, fields ...Field)
	Fatal(ctx context.Context, msg string, fields ...Field)

	Enabled(level Level) bool

	WithContext(ctx context.Context) Logger
	WithTraceID(traceID string) Logger
	WithService(service, module string) Logger
	WithFields(fields map[string]interface{}) Logger
	WithField(key string, value interface{}) Logger

	Sugar() SugarLogger
	Sync() error
	Clone() Logger
}

// SugarLogger defines the sugared logging interface for easier usage
type SugarLogger interface {
	Debugf(template string, args ...interface{})
	Infof(template string, args ...interface{})
	Warnf(template string, args ...interface{})
	Errorf(template string, args ...interface{})
	DPanicf(template string, args ...interface{})
	Panicf(template string, args ...interface{})
	Fatalf(template string, args ...interface{})

	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	DPanic(args ...interface{})
	Panic(args ...interface{})
	Fatal(args ...interface{})

	With(args ...interface{}) SugarLogger
	WithOptions(opts ...interface{}) SugarLogger

	Enabled(level Level) bool
	Sync() error
	Desugar() Logger
}

// Config holds the configuration for logger
type Config struct {
	Level             string          `yaml:"level" json:"level"`
	Format            string          `yaml:"format" json:"format"`
	DisableCaller     bool            `yaml:"disable_caller" json:"disable_caller"`
	DisableStacktrace bool            `yaml:"disable_stacktrace" json:"disable_stacktrace"`
	OutputPaths       []string        `yaml:"output_paths" json:"output_paths"`
	ErrorOutputPaths  []string        `yaml:"error_output_paths" json:"error_output_paths"`
	Rotation          *RotationConfig `yaml:"rotation" json:"rotation"`
}

// LoggerFactory provides a factory for creating logger instances with different configurations
type LoggerFactory struct {
	config *Config
}

// ClogOption defines a functional option for configuring logger creation
type ClogOption func(*clogOptions)

// LoggerType defines the type of logger to create
type LoggerType string

const (
	// ZapLogger creates a standard zap logger
	ZapLogger LoggerType = "zap"
	// KratosLogger creates a kratos-compatible logger
	KratosLogger LoggerType = "kratos"
	// StdLogger creates a standard library compatible logger
	StdLogger LoggerType = "std"
)

// clogOptions holds the configuration clogOptions for creating a logger instance
type clogOptions struct {
	name       string
	callerSkip int
	service    string
	module     string
	fields     map[string]interface{}
	loggerType LoggerType
	config     *Config // Override factory config if needed
}

// defaultOptions returns the default options
func defaultOptions() *clogOptions {
	return &clogOptions{
		name:       "",
		callerSkip: 0,
		service:    "",
		module:     "",
		fields:     make(map[string]interface{}),
		loggerType: ZapLogger, // Default to zap logger
		config:     nil,       // Use factory config
	}
}

// WithName sets the logger name
func WithName(name string) ClogOption {
	return func(o *clogOptions) {
		o.name = name
	}
}

// WithCallerSkip sets the caller skip for the logger
func WithCallerSkip(skip int) ClogOption {
	return func(o *clogOptions) {
		o.callerSkip = skip
	}
}

// WithLoggerService sets the service and module for the logger
func WithLoggerService(service, module string) ClogOption {
	return func(o *clogOptions) {
		o.service = service
		o.module = module
	}
}

// WithLoggerFields adds fields to the logger
func WithLoggerFields(fields map[string]interface{}) ClogOption {
	return func(o *clogOptions) {
		for k, v := range fields {
			o.fields[k] = v
		}
	}
}

// WithLoggerField adds a single field to the logger
func WithLoggerField(key string, value interface{}) ClogOption {
	return func(o *clogOptions) {
		o.fields[key] = value
	}
}

// WithLoggerType sets the type of logger to create
func WithLoggerType(loggerType LoggerType) ClogOption {
	return func(o *clogOptions) {
		o.loggerType = loggerType
	}
}

// WithConfig overrides the factory configuration for this logger instance
func WithConfig(config *Config) ClogOption {
	return func(o *clogOptions) {
		o.config = config
	}
}

// RotationConfig holds log rotation settings using lumberjack
type RotationConfig struct {
	MaxSize    int  `yaml:"max_size" json:"max_size"`
	MaxBackups int  `yaml:"max_backups" json:"max_backups"`
	MaxAge     int  `yaml:"max_age" json:"max_age"`
	Compress   bool `yaml:"compress" json:"compress"`
}

// DefaultConfig returns the default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:             "info",
		Format:            "json",
		DisableCaller:     false,
		DisableStacktrace: false,
		OutputPaths:       []string{"stdout"},
		ErrorOutputPaths:  []string{"stderr"},
		Rotation: &RotationConfig{
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     7,
			Compress:   true,
		},
	}
}

// NewLoggerFactory creates a new logger factory with the given configuration
func NewLoggerFactory(config *Config) *LoggerFactory {
	if config == nil {
		config = DefaultConfig()
	}
	return &LoggerFactory{
		config: config,
	}
}

// NewDefaultLoggerFactory creates a new logger factory with default configuration
func NewDefaultLoggerFactory() *LoggerFactory {
	return NewLoggerFactory(DefaultConfig())
}

// New creates a new logger instance with the provided options
func (f *LoggerFactory) New(opts ...ClogOption) (Logger, error) {
	// Start with default options
	options := defaultOptions()

	// Apply provided options
	for _, opt := range opts {
		opt(options)
	}

	// Determine which config to use
	config := f.config
	if options.config != nil {
		config = options.config
	}

	// Create logger based on type
	var logger Logger
	var err error

	switch options.loggerType {
	case ZapLogger:
		logger, err = NewLoggerWithCallerSkip(options.name, config, options.callerSkip)
	case KratosLogger:
		// For Kratos, we create a wrapper that implements our Logger interface
		// but optimizes for Kratos usage patterns
		logger, err = NewKratosCompatibleLogger(options.name, config, options.callerSkip)
	default:
		// Fallback to zap logger
		logger, err = NewLoggerWithCallerSkip(options.name, config, options.callerSkip)
	}

	if err != nil {
		return nil, err
	}

	// Apply additional configurations
	if options.service != "" || options.module != "" {
		logger = logger.WithService(options.service, options.module)
	}

	if len(options.fields) > 0 {
		logger = logger.WithFields(options.fields)
	}

	return logger, nil
}

// GetConfig returns the factory's configuration
func (f *LoggerFactory) GetConfig() *Config {
	return f.config
}

// Default logger factory instance for convenience
var defaultFactory *LoggerFactory

func init() {
	defaultFactory = NewDefaultLoggerFactory()
}

// NewWithOptions creates a new logger using the default factory with the provided options
func NewWithOptions(opts ...ClogOption) (Logger, error) {
	return defaultFactory.New(opts...)
}

// NewFactoryLogger creates a new logger with a custom factory and options
func NewFactoryLogger(config *Config, opts ...ClogOption) (Logger, error) {
	factory := NewLoggerFactory(config)
	return factory.New(opts...)
}

// NewKratosCompatibleLogger creates a logger optimized for Kratos usage
func NewKratosCompatibleLogger(name string, config *Config, callerSkip int) (Logger, error) {
	// Create base logger with extra caller skip for Kratos middleware
	return NewLoggerWithCallerSkip(name, config, callerSkip+1)
}

// Predefined logger configurations for common use cases

// APIConfig returns a configuration optimized for API logging
func APIConfig() *Config {
	config := DefaultConfig()
	config.Level = "info"
	config.Format = "json"
	return config
}

// DevConfig returns a configuration optimized for development
func DevConfig() *Config {
	config := DefaultConfig()
	config.Level = "debug"
	config.Format = "console"
	return config
}

// ProductionConfig returns a configuration optimized for production
func ProductionConfig() *Config {
	config := DefaultConfig()
	config.Level = "info"
	config.Format = "json"
	config.OutputPaths = []string{"stdout", "/var/log/app.log"}
	config.ErrorOutputPaths = []string{"stderr", "/var/log/app-error.log"}
	return config
}

// Convenient factory creation methods

// NewAPILoggerFactory creates a factory optimized for API usage
func NewAPILoggerFactory() *LoggerFactory {
	return NewLoggerFactory(APIConfig())
}

// NewDevLoggerFactory creates a factory optimized for development
func NewDevLoggerFactory() *LoggerFactory {
	return NewLoggerFactory(DevConfig())
}

// NewProductionLoggerFactory creates a factory optimized for production
func NewProductionLoggerFactory() *LoggerFactory {
	return NewLoggerFactory(ProductionConfig())
}

// Convenient typed logger creation methods

// NewAPILogger creates a logger optimized for API usage with specified caller skip
func NewAPILogger(name string, callerSkip int) (Logger, error) {
	factory := NewAPILoggerFactory()
	return factory.New(
		WithName(name),
		WithCallerSkip(callerSkip),
		WithLoggerType(ZapLogger),
	)
}

// NewKratosLogger creates a logger optimized for Kratos usage
func NewKratosLogger(name string, callerSkip int) (Logger, error) {
	factory := NewDefaultLoggerFactory()
	return factory.New(
		WithName(name),
		WithCallerSkip(callerSkip),
		WithLoggerType(KratosLogger),
	)
}

// NewServiceLogger creates a logger for business services
func NewServiceLogger(serviceName, moduleName string, callerSkip int) (Logger, error) {
	factory := NewDefaultLoggerFactory()
	return factory.New(
		WithName(fmt.Sprintf("%s-%s", serviceName, moduleName)),
		WithCallerSkip(callerSkip),
		WithLoggerService(serviceName, moduleName),
		WithLoggerType(ZapLogger),
	)
}

// NewDatabaseLogger creates a logger optimized for database operations
func NewDatabaseLogger(dbName string, callerSkip int) (Logger, error) {
	factory := NewDefaultLoggerFactory()
	return factory.New(
		WithName(fmt.Sprintf("db-%s", dbName)),
		WithCallerSkip(callerSkip),
		WithLoggerFields(map[string]interface{}{
			"component": "database",
			"db_name":   dbName,
		}),
		WithLoggerType(ZapLogger),
	)
}

// ZapLoggerImpl implements Logger interface using zap
type ZapLoggerImpl struct {
	logger     *zap.Logger
	config     *Config
	name       string
	callerSkip int
}

// NewLogger creates a new zap-based logger implementation
func NewLogger(config *Config) (Logger, error) {
	return NewLoggerWithName("", config)
}

// NewLoggerWithName creates a new zap-based logger with a name
func NewLoggerWithName(name string, config *Config) (Logger, error) {
	return NewLoggerWithCallerSkip(name, config, 0)
}

// NewLoggerWithCallerSkip creates a new zap-based logger with a name and caller skip
func NewLoggerWithCallerSkip(name string, config *Config, callerSkip int) (Logger, error) {
	if config == nil {
		config = DefaultConfig()
	}

	level, err := zapcore.ParseLevel(config.Level)
	if err != nil {
		return nil, err
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   customCallerEncoder,
	}

	var encoder zapcore.Encoder
	switch config.Format {
	case "json":
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	case "console":
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	default:
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	writeSyncers := getWriteSyncers(config.OutputPaths, config.Rotation)
	multiWriteSyncer := zapcore.NewMultiWriteSyncer(writeSyncers...)
	normalCore := zapcore.NewCore(encoder, multiWriteSyncer, level)

	var core zapcore.Core
	if len(config.ErrorOutputPaths) > 0 {
		errorWriteSyncers := getWriteSyncers(config.ErrorOutputPaths, config.Rotation)
		errorMultiWriteSyncer := zapcore.NewMultiWriteSyncer(errorWriteSyncers...)
		errorCore := zapcore.NewCore(encoder, errorMultiWriteSyncer, zapcore.ErrorLevel)

		// 使用 Tee 将普通日志和错误日志结合
		core = zapcore.NewTee(normalCore, errorCore)
	} else {
		// 如果没有配置错误输出路径，只使用普通 core
		core = normalCore
	}

	options := []zap.Option{}
	if !config.DisableCaller {
		options = append(options, zap.AddCaller())
		if callerSkip > 0 {
			options = append(options, zap.AddCallerSkip(callerSkip))
		}
	}
	if !config.DisableStacktrace {
		options = append(options, zap.AddStacktrace(zapcore.ErrorLevel))
	}

	logger := zap.New(core, options...)
	if name != "" {
		logger = logger.Named(name)
	}

	return &ZapLoggerImpl{
		logger:     logger,
		config:     config,
		name:       name,
		callerSkip: callerSkip,
	}, nil
}

// customCallerEncoder encodes caller information with 3 path segments
func customCallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	if !caller.Defined {
		enc.AppendString("undefined")
		return
	}

	parts := strings.Split(caller.File, "/")
	var pathSegments []string

	if len(parts) >= 3 {
		pathSegments = parts[len(parts)-3:]
	} else {
		pathSegments = parts
	}

	trimmed := strings.Join(pathSegments, "/")
	enc.AppendString(fmt.Sprintf("%s:%d", trimmed, caller.Line))
}

// getWriteSyncers converts output paths to WriteSyncers
func getWriteSyncers(paths []string, rotationConfig *RotationConfig) []zapcore.WriteSyncer {
	var syncers []zapcore.WriteSyncer
	for _, path := range paths {
		switch path {
		case "stdout":
			syncers = append(syncers, zapcore.AddSync(os.Stdout))
		case "stderr":
			syncers = append(syncers, zapcore.AddSync(os.Stderr))
		default:
			if rotationConfig != nil && isLogFile(path) {
				lumberjackLogger := &lumberjack.Logger{
					Filename:   path,
					MaxSize:    rotationConfig.MaxSize,
					MaxBackups: rotationConfig.MaxBackups,
					MaxAge:     rotationConfig.MaxAge,
					Compress:   rotationConfig.Compress,
				}
				syncers = append(syncers, zapcore.AddSync(lumberjackLogger))
			} else {
				if file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644); err == nil {
					syncers = append(syncers, zapcore.AddSync(file))
				}
			}
		}
	}
	return syncers
}

// isLogFile checks if the path is a file path
func isLogFile(path string) bool {
	return filepath.IsAbs(path) || (path != "stdout" && path != "stderr" && filepath.Ext(path) != "")
}

// Logger implementation methods
func (l *ZapLoggerImpl) Debug(ctx context.Context, msg string, fields ...Field) {
	l.logWithContext(ctx, zapcore.DebugLevel, msg, fields...)
}

func (l *ZapLoggerImpl) Info(ctx context.Context, msg string, fields ...Field) {
	l.logWithContext(ctx, zapcore.InfoLevel, msg, fields...)
}

func (l *ZapLoggerImpl) Warn(ctx context.Context, msg string, fields ...Field) {
	l.logWithContext(ctx, zapcore.WarnLevel, msg, fields...)
}

func (l *ZapLoggerImpl) Error(ctx context.Context, msg string, fields ...Field) {
	l.logWithContext(ctx, zapcore.ErrorLevel, msg, fields...)
}

func (l *ZapLoggerImpl) DPanic(ctx context.Context, msg string, fields ...Field) {
	l.logWithContext(ctx, zapcore.DPanicLevel, msg, fields...)
}

func (l *ZapLoggerImpl) Panic(ctx context.Context, msg string, fields ...Field) {
	l.logWithContext(ctx, zapcore.PanicLevel, msg, fields...)
}

func (l *ZapLoggerImpl) Fatal(ctx context.Context, msg string, fields ...Field) {
	l.logWithContext(ctx, zapcore.FatalLevel, msg, fields...)
}

// logWithContext is a helper method to log with context and trace ID
func (l *ZapLoggerImpl) logWithContext(ctx context.Context, level zapcore.Level, msg string, fields ...Field) {
	// Extract OpenTelemetry trace information first (for better field ordering)
	traceID := trace.TraceIDFromContext(ctx)
	spanID := trace.SpanIDFromContext(ctx)

	// Start with trace fields, then add user fields
	var zapFields []zap.Field
	if traceID != "" {
		zapFields = append(zapFields, zap.String("trace_id", traceID))
	}
	if spanID != "" {
		zapFields = append(zapFields, zap.String("span_id", spanID))
	}

	// Add user-provided fields after trace fields
	zapFields = append(zapFields, ToZapFields(fields)...)

	switch level {
	case zapcore.DebugLevel:
		l.logger.Debug(msg, zapFields...)
	case zapcore.InfoLevel:
		l.logger.Info(msg, zapFields...)
	case zapcore.WarnLevel:
		l.logger.Warn(msg, zapFields...)
	case zapcore.ErrorLevel:
		l.logger.Error(msg, zapFields...)
	case zapcore.DPanicLevel:
		l.logger.DPanic(msg, zapFields...)
	case zapcore.PanicLevel:
		l.logger.Panic(msg, zapFields...)
	case zapcore.FatalLevel:
		l.logger.Fatal(msg, zapFields...)
	}
}

func (l *ZapLoggerImpl) Enabled(level Level) bool {
	return l.logger.Core().Enabled(l.convertLevel(level))
}

func (l *ZapLoggerImpl) convertLevel(level Level) zapcore.Level {
	switch level {
	case DebugLevel:
		return zapcore.DebugLevel
	case InfoLevel:
		return zapcore.InfoLevel
	case WarnLevel:
		return zapcore.WarnLevel
	case ErrorLevel:
		return zapcore.ErrorLevel
	case DPanicLevel:
		return zapcore.DPanicLevel
	case PanicLevel:
		return zapcore.PanicLevel
	case FatalLevel:
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func (l *ZapLoggerImpl) WithContext(ctx context.Context) Logger {
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
		return &ZapLoggerImpl{
			logger:     l.logger.With(fields...),
			config:     l.config,
			name:       l.name,
			callerSkip: l.callerSkip,
		}
	}
	return l
}

func (l *ZapLoggerImpl) WithTraceID(traceID string) Logger {
	return &ZapLoggerImpl{
		logger:     l.logger.With(zap.String("trace_id", traceID)),
		config:     l.config,
		name:       l.name,
		callerSkip: l.callerSkip,
	}
}

func (l *ZapLoggerImpl) WithService(service, module string) Logger {
	return &ZapLoggerImpl{
		logger:     l.logger.With(zap.String("service", service), zap.String("module", module)),
		config:     l.config,
		name:       l.name,
		callerSkip: l.callerSkip,
	}
}

func (l *ZapLoggerImpl) WithFields(fields map[string]interface{}) Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return &ZapLoggerImpl{
		logger:     l.logger.With(zapFields...),
		config:     l.config,
		name:       l.name,
		callerSkip: l.callerSkip,
	}
}

func (l *ZapLoggerImpl) WithField(key string, value interface{}) Logger {
	return &ZapLoggerImpl{
		logger:     l.logger.With(zap.Any(key, value)),
		config:     l.config,
		name:       l.name,
		callerSkip: l.callerSkip,
	}
}

func (l *ZapLoggerImpl) Sugar() SugarLogger {
	return &ZapSugarLoggerImpl{
		sugar:  l.logger.Sugar(),
		logger: l,
	}
}

func (l *ZapLoggerImpl) Sync() error {
	return l.logger.Sync()
}

func (l *ZapLoggerImpl) Clone() Logger {
	return &ZapLoggerImpl{
		logger:     l.logger,
		config:     l.config,
		name:       l.name,
		callerSkip: l.callerSkip,
	}
}

func (l *ZapLoggerImpl) GetZapLogger() *zap.Logger {
	return l.logger
}

// ZapSugarLoggerImpl implements SugarLogger interface using zap.SugaredLogger
type ZapSugarLoggerImpl struct {
	sugar  *zap.SugaredLogger
	logger *ZapLoggerImpl
}

func (s *ZapSugarLoggerImpl) Debugf(template string, args ...interface{}) {
	s.sugar.Debugf(template, args...)
}

func (s *ZapSugarLoggerImpl) Infof(template string, args ...interface{}) {
	s.sugar.Infof(template, args...)
}

func (s *ZapSugarLoggerImpl) Warnf(template string, args ...interface{}) {
	s.sugar.Warnf(template, args...)
}

func (s *ZapSugarLoggerImpl) Errorf(template string, args ...interface{}) {
	s.sugar.Errorf(template, args...)
}

func (s *ZapSugarLoggerImpl) DPanicf(template string, args ...interface{}) {
	s.sugar.DPanicf(template, args...)
}

func (s *ZapSugarLoggerImpl) Panicf(template string, args ...interface{}) {
	s.sugar.Panicf(template, args...)
}

func (s *ZapSugarLoggerImpl) Fatalf(template string, args ...interface{}) {
	s.sugar.Fatalf(template, args...)
}

func (s *ZapSugarLoggerImpl) Debug(args ...interface{}) {
	s.sugar.Debug(args...)
}

func (s *ZapSugarLoggerImpl) Info(args ...interface{}) {
	s.sugar.Info(args...)
}

func (s *ZapSugarLoggerImpl) Warn(args ...interface{}) {
	s.sugar.Warn(args...)
}

func (s *ZapSugarLoggerImpl) Error(args ...interface{}) {
	s.sugar.Error(args...)
}

func (s *ZapSugarLoggerImpl) DPanic(args ...interface{}) {
	s.sugar.DPanic(args...)
}

func (s *ZapSugarLoggerImpl) Panic(args ...interface{}) {
	s.sugar.Panic(args...)
}

func (s *ZapSugarLoggerImpl) Fatal(args ...interface{}) {
	s.sugar.Fatal(args...)
}

func (s *ZapSugarLoggerImpl) With(args ...interface{}) SugarLogger {
	return &ZapSugarLoggerImpl{
		sugar:  s.sugar.With(args...),
		logger: s.logger,
	}
}

func (s *ZapSugarLoggerImpl) WithOptions(opts ...interface{}) SugarLogger {
	return s
}

func (s *ZapSugarLoggerImpl) Enabled(level Level) bool {
	return s.logger.Enabled(level)
}

func (s *ZapSugarLoggerImpl) Sync() error {
	return s.sugar.Sync()
}

func (s *ZapSugarLoggerImpl) Desugar() Logger {
	return &ZapLoggerImpl{
		logger:     s.sugar.Desugar(),
		config:     s.logger.config,
		name:       s.logger.name,
		callerSkip: s.logger.callerSkip,
	}
}

// Helper functions for creating fields
func String(key, val string) Field {
	return Field{Key: key, Value: val}
}

func Int(key string, val int) Field {
	return Field{Key: key, Value: val}
}

func Int8(key string, val int8) Field {
	return Field{Key: key, Value: val}
}

func Int16(key string, val int16) Field {
	return Field{Key: key, Value: val}
}

func Int32(key string, val int32) Field {
	return Field{Key: key, Value: val}
}

func Int64(key string, val int64) Field {
	return Field{Key: key, Value: val}
}

func Uint(key string, val uint) Field {
	return Field{Key: key, Value: val}
}

func Uint8(key string, val uint8) Field {
	return Field{Key: key, Value: val}
}

func Uint16(key string, val uint16) Field {
	return Field{Key: key, Value: val}
}

func Uint32(key string, val uint32) Field {
	return Field{Key: key, Value: val}
}

func Uint64(key string, val uint64) Field {
	return Field{Key: key, Value: val}
}

func Uintptr(key string, val uintptr) Field {
	return Field{Key: key, Value: val}
}

func Float32(key string, val float32) Field {
	return Field{Key: key, Value: val}
}

func Float64(key string, val float64) Field {
	return Field{Key: key, Value: val}
}

func Complex64(key string, val complex64) Field {
	return Field{Key: key, Value: val}
}

func Complex128(key string, val complex128) Field {
	return Field{Key: key, Value: val}
}

func Bool(key string, val bool) Field {
	return Field{Key: key, Value: val}
}

func Duration(key string, val time.Duration) Field {
	return Field{Key: key, Value: val}
}

func Time(key string, val time.Time) Field {
	return Field{Key: key, Value: val}
}

// nilField returns a field that represents a nil value
func nilField(key string) Field {
	return Field{Key: key, Value: nil}
}

// Pointer type field functions
func Intp(key string, val *int) Field {
	if val == nil {
		return nilField(key)
	}
	return Int(key, *val)
}

func Int8p(key string, val *int8) Field {
	if val == nil {
		return nilField(key)
	}
	return Int8(key, *val)
}

func Int16p(key string, val *int16) Field {
	if val == nil {
		return nilField(key)
	}
	return Int16(key, *val)
}

func Int32p(key string, val *int32) Field {
	if val == nil {
		return nilField(key)
	}
	return Int32(key, *val)
}

func Int64p(key string, val *int64) Field {
	if val == nil {
		return nilField(key)
	}
	return Int64(key, *val)
}

func Uintp(key string, val *uint) Field {
	if val == nil {
		return nilField(key)
	}
	return Uint(key, *val)
}

func Uint8p(key string, val *uint8) Field {
	if val == nil {
		return nilField(key)
	}
	return Uint8(key, *val)
}

func Uint16p(key string, val *uint16) Field {
	if val == nil {
		return nilField(key)
	}
	return Uint16(key, *val)
}

func Uint32p(key string, val *uint32) Field {
	if val == nil {
		return nilField(key)
	}
	return Uint32(key, *val)
}

func Uint64p(key string, val *uint64) Field {
	if val == nil {
		return nilField(key)
	}
	return Uint64(key, *val)
}

func Float32p(key string, val *float32) Field {
	if val == nil {
		return nilField(key)
	}
	return Float32(key, *val)
}

func Float64p(key string, val *float64) Field {
	if val == nil {
		return nilField(key)
	}
	return Float64(key, *val)
}

func Complex64p(key string, val *complex64) Field {
	if val == nil {
		return nilField(key)
	}
	return Complex64(key, *val)
}

func Complex128p(key string, val *complex128) Field {
	if val == nil {
		return nilField(key)
	}
	return Complex128(key, *val)
}

func Boolp(key string, val *bool) Field {
	if val == nil {
		return nilField(key)
	}
	return Bool(key, *val)
}

func Durationp(key string, val *time.Duration) Field {
	if val == nil {
		return nilField(key)
	}
	return Duration(key, *val)
}

func Timep(key string, val *time.Time) Field {
	if val == nil {
		return nilField(key)
	}
	return Time(key, *val)
}

func Stringp(key string, val *string) Field {
	if val == nil {
		return nilField(key)
	}
	return String(key, *val)
}

func Any(key string, val interface{}) Field {
	return Field{Key: key, Value: val}
}

func Err(err error) Field {
	return Field{Key: "error", Value: err}
}

func Stack(key string) Field {
	return Field{Key: key, Value: "stack_trace"}
}

// Array type field functions
func Strings(key string, val []string) Field {
	return Field{Key: key, Value: val}
}

func Ints(key string, val []int) Field {
	return Field{Key: key, Value: val}
}

func Int8s(key string, val []int8) Field {
	return Field{Key: key, Value: val}
}

func Int16s(key string, val []int16) Field {
	return Field{Key: key, Value: val}
}

func Int32s(key string, val []int32) Field {
	return Field{Key: key, Value: val}
}

func Int64s(key string, val []int64) Field {
	return Field{Key: key, Value: val}
}

func Uints(key string, val []uint) Field {
	return Field{Key: key, Value: val}
}

func Uint8s(key string, val []uint8) Field {
	return Field{Key: key, Value: val}
}

func Uint16s(key string, val []uint16) Field {
	return Field{Key: key, Value: val}
}

func Uint32s(key string, val []uint32) Field {
	return Field{Key: key, Value: val}
}

func Uint64s(key string, val []uint64) Field {
	return Field{Key: key, Value: val}
}

func Float32s(key string, val []float32) Field {
	return Field{Key: key, Value: val}
}

func Float64s(key string, val []float64) Field {
	return Field{Key: key, Value: val}
}

func Complex64s(key string, val []complex64) Field {
	return Field{Key: key, Value: val}
}

func Complex128s(key string, val []complex128) Field {
	return Field{Key: key, Value: val}
}

func Bools(key string, val []bool) Field {
	return Field{Key: key, Value: val}
}

func Durations(key string, val []time.Duration) Field {
	return Field{Key: key, Value: val}
}

func Times(key string, val []time.Time) Field {
	return Field{Key: key, Value: val}
}

// Convert our Field to zap.Field
func (f Field) ToZapField() zap.Field {
	return zap.Any(f.Key, f.Value)
}

// Convert multiple fields to zap fields
func ToZapFields(fields []Field) []zap.Field {
	zapFields := make([]zap.Field, len(fields))
	for i, field := range fields {
		zapFields[i] = field.ToZapField()
	}
	return zapFields
}

// Convert zap.Field to our Field
func FromZapField(zapField zap.Field) Field {
	return Field{
		Key:   zapField.Key,
		Value: zapField.Interface,
	}
}
