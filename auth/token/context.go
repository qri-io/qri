package token

import (
	"context"
	"net/http"
	"strings"
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

const (
	// httpAuthorizationHeader is the http header field to check for tokens,
	// follows OAuth 2.0 spec
	httpAuthorizationHeader = "authorization"
	// httpAuthorizationBearerPrefix is a prefix before a token in the
	// Authorization header field. Follows OAuth 2.0 spec
	httpAuthorizationBearerPrefix = "Bearer "
)

// OAuthTokenMiddleware parses any "authorization" header containing a Bearer
// token & adds it to the request context
func OAuthTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqToken := r.Header.Get(httpAuthorizationHeader)
		if reqToken == "" && r.FormValue(httpAuthorizationHeader) != "" {
			reqToken = r.FormValue(httpAuthorizationHeader)
		}
		if reqToken == "" {
			next.ServeHTTP(w, r)
			return
		}

		if !strings.HasPrefix(reqToken, httpAuthorizationBearerPrefix) {
			next.ServeHTTP(w, r)
			return
		}

		tokenStr := strings.TrimPrefix(reqToken, httpAuthorizationBearerPrefix)
		ctx := AddToContext(r.Context(), tokenStr)

		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

// AddContextTokenToRequest checks the supplied context for an auth token and
// adds it to an http request, returns true if a token is added
func AddContextTokenToRequest(ctx context.Context, r *http.Request) (*http.Request, bool) {
	if s := FromCtx(ctx); s != "" {
		r.Header.Set(httpAuthorizationHeader, strings.Join([]string{httpAuthorizationBearerPrefix, s}, ""))
		return r, true
	}
	return r, false
}
