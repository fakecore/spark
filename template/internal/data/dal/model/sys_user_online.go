package model

const TableNameSysUserOnline = "sys_user_online"

// SysUserOnline 用户在线状态表
type SysUserOnline struct {
	BaseModelNoSoftDelete
	UUID     string `gorm:"size:64;not null;comment:UUID" json:"uuid"`
	Token    string `gorm:"size:512;not null;comment:Token" json:"token"`
	UserName string `gorm:"size:64;not null;comment:用户名" json:"user_name"`
	IP       string `gorm:"size:64;not null;comment:IP" json:"ip"`
	Explorer string `gorm:"size:128;not null;comment:浏览器" json:"explorer"`
	Os       string `gorm:"size:128;not null;comment:操作系统" json:"os"`
}

func (SysUserOnline) TableName() string {
	return TableNameSysUserOnline
}
