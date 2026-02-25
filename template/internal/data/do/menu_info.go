package do

import "spark/internal/data/dal/model"

// MenuInfo 菜单信息（包含树形结构）
type MenuInfo struct {
	model.SysMenu
	Children []*MenuInfo `json:"children,omitempty"` // 子菜单列表
}
