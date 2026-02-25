package server

import (
	"context"
	"testing"

	"spark/internal/data/dal/model"
	"spark/pkg/viewer"
)

func TestIsInfraAdmin(t *testing.T) {
	t.Run("MissingViewerDenied", func(t *testing.T) {
		if isInfraAdmin(context.Background()) {
			t.Fatalf("expected false for missing viewer")
		}
	})

	t.Run("SuperAdminAllowed", func(t *testing.T) {
		ctx := viewer.NewContext(context.Background(), viewer.NewNormalViewer(&model.SysUser{BaseModel: model.BaseModel{ID: 1}}))
		if !isInfraAdmin(ctx) {
			t.Fatalf("expected true for user id 1")
		}
	})

	t.Run("RoleAdminAllowed", func(t *testing.T) {
		ctx := viewer.NewContext(context.Background(), viewer.NewNormalViewer(&model.SysUser{
			BaseModel: model.BaseModel{ID: 22},
			IsAdmin:   1,
		}))
		if !isInfraAdmin(ctx) {
			t.Fatalf("expected true for is_admin=1")
		}
	})

	t.Run("NormalUserDenied", func(t *testing.T) {
		ctx := viewer.NewContext(context.Background(), viewer.NewNormalViewer(&model.SysUser{
			BaseModel: model.BaseModel{ID: 22},
			IsAdmin:   0,
		}))
		if isInfraAdmin(ctx) {
			t.Fatalf("expected false for normal user")
		}
	})
}
