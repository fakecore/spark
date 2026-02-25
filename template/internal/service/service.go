package service

import (
	"spark/internal/service/system"

	"go.uber.org/fx"
)

// Module provides service layer dependencies
var Module = fx.Options(
	fx.Provide(
		// System services
		system.NewAuthService,
		system.NewConfigService,
		system.NewDeptService,
		system.NewLogService,
		system.NewMenuService,
		system.NewPostService,
		system.NewUserService,
		system.NewDictService,
		system.NewRoleService,
		system.NewFileService,
		system.NewAnnouncementService,
		system.NewMessageService,
		system.NewOAuthService,
		system.NewPackageService,
		system.NewUserOnlineService,
	),
)
