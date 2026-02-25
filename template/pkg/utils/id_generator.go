package utils

import (
	"fmt"
	"math/rand"
	"time"
)

// GenerateVirtCustomerID 生成12位虚拟客户ID
// 格式：9999 + 时间戳后6位 + 随机数2位
// 示例：999912345678 (9999=虚拟客户标识, 123456=时间戳后6位, 78=随机数2位)
func GenerateVirtCustomerID() int64 {
	// 获取当前时间戳的后6位
	timestamp := time.Now().UnixNano() / 1000000000 // 1秒时间戳
	timestampSuffix := timestamp % 1000000          // 取后6位

	// 生成2位随机数
	randomNum := rand.Intn(100)

	// 组合成12位数字：9999 + 时间戳后6位 + 随机数2位
	virtCustomerID := 999900000000 + timestampSuffix*100 + int64(randomNum)

	return virtCustomerID
}

// GenerateVirtCustomerIDString 生成字符串格式的虚拟客户ID
func GenerateVirtCustomerIDString() string {
	id := GenerateVirtCustomerID()
	return fmt.Sprintf("%012d", id)
}
