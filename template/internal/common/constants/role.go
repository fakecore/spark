package constants

const (
	RoleCodeSuperAdmin = "super_admin"
	RoleCodeStaff      = "staff"
)

func IsRoleReadonly(code string) bool {
	switch code {
	case RoleCodeSuperAdmin, RoleCodeStaff:
		return true
	default:
		return false
	}
}
