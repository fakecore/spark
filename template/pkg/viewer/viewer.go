package viewer

import (
	"context"

	"spark/internal/data/dal/model"
)

// viewerKey is the context key type for viewer
type viewerKey struct{}

// Role represents viewer permission level
type Role uint8

const (
	NoRole Role = iota
	View
	Admin
)

// Viewer describes the query/mutation viewer-context
type Viewer interface {
	Admin() bool
}

// UserViewer implements the Viewer interface
type UserViewer struct {
	user *model.SysUser
	role Role
}

func (v UserViewer) Admin() bool {
	return v.role == Admin
}

// GetUser returns the underlying user
func (v UserViewer) GetUser() *model.SysUser {
	return v.user
}

// NewViewer creates a new UserViewer
func NewViewer(user *model.SysUser, role Role) UserViewer {
	return UserViewer{
		user: user,
		role: role,
	}
}

// Convenience constructors
func NewNormalViewer(user *model.SysUser) UserViewer {
	return NewViewer(user, View)
}

// Context handling
func FromContext(ctx context.Context) Viewer {
	v, _ := ctx.Value(viewerKey{}).(Viewer)
	return v
}

func UserViewFromContext(ctx context.Context) (UserViewer, bool) {
	v, ok := ctx.Value(viewerKey{}).(UserViewer)
	return v, ok
}

func MustGetUserViewFromContext(ctx context.Context) UserViewer {
	userView, ok := UserViewFromContext(ctx)
	if !ok {
		panic("user view not found in context")
	}
	return userView
}

func NewContext(parent context.Context, v Viewer) context.Context {
	return context.WithValue(parent, viewerKey{}, v)
}

// WithViewer adds viewer to context
func WithViewer(ctx context.Context, v Viewer) context.Context {
	return context.WithValue(ctx, viewerKey{}, v)
}
