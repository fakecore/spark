package model

const TableNameSysLoginLog = "sys_login_log"

// SysLoginLog 系统访问记录
type SysLoginLog struct {
	BaseModelCreateOnly
	LoginName     *string `gorm:"size:64;comment:登录名" json:"login_name"`
	Ipaddr        *string `gorm:"size:64;comment:IP地址" json:"ipaddr"`
	LoginLocation *string `gorm:"size:256;comment:登录地点" json:"login_location"`
	Browser       *string `gorm:"size:128;comment:浏览器" json:"browser"`
	Os            *string `gorm:"size:128;comment:操作系统" json:"os"`
	Status        int32   `gorm:"type:smallint;not null;default:1;comment:状态 0失败 1成功" json:"status"`
	Msg           *string `gorm:"size:512;comment:消息" json:"msg"`
	Module        *string `gorm:"size:64;comment:模块" json:"module"`
}

func (SysLoginLog) TableName() string {
	return TableNameSysLoginLog
}
