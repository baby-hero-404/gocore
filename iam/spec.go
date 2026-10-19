package iam

import "context"

// TokenValidator defines the contract for verifying bearer tokens (JWT, OAuth2, Firebase, etc.).
type TokenValidator interface {
	Validate(ctx context.Context, token string) (Identity, error)
}

// Enforcer abstracts authorization policy enforcement (e.g. Casbin).
type Enforcer interface {
	Enforce(rvals ...any) (bool, error)
}
