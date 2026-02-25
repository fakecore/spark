package model

const TableNameSysDictType = "sys_dict_type"

// SysDictType 字典类型表
type SysDictType struct {
	BaseModelNoSoftDelete
	Name     string  `gorm:"size:128;not null;comment:字典名称" json:"name"`
	TypeCode string  `gorm:"size:64;not null;uniqueIndex;comment:字典类型编码" json:"type_code"`
	Status   int32   `gorm:"type:smallint;not null;default:1;comment:状态 0禁用 1正常" json:"status"`
	Remark   *string `gorm:"size:512;comment:备注" json:"remark"`
}

func (SysDictType) TableName() string {
	return TableNameSysDictType
}
