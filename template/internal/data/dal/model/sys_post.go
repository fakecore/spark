package model

const TableNameSysPost = "sys_post"

// SysPost 岗位信息表
type SysPost struct {
	BaseModel
	PostCode string  `gorm:"size:64;not null;comment:岗位编码" json:"post_code"`
	PostName string  `gorm:"size:64;not null;comment:岗位名称" json:"post_name"`
	Sort     int32   `gorm:"not null;comment:显示顺序" json:"sort"`
	Status   int32   `gorm:"type:smallint;not null;default:1;comment:状态 0停用 1正常" json:"status"`
	Remark   *string `gorm:"size:512;comment:备注" json:"remark"`
}

func (SysPost) TableName() string {
	return TableNameSysPost
}
