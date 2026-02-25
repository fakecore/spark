package model

const TableNameSysRoleDept = "sys_role_dept"

// SysRoleDept 角色和部门关联表
type SysRoleDept struct {
	ID     int64 `gorm:"primaryKey;autoIncrement" json:"id"`
	RoleID int64 `gorm:"not null;index;comment:角色ID" json:"role_id"`
	DeptID int64 `gorm:"not null;index;comment:部门ID" json:"dept_id"`
}

func (SysRoleDept) TableName() string {
	return TableNameSysRoleDept
}
