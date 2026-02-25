package do

type SysRole struct {
	ID               int64   `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	Status           *int32  `gorm:"column:status;not null;comment:状态 0 禁用 1 正常" json:"status,omitempty"`                                                    // 状态 0 禁用 1 正常
	Sort             *int32  `gorm:"column:sort;not null;comment:排序" json:"sort,omitempty"`                                                                  // 排序
	Name             *string `gorm:"column:name;not null;comment:角色名称" json:"name,omitempty"`                                                                // 角色名称
	Code             *string `gorm:"column:code;not null;comment:角色编码" json:"code,omitempty"`                                                                // 角色编码
	DataScope        *int32  `gorm:"column:data_scope;not null;default:3;comment:数据范围 1:全部数据权限 2:自定数据权限 3:本部门数据权限 4:本部门及以下数据权限" json:"data_scope,omitempty"` // 数据范围 1:全部数据权限 2:自定数据权限 3:本部门数据权限 4:本部门及以下数据权限
	Remark           *string `gorm:"column:remark;not null;comment:备注" json:"remark,omitempty"`                                                              // 备注
	CreatedBy        *int64  `gorm:"column:created_by;comment:创建者" json:"created_by,omitempty"`                                                              // 创建者
	UpdatedBy        *int64  `gorm:"column:updated_by;comment:更新者" json:"updated_by,omitempty"`                                                              // 更新者
	CreatedAt        *int64  `gorm:"column:created_at;not null;autoCreateTime:milli;comment:创建时间" json:"created_at,omitempty"`                               // 创建时间
	UpdatedAt        *int64  `gorm:"column:updated_at;not null;autoUpdateTime:milli;comment:更新时间" json:"updated_at,omitempty"`                               // 更新时间
	MenuIds          []int64 `gorm:"-" json:"menu_ids,omitempty"`                                                                                            // 菜单ID列表，用于角色菜单权限分配
	DataScopeDeptIDs []int64 `gorm:"-" json:"data_scope_dept_ids,omitempty"`                                                                                 // 自定义数据范围部门ID列表
}
