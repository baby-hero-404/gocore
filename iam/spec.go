package iam

import "context"

// KeyUnauthenticated is the i18n key carried by every authentication failure raised by this
// package, so clients and translators can match it regardless of the validator in use.
const KeyUnauthenticated = "error.unauthenticated"

// TokenValidator defines the contract for verifying bearer tokens (JWT, OAuth2, Firebase, etc.).
type TokenValidator interface {
	Validate(ctx context.Context, token string) (Identity, error)
}

// Enforcer abstracts authorization policy enforcement (e.g. Casbin).
type Enforcer interface {
	Enforce(rvals ...any) (bool, error)
}
