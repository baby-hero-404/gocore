// Package iamgin adapts the iam package to Gin: authentication and authorization middleware.
package iamgin

import (
	"fmt"

	"github.com/baby-hero-404/gocore/iam"
	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdres/stdgin"
	"github.com/gin-gonic/gin"
)

// contextKeyIdentityString is the Gin context key under which the Identity is stored.
const contextKeyIdentityString = "gocore_iam_identity"

// IdentityFromGin extracts the Identity from the Gin context.
func IdentityFromGin(c *gin.Context) (iam.Identity, bool) {
	if val, exists := c.Get(contextKeyIdentityString); exists {
		if id, ok := val.(iam.Identity); ok {
			return id, true
		}
	}
	return iam.IdentityFromContext(c.Request.Context())
}

// JWTMiddleware extracts the token, validates it, and injects Identity into the context.
// By default the token comes from the "Authorization: Bearer" header; WithCookie makes the
// named cookie take precedence. Rejects unauthenticated requests with HTTP 401 using stdgin.Error.
func JWTMiddleware(validator iam.TokenValidator, opts ...Option) gin.HandlerFunc {
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}

	return func(c *gin.Context) {
		token, err := extractToken(c, o)
		if err != nil {
			stdgin.Error(c, err)
			return
		}

		identity, err := validator.Validate(c.Request.Context(), token)
		if err != nil {
			stdgin.Error(c, err)
			return
		}

		// Inject Identity into Gin context and Request Context
		c.Set(contextKeyIdentityString, identity)
		c.Request = c.Request.WithContext(iam.WithIdentity(c.Request.Context(), identity))

		c.Next()
	}
}

// extractToken reads the token from the configured cookie, falling back to the Bearer header.
func extractToken(c *gin.Context, o options) (string, error) {
	if o.cookieName != "" {
		if v, err := c.Cookie(o.cookieName); err == nil && v != "" {
			return v, nil
		}
	}
	return iam.BearerToken(c.GetHeader("Authorization"))
}

// CasbinResolver allows customizing how sub, obj, and act are resolved from a request.
type CasbinResolver func(c *gin.Context, id iam.Identity) (sub string, obj string, act string)

// DefaultCasbinResolver uses role/user ID, URL path, and HTTP method.
func DefaultCasbinResolver(c *gin.Context, id iam.Identity) (string, string, string) {
	return iam.Subject(id), c.Request.URL.Path, c.Request.Method
}

// CasbinMiddleware enforces RBAC/ABAC authorization rules using Casbin.
// Rejects unauthorized requests with standardized HTTP 403 stderr error.
func CasbinMiddleware(enforcer iam.Enforcer, resolver ...CasbinResolver) gin.HandlerFunc {
	r := DefaultCasbinResolver
	if len(resolver) > 0 && resolver[0] != nil {
		r = resolver[0]
	}

	return func(c *gin.Context) {
		if enforcer == nil {
			stdgin.Error(c, stderr.NewInternal("casbin enforcer is not configured"))
			return
		}

		identity, ok := IdentityFromGin(c)
		if !ok {
			stdgin.Error(c, stderr.NewUnauthorized("unauthenticated request").WithKey(iam.KeyUnauthenticated))
			return
		}

		sub, obj, act := r(c, identity)
		allowed, err := iam.Authorize(enforcer, identity, sub, obj, act)
		if err != nil {
			stdgin.Error(c, err)
			return
		}
		if !allowed {
			stdgin.Error(c, stderr.NewForbidden(fmt.Sprintf("access denied: %s cannot %s %s", sub, act, obj)))
			return
		}

		c.Next()
	}
}
