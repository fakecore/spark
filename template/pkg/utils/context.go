package utils

import (
	"time"
)

type contextKey string

// GetTodayStartTimestamp 获取今天零点的时间戳（毫秒）- 使用UTC+8时区
func GetTodayStartTimestamp() int64 {
	// 使用 UTC+8 (中国标准时间)
	loc := GetChinaLocation()
	now := time.Now().In(loc)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	return todayStart.UnixMilli()
}
