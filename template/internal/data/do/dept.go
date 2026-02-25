package do

// SysDept 部门更新对象
type SysDept struct {
	ID        int64   `json:"id"`                  // 部门ID
	ParentID  *int64  `json:"parent_id,omitempty"` // 父部门ID
	Ancestors *string `json:"ancestors,omitempty"` // 祖级列表
	DeptName  *string `json:"dept_name,omitempty"` // 部门名称
	OrderNum  *int32  `json:"order_num,omitempty"` // 显示顺序
	Leader    *string `json:"leader,omitempty"`    // 负责人
	Phone     *string `json:"phone,omitempty"`     // 联系电话
	Email     *string `json:"email,omitempty"`     // 邮箱
	Status    *int32  `json:"status,omitempty"`    // 部门状态
}
