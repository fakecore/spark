package utils

import (
	"testing"
	"time"
)

func TestParseDateToStartOfDay(t *testing.T) {
	// 测试：2025-10-10 -> 2025-10-10 00:00:00
	dateStr := "2025-10-10"

	result, err := ParseDateToStartOfDay(dateStr)
	if err != nil {
		t.Errorf("ParseDateToStartOfDay() error = %v", err)
		return
	}

	resultTime := time.UnixMilli(result)
	expected, _ := time.ParseInLocation("2006-01-02 15:04:05", "2025-10-10 00:00:00", GetChinaLocation())

	if resultTime.UnixMilli() != expected.UnixMilli() {
		t.Errorf("ParseDateToStartOfDay() = %s, want %s",
			resultTime.Format("2006-01-02 15:04:05"),
			expected.Format("2006-01-02 15:04:05"))
	}

	t.Logf("输入日期: %s", dateStr)
	t.Logf("输出时间: %s (timestamp: %d)", resultTime.Format("2006-01-02 15:04:05"), result)
}

func TestParseDateToEndOfDay(t *testing.T) {
	// 测试：2025-10-10 -> 2025-10-10 23:59:59
	dateStr := "2025-10-10"

	result, err := ParseDateToEndOfDay(dateStr)
	if err != nil {
		t.Errorf("ParseDateToEndOfDay() error = %v", err)
		return
	}

	resultTime := time.UnixMilli(result)
	expected, _ := time.ParseInLocation("2006-01-02 15:04:05", "2025-10-10 23:59:59", GetChinaLocation())

	if resultTime.UnixMilli() != expected.UnixMilli() {
		t.Errorf("ParseDateToEndOfDay() = %s, want %s",
			resultTime.Format("2006-01-02 15:04:05"),
			expected.Format("2006-01-02 15:04:05"))
	}

	t.Logf("输入日期: %s", dateStr)
	t.Logf("输出时间: %s (timestamp: %d)", resultTime.Format("2006-01-02 15:04:05"), result)
}

func TestFullDayRange(t *testing.T) {
	// 测试完整一天的转换
	dateStr := "2025-10-10"

	start, err := ParseDateToStartOfDay(dateStr)
	if err != nil {
		t.Errorf("ParseDateToStartOfDay() error = %v", err)
		return
	}

	end, err := ParseDateToEndOfDay(dateStr)
	if err != nil {
		t.Errorf("ParseDateToEndOfDay() error = %v", err)
		return
	}

	startTime := time.UnixMilli(start)
	endTime := time.UnixMilli(end)

	t.Logf("日期字符串: %s", dateStr)
	t.Logf("开始时间: %s (timestamp: %d)", startTime.Format("2006-01-02 15:04:05"), start)
	t.Logf("结束时间: %s (timestamp: %d)", endTime.Format("2006-01-02 15:04:05"), end)

	// 验证时间差应该是一天减1秒（23小时59分59秒 = 86399秒）
	timeDiff := end - start
	expectedDiff := int64(86399 * 1000) // 23:59:59 = 86399 seconds = 86399000 milliseconds
	if timeDiff != expectedDiff {
		t.Errorf("时间差应该是 %d 毫秒 (23小时59分59秒), 得到 %d 毫秒", expectedDiff, timeDiff)
	}

	// 验证开始时间小于结束时间
	if start >= end {
		t.Errorf("开始时间应该小于结束时间")
	}

	t.Logf("时间跨度: %d 毫秒 (约 %.2f 小时)", timeDiff, float64(timeDiff)/3600000.0)
}

func TestInvalidDateFormat(t *testing.T) {
	invalidDates := []string{
		"2025/10/10",
		"10-10-2025",
		"2025-10-10 00:00:00",
		"invalid",
		"",
	}

	for _, dateStr := range invalidDates {
		_, err := ParseDateToStartOfDay(dateStr)
		if err == nil {
			t.Errorf("ParseDateToStartOfDay(%s) 应该返回错误", dateStr)
		}

		_, err = ParseDateToEndOfDay(dateStr)
		if err == nil {
			t.Errorf("ParseDateToEndOfDay(%s) 应该返回错误", dateStr)
		}
	}
}

func TestDateComparison(t *testing.T) {
	// 测试日期比较
	start, _ := ParseDateToStartOfDay("2025-10-10")
	end, _ := ParseDateToEndOfDay("2025-10-15")

	if start > end {
		t.Errorf("2025-10-10 00:00:00 应该小于 2025-10-15 23:59:59")
	}

	// 测试同一天
	sameStart, _ := ParseDateToStartOfDay("2025-10-10")
	sameEnd, _ := ParseDateToEndOfDay("2025-10-10")

	if sameStart > sameEnd {
		t.Errorf("同一天的开始时间应该小于结束时间")
	}

	t.Logf("2025-10-10 开始: %d, 结束: %d, 差值: %d 毫秒",
		sameStart, sameEnd, sameEnd-sameStart)
}
