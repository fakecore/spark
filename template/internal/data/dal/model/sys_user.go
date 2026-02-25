package model

const TableNameSysUser = "sys_user"

// SysUser 用户表
type SysUser struct {
	BaseModel
	Name          string  `gorm:"size:64;not null;comment:用户名" json:"name"`
	Nickname      string  `gorm:"size:64;not null;comment:用户昵称" json:"nickname"`
	Mobile        string  `gorm:"size:20;not null;comment:手机号" json:"mobile"`
	Birthday      int32   `gorm:"not null;comment:生日" json:"birthday"`
	Password      string  `gorm:"size:255;not null;comment:登录密码" json:"password"`
	Status        int32   `gorm:"type:smallint;not null;default:1;comment:状态 0禁用 1正常 2未验证" json:"status"`
	Email         string  `gorm:"size:128;not null;comment:邮箱" json:"email"`
	Sex           int32   `gorm:"type:smallint;not null;default:0;comment:性别 0保密 1男 2女" json:"sex"`
	Avatar        string  `gorm:"size:512;not null;comment:头像" json:"avatar"`
	DeptID        int64   `gorm:"not null;index;comment:部门ID" json:"dept_id"`
	IsAdmin       int32   `gorm:"type:smallint;not null;default:0;comment:是否管理员" json:"is_admin"`
	Address       *string `gorm:"size:256;comment:联系地址" json:"address"`
	Remark        *string `gorm:"size:512;comment:备注" json:"remark"`
	LastLoginIP   *string `gorm:"size:64;comment:最后登录IP" json:"last_login_ip"`
	LastLoginTime *int64  `gorm:"comment:最后登录时间" json:"last_login_time"`
}

func (SysUser) TableName() string {
	return TableNameSysUser
}
