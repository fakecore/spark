package do

// SysDictValue 字典数据更新对象
type SysDictValue struct {
	ID         int64   `json:"id"`
	DictTypeID *int64  `json:"dict_type_id,omitempty"`
	DictLabel  *string `json:"dict_label,omitempty"`
	DictValue  *string `json:"dict_value,omitempty"`
	DictType   *string `json:"dict_type,omitempty"`
	Sort       *int32  `json:"sort,omitempty"`
	CSSClass   *string `json:"css_class,omitempty"`
	ListClass  *string `json:"list_class,omitempty"`
	IsDefault  *bool   `json:"is_default,omitempty"`
	Status     *int32  `json:"status,omitempty"`
	Remark     *string `json:"remark,omitempty"`
}
