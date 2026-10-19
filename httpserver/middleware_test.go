package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdres"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRequestIDMiddleware(t *testing.T) {
	engine := gin.New()
	engine.Use(RequestIDMiddleware())
	engine.GET("/ping", func(c *gin.Context) {
		reqID := c.GetString("request_id")
		c.String(http.StatusOK, reqID)
	})

	// Case 1: Without incoming X-Request-ID header
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	respID := w.Header().Get(HeaderRequestID)
	if respID == "" {
		t.Error("expected non-empty X-Request-ID header in response")
	}
	if w.Body.String() != respID {
		t.Errorf("expected context request_id %q to match response header %q", w.Body.String(), respID)
	}

	// Case 2: With incoming X-Request-ID header
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req2.Header.Set(HeaderRequestID, "custom-id-999")
	engine.ServeHTTP(w2, req2)

	if w2.Header().Get(HeaderRequestID) != "custom-id-999" {
		t.Errorf("expected custom-id-999, got %q", w2.Header().Get(HeaderRequestID))
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	engine := gin.New()
	engine.Use(RecoveryMiddleware())
	engine.GET("/panic", func(c *gin.Context) {
		panic("simulated catastrophic panic")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}

	var res stdres.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode error envelope: %v", err)
	}

	if res.Code != int(stderr.CodeInternal) {
		t.Errorf("expected code %d, got %d", stderr.CodeInternal, res.Code)
	}
}

func TestCORSMiddleware(t *testing.T) {
	cfg := Config{
		AllowOrigins: []string{"https://example.com"},
		AllowMethods: []string{"GET", "POST"},
	}

	engine := gin.New()
	engine.Use(CORSMiddleware(cfg))
	engine.GET("/cors", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Preflight OPTIONS request
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/cors", nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204 for OPTIONS preflight, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("unexpected origin header: %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestServer_Initialization(t *testing.T) {
	srv := New(Config{Port: 8888}, WithMiddlewares(func(c *gin.Context) { c.Next() }))
	if srv.Engine() == nil {
		t.Error("expected non-nil gin engine")
	}
}
