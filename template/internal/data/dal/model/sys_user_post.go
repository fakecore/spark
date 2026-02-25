package model

const TableNameSysUserPost = "sys_user_post"

// SysUserPost 用户与岗位关联表
type SysUserPost struct {
	ID     int64 `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID int64 `gorm:"not null;index;comment:用户ID" json:"user_id"`
	PostID int64 `gorm:"not null;index;comment:岗位ID" json:"post_id"`
}

func (SysUserPost) TableName() string {
	return TableNameSysUserPost
}
