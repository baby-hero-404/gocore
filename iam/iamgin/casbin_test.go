package iamgin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baby-hero-404/gocore/iam"
	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	"github.com/gin-gonic/gin"
)

func createTestEnforcer(t *testing.T) *casbin.Enforcer {
	text := `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = r.sub == p.sub && r.obj == p.obj && r.act == p.act
`
	m, err := model.NewModelFromString(text)
	if err != nil {
		t.Fatalf("failed to create casbin model: %v", err)
	}

	enforcer, err := casbin.NewEnforcer(m)
	if err != nil {
		t.Fatalf("failed to create casbin enforcer: %v", err)
	}

	// Policy: admin can GET /admin/dashboard
	_, _ = enforcer.AddPolicy("admin", "/admin/dashboard", "GET")
	return enforcer
}

func TestCasbinMiddleware(t *testing.T) {
	enforcer := createTestEnforcer(t)

	engine := gin.New()
	// Simulated auth middleware setting Identity
	engine.Use(func(c *gin.Context) {
		role := c.GetHeader("X-Role")
		if role != "" {
			c.Set(contextKeyIdentityString, iam.Identity{
				ID:    "user-1",
				Roles: []string{role},
			})
		}
		c.Next()
	})
	engine.Use(CasbinMiddleware(enforcer))

	engine.GET("/admin/dashboard", func(c *gin.Context) {
		c.String(http.StatusOK, "welcome admin")
	})

	// Case 1: Unauthenticated request -> 401
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/admin/dashboard", nil)
	engine.ServeHTTP(w1, req1)
	if w1.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated request, got %d", w1.Code)
	}

	// Case 2: Unauthorized role (user) -> 403
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/admin/dashboard", nil)
	req2.Header.Set("X-Role", "user")
	engine.ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Errorf("expected 403 for unauthorized role, got %d", w2.Code)
	}

	// Case 3: Authorized role (admin) -> 200
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/admin/dashboard", nil)
	req3.Header.Set("X-Role", "admin")
	engine.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Errorf("expected 200 for authorized role, got %d", w3.Code)
	}
	if w3.Body.String() != "welcome admin" {
		t.Errorf("expected body welcome admin, got %s", w3.Body.String())
	}
}
