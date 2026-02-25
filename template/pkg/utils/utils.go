package utils

import (
	"spark/pkg/utils/oss"

	"go.uber.org/fx"
)

// GetStringOrDefault safely dereferences a *string, returning the value or an empty string if nil.
func GetStringOrDefault(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}

// GetInt64OrDefault safely dereferences a *int64, returning the value or 0 if nil.
func GetInt64OrDefault(ptr *int64) int64 {
	if ptr != nil {
		return *ptr
	}
	return 0
}

// GetInt32OrDefault safely dereferences a *int32, returning the value or 0 if nil.
func GetInt32OrDefault(ptr *int32) int32 {
	if ptr != nil {
		return *ptr
	}
	return 0
}

// Module provides service layer dependencies
var Module = fx.Options(
	fx.Provide(
		oss.NewOssProvider,
		func(provider oss.OssProvider) oss.OssService {
			return provider
		},
	),
)
