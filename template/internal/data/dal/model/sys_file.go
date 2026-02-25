package model

import "gorm.io/plugin/soft_delete"

const TableNameSysFile = "sys_file"

// SysFile 文件管理表
type SysFile struct {
	ID         int64                 `gorm:"primaryKey;autoIncrement" json:"id"`
	FileName   string                `gorm:"size:256;not null;comment:文件名" json:"file_name"`
	FileSize   int64                 `gorm:"not null;comment:文件大小" json:"file_size"`
	MimeType   *string               `gorm:"size:128;comment:MIME类型" json:"mime_type"`
	BizType    int32                 `gorm:"type:smallint;not null;default:1;comment:业务类型 1公开 2私有" json:"biz_type"`
	UsageType  *string               `gorm:"size:64;comment:使用类型" json:"usage_type"`
	Md5        *string               `gorm:"size:64;comment:MD5" json:"md5"`
	OssBucket  string                `gorm:"size:128;not null;comment:OSS Bucket" json:"oss_bucket"`
	OssPath    string                `gorm:"size:1024;not null;comment:OSS路径" json:"oss_path"`
	Status     int32                 `gorm:"type:smallint;not null;comment:状态 0禁用 1已上传 2已使用 3待处理" json:"status"`
	CreateTime int64                 `gorm:"not null;comment:创建时间" json:"create_time"`
	DeletedAt  soft_delete.DeletedAt `gorm:"softDelete:milli" json:"deleted_at"`
}

func (SysFile) TableName() string {
	return TableNameSysFile
}
