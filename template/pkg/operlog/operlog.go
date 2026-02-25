package operlog

import "context"

type logMetaKey struct{}

// LogMeta holds custom log information
type LogMeta struct {
	Title        string
	BusinessType int32
	OperName     string
	OperParam    string
	JsonResult   string
}

// SetLogTitle sets the operation title
func SetLogTitle(ctx context.Context, title string) {
	if meta, ok := ctx.Value(logMetaKey{}).(*LogMeta); ok {
		meta.Title = title
	}
}

// SetLogBusinessType sets the business type
func SetLogBusinessType(ctx context.Context, businessType int32) {
	if meta, ok := ctx.Value(logMetaKey{}).(*LogMeta); ok {
		meta.BusinessType = businessType
	}
}

// SetLogOperParam sets custom operation parameters
func SetLogOperParam(ctx context.Context, param string) {
	if meta, ok := ctx.Value(logMetaKey{}).(*LogMeta); ok {
		meta.OperParam = param
	}
}

// SetLogResult sets custom operation result
func SetLogResult(ctx context.Context, result string) {
	if meta, ok := ctx.Value(logMetaKey{}).(*LogMeta); ok {
		meta.JsonResult = result
	}
}

// NewContext injects LogMeta into context
func NewContext(ctx context.Context) (context.Context, *LogMeta) {
	meta := &LogMeta{}
	return context.WithValue(ctx, logMetaKey{}, meta), meta
}
