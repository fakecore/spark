package do

// SysDictType 字典类型更新对象
type SysDictType struct {
	ID       int64   `json:"id"`
	Name     *string `json:"name,omitempty"`
	TypeCode *string `json:"type_code,omitempty"`
	Status   *int32  `json:"status,omitempty"`
	Remark   *string `json:"remark,omitempty"`
}
