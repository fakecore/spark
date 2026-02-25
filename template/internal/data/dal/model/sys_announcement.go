package model

const TableNameSysAnnouncement = "sys_announcement"

// SysAnnouncement 系统公告表
type SysAnnouncement struct {
	BaseModelCreateOnly
	Title   string `gorm:"size:256;not null;comment:标题" json:"title"`
	Content string `gorm:"type:text;not null;comment:内容" json:"content"`
	URL     string `gorm:"size:512;not null;comment:图片URL" json:"url"`
	Status  int32  `gorm:"type:smallint;not null;comment:状态 0未发布 1已发布" json:"status"`
}

func (SysAnnouncement) TableName() string {
	return TableNameSysAnnouncement
}
