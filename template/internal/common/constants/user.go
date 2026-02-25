package constants

import (
	"crypto/md5"
	"fmt"
)

type (
	UserInfoTransportKey struct{}
)

const (
	USER_PASSWORD        string = "123456"
	USER_TOKEN_REDIS_KEY string = "token:"
)

type UserStatus int32

const (
	UserStatus_Ban      UserStatus = 0
	UserStatus_Active   UserStatus = 1
	UserStatus_Unpermit UserStatus = 2
)

func (e UserStatus) IsActive() bool {
	return e == UserStatus_Active
}

func (e UserStatus) IsNotActive() bool {
	return !e.IsActive()
}

func (e UserStatus) ToInt32() int32 {
	return int32(e)
}

// GenTokenHashRedisKey 生成基于token hash的Redis key，支持多设备登录
func GenTokenHashRedisKey(userID int64, tokenHash string) string {
	return fmt.Sprintf("%s%d:%s", USER_TOKEN_REDIS_KEY, userID, tokenHash)
}

// GenerateTokenHash 生成token的短hash值(16位)
func GenerateTokenHash(token string) string {
	hash := md5.Sum([]byte(token))
	return fmt.Sprintf("%x", hash)[:16] // 取前16位
}

// GenUserTokenPattern 生成用户所有token的匹配模式
func GenUserTokenPattern(userID int64) string {
	return fmt.Sprintf("%s%d:*", USER_TOKEN_REDIS_KEY, userID)
}
