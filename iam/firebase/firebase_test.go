package firebase

import (
	"context"
	"errors"
	"testing"

	"firebase.google.com/go/v4/auth"
)

type mockFirebaseAuthClient struct {
	verifyFunc func(ctx context.Context, idToken string) (*auth.Token, error)
}

func (m *mockFirebaseAuthClient) VerifyIDToken(ctx context.Context, idToken string) (*auth.Token, error) {
	return m.verifyFunc(ctx, idToken)
}

func TestFirebaseValidator(t *testing.T) {
	mockClient := &mockFirebaseAuthClient{
		verifyFunc: func(ctx context.Context, idToken string) (*auth.Token, error) {
			if idToken == "valid-firebase-token" {
				return &auth.Token{
					UID: "fb-user-777",
					Claims: map[string]any{
						"email": "firebase@example.com",
						"roles": []any{"user", "driver"},
					},
				}, nil
			}
			return nil, errors.New("firebase token signature invalid")
		},
	}

	validator := NewFirebaseValidator(mockClient)

	// Case 1: Valid token
	identity, err := validator.Validate(context.Background(), "valid-firebase-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if identity.ID != "fb-user-777" {
		t.Errorf("expected ID fb-user-777, got %s", identity.ID)
	}
	if identity.Email != "firebase@example.com" {
		t.Errorf("expected email firebase@example.com, got %s", identity.Email)
	}
	if !identity.HasRole("driver") {
		t.Error("expected identity to have driver role")
	}

	// Case 2: Invalid token
	_, err = validator.Validate(context.Background(), "bad-token")
	if err == nil {
		t.Fatal("expected error for bad token, got nil")
	}

	// Case 3: Empty token
	_, err = validator.Validate(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
}
