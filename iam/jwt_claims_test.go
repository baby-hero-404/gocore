package iam

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const claimsTestSecret = "claims-config-secret" // gitleaks:allow (test fixture, not a real secret)

func tokenWith(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	if _, ok := claims["exp"]; !ok {
		claims["exp"] = time.Now().Add(time.Hour).Unix()
	}
	return signWith(t, jwt.SigningMethodHS256, claimsTestSecret, claims)
}

func TestJWTValidator_CustomClaimNames(t *testing.T) {
	tests := []struct {
		name      string
		cfg       JWTConfig
		claims    jwt.MapClaims
		wantID    string
		wantRoles []string
	}{
		{"custom id and single role", JWTConfig{IDClaim: "userId", RolesClaim: "role"}, jwt.MapClaims{"userId": "u1", "role": "admin"}, "u1", []string{"admin"}},
		{"custom roles list", JWTConfig{RolesClaim: "perms"}, jwt.MapClaims{"sub": "u2", "perms": []any{"a", 3, "b"}}, "u2", []string{"a", "b"}},
		{"custom numeric id", JWTConfig{IDClaim: "uid"}, jwt.MapClaims{"uid": float64(42)}, "42", nil},
		{"custom id claim ignores sub", JWTConfig{IDClaim: "userId"}, jwt.MapClaims{"sub": "other", "userId": "u3"}, "u3", nil},
		{"custom roles claim ignores roles", JWTConfig{RolesClaim: "perms"}, jwt.MapClaims{"sub": "u", "roles": []any{"x"}}, "u", nil},
		{"defaults still read sub and roles", JWTConfig{}, jwt.MapClaims{"sub": "u4", "roles": []any{"r"}}, "u4", []string{"r"}},
		{"defaults still fall back to id and role", JWTConfig{}, jwt.MapClaims{"id": "u5", "role": "viewer"}, "u5", []string{"viewer"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cfg.Secret = claimsTestSecret
			id, err := NewJWTValidator(tt.cfg).Validate(context.Background(), tokenWith(t, tt.claims))
			if err != nil {
				t.Fatal(err)
			}
			if id.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", id.ID, tt.wantID)
			}
			if len(id.Roles) != len(tt.wantRoles) {
				t.Fatalf("roles = %v, want %v", id.Roles, tt.wantRoles)
			}
			for i := range tt.wantRoles {
				if id.Roles[i] != tt.wantRoles[i] {
					t.Errorf("roles = %v, want %v", id.Roles, tt.wantRoles)
				}
			}
		})
	}
}

func TestJWTTokenType(t *testing.T) {
	cfgFor := func(typ string) JWTConfig { return JWTConfig{Secret: claimsTestSecret, TokenType: typ} }
	signerFor := func(typ string) *JWTSigner {
		s, err := NewJWTSigner(cfgFor(typ))
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	ctx := context.Background()

	access, err := signerFor("access").Sign("u1", time.Hour, map[string]any{"typ": "forged"})
	if err != nil {
		t.Fatal(err)
	}
	refresh, _ := signerFor("refresh").Sign("u1", time.Hour, nil)
	untyped, _ := signerFor("").Sign("u1", time.Hour, nil)

	tests := []struct {
		name      string
		validator JWTConfig
		token     string
		ok        bool
	}{
		{"access accepted by access validator", cfgFor("access"), access, true},
		{"refresh accepted by refresh validator", cfgFor("refresh"), refresh, true},
		{"refresh rejected by access validator", cfgFor("access"), refresh, false},
		{"access rejected by refresh validator", cfgFor("refresh"), access, false},
		{"untyped token rejected by typed validator", cfgFor("access"), untyped, false},
		{"typed token accepted by untyped validator", cfgFor(""), access, true},
		{"untyped token accepted by untyped validator", cfgFor(""), untyped, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewJWTValidator(tt.validator).Validate(ctx, tt.token)
			if tt.ok {
				if err != nil {
					t.Fatalf("expected valid, got %v", err)
				}
				return
			}
			unauthorized(t, err)
		})
	}
}

func TestJWTSigner_WritesConfiguredIDClaim(t *testing.T) {
	cfg := JWTConfig{Secret: claimsTestSecret, IDClaim: "userId", RolesClaim: "role"}
	s, err := NewJWTSigner(cfg)
	if err != nil {
		t.Fatal(err)
	}
	tok, err := s.Sign("u9", time.Hour, map[string]any{"role": "admin"})
	if err != nil {
		t.Fatal(err)
	}
	id, err := NewJWTValidator(cfg).Validate(context.Background(), tok)
	if err != nil || id.ID != "u9" || len(id.Roles) != 1 || id.Roles[0] != "admin" {
		t.Fatalf("round trip with custom claims failed: %+v %v", id, err)
	}
	if id.Metadata["userId"] != "u9" || id.Metadata["sub"] != "u9" {
		t.Errorf("signer must write both sub and the configured id claim: %v", id.Metadata)
	}
}

func TestJWTValidator_ConfiguredIDClaimIsRequired(t *testing.T) {
	v := NewJWTValidator(JWTConfig{Secret: claimsTestSecret, IDClaim: "userId"})
	tests := []struct {
		name   string
		claims jwt.MapClaims
	}{
		{"claim missing", jwt.MapClaims{"sub": "other"}},
		{"claim empty", jwt.MapClaims{"userId": ""}},
		{"claim has the wrong type", jwt.MapClaims{"userId": true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := v.Validate(context.Background(), tokenWith(t, tt.claims))
			unauthorized(t, err)
		})
	}

	// Without IDClaim the v0.2.0 behaviour stays: a token without any id still yields an Identity.
	if id, err := NewJWTValidator(JWTConfig{Secret: claimsTestSecret}).Validate(context.Background(), tokenWith(t, jwt.MapClaims{"email": "x@y.z"})); err != nil || id.ID != "" {
		t.Fatalf("default config must keep accepting id-less tokens: %+v %v", id, err)
	}
}
