package server

import (
	"context"

	"spark/pkg/viewer"
)

func isInfraAdmin(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, ok := viewer.UserViewFromContext(ctx)
	if !ok {
		return false
	}
	u := v.GetUser()
	if u == nil {
		return false
	}
	if u.ID == 1 || u.IsAdmin == 1 {
		return true
	}
	return false
}
