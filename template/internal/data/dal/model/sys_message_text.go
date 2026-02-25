package model

const TableNameSysMessageText = "sys_message_text"

// SysMessageText 系统站内信详细表
type SysMessageText struct {
	BaseModelCreateOnly
	Title   string `gorm:"size:256;not null;comment:标题" json:"title"`
	Content string `gorm:"type:text;not null;comment:内容" json:"content"`
}

func (SysMessageText) TableName() string {
	return TableNameSysMessageText
}
