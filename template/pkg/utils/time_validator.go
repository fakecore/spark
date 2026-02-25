package utils

import (
	"fmt"
	"time"
)

const (
	// 毫秒时间戳的范围 (13位数字)
	// 1000000000000 约为 2001-09-09
	// 9999999999999 约为 2286-11-20
	MinMillisecondTimestamp = 1000000000000
	MaxMillisecondTimestamp = 9999999999999

	// 秒时间戳的范围 (10位数字)
	// 1000000000 约为 2001-09-09
	// 9999999999 约为 2286-11-20
	MinSecondTimestamp = 1000000000
	MaxSecondTimestamp = 9999999999
)

// ValidateMillisecondTimestamp 验证时间戳是否为毫秒级格式
// 毫秒时间戳应该是13位数字，大约在 1000000000000 (2001年) 到 9999999999999 (2286年) 之间
func ValidateMillisecondTimestamp(timestamp int64) error {
	if timestamp < MinMillisecondTimestamp || timestamp > MaxMillisecondTimestamp {
		return fmt.Errorf("timestamp must be in milliseconds (13-digit number), got: %d", timestamp)
	}
	return nil
}

// ValidateSecondTimestamp 验证时间戳是否为秒级格式
// 秒时间戳应该是10位数字，大约在 1000000000 (2001年) 到 9999999999 (2286年) 之间
func ValidateSecondTimestamp(timestamp int64) error {
	if timestamp < MinSecondTimestamp || timestamp > MaxSecondTimestamp {
		return fmt.Errorf("timestamp must be in seconds (10-digit number), got: %d", timestamp)
	}
	return nil
}

// ValidateTimestampRange 验证时间戳范围是否合理
// startTime: 开始时间（毫秒）
// endTime: 结束时间（毫秒）
// maxDuration: 最大允许的时长（毫秒），例如 24小时 = 24 * 60 * 60 * 1000 = 86400000
func ValidateTimestampRange(startTime, endTime, maxDuration int64) error {
	// 验证开始时间
	if err := ValidateMillisecondTimestamp(startTime); err != nil {
		return fmt.Errorf("invalid start_time: %w", err)
	}

	// 验证结束时间
	if err := ValidateMillisecondTimestamp(endTime); err != nil {
		return fmt.Errorf("invalid end_time: %w", err)
	}

	// 验证结束时间应该在开始时间之后
	if endTime <= startTime {
		return fmt.Errorf("end_time (%d) must be after start_time (%d)", endTime, startTime)
	}

	// 验证时长不超过最大限制
	duration := endTime - startTime
	if maxDuration > 0 && duration > maxDuration {
		return fmt.Errorf("duration (%d ms) exceeds maximum allowed duration (%d ms)", duration, maxDuration)
	}

	return nil
}

// ConvertMillisecondToSecond 将毫秒时间戳转换为秒时间戳
func ConvertMillisecondToSecond(millisecond int64) int64 {
	return millisecond / 1000
}

// ConvertSecondToMillisecond 将秒时间戳转换为毫秒时间戳
func ConvertSecondToMillisecond(second int64) int64 {
	return second * 1000
}

// FormatMillisecondTimestamp 将毫秒时间戳格式化为可读字符串
func FormatMillisecondTimestamp(millisecond int64) string {
	t := time.UnixMilli(millisecond)
	return t.Format("2006-01-02 15:04:05")
}
