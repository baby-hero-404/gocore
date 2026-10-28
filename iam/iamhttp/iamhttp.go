package iamhttp

import (
	"fmt"
	"net/http"

	"github.com/baby-hero-404/gocore/iam"
	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdres"
)

// IdentityFromRequest returns the Identity stored by JWT, if any.
func IdentityFromRequest(r *http.Request) (iam.Identity, bool) {
	return iam.IdentityFromContext(r.Context())
}

// JWT extracts the token, validates it and stores the Identity in the request context (read it with
// IdentityFromRequest or iam.IdentityFromContext). By default the token comes from the
// "Authorization: Bearer" header; WithCookie makes the named cookie take precedence. Unauthenticated
// requests are answered with the standard 401 envelope and never reach the next handler.
func JWT(validator iam.TokenValidator, opts ...Option) func(http.Handler) http.Handler {
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := extractToken(r, o)
			if err != nil {
				stdres.WriteError(w, r, err)
				return
			}

			identity, err := validator.Validate(r.Context(), token)
			if err != nil {
				stdres.WriteError(w, r, err)
				return
			}

			next.ServeHTTP(w, r.WithContext(iam.WithIdentity(r.Context(), identity)))
		})
	}
}

// extractToken reads the token from the configured cookie, falling back to the Bearer header.
func extractToken(r *http.Request, o options) (string, error) {
	if o.cookieName != "" {
		if c, err := r.Cookie(o.cookieName); err == nil && c.Value != "" {
			return c.Value, nil
		}
	}
	return iam.BearerToken(r.Header.Get("Authorization"))
}

// Resolver customizes how the Casbin subject, object and action are derived from a request.
type Resolver func(r *http.Request, id iam.Identity) (sub, obj, act string)

// DefaultResolver uses the identity's role (or id), the URL path and the HTTP method.
func DefaultResolver(r *http.Request, id iam.Identity) (string, string, string) {
	return iam.Subject(id), r.URL.Path, r.Method
}

// Casbin enforces authorization with the enforcer and must run after JWT. It answers 401 without an
// identity, 500 when the enforcer is missing or fails, and 403 when the policy denies the request.
func Casbin(enforcer iam.Enforcer, resolver ...Resolver) func(http.Handler) http.Handler {
	resolve := DefaultResolver
	if len(resolver) > 0 && resolver[0] != nil {
		resolve = resolver[0]
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if enforcer == nil {
				stdres.WriteError(w, r, stderr.NewInternal("casbin enforcer is not configured"))
				return
			}

			identity, ok := IdentityFromRequest(r)
			if !ok {
				stdres.WriteError(w, r, stderr.NewUnauthorized("unauthenticated request").WithKey(iam.KeyUnauthenticated))
				return
			}

			sub, obj, act := resolve(r, identity)
			allowed, err := iam.Authorize(enforcer, identity, sub, obj, act)
			if err != nil {
				stdres.WriteError(w, r, err)
				return
			}
			if !allowed {
				stdres.WriteError(w, r, stderr.NewForbidden(fmt.Sprintf("access denied: %s cannot %s %s", sub, act, obj)))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
