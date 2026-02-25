package principal

type DataScope string

const (
	ScopeAll      DataScope = "ALL"
	ScopeCustom   DataScope = "CUSTOM"
	ScopeDept     DataScope = "DEPT"
	ScopeDeptTree DataScope = "DEPT_TREE"
	ScopeSelf     DataScope = "SELF"
)

type Principal struct {
	UserID       int64
	RoleID       int64
	RoleCode     string
	DeptID       int64
	Scope        DataScope
	IsSuperAdmin bool
	ClientType   string
}

func (p *Principal) IsScopedTo(scope DataScope) bool {
	if p == nil || p.IsSuperAdmin {
		return false
	}
	return p.Scope == scope
}
