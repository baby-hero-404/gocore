// Package firebase validates Firebase ID tokens into iam.Identity.
package firebase

import (
	"context"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/baby-hero-404/gocore/iam"
	"github.com/baby-hero-404/gocore/stderr"
)

// FirebaseAuthClient defines the interface for verifying Firebase ID tokens.
type FirebaseAuthClient interface {
	VerifyIDToken(ctx context.Context, idToken string) (*auth.Token, error)
}

// FirebaseValidator verifies Firebase authentication ID tokens.
type FirebaseValidator struct {
	client FirebaseAuthClient
}

// NewFirebaseValidator creates a new FirebaseValidator wrapping a Firebase Auth client.
func NewFirebaseValidator(client FirebaseAuthClient) *FirebaseValidator {
	return &FirebaseValidator{client: client}
}

// Validate verifies the Firebase ID token and extracts the iam.Identity.
func (v *FirebaseValidator) Validate(ctx context.Context, tokenStr string) (iam.Identity, error) {
	if strings.TrimSpace(tokenStr) == "" {
		return iam.Identity{}, stderr.NewUnauthorized("missing firebase id token")
	}
	if v.client == nil {
		return iam.Identity{}, stderr.NewInternal("firebase auth client is not configured")
	}

	token, err := v.client.VerifyIDToken(ctx, tokenStr)
	if err != nil {
		return iam.Identity{}, stderr.Wrap(stderr.CodeUnauthorized, err, "invalid or expired firebase token")
	}

	var email string
	if emailVal, ok := token.Claims["email"].(string); ok {
		email = emailVal
	}

	var roles []string
	if rolesVal, ok := token.Claims["roles"].([]any); ok {
		for _, r := range rolesVal {
			if strR, ok := r.(string); ok {
				roles = append(roles, strR)
			}
		}
	} else if roleVal, ok := token.Claims["role"].(string); ok {
		roles = append(roles, roleVal)
	}

	return iam.Identity{
		ID:       token.UID,
		Email:    email,
		Roles:    roles,
		Metadata: token.Claims,
	}, nil
}
