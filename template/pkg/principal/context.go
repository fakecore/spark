package principal

import "context"

type ctxKey struct{}

func NewContext(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

func FromContext(ctx context.Context) (*Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(*Principal)
	return p, ok
}

func MustFromContext(ctx context.Context) *Principal {
	p, ok := FromContext(ctx)
	if !ok || p == nil {
		panic("principal not in context")
	}
	return p
}
