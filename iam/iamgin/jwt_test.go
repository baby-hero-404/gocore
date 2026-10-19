package iamgin

import (
	"net/http"
	"net/http/httptest"
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
