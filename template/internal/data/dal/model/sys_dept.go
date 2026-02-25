package model

const TableNameSysDept = "sys_dept"

// SysDept 部门表
type SysDept struct {
	BaseModel
	ParentID  int64   `gorm:"not null;comment:父部门ID" json:"parent_id"`
	Ancestors string  `gorm:"size:512;not null;comment:祖级列表" json:"ancestors"`
	DeptName  string  `gorm:"size:64;not null;comment:部门名称" json:"dept_name"`
	Sort      int32   `gorm:"not null;comment:显示顺序" json:"sort"`
	Leader    *string `gorm:"size:64;comment:负责人" json:"leader"`
	Phone     *string `gorm:"size:20;comment:联系电话" json:"phone"`
	Email     *string `gorm:"size:128;comment:邮箱" json:"email"`
	Status    int32   `gorm:"type:smallint;not null;default:1;comment:状态 0停用 1正常" json:"status"`

	// 关系
	Users []SysUser `gorm:"foreignKey:DeptID" json:"users"`
}

func (SysDept) TableName() string {
	return TableNameSysDept
}
