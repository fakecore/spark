package utils

import "time"

// GetChinaLocation 获取中国时区 (UTC+8)
func GetChinaLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		// 如果加载失败，使用固定的 UTC+8
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}

func ConvertTimeToUnix(t time.Time) int64 {
	return t.UnixMilli()
}

func ConvertUnixToTime(unix int64) time.Time {
	return time.UnixMilli(unix)
}

func ConvertTimeToUnixString(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func ConvertUnixToTimeString(unix int64) string {
	return time.UnixMilli(unix).Format("2006-01-02 15:04:05")
}

func ConvertTimeStringToUnix(timeString string) (int64, error) {
	t, err := time.Parse("2006-01-02 15:04:05", timeString)
	if err != nil {
		return 0, err
	}
	return t.UnixMilli(), nil
}

// NormalizeStartOfDay 将时间戳标准化为当天的00:00:00（毫秒级）
// 例如：2025-10-10 15:30:45 -> 2025-10-10 00:00:00
func NormalizeStartOfDay(timestamp int64) int64 {
	t := ConvertUnixToTime(timestamp)
	// 创建当天的00:00:00
	year, month, day := t.Date()
	location := t.Location()
	startOfDay := time.Date(year, month, day, 0, 0, 0, 0, location)
	return startOfDay.UnixMilli()
}

// NormalizeEndOfDay 将时间戳标准化为当天的23:59:59（毫秒级）
// 例如：2025-10-10 15:30:45 -> 2025-10-10 23:59:59
func NormalizeEndOfDay(timestamp int64) int64 {
	t := ConvertUnixToTime(timestamp)
	// 创建当天的23:59:59
	year, month, day := t.Date()
	location := t.Location()
	endOfDay := time.Date(year, month, day, 23, 59, 59, 0, location)
	return endOfDay.UnixMilli()
}

// ParseDateToStartOfDay 将日期字符串（格式：2025-10-10）转换为当天00:00:00的毫秒级时间戳（使用UTC+8时区）
func ParseDateToStartOfDay(dateStr string) (int64, error) {
	loc := GetChinaLocation()
	t, err := time.ParseInLocation("2006-01-02", dateStr, loc)
	if err != nil {
		return 0, err
	}
	// 确保是00:00:00
	year, month, day := t.Date()
	startOfDay := time.Date(year, month, day, 0, 0, 0, 0, loc)
	return startOfDay.UnixMilli(), nil
}

// ParseDateToEndOfDay 将日期字符串（格式：2025-10-10）转换为当天23:59:59的毫秒级时间戳（使用UTC+8时区）
func ParseDateToEndOfDay(dateStr string) (int64, error) {
	loc := GetChinaLocation()
	t, err := time.ParseInLocation("2006-01-02", dateStr, loc)
	if err != nil {
		return 0, err
	}
	// 设置为23:59:59
	year, month, day := t.Date()
	endOfDay := time.Date(year, month, day, 23, 59, 59, 0, loc)
	return endOfDay.UnixMilli(), nil
}

// ParseDateString 将日期字符串（格式：2025-10-10）解析为 time.Time（使用UTC+8时区）
func ParseDateString(dateStr string) (time.Time, error) {
	loc := GetChinaLocation()
	return time.ParseInLocation("2006-01-02", dateStr, loc)
}

// TimeToStartOfDay 将 time.Time 转换为当天00:00:00的毫秒级时间戳
func TimeToStartOfDay(t time.Time) int64 {
	year, month, day := t.Date()
	location := t.Location()
	startOfDay := time.Date(year, month, day, 0, 0, 0, 0, location)
	return startOfDay.UnixMilli()
}

// TimeToEndOfDay 将 time.Time 转换为当天23:59:59的毫秒级时间戳
func TimeToEndOfDay(t time.Time) int64 {
	year, month, day := t.Date()
	location := t.Location()
	endOfDay := time.Date(year, month, day, 23, 59, 59, 0, location)
	return endOfDay.UnixMilli()
}
