package token

import (
	"context"
)

// CtxKey defines a distinct type for context keys used by the access
// package
type CtxKey string

// tokenCtxKey is the key for adding an access token to a context.Context
const tokenCtxKey CtxKey = "Token"

// AddToContext adds a token string to a context
func AddToContext(ctx context.Context, s string) context.Context {
	return context.WithValue(ctx, tokenCtxKey, s)
}

// FromCtx extracts the JWT from a given
// context if one is set, returning nil otherwise
func FromCtx(ctx context.Context) string {
	iface := ctx.Value(tokenCtxKey)
	if s, ok := iface.(string); ok {
		return s
	}
	return ""
}
