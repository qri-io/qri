package profile

import (
	"context"
)

// CtxKey defines a distinct type for context keys used by the profile
// package
type CtxKey string

// profileCtxKey is the key for adding a profile identifier to a context.Context
const profileCtxKey CtxKey = "Profile"

// AddIDToContext adds a token string to a context
func AddIDToContext(ctx context.Context, s string) context.Context {
	return context.WithValue(ctx, profileCtxKey, s)
}

// IDFromCtx extracts a profile identifier from a given context if one is set
func IDFromCtx(ctx context.Context) string {
	i := ctx.Value(profileCtxKey)
	if s, ok := i.(string); ok {
		return s
	}
	return ""
}
