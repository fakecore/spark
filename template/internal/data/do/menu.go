package do

type SysMenu struct {
	ID         int64   `json:"id"`
	Pid        *int64  `json:"pid,omitempty"`
	Name       *string `json:"name,omitempty"`
	Title      *string `json:"title,omitempty"`
	Icon       *string `json:"icon,omitempty"`
	Condition  *string `json:"condition,omitempty"`
	Remark     *string `json:"remark,omitempty"`
	MenuType   *int32  `json:"menu_type,omitempty"`
	Weight     *int32  `json:"weight,omitempty"`
	IsShow     *int32  `json:"is_show,omitempty"`
	Path       *string `json:"path,omitempty"`
	Component  *string `json:"component,omitempty"`
	IsLink     *int32  `json:"is_link,omitempty"`
	ModuleType *string `json:"module_type,omitempty"`
	ModelID    *int32  `json:"model_id,omitempty"`
	IsIframe   *int32  `json:"is_iframe,omitempty"`
	IsCached   *int32  `json:"is_cached,omitempty"`
	Redirect   *string `json:"redirect,omitempty"`
	IsAffix    *int32  `json:"is_affix,omitempty"`
	LinkUrl    *string `json:"link_url,omitempty"`
}
