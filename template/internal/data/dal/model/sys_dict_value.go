package model

const TableNameSysDictValue = "sys_dict_value"

// SysDictValue 字典数据表
type SysDictValue struct {
	BaseModel
	DictCode  string  `gorm:"size:64;not null;index;comment:字典类型编码" json:"dict_code"`
	Sort      int32   `gorm:"not null;comment:显示顺序" json:"sort"`
	Label     string  `gorm:"size:128;not null;comment:字典标签" json:"label"`
	Value     string  `gorm:"size:256;not null;comment:字典值" json:"value"`
	CSSClass  *string `gorm:"size:128;comment:样式类" json:"css_class"`
	ListClass *string `gorm:"size:128;comment:列表样式" json:"list_class"`
	IsDefault int32   `gorm:"type:smallint;not null;comment:是否默认" json:"is_default"`
	Status    int32   `gorm:"type:smallint;not null;default:1;comment:状态 0禁用 1正常" json:"status"`
	Remark    *string `gorm:"size:512;comment:备注" json:"remark"`
}

func (SysDictValue) TableName() string {
	return TableNameSysDictValue
}
