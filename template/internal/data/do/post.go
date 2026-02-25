package do

type SysPost struct {
	ID       int64   `json:"id"`
	Status   *int32  `json:"status,omitempty"`
	PostSort *int32  `json:"post_sort,omitempty"`
	PostCode *string `json:"post_code,omitempty"`
	PostName *string `json:"post_name,omitempty"`
	Remark   *string `json:"remark,omitempty"`
}
