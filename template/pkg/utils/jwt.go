package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

type JWTUtils struct {
	secretKey []byte
	expiresIn time.Duration
}

type CustomClaims struct {
	UserID     int64  `json:"userId"`
	RoleID     int64  `json:"roleId"`
	ClientType string `json:"clientType,omitempty"` // 客户端类型: "web", "desktop"
	jwt.RegisteredClaims
}

func NewJWTUtils(secretKey string, expiresIn time.Duration) *JWTUtils {
	return &JWTUtils{
		secretKey: []byte(secretKey),
		expiresIn: expiresIn,
	}
}

// TokenOptions 可选参数，用于生成特定类型的token
type TokenOptions struct {
	ClientType string // 客户端类型: "web", "desktop"
}

// GenerateToken 生成新的 token
func (j *JWTUtils) GenerateToken(userID, roleId int64) (string, time.Time, error) {
	return j.GenerateTokenWithOptions(userID, roleId, nil)
}

// GenerateTokenWithOptions 生成带可选参数的 token
func (j *JWTUtils) GenerateTokenWithOptions(userID, roleId int64, opts *TokenOptions) (string, time.Time, error) {
	claims := CustomClaims{
		UserID: userID,
		RoleID: roleId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.expiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	// 如果有可选参数，设置额外字段
	if opts != nil {
		claims.ClientType = opts.ClientType
	}

	tokenGen := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tokenGen.SignedString(j.secretKey)

	return token, claims.ExpiresAt.Time, err
}

// ParseToken 解析 token
func (j *JWTUtils) ParseToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// RenewToken 续签 token
func (j *JWTUtils) RenewToken(tokenString string) (string, error) {
	claims, err := j.ParseToken(tokenString)
	if err != nil && err != ErrExpiredToken {
		return "", err
	}

	// 创建新的 claims,保留原有的 UserID、RoleID 和 ClientType
	newClaims := CustomClaims{
		UserID:     claims.UserID,
		RoleID:     claims.RoleID,
		ClientType: claims.ClientType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.expiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	return token.SignedString(j.secretKey)
}

func (j *JWTUtils) IsTokenExpiredWithin(claims *CustomClaims, duration time.Duration) bool {
	expirationTime := claims.ExpiresAt.Time
	now := time.Now()
	return now.After(expirationTime) && now.Before(expirationTime.Add(duration))
}
