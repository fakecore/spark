package do

import "spark/internal/data/dal/model"

type SysUser struct {
	ID            int64   `gorm:"column:id;primaryKey;autoIncrement:true" json:"id,omitempty"`
	Name          *string `gorm:"column:name;not null;comment:用户名" json:"name,omitempty"`                                  // 用户名
	Nickname      *string `gorm:"column:nickname;not null;comment:用户昵称" json:"nickname,omitempty"`                         // 用户昵称
	Mobile        *string `gorm:"column:mobile;not null;comment:中国手机不带国家代码,国际手机号格式为：国家代码-手机号" json:"mobile,omitempty"`     // 中国手机不带国家代码,国际手机号格式为：国家代码-手机号
	Birthday      *int32  `gorm:"column:birthday;not null;comment:生日" json:"birthday,omitempty"`                           // 生日
	Password      *string `gorm:"column:password;not null;comment:登录密码;cmf_password加密" json:"password,omitempty"`          // 登录密码;cmf_password加密
	Status        *int32  `gorm:"column:status;not null;default:1;comment:用户状态;0:禁用,1:正常,2:未验证" json:"status,omitempty"`   // 用户状态;0:禁用,1:正常,2:未验证
	Email         *string `gorm:"column:email;not null;comment:用户登录邮箱" json:"email,omitempty"`                             // 用户登录邮箱
	Sex           *int32  `gorm:"column:sex;not null;comment:性别;0:保密,1:男,2:女" json:"sex,omitempty"`                        // 性别;0:保密,1:男,2:女
	Avatar        *string `gorm:"column:avatar;not null;comment:用户头像" json:"avatar,omitempty"`                             // 用户头像
	DeptID        *int64  `gorm:"column:dept_id;not null;comment:部门id" json:"dept_id,omitempty"`                           // 部门id
	IsAdmin       *int32  `gorm:"column:is_admin;not null;default:1;comment:是否后台管理员 1 是  0   否" json:"is_admin,omitempty"` // 是否后台管理员 1 是  0   否
	Address       *string `gorm:"column:address;not null;comment:联系地址" json:"address,omitempty"`                           // 联系地址
	Describe      *string `gorm:"column:describe;not null;comment: 描述信息" json:"describe,omitempty"`                        //  描述信息
	Remark        *string `gorm:"column:remark;not null;comment:备注" json:"remark,omitempty"`                               // 备注
	LastLoginIP   *string `gorm:"column:last_login_ip;comment:最后登录ip" json:"last_login_ip,omitempty"`                      // 最后登录ip
	LastLoginTime *int64  `gorm:"column:last_login_time;comment:最后登录时间" json:"last_login_time,omitempty"`                  // 最后登录时间
	CreatedBy     *int64  `gorm:"column:created_by;comment:创建者" json:"created_by,omitempty"`                               // 创建者
	UpdatedBy     *int64  `gorm:"column:updated_by;comment:更新者" json:"updated_by,omitempty"`                               // 更新者
	DeletedBy     *int64  `gorm:"column:deleted_by;comment:删除者" json:"deleted_by,omitempty"`                               // 删除者
	CreatedAt     *int64  `gorm:"column:created_at;not null;comment:创建时间" json:"created_at,omitempty"`                     // 创建时间
	UpdatedAt     *int64  `gorm:"column:updated_at;not null;comment:更新时间" json:"updated_at,omitempty"`                     // 更新时间
	DeletedAt     *int64  `gorm:"column:deleted_at;comment:删除时间" json:"deleted_at,omitempty"`                              // 删除时间
	RoleIds       []int64 `gorm:"column:role_ids;not null;comment:角色id" json:"role_ids,omitempty"`                         // 角色id
	PostIds       []int64 `gorm:"column:post_ids;not null;comment:岗位id" json:"post_ids,omitempty"`                         // 岗位id
}

type SysUserProfile struct {
	ID            int64            `gorm:"column:id;primaryKey;autoIncrement:true" json:"id,omitempty"`
	Name          *string          `gorm:"column:name;not null;comment:用户名" json:"name,omitempty"`                                  // 用户名
	Nickname      *string          `gorm:"column:nickname;not null;comment:用户昵称" json:"nickname,omitempty"`                         // 用户昵称
	Mobile        *string          `gorm:"column:mobile;not null;comment:中国手机不带国家代码,国际手机号格式为：国家代码-手机号" json:"mobile,omitempty"`     // 中国手机不带国家代码,国际手机号格式为：国家代码-手机号
	Birthday      *int32           `gorm:"column:birthday;not null;comment:生日" json:"birthday,omitempty"`                           // 生日
	Password      *string          `gorm:"column:password;not null;comment:登录密码;cmf_password加密" json:"password,omitempty"`          // 登录密码;cmf_password加密
	Status        *int32           `gorm:"column:status;not null;default:1;comment:用户状态;0:禁用,1:正常,2:未验证" json:"status,omitempty"`   // 用户状态;0:禁用,1:正常,2:未验证
	Email         *string          `gorm:"column:email;not null;comment:用户登录邮箱" json:"email,omitempty"`                             // 用户登录邮箱
	Sex           *int32           `gorm:"column:sex;not null;comment:性别;0:保密,1:男,2:女" json:"sex,omitempty"`                        // 性别;0:保密,1:男,2:女
	Avatar        *string          `gorm:"column:avatar;not null;comment:用户头像" json:"avatar,omitempty"`                             // 用户头像
	DeptID        *int64           `gorm:"column:dept_id;not null;comment:部门id" json:"dept_id,omitempty"`                           // 部门id
	IsAdmin       *int32           `gorm:"column:is_admin;not null;default:1;comment:是否后台管理员 1 是  0   否" json:"is_admin,omitempty"` // 是否后台管理员 1 是  0   否
	Address       *string          `gorm:"column:address;not null;comment:联系地址" json:"address,omitempty"`                           // 联系地址
	Describe      *string          `gorm:"column:describe;not null;comment: 描述信息" json:"describe,omitempty"`                        //  描述信息
	Remark        *string          `gorm:"column:remark;not null;comment:备注" json:"remark,omitempty"`                               // 备注
	LastLoginIP   *string          `gorm:"column:last_login_ip;comment:最后登录ip" json:"last_login_ip,omitempty"`                      // 最后登录ip
	LastLoginTime *int64           `gorm:"column:last_login_time;comment:最后登录时间" json:"last_login_time,omitempty"`                  // 最后登录时间
	CreatedBy     *int64           `gorm:"column:created_by;comment:创建者" json:"created_by,omitempty"`                               // 创建者
	UpdatedBy     *int64           `gorm:"column:updated_by;comment:更新者" json:"updated_by,omitempty"`                               // 更新者
	DeletedBy     *int64           `gorm:"column:deleted_by;comment:删除者" json:"deleted_by,omitempty"`                               // 删除者
	CreatedAt     *int64           `gorm:"column:created_at;not null;comment:创建时间" json:"created_at,omitempty"`                     // 创建时间
	UpdatedAt     *int64           `gorm:"column:updated_at;not null;comment:更新时间" json:"updated_at,omitempty"`                     // 更新时间
	DeletedAt     *int64           `gorm:"column:deleted_at;comment:删除时间" json:"deleted_at,omitempty"`                              // 删除时间
	RoleIds       []int64          `gorm:"column:role_ids;not null;comment:角色id" json:"role_ids,omitempty"`                         // 角色id
	PostIds       []int64          `gorm:"column:post_ids;not null;comment:岗位id" json:"post_ids,omitempty"`                         // 岗位id
	Roles         []*model.SysRole `gorm:"column:roles;not null;comment:角色名称列表" json:"roles,omitempty"`                             // 角色名称列表
	Posts         []*model.SysPost `gorm:"column:posts;not null;comment:岗位名称列表" json:"posts,omitempty"`                             // 岗位名称列表
	DeptName      *string          `gorm:"column:dept_name;not null;comment:部门名称" json:"dept_name,omitempty"`                       // 部门名称
	Menus         []*model.SysMenu `gorm:"column:menus;not null;comment:菜单列表" json:"menus,omitempty"`                               // 菜单列表
}
