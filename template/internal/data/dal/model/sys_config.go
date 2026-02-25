package model

const TableNameSysConfig = "sys_config"

// SysConfig 系统配置表
type SysConfig struct {
	BaseModelNoSoftDelete
	Name   string  `gorm:"size:128;not null;comment:配置名称" json:"name"`
	Key    *string `gorm:"size:128;uniqueIndex;comment:配置键" json:"key"`
	Value  *string `gorm:"type:text;comment:配置值" json:"value"`
	Kind   int32   `gorm:"type:smallint;not null;comment:类型 0系统 1用户" json:"kind"`
	Status int32   `gorm:"type:smallint;not null;default:1;comment:状态 0禁用 1正常" json:"status"`
	Remark *string `gorm:"size:512;comment:备注" json:"remark"`
}

func (SysConfig) TableName() string {
	return TableNameSysConfig
}
