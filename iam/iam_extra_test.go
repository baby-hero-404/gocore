package iam

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/golang-jwt/jwt/v5"
)

func unauthorized(t *testing.T, err error) {
	t.Helper()
	var se *stderr.Error
	if !errors.As(err, &se) || se.Code != stderr.CodeUnauthorized {
		t.Fatalf("want CodeUnauthorized, got %v", err)
	}
}

func TestIdentityContextRoundTrip(t *testing.T) {
	id := Identity{ID: "u1", Roles: []string{"admin", "user"}}

	got, ok := IdentityFromContext(WithIdentity(context.Background(), id))
	if !ok || got.ID != "u1" {
		t.Fatalf("got %+v ok=%v", got, ok)
	}
	if _, ok := IdentityFromContext(context.Background()); ok {
		t.Error("empty context must not yield an identity")
	}
	if _, ok := IdentityFromContext(nil); ok { //nolint:staticcheck // nil context is tolerated by contract
		t.Error("nil context must not yield an identity")
	}
}

func TestIdentityHasRole(t *testing.T) {
	id := Identity{Roles: []string{"admin", "user"}}
	for role, want := range map[string]bool{"admin": true, "user": true, "root": false, "": false} {
		if got := id.HasRole(role); got != want {
			t.Errorf("HasRole(%q) = %v, want %v", role, got, want)
		}
	}
	if (Identity{}).HasRole("admin") {
		t.Error("identity without roles has no role")
	}
}

func TestDefaultConfigs(t *testing.T) {
	jc := DefaultJWTConfig()
	if jc.SigningMethod != "HS256" || jc.ClockSkew != 5*time.Second {
		t.Errorf("unexpected JWT defaults: %+v", jc)
	}
	if DefaultOAuth2Config().Timeout != 5*time.Second {
		t.Errorf("unexpected OAuth2 defaults: %+v", DefaultOAuth2Config())
	}
}

func signWith(t *testing.T, method jwt.SigningMethod, secret string, claims jwt.MapClaims) string {
	t.Helper()
	tok, err := jwt.NewWithClaims(method, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestJWTValidator_EnforcesConfiguredClaims(t *testing.T) {
	const secret = "validator-test-secret" // gitleaks:allow (test fixture, not a real secret)
	exp := time.Now().Add(time.Hour).Unix()
	base := func(extra jwt.MapClaims) jwt.MapClaims {
		c := jwt.MapClaims{"sub": "u1", "exp": exp}
		for k, v := range extra {
			c[k] = v
		}
		return c
	}

	tests := []struct {
		name  string
		cfg   JWTConfig
		token string
		ok    bool
	}{
		{"issuer matches", JWTConfig{Secret: secret, Issuer: "gocore"}, signWith(t, jwt.SigningMethodHS256, secret, base(jwt.MapClaims{"iss": "gocore"})), true},
		{"issuer mismatch", JWTConfig{Secret: secret, Issuer: "gocore"}, signWith(t, jwt.SigningMethodHS256, secret, base(jwt.MapClaims{"iss": "evil"})), false},
		{"issuer missing", JWTConfig{Secret: secret, Issuer: "gocore"}, signWith(t, jwt.SigningMethodHS256, secret, base(nil)), false},
		{"issuer not enforced when unset", JWTConfig{Secret: secret}, signWith(t, jwt.SigningMethodHS256, secret, base(jwt.MapClaims{"iss": "anything"})), true},
		{"audience matches", JWTConfig{Secret: secret, Audience: "api"}, signWith(t, jwt.SigningMethodHS256, secret, base(jwt.MapClaims{"aud": "api"})), true},
		{"audience mismatch", JWTConfig{Secret: secret, Audience: "api"}, signWith(t, jwt.SigningMethodHS256, secret, base(jwt.MapClaims{"aud": "other"})), false},
		{"signing method must match config", JWTConfig{Secret: secret, SigningMethod: "HS512"}, signWith(t, jwt.SigningMethodHS256, secret, base(nil)), false},
		{"signing method matches config", JWTConfig{Secret: secret, SigningMethod: "HS512"}, signWith(t, jwt.SigningMethodHS512, secret, base(nil)), true},
		{"clock skew tolerates a just-expired token", JWTConfig{Secret: secret, ClockSkew: time.Minute}, signWith(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{"sub": "u1", "exp": time.Now().Add(-10 * time.Second).Unix()}), true},
		{"expired beyond skew is rejected", JWTConfig{Secret: secret, ClockSkew: time.Second}, signWith(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{"sub": "u1", "exp": time.Now().Add(-time.Minute).Unix()}), false},
		{"wrong secret", JWTConfig{Secret: secret}, signWith(t, jwt.SigningMethodHS256, "another-secret", base(nil)), false},
		{"alg none is rejected", JWTConfig{Secret: secret}, func() string {
			tok, _ := jwt.NewWithClaims(jwt.SigningMethodNone, base(nil)).SignedString(jwt.UnsafeAllowNoneSignatureType)
			return tok
		}(), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewJWTValidator(tt.cfg).Validate(context.Background(), tt.token)
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

func TestJWTValidator_ClaimShapes(t *testing.T) {
	const secret = "claims-test-secret" // gitleaks:allow (test fixture, not a real secret)
	exp := time.Now().Add(time.Hour).Unix()
	tests := []struct {
		name      string
		claims    jwt.MapClaims
		wantID    string
		wantRoles []string
	}{
		{"id claim when sub missing", jwt.MapClaims{"id": "from-id", "exp": exp}, "from-id", nil},
		{"numeric sub", jwt.MapClaims{"sub": float64(42), "exp": exp}, "", nil},
		{"single role claim", jwt.MapClaims{"sub": "u", "role": "admin", "exp": exp}, "u", []string{"admin"}},
		{"roles array ignores non strings", jwt.MapClaims{"sub": "u", "roles": []any{"a", 5, "b"}, "exp": exp}, "u", []string{"a", "b"}},
	}
	v := NewJWTValidator(JWTConfig{Secret: secret})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := v.Validate(context.Background(), signWith(t, jwt.SigningMethodHS256, secret, tt.claims))
			if err != nil {
				t.Fatal(err)
			}
			if tt.name != "numeric sub" && id.ID != tt.wantID {
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
	if id, _ := v.Validate(context.Background(), signWith(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{"sub": float64(42), "exp": exp})); id.ID != "42" {
		t.Errorf("numeric sub = %q, want 42", id.ID)
	}
	if _, err := v.Validate(context.Background(), "   "); err == nil {
		t.Error("blank token must be rejected")
	}
}

func TestOAuth2Validator_Cases(t *testing.T) {
	var statuses = map[string]struct {
		status int
		body   string
	}{
		"ok-sub":   {200, `{"sub":"s1","email":"a@b.c"}`},
		"ok-id":    {200, `{"id":"i1"}`},
		"rejected": {403, ``},
		"badjson":  {200, `{`},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, ok := statuses[r.Header.Get("Authorization")[len("Bearer "):]]
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(resp.status)
		_, _ = w.Write([]byte(resp.body))
	}))
	defer srv.Close()

	tests := []struct {
		name     string
		cfg      OAuth2Config
		token    string
		wantID   string
		wantCode stderr.Code
	}{
		{"sub claim", OAuth2Config{UserInfoURL: srv.URL}, "ok-sub", "s1", 0},
		{"id claim fallback", OAuth2Config{UserInfoURL: srv.URL}, "ok-id", "i1", 0},
		{"blank token", OAuth2Config{UserInfoURL: srv.URL}, " ", "", stderr.CodeUnauthorized},
		{"missing url", OAuth2Config{}, "ok-sub", "", stderr.CodeInternal},
		{"provider rejects", OAuth2Config{UserInfoURL: srv.URL}, "rejected", "", stderr.CodeUnauthorized},
		{"malformed body", OAuth2Config{UserInfoURL: srv.URL}, "badjson", "", stderr.CodeInternal},
		{"unreachable provider", OAuth2Config{UserInfoURL: "http://127.0.0.1:1", Timeout: 200 * time.Millisecond}, "ok-sub", "", stderr.CodeUnavailable},
		{"invalid url", OAuth2Config{UserInfoURL: "http://bad host/"}, "ok-sub", "", stderr.CodeInternal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := NewOAuth2Validator(tt.cfg).Validate(context.Background(), tt.token)
			if tt.wantCode == 0 {
				if err != nil || id.ID != tt.wantID {
					t.Fatalf("id=%q err=%v", id.ID, err)
				}
				return
			}
			var se *stderr.Error
			if !errors.As(err, &se) || se.Code != tt.wantCode {
				t.Fatalf("want %v, got %v", tt.wantCode, err)
			}
		})
	}
}

func TestOAuth2Validator_CustomClientAndTimeout(t *testing.T) {
	custom := &http.Client{Timeout: time.Second}
	if v := NewOAuth2Validator(OAuth2Config{}, custom); v.httpClient != custom {
		t.Error("custom client must be used")
	}
	if v := NewOAuth2Validator(OAuth2Config{}, nil); v.httpClient.Timeout != 5*time.Second {
		t.Errorf("nil client must fall back to the default, got %v", v.httpClient.Timeout)
	}
	if v := NewOAuth2Validator(OAuth2Config{Timeout: 2 * time.Second}); v.httpClient.Timeout != 2*time.Second {
		t.Errorf("config timeout not applied: %v", v.httpClient.Timeout)
	}
}

func TestJWTSigner_UnsignableMethod(t *testing.T) {
	s, err := NewJWTSigner(JWTConfig{Secret: "s", SigningMethod: "HS384", Issuer: "i", Audience: "a"})
	if err != nil {
		t.Fatal(err)
	}
	tok, err := s.Sign("u", time.Hour, map[string]any{"sub": "ignored", "sid": "x"})
	if err != nil {
		t.Fatal(err)
	}
	id, err := NewJWTValidator(JWTConfig{Secret: "s", SigningMethod: "HS384", Issuer: "i", Audience: "a"}).Validate(context.Background(), tok)
	if err != nil || id.ID != "u" || id.Metadata["sid"] != "x" {
		t.Fatalf("round trip with issuer/audience failed: %+v %v", id, err)
	}
}
