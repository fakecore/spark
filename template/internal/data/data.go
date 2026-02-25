package data

import (
	"spark/internal/data/repository"

	"go.uber.org/fx"
)

// Module provides data layer dependencies
var Module = fx.Options(
	fx.Provide(
		NewDBQuery,
	),
	repository.Module,
)
