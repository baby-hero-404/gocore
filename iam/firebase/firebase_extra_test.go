package firebase

import (
	"context"
	"errors"
	"testing"

	"firebase.google.com/go/v4/auth"
	"github.com/baby-hero-404/gocore/stderr"
)

func TestFirebaseValidator_ClaimsAndConfiguration(t *testing.T) {
	client := &mockFirebaseAuthClient{verifyFunc: func(_ context.Context, tok string) (*auth.Token, error) {
		claims := map[string]any{}
		switch tok {
		case "single-role":
			claims["role"] = "admin"
		case "mixed-roles":
			claims["roles"] = []any{"a", 7, "b"}
		case "no-claims":
		}
		return &auth.Token{UID: "uid-" + tok, Claims: claims}, nil
	}}
	v := NewFirebaseValidator(client)

	tests := []struct {
		token     string
		wantRoles []string
	}{
		{"single-role", []string{"admin"}},
		{"mixed-roles", []string{"a", "b"}},
		{"no-claims", nil},
	}
	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			id, err := v.Validate(context.Background(), tt.token)
			if err != nil {
				t.Fatal(err)
			}
			if id.ID != "uid-"+tt.token || len(id.Roles) != len(tt.wantRoles) {
				t.Fatalf("got %+v, want roles %v", id, tt.wantRoles)
			}
			for i, r := range tt.wantRoles {
				if id.Roles[i] != r {
					t.Errorf("roles = %v, want %v", id.Roles, tt.wantRoles)
				}
			}
		})
	}

	t.Run("nil client is an internal error", func(t *testing.T) {
		_, err := NewFirebaseValidator(nil).Validate(context.Background(), "tok")
		var se *stderr.Error
		if !errors.As(err, &se) || se.Code != stderr.CodeInternal {
			t.Fatalf("want CodeInternal, got %v", err)
		}
	})
}
