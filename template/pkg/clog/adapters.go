package clog

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	kratosLog "github.com/go-kratos/kratos/v2/log"
	"go.uber.org/fx/fxevent"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

// Fx Adapter
// =============================================================================

// FxLoggerAdapter adapts our Logger interface to Fx's logger interface

type FxLoggerAdapter struct {
	logger Logger
}

// NewFxLoggerAdapter creates a new Fx logger adapter
func NewFxLoggerAdapter(logger Logger) *FxLoggerAdapter {
	return &FxLoggerAdapter{
		logger: logger,
	}
}

// NewFxLoggerAdapterWithCallerSkip creates a new FxLoggerAdapter with caller skip
func NewFxLoggerAdapterWithCallerSkip(config *Config, callerSkip int) (*FxLoggerAdapter, error) {
	logger, err := NewLoggerWithCallerSkip("fx", config, callerSkip)
	if err != nil {
		return nil, err
	}
	return &FxLoggerAdapter{
		logger: logger,
	}, nil
}

// LogEvent implements fxevent.Logger interface
func (l *FxLoggerAdapter) LogEvent(event fxevent.Event) {
	ctx := context.Background()

	switch e := event.(type) {
	case *fxevent.OnStartExecuting:
		l.logger.Info(ctx, fmt.Sprintf("Starting: %s", e.FunctionName), String("caller", e.CallerName))
	case *fxevent.OnStartExecuted:
		if e.Err != nil {
			l.logger.Error(ctx, fmt.Sprintf("Failed to start: %s", e.FunctionName), Err(e.Err), String("caller", e.CallerName))
		} else {
			l.logger.Info(ctx, fmt.Sprintf("Started: %s", e.FunctionName), String("caller", e.CallerName), Duration("runtime", e.Runtime))
		}
	case *fxevent.OnStopExecuting:
		l.logger.Info(ctx, fmt.Sprintf("Stopping: %s", e.FunctionName), String("caller", e.CallerName))
	case *fxevent.OnStopExecuted:
		if e.Err != nil {
			l.logger.Error(ctx, fmt.Sprintf("Failed to stop: %s", e.FunctionName), Err(e.Err), String("caller", e.CallerName))
		} else {
			l.logger.Info(ctx, fmt.Sprintf("Stopped: %s", e.FunctionName), String("caller", e.CallerName), Duration("runtime", e.Runtime))
		}
	case *fxevent.Supplied:
		if e.Err != nil {
			l.logger.Error(ctx, "Failed to supply", Err(e.Err), String("type", e.TypeName))
		} else {
			l.logger.Info(ctx, "Supplied", String("type", e.TypeName))
		}
	case *fxevent.Provided:
		if e.Err != nil {
			l.logger.Error(ctx, "Failed to provide", Err(e.Err), String("types", fmt.Sprintf("%v", e.OutputTypeNames)))
		} else {
			l.logger.Info(ctx, "Provided", String("types", fmt.Sprintf("%v", e.OutputTypeNames)))
		}
	}
}

// GORM Adapter
// =============================================================================

// GormLoggerAdapter adapts our Logger interface to GORM's logger interface
type GormLoggerAdapter struct {
	logger        Logger
	logLevel      gormLogger.LogLevel
	slowThreshold time.Duration
}

// NewGormLoggerAdapter creates a new GORM logger adapter
func NewGormLoggerAdapter(logger Logger) gormLogger.Interface {
	return &GormLoggerAdapter{
		logger:        logger,
		logLevel:      gormLogger.Info,
		slowThreshold: 200 * time.Millisecond,
	}
}

// NewGormLoggerAdapterWithCallerSkip creates a new GORM logger adapter with caller skip
func NewGormLoggerAdapterWithCallerSkip(config *Config, callerSkip int) (gormLogger.Interface, error) {
	logger, err := NewLoggerWithCallerSkip("", config, callerSkip)
	if err != nil {
		return nil, err
	}
	return &GormLoggerAdapter{
		logger:        logger,
		logLevel:      gormLogger.Info,
		slowThreshold: 200 * time.Millisecond,
	}, nil
}

// NewGormLoggerAdapterWithConfig creates a new GORM logger adapter with configuration
func NewGormLoggerAdapterWithConfig(logger Logger, level gormLogger.LogLevel, slowThreshold time.Duration) gormLogger.Interface {
	return &GormLoggerAdapter{
		logger:        logger,
		logLevel:      level,
		slowThreshold: slowThreshold,
	}
}

// LogMode implements gorm logger.Interface
func (l *GormLoggerAdapter) LogMode(level gormLogger.LogLevel) gormLogger.Interface {
	return &GormLoggerAdapter{
		logger:        l.logger,
		logLevel:      level,
		slowThreshold: l.slowThreshold,
	}
}

// Info implements gorm logger.Interface
func (l *GormLoggerAdapter) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= gormLogger.Info {
		l.logger.Info(ctx, msg, Any("data", data))
	}
}

// Warn implements gorm logger.Interface
func (l *GormLoggerAdapter) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= gormLogger.Warn {
		l.logger.Warn(ctx, msg, Any("data", data))
	}
}

// Error implements gorm logger.Interface
func (l *GormLoggerAdapter) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= gormLogger.Error {
		l.logger.Error(ctx, msg, Any("data", data))
	}
}

// Trace implements gorm logger.Interface for SQL tracing
func (l *GormLoggerAdapter) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.logLevel <= gormLogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := []Field{
		String("sql", sql),
		Duration("elapsed", elapsed),
		Int64("rows", rows),
	}

	// 动态查找业务代码位置
	// 从调用栈中找到第一个非 GORM/Gen/DAL 的代码位置
	callerFile, callerLine := l.findBusinessCodeCaller()

	// 构建包含 caller 信息的消息
	var msgPrefix string
	if callerFile != "" {
		msgPrefix = fmt.Sprintf("%s:%d GORM ", callerFile, callerLine)
	}

	switch {
	case err != nil && l.logLevel >= gormLogger.Error && (!errors.Is(err, gorm.ErrRecordNotFound)):
		l.logger.Error(ctx, msgPrefix+"SQL execution error", append(fields, Err(err))...)
	case elapsed > l.slowThreshold && l.slowThreshold != 0 && l.logLevel >= gormLogger.Warn:
		l.logger.Warn(ctx, msgPrefix+"Slow SQL query", append(fields, String("threshold", l.slowThreshold.String()))...)
	case l.logLevel == gormLogger.Info:
		l.logger.Info(ctx, msgPrefix+"SQL query", fields...)
	}
}

// findBusinessCodeCaller 查找业务代码的调用位置
// 返回文件路径和行号，如果没找到则返回空字符串和0
func (l *GormLoggerAdapter) findBusinessCodeCaller() (string, int) {
	// 从 Frame 2 开始查找（跳过 runtime.Caller 和 findBusinessCodeCaller）
	for i := 2; i < 30; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		// 标准化路径为正斜杠（跨平台兼容）
		file = filepath.ToSlash(file)

		// 跳过框架和基础设施代码
		skipPatterns := []string{
			"gorm.io/",
			"gorm.io/gen/",
			"/internal/data/dal/",
			"/pkg/clog/",
			"/go-kratos/",
			"runtime/",
			"reflect/",
		}

		shouldSkip := false
		for _, pattern := range skipPatterns {
			if strings.Contains(file, pattern) {
				shouldSkip = true
				break
			}
		}

		if shouldSkip {
			continue
		}

		// 找到第一个业务代码，返回简洁的路径
		return formatCallerPath(file), line
	}
	return "", 0
}

// formatCallerPath 格式化 caller 路径，只保留最后三层路径
func formatCallerPath(file string) string {
	// 按路径分隔符拆分
	parts := strings.Split(file, "/")

	// 只保留最后3层路径
	if len(parts) >= 3 {
		return strings.Join(parts[len(parts)-3:], "/")
	}

	// 如果不足3层，返回完整路径
	return strings.Join(parts, "/")
}

// GormConfig holds configuration for GORM logger adapter
type GormConfig struct {
	LogLevel                  gormLogger.LogLevel `yaml:"log_level" json:"log_level"`
	SlowThreshold             time.Duration       `yaml:"slow_threshold" json:"slow_threshold"`
	IgnoreRecordNotFoundError bool                `yaml:"ignore_record_not_found_error" json:"ignore_record_not_found_error"`
	EnableParamsFilter        bool                `yaml:"enable_params_filter" json:"enable_params_filter"`
	Colorful                  bool                `yaml:"colorful" json:"colorful"`
}

// DefaultGormConfig returns default GORM logger configuration
func DefaultGormConfig() *GormConfig {
	return &GormConfig{
		LogLevel:                  gormLogger.Warn,
		SlowThreshold:             200 * time.Millisecond,
		IgnoreRecordNotFoundError: false,
		EnableParamsFilter:        false,
		Colorful:                  false,
	}
}

// NewGormLoggerAdapterFromConfig creates GORM logger adapter from configuration
func NewGormLoggerAdapterFromConfig(logger Logger, config *GormConfig) gormLogger.Interface {
	if config == nil {
		config = DefaultGormConfig()
	}

	return &GormLoggerAdapter{
		logger:        logger,
		logLevel:      config.LogLevel,
		slowThreshold: config.SlowThreshold,
	}
}

// Kratos Adapter
// =============================================================================

// KratosLoggerAdapter adapts our Logger interface to Kratos's logger interface
type KratosLoggerAdapter struct {
	logger Logger
	ctx    context.Context
}

// NewKratosLoggerAdapter creates a new Kratos logger adapter
func NewKratosLoggerAdapter(logger Logger) kratosLog.Logger {
	return &KratosLoggerAdapter{
		logger: logger,
		ctx:    context.Background(),
	}
}

// NewKratosLoggerAdapterWithCallerSkip creates a new Kratos logger adapter with caller skip
func NewKratosLoggerAdapterWithCallerSkip(config *Config, callerSkip int) (kratosLog.Logger, error) {
	logger, err := NewLoggerWithCallerSkip("kratos", config, callerSkip)
	if err != nil {
		return nil, err
	}
	return &KratosLoggerAdapter{
		logger: logger,
		ctx:    context.Background(),
	}, nil
}

// NewKratosLoggerAdapterWithContext creates a new Kratos logger adapter with context
func NewKratosLoggerAdapterWithContext(logger Logger, ctx context.Context) kratosLog.Logger {
	return &KratosLoggerAdapter{
		logger: logger,
		ctx:    ctx,
	}
}

// Log implements kratosLog.Logger interface
func (l *KratosLoggerAdapter) Log(level kratosLog.Level, keyvals ...interface{}) error {
	if len(keyvals) == 0 {
		return nil
	}

	if len(keyvals)%2 != 0 {
		keyvals = append(keyvals, "")
	}

	// Convert keyvals to our fields
	fields := make([]Field, 0, len(keyvals)/2)
	msg := ""

	for i := 0; i < len(keyvals); i += 2 {
		key := fmt.Sprint(keyvals[i])
		val := keyvals[i+1]

		if key == "msg" || key == "message" {
			msg = fmt.Sprint(val)
		} else {
			fields = append(fields, Any(key, val))
		}
	}

	// Use the stored context
	ctx := l.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	// Log with appropriate level
	switch level {
	case kratosLog.LevelDebug:
		l.logger.Debug(ctx, msg, fields...)
	case kratosLog.LevelInfo:
		l.logger.Info(ctx, msg, fields...)
	case kratosLog.LevelWarn:
		l.logger.Warn(ctx, msg, fields...)
	case kratosLog.LevelError:
		l.logger.Error(ctx, msg, fields...)
	case kratosLog.LevelFatal:
		l.logger.Fatal(ctx, msg, fields...)
	default:
		l.logger.Info(ctx, msg, fields...)
	}

	return nil
}

// KratosContextualLoggerAdapter wraps KratosLoggerAdapter to work with Kratos WithContext
type KratosContextualLoggerAdapter struct {
	*KratosLoggerAdapter
	ctx context.Context
}

// Log implements kratosLog.Logger interface for KratosContextualLoggerAdapter
func (l *KratosContextualLoggerAdapter) Log(level kratosLog.Level, keyvals ...interface{}) error {
	if len(keyvals) == 0 {
		return nil
	}

	if len(keyvals)%2 != 0 {
		keyvals = append(keyvals, "")
	}

	// Convert keyvals to our fields
	fields := make([]Field, 0, len(keyvals)/2)
	msg := ""

	for i := 0; i < len(keyvals); i += 2 {
		key := fmt.Sprint(keyvals[i])
		val := keyvals[i+1]

		if key == "msg" || key == "message" {
			msg = fmt.Sprint(val)
		} else {
			fields = append(fields, Any(key, val))
		}
	}

	// Use the stored context from Kratos WithContext
	ctx := l.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	switch level {
	case kratosLog.LevelDebug:
		l.KratosLoggerAdapter.logger.Debug(ctx, msg, fields...)
	case kratosLog.LevelInfo:
		l.KratosLoggerAdapter.logger.Info(ctx, msg, fields...)
	case kratosLog.LevelWarn:
		l.KratosLoggerAdapter.logger.Warn(ctx, msg, fields...)
	case kratosLog.LevelError:
		l.KratosLoggerAdapter.logger.Error(ctx, msg, fields...)
	case kratosLog.LevelFatal:
		l.KratosLoggerAdapter.logger.Fatal(ctx, msg, fields...)
	default:
		l.KratosLoggerAdapter.logger.Info(ctx, msg, fields...)
	}

	return nil
}

// WithContext creates a context-aware logger that works with Kratos WithContext
func (l *KratosLoggerAdapter) WithContext(ctx context.Context) kratosLog.Logger {
	return &KratosContextualLoggerAdapter{
		KratosLoggerAdapter: l,
		ctx:                 ctx,
	}
}

// KratosHelperLoggerAdapter provides additional helper methods for easier logging
type KratosHelperLoggerAdapter struct {
	*KratosLoggerAdapter
}

// NewKratosHelperLoggerAdapter creates a Kratos logger adapter with helper methods
func NewKratosHelperLoggerAdapter(logger Logger) *KratosHelperLoggerAdapter {
	return &KratosHelperLoggerAdapter{
		KratosLoggerAdapter: &KratosLoggerAdapter{
			logger: logger,
			ctx:    context.Background(),
		},
	}
}

// Helper methods for easier logging
func (l *KratosHelperLoggerAdapter) Infof(template string, args ...interface{}) {
	l.logger.Info(l.ctx, fmt.Sprintf(template, args...))
}

func (l *KratosHelperLoggerAdapter) Errorf(template string, args ...interface{}) {
	l.logger.Error(l.ctx, fmt.Sprintf(template, args...))
}

func (l *KratosHelperLoggerAdapter) Debugf(template string, args ...interface{}) {
	l.logger.Debug(l.ctx, fmt.Sprintf(template, args...))
}

func (l *KratosHelperLoggerAdapter) Warnf(template string, args ...interface{}) {
	l.logger.Warn(l.ctx, fmt.Sprintf(template, args...))
}

func (l *KratosHelperLoggerAdapter) Fatalf(template string, args ...interface{}) {
	l.logger.Fatal(l.ctx, fmt.Sprintf(template, args...))
}

// WithContext returns a new logger with the given context
func (l *KratosHelperLoggerAdapter) WithContext(ctx context.Context) *KratosHelperLoggerAdapter {
	return &KratosHelperLoggerAdapter{
		KratosLoggerAdapter: &KratosLoggerAdapter{
			logger: l.logger,
			ctx:    ctx,
		},
	}
}

// GetLogger returns the underlying Logger for advanced usage
func (l *KratosHelperLoggerAdapter) GetLogger() Logger {
	return l.logger
}

// KratosConfig holds configuration for Kratos logger adapter
type KratosConfig struct {
	EnableHelper    bool `yaml:"enable_helper" json:"enable_helper"`
	DefaultContext  bool `yaml:"default_context" json:"default_context"`
	EnableWithValue bool `yaml:"enable_with_value" json:"enable_with_value"`
}

// DefaultKratosConfig returns default Kratos logger configuration
func DefaultKratosConfig() *KratosConfig {
	return &KratosConfig{
		EnableHelper:    true,
		DefaultContext:  true,
		EnableWithValue: false,
	}
}

// NewKratosLoggerAdapterFromConfig creates Kratos logger adapter from configuration
func NewKratosLoggerAdapterFromConfig(logger Logger, config *KratosConfig) kratosLog.Logger {
	if config == nil {
		config = DefaultKratosConfig()
	}

	base := &KratosLoggerAdapter{
		logger: logger,
		ctx:    context.Background(),
	}

	if config.EnableHelper {
		return &KratosHelperLoggerAdapter{
			KratosLoggerAdapter: base,
		}
	}

	return base
}

// Casbin Adapter
// =============================================================================

// CasbinLoggerAdapter adapts our Logger interface to Casbin's logger interface
type CasbinLoggerAdapter struct {
	logger  Logger
	enabled bool
}

// NewCasbinZapAdapter creates a new Casbin logger adapter
func NewCasbinZapAdapter(logger Logger) *CasbinLoggerAdapter {
	return &CasbinLoggerAdapter{
		logger:  logger,
		enabled: true,
	}
}

// NewCasbinZapAdapterWithCallerSkip creates a new Casbin logger adapter with caller skip
func NewCasbinZapAdapterWithCallerSkip(config *Config, callerSkip int) (*CasbinLoggerAdapter, error) {
	logger, err := NewLoggerWithCallerSkip("casbin", config, callerSkip)
	if err != nil {
		return nil, err
	}
	return &CasbinLoggerAdapter{
		logger:  logger,
		enabled: true,
	}, nil
}

// EnableLog implements casbin log.Logger interface - controls whether to print messages
func (l *CasbinLoggerAdapter) EnableLog(enable bool) {
	l.enabled = enable
}

// IsEnabled implements casbin log.Logger interface - returns if logger is enabled
func (l *CasbinLoggerAdapter) IsEnabled() bool {
	return l.enabled
}

// LogModel implements casbin log.Logger interface - logs info related to model
func (l *CasbinLoggerAdapter) LogModel(model [][]string) {
	if !l.enabled {
		return
	}
	ctx := context.Background()
	l.logger.Debug(ctx, "Casbin model loaded", Any("model", model))
}

// LogEnforce implements casbin log.Logger interface - logs info related to enforce
func (l *CasbinLoggerAdapter) LogEnforce(matcher string, request []interface{}, result bool, explains [][]string) {
	if !l.enabled {
		return
	}
	ctx := context.Background()
	l.logger.Debug(ctx, "Casbin enforce",
		String("matcher", matcher),
		Any("request", request),
		Bool("result", result),
		Any("explains", explains),
	)
}

// LogRole implements casbin log.Logger interface - logs info related to role
func (l *CasbinLoggerAdapter) LogRole(roles []string) {
	if !l.enabled {
		return
	}
	ctx := context.Background()
	l.logger.Debug(ctx, "Casbin role operation", Strings("roles", roles))
}

// LogPolicy implements casbin log.Logger interface - logs info related to policy
func (l *CasbinLoggerAdapter) LogPolicy(policy map[string][][]string) {
	if !l.enabled {
		return
	}
	ctx := context.Background()
	l.logger.Debug(ctx, "Casbin policy operation", Any("policy", policy))
}

// LogError implements casbin log.Logger interface - logs info related to error
func (l *CasbinLoggerAdapter) LogError(err error, msg ...string) {
	if !l.enabled {
		return
	}
	ctx := context.Background()
	message := "Casbin error"
	if len(msg) > 0 {
		message = msg[0]
	}
	l.logger.Error(ctx, message, Err(err))
}

// CasbinConfig holds configuration for Casbin logger adapter
type CasbinConfig struct {
	EnableLog  bool   `yaml:"enable_log" json:"enable_log"`
	LogLevel   string `yaml:"log_level" json:"log_level"`
	LogModel   bool   `yaml:"log_model" json:"log_model"`
	LogEnforce bool   `yaml:"log_enforce" json:"log_enforce"`
	LogRole    bool   `yaml:"log_role" json:"log_role"`
	LogPolicy  bool   `yaml:"log_policy" json:"log_policy"`
}

// DefaultCasbinConfig returns default Casbin logger configuration
func DefaultCasbinConfig() *CasbinConfig {
	return &CasbinConfig{
		EnableLog:  true,
		LogLevel:   "debug",
		LogModel:   false,
		LogEnforce: false,
		LogRole:    false,
		LogPolicy:  false,
	}
}

// NewCasbinZapAdapterFromConfig creates Casbin logger adapter from configuration
func NewCasbinZapAdapterFromConfig(logger Logger, config *CasbinConfig) *CasbinLoggerAdapter {
	if config == nil {
		config = DefaultCasbinConfig()
	}

	return &CasbinLoggerAdapter{
		logger:  logger,
		enabled: config.EnableLog,
	}
}
