package iam

import (
	"context"
)

// identityCtxKey is a typed context key to avoid collision with other packages.
type identityCtxKey struct{}

// IdentityCtxKey is the typed context key for storing/retrieving Identity from context.Context.
var IdentityCtxKey = identityCtxKey{}

// Identity represents the authenticated principal.
type Identity struct {
	ID       string         `json:"id"`
	Email    string         `json:"email,omitempty"`
	Roles    []string       `json:"roles,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// HasRole checks if the identity possesses a given role.
func (i Identity) HasRole(role string) bool {
	for _, r := range i.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// WithIdentity injects the Identity into the Go standard context.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, IdentityCtxKey, id)
}

// IdentityFromContext extracts the Identity from the Go standard context.
func IdentityFromContext(ctx context.Context) (Identity, bool) {
	if ctx == nil {
		return Identity{}, false
	}
	id, ok := ctx.Value(IdentityCtxKey).(Identity)
	return id, ok
}
