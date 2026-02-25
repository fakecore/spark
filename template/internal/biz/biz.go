package biz

import (
	"spark/internal/biz/system"

	"go.uber.org/fx"
)

// Module provides business layer dependencies
var Module = fx.Options(
	fx.Provide(
		// System usecases
		system.NewSystemConfigUsecase,
		system.NewSystemDeptUsecase,
		system.NewSystemDictUsecase,
		system.NewSystemFileUsecase,
		system.NewSystemMenuUsecase,
		system.NewSystemPermissionUsecase,
		system.NewSystemPostUsecase,
		system.NewSystemRoleUsecase,
		system.NewSystemUserUsecase,
		// Provide SystemDeptUsecase as DeptRepoGetter for SystemUserUsecase
		func(uc *system.SystemDeptUsecase) system.DeptRepoGetter {
			return uc
		},
	),
)
