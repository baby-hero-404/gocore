package iamhttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/baby-hero-404/gocore/iam"
	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdres"
)

const secret = "iamhttp-test-secret-0123456789abc" // gitleaks:allow (test fixture, not a real secret)

type fakeValidator struct {
	id  iam.Identity
	err error
}

func (f fakeValidator) Validate(context.Context, string) (iam.Identity, error) { return f.id, f.err }

type fakeEnforcer struct {
	allow func(sub, obj, act string) (bool, error)
}

func (f fakeEnforcer) Enforce(r ...any) (bool, error) {
	return f.allow(r[0].(string), r[1].(string), r[2].(string))
}

func envelope(t *testing.T, w *httptest.ResponseRecorder) stdres.Envelope {
	t.Helper()
	var env stdres.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("body is not an envelope: %q (%v)", w.Body.String(), err)
	}
	return env
}

func echoID(w http.ResponseWriter, r *http.Request) {
	id, ok := IdentityFromRequest(r)
	if !ok {
		_, _ = w.Write([]byte("<none>"))
		return
	}
	fromCtx, _ := iam.IdentityFromContext(r.Context())
	_, _ = w.Write([]byte(id.ID + "|" + fromCtx.ID))
}

func TestJWT(t *testing.T) {
	signer, err := iam.NewJWTSigner(iam.JWTConfig{Secret: secret, TokenType: "access"})
	if err != nil {
		t.Fatal(err)
	}
	valid, _ := signer.Sign("user-1", time.Hour, nil)
	expired, _ := signer.Sign("user-1", time.Nanosecond, nil)
	time.Sleep(1100 * time.Millisecond)
	refreshSigner, _ := iam.NewJWTSigner(iam.JWTConfig{Secret: secret, TokenType: "refresh"})
	refresh, _ := refreshSigner.Sign("user-1", time.Hour, nil)

	validator := iam.NewJWTValidator(iam.JWTConfig{Secret: secret, TokenType: "access"})

	tests := []struct {
		name       string
		opts       []Option
		header     string
		cookie     *http.Cookie
		wantStatus int
		wantBody   string
	}{
		{"valid bearer token", nil, "Bearer " + valid, nil, http.StatusOK, "user-1|user-1"},
		{"missing header", nil, "", nil, http.StatusUnauthorized, ""},
		{"wrong scheme", nil, "Basic abc", nil, http.StatusUnauthorized, ""},
		{"garbage token", nil, "Bearer nope", nil, http.StatusUnauthorized, ""},
		{"expired token", nil, "Bearer " + expired, nil, http.StatusUnauthorized, ""},
		{"refresh token is rejected as access", nil, "Bearer " + refresh, nil, http.StatusUnauthorized, ""},
		{"cookie is used when configured", []Option{WithCookie("sid")}, "", &http.Cookie{Name: "sid", Value: valid}, http.StatusOK, "user-1|user-1"},
		{"cookie wins over header", []Option{WithCookie("sid")}, "Bearer nope", &http.Cookie{Name: "sid", Value: valid}, http.StatusOK, "user-1|user-1"},
		{"empty cookie falls back to the header", []Option{WithCookie("sid")}, "Bearer " + valid, &http.Cookie{Name: "sid", Value: ""}, http.StatusOK, "user-1|user-1"},
		{"cookie is ignored unless configured", nil, "", &http.Cookie{Name: "sid", Value: valid}, http.StatusUnauthorized, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			h := JWT(validator, tt.opts...)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; echoID(w, r) }))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			if w.Code != tt.wantStatus || called != (tt.wantStatus == http.StatusOK) {
				t.Fatalf("status = %d handler called = %v (%s)", w.Code, called, w.Body.String())
			}
			if tt.wantStatus == http.StatusOK && w.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", w.Body.String(), tt.wantBody)
			}
			if tt.wantStatus == http.StatusUnauthorized {
				if env := envelope(t, w); env.Code != int(stderr.CodeUnauthorized) || env.Key != iam.KeyUnauthenticated {
					t.Errorf("envelope = %+v, want unauthorized with key %q", env, iam.KeyUnauthenticated)
				}
			}
		})
	}
}

func TestJWT_ValidatorFailureIsAnEnvelope(t *testing.T) {
	h := JWT(fakeValidator{err: errors.New("jwks endpoint down: password=hunter2")})(http.HandlerFunc(echoID))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer abc")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if env := envelope(t, w); env.Message != "internal server error" {
		t.Errorf("unclassified validator errors must stay generic, got %q", env.Message)
	}
}

func TestCasbin(t *testing.T) {
	user := iam.Identity{ID: "u1", Roles: []string{"member"}}
	allowAll := fakeEnforcer{allow: func(string, string, string) (bool, error) { return true, nil }}
	denyAll := fakeEnforcer{allow: func(string, string, string) (bool, error) { return false, nil }}
	broken := fakeEnforcer{allow: func(string, string, string) (bool, error) { return false, errors.New("policy store down") }}
	exact := fakeEnforcer{allow: func(sub, obj, act string) (bool, error) {
		return sub == "member" && obj == "/orders" && act == "GET", nil
	}}

	tests := []struct {
		name     string
		enforcer iam.Enforcer
		resolver []Resolver
		identity *iam.Identity
		path     string
		want     int
	}{
		{"nil enforcer", nil, nil, &user, "/orders", http.StatusInternalServerError},
		{"unauthenticated", allowAll, nil, nil, "/orders", http.StatusUnauthorized},
		{"enforcer error", broken, nil, &user, "/orders", http.StatusInternalServerError},
		{"denied", denyAll, nil, &user, "/orders", http.StatusForbidden},
		{"allowed", allowAll, nil, &user, "/orders", http.StatusOK},
		{"default resolver uses role, path and method", exact, nil, &user, "/orders", http.StatusOK},
		{"default resolver denies another path", exact, nil, &user, "/admin", http.StatusForbidden},
		{"custom resolver", fakeEnforcer{allow: func(sub, obj, act string) (bool, error) { return sub == "t7" && obj == "orders" && act == "read", nil }},
			[]Resolver{func(*http.Request, iam.Identity) (string, string, string) { return "t7", "orders", "read" }}, &user, "/anything", http.StatusOK},
		{"nil custom resolver falls back to the default", exact, []Resolver{nil}, &user, "/orders", http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			h := Casbin(tt.enforcer, tt.resolver...)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.identity != nil {
				req = req.WithContext(iam.WithIdentity(req.Context(), *tt.identity))
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != tt.want || called != (tt.want == http.StatusOK) {
				t.Errorf("status = %d (want %d), handler called = %v", w.Code, tt.want, called)
			}
			if tt.want != http.StatusOK {
				if env := envelope(t, w); env.Code == 0 {
					t.Errorf("error responses must be envelopes: %s", w.Body.String())
				}
			}
		})
	}
}

func TestJWTThenCasbin_Chain(t *testing.T) {
	validator := fakeValidator{id: iam.Identity{ID: "u1", Roles: []string{"admin"}}}
	enforcer := fakeEnforcer{allow: func(sub, obj, act string) (bool, error) { return sub == "admin", nil }}
	h := JWT(validator)(Casbin(enforcer)(http.HandlerFunc(echoID)))

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer anything")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK || w.Body.String() != "u1|u1" {
		t.Errorf("got %d %q", w.Code, w.Body.String())
	}
}

func TestIdentityFromRequest_NoIdentity(t *testing.T) {
	if _, ok := IdentityFromRequest(httptest.NewRequest(http.MethodGet, "/", nil)); ok {
		t.Error("a request that did not pass JWT has no identity")
	}
}
