package iamgin

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baby-hero-404/gocore/iam"
	"github.com/gin-gonic/gin"
)

type fakeEnforcer struct {
	allow func(sub, obj, act string) (bool, error)
}

func (f fakeEnforcer) Enforce(rvals ...any) (bool, error) {
	return f.allow(rvals[0].(string), rvals[1].(string), rvals[2].(string))
}

func serve(t *testing.T, mw gin.HandlerFunc, id *iam.Identity) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	if id != nil {
		engine.Use(func(c *gin.Context) {
			c.Set(contextKeyIdentityString, *id)
			c.Next()
		})
	}
	engine.Use(mw)
	engine.GET("/orders", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orders", nil))
	return w
}

func TestCasbinMiddleware_Branches(t *testing.T) {
	user := &iam.Identity{ID: "u1", Roles: []string{"member"}}
	allowAll := fakeEnforcer{allow: func(string, string, string) (bool, error) { return true, nil }}
	denyAll := fakeEnforcer{allow: func(string, string, string) (bool, error) { return false, nil }}
	broken := fakeEnforcer{allow: func(string, string, string) (bool, error) { return false, errors.New("policy store down") }}

	tests := []struct {
		name     string
		enforcer iam.Enforcer
		resolver []CasbinResolver
		id       *iam.Identity
		want     int
	}{
		{"nil enforcer", nil, nil, user, http.StatusInternalServerError},
		{"unauthenticated", allowAll, nil, nil, http.StatusUnauthorized},
		{"enforcer error", broken, nil, user, http.StatusInternalServerError},
		{"denied", denyAll, nil, user, http.StatusForbidden},
		{"allowed", allowAll, nil, user, http.StatusOK},
		{"custom resolver is used", fakeEnforcer{allow: func(sub, obj, act string) (bool, error) {
			return sub == "tenant-7" && obj == "orders" && act == "read", nil
		}}, []CasbinResolver{func(*gin.Context, iam.Identity) (string, string, string) { return "tenant-7", "orders", "read" }}, user, http.StatusOK},
		{"nil custom resolver falls back to default", allowAll, []CasbinResolver{nil}, user, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if w := serve(t, CasbinMiddleware(tt.enforcer, tt.resolver...), tt.id); w.Code != tt.want {
				t.Errorf("status = %d, want %d (%s)", w.Code, tt.want, w.Body.String())
			}
		})
	}
}

func TestIdentityFromGin_FallsBackToRequestContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	if _, ok := IdentityFromGin(c); ok {
		t.Fatal("no identity anywhere must report false")
	}

	c.Set(contextKeyIdentityString, "not-an-identity")
	if _, ok := IdentityFromGin(c); ok {
		t.Fatal("a value of the wrong type must be ignored")
	}

	c.Request = c.Request.WithContext(iam.WithIdentity(c.Request.Context(), iam.Identity{ID: "from-ctx"}))
	if id, ok := IdentityFromGin(c); !ok || id.ID != "from-ctx" {
		t.Fatalf("got %+v ok=%v, want identity from request context", id, ok)
	}
}
