package principal

import (
	"context"
	"testing"
)

func TestContextRoundTrip(t *testing.T) {
	p := &Principal{
		UserID:       12,
		RoleID:       3,
		RoleCode:     "super_admin",
		DeptID:       9,
		Scope:        ScopeDeptTree,
		IsSuperAdmin: true,
		ClientType:   "web",
	}

	ctx := NewContext(context.Background(), p)
	got, ok := FromContext(ctx)
	if !ok {
		t.Fatal("expected principal in context")
	}
	if got.UserID != p.UserID || got.RoleCode != p.RoleCode || got.Scope != p.Scope {
		t.Fatalf("unexpected principal: %+v", got)
	}
}
