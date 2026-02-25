package model

const TableNameSysMenu = "sys_menu"

// SysMenu 菜单节点表
type SysMenu struct {
	BaseModelNoSoftDelete
	Pid        int64   `gorm:"not null;comment:父ID" json:"pid"`
	Name       string  `gorm:"size:64;not null;comment:规则名称" json:"name"`
	Title      string  `gorm:"size:64;not null;comment:标题" json:"title"`
	Icon       string  `gorm:"size:300;not null;comment:图标" json:"icon"`
	Condition  string  `gorm:"size:256;not null;comment:条件" json:"condition"`
	Remark     *string `gorm:"size:512;comment:备注" json:"remark"`
	MenuType   int32   `gorm:"type:smallint;not null;comment:类型 0目录 1菜单 2按钮" json:"menu_type"`
	Weight     int32   `gorm:"not null;comment:权重" json:"weight"`
	IsShow     int32   `gorm:"type:smallint;not null;default:1;comment:显示状态 0隐藏 1显示" json:"is_show"`
	Path       string  `gorm:"size:256;not null;comment:路由地址" json:"path"`
	Component  string  `gorm:"size:256;not null;comment:组件路径" json:"component"`
	IsLink     int32   `gorm:"type:smallint;not null;comment:是否外链" json:"is_link"`
	ModuleType string  `gorm:"size:64;not null;comment:所属模块" json:"module_type"`
	ModelID    int32   `gorm:"not null;comment:模型ID" json:"model_id"`
	IsIframe   int32   `gorm:"type:smallint;not null;comment:是否内嵌iframe" json:"is_iframe"`
	IsCached   int32   `gorm:"type:smallint;not null;comment:是否缓存" json:"is_cached"`
	Redirect   string  `gorm:"size:256;not null;comment:路由重定向" json:"redirect"`
	IsAffix    int32   `gorm:"type:smallint;not null;comment:是否固定" json:"is_affix"`
	LinkURL    string  `gorm:"size:512;not null;comment:链接地址" json:"link_url"`
}

func (SysMenu) TableName() string {
	return TableNameSysMenu
}
