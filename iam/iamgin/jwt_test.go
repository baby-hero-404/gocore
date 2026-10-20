package iamgin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/baby-hero-404/gocore/iam"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func generateTestToken(secret string, sub string, email string, roles []string, exp time.Duration) string {
	claims := jwt.MapClaims{"sub": sub, "email": email, "roles": roles, "exp": time.Now().Add(exp).Unix()}
	tokenString, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	return tokenString
}

func TestJWTMiddleware(t *testing.T) {
	secret := "jwt-middleware-secret"
	validator := iam.NewJWTValidator(iam.JWTConfig{Secret: secret})

	engine := gin.New()
	engine.Use(JWTMiddleware(validator))
	engine.GET("/protected", func(c *gin.Context) {
		id, ok := IdentityFromGin(c)
		if !ok {
			c.String(http.StatusInternalServerError, "no identity")
			return
		}
		c.String(http.StatusOK, id.ID)
	})

	// Case 1: No auth header -> 401
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for missing header, got %d", w.Code)
	}

	// Case 2: Valid Bearer token -> 200
	validToken := generateTestToken(secret, "user-abc", "abc@example.com", []string{"editor"}, time.Hour)
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req2.Header.Set("Authorization", "Bearer "+validToken)
	engine.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w2.Code)
	}
	if w2.Body.String() != "user-abc" {
		t.Errorf("expected body user-abc, got %s", w2.Body.String())
	}
}

func TestJWTMiddleware_WithCookie(t *testing.T) {
	secret := "jwt-cookie-secret"
	validator := iam.NewJWTValidator(iam.JWTConfig{Secret: secret})

	engine := gin.New()
	engine.Use(JWTMiddleware(validator, WithCookie("session")))
	engine.GET("/protected", func(c *gin.Context) {
		id, _ := IdentityFromGin(c)
		c.String(http.StatusOK, id.ID)
	})

	cookieToken := generateTestToken(secret, "from-cookie", "", nil, time.Hour)
	headerToken := generateTestToken(secret, "from-header", "", nil, time.Hour)

	tests := []struct {
		name     string
		cookie   string
		header   string
		wantCode int
		wantBody string
	}{
		{"cookie only", cookieToken, "", http.StatusOK, "from-cookie"},
		{"cookie takes precedence over header", cookieToken, "Bearer " + headerToken, http.StatusOK, "from-cookie"},
		{"falls back to header when cookie absent", "", "Bearer " + headerToken, http.StatusOK, "from-header"},
		{"invalid cookie token rejected", "not-a-jwt", "", http.StatusUnauthorized, ""},
		{"neither cookie nor header", "", "", http.StatusUnauthorized, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "session", Value: tt.cookie})
			}
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			engine.ServeHTTP(w, req)
			if w.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantCode)
			}
			if tt.wantBody != "" && w.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", w.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestJWTMiddleware_UnauthenticatedCarriesKey(t *testing.T) {
	engine := gin.New()
	engine.Use(JWTMiddleware(iam.NewJWTValidator(iam.JWTConfig{Secret: "s"}), WithCookie("session")))
	engine.GET("/p", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	for name, setup := range map[string]func(*http.Request){
		"no credentials": func(*http.Request) {},
		"bad cookie":     func(r *http.Request) { r.AddCookie(&http.Cookie{Name: "session", Value: "junk"}) },
		"bad bearer":     func(r *http.Request) { r.Header.Set("Authorization", "Bearer junk") },
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/p", nil)
		setup(req)
		engine.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), `"key":"`+iam.KeyUnauthenticated+`"`) {
			t.Errorf("%s: status=%d body=%s", name, w.Code, w.Body.String())
		}
	}
}
