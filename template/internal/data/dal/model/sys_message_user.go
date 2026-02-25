package model

const TableNameSysMessageUser = "sys_message_user"

// SysMessageUser 系统站内信-用户表
type SysMessageUser struct {
	BaseModelNoSoftDelete
	MessageID int64 `gorm:"not null;index;comment:消息ID" json:"message_id"`
	SendID    int64 `gorm:"not null;index;comment:发送者ID" json:"send_id"`
	RecID     int64 `gorm:"not null;index;comment:接收者ID" json:"rec_id"`
	Status    int32 `gorm:"type:smallint;not null;comment:状态 0未读 1已读" json:"status"`
}

func (SysMessageUser) TableName() string {
	return TableNameSysMessageUser
}
