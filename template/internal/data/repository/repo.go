package repository

import (
	bizsystem "spark/internal/biz/system"
	"spark/internal/data/repository/system"

	"go.uber.org/fx"
)

// Module provides repository layer dependencies
var Module = fx.Options(
	fx.Provide(
		// System repositories
		system.NewSystemUserRepo,
		system.NewSystemDeptRepo,
		system.NewSystemDictRepo,
		system.NewSystemFileRepo,
		system.NewSystemMenuRepo,
		system.NewSystemPostRepo,
		system.NewSystemRoleRepo,
		system.NewSystemConfigRepo,
		// Casbin rule repository
		system.NewCasbinRuleRepo,
		// User post repository
		system.NewSystemUserPostRepo,
		// Provide SystemMenuRepo as SystemRoleMenuRepo for RoleUsecase
		func(repo bizsystem.SystemMenuRepo) bizsystem.SystemRoleMenuRepo {
			return repo
		},
	),
)
