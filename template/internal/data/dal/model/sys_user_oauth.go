package model

const TableNameSysUserOAuth = "sys_user_o_auth"

// SysUserOAuth 用户OAuth表
type SysUserOAuth struct {
	ID               int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	UID              int64  `gorm:"not null;index;comment:用户ID" json:"uid"`
	OauthType        string `gorm:"size:32;not null;comment:OAuth类型" json:"oauth_type"`
	OauthID          string `gorm:"size:128;not null;comment:OAuth ID" json:"oauth_id"`
	OauthAccessToken string `gorm:"size:512;not null;comment:Access Token" json:"oauth_access_token"`
	OauthExpire      int64  `gorm:"not null;default:86400;comment:过期时间" json:"oauth_expire"`
}

func (SysUserOAuth) TableName() string {
	return TableNameSysUserOAuth
}
