package model

import "gorm.io/plugin/soft_delete"

// BaseModel 所有需要软删除和审计的实体继承此结构
type BaseModel struct {
	ID        int64                 `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedBy *int64                `gorm:"comment:创建者" json:"created_by"`
	UpdatedBy *int64                `gorm:"comment:更新者" json:"updated_by"`
	DeletedBy *int64                `gorm:"comment:删除者" json:"deleted_by"`
	CreatedAt int64                 `gorm:"autoCreateTime:milli;comment:创建时间" json:"created_at"`
	UpdatedAt int64                 `gorm:"autoUpdateTime:milli;comment:更新时间" json:"updated_at"`
	DeletedAt soft_delete.DeletedAt `gorm:"softDelete:milli;comment:删除时间" json:"deleted_at"`
}

// TableName 指定表名 - 防止被当作独立表创建
type baseModel struct{}

func (baseModel) TableName() string { return "" }

// BaseModelNoSoftDelete 不需要软删除的实体继承此结构
type BaseModelNoSoftDelete struct {
	ID        int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedBy *int64 `gorm:"comment:创建者" json:"created_by"`
	UpdatedBy *int64 `gorm:"comment:更新者" json:"updated_by"`
	CreatedAt int64  `gorm:"autoCreateTime:milli;comment:创建时间" json:"created_at"`
	UpdatedAt int64  `gorm:"autoUpdateTime:milli;comment:更新时间" json:"updated_at"`
}

// BaseModelCreateOnly 只有创建时间的实体继承此结构
type BaseModelCreateOnly struct {
	ID        int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedBy *int64 `gorm:"comment:创建者" json:"created_by"`
	CreatedAt int64  `gorm:"autoCreateTime:milli;comment:创建时间" json:"created_at"`
}
