package model

const TableNameSysRole = "sys_role"

// SysRole 角色表
type SysRole struct {
	BaseModelNoSoftDelete
	Status    int32   `gorm:"type:smallint;not null;comment:状态 0禁用 1正常" json:"status"`
	Sort      int32   `gorm:"not null;comment:显示顺序" json:"sort"`
	Name      string  `gorm:"size:64;not null;comment:角色名称" json:"name"`
	Code      string  `gorm:"size:64;not null;uniqueIndex;comment:角色编码" json:"code"`
	DataScope int32   `gorm:"type:smallint;not null;default:3;comment:数据范围 1全部 2自定 3本部门 4本部门及以下" json:"data_scope"`
	Remark    *string `gorm:"size:512;comment:备注" json:"remark"`
	IsAdmin   int32   `gorm:"type:smallint;not null;default:0;comment:是否管理员 0否 1是" json:"is_admin"`
}

func (SysRole) TableName() string {
	return TableNameSysRole
}
