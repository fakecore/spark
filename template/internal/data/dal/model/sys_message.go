package model

const TableNameSysMessage = "sys_message"

// SysMessage 系统站内信表
type SysMessage struct {
	BaseModelCreateOnly
	Title   string `gorm:"size:256;not null;comment:标题" json:"title"`
	Kind    int32  `gorm:"type:smallint;not null;comment:类型 0系统 1一对多" json:"kind"`
	Content string `gorm:"type:text;not null;comment:内容" json:"content"`
}

func (SysMessage) TableName() string {
	return TableNameSysMessage
}
