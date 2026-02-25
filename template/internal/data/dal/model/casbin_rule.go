package model

const TableNameCasbinRule = "casbin_rule"

// CasbinRule Casbin规则表
type CasbinRule struct {
	ID    int64   `gorm:"primaryKey;autoIncrement" json:"id"`
	Ptype *string `gorm:"size:64;comment:策略类型" json:"ptype"`
	V0    *string `gorm:"size:256;comment:V0" json:"v0"`
	V1    *string `gorm:"size:256;comment:V1" json:"v1"`
	V2    *string `gorm:"size:256;comment:V2" json:"v2"`
	V3    *string `gorm:"size:256;comment:V3" json:"v3"`
	V4    *string `gorm:"size:256;comment:V4" json:"v4"`
	V5    *string `gorm:"size:256;comment:V5" json:"v5"`
}

func (CasbinRule) TableName() string {
	return TableNameCasbinRule
}
