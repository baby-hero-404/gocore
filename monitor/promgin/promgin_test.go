package promgin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baby-hero-404/gocore/monitor/prom"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestMiddlewareAndHandler(t *testing.T) {
	reg := prometheus.NewRegistry()
	recorder := prom.NewRecorder(reg)

	engine := gin.New()
	engine.Use(Middleware(recorder))
	engine.GET("/test-metric", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})
	engine.GET("/metrics", Handler())

	// Trigger a request to record metric
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test-metric", nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Fetch /metrics
	wMetrics := httptest.NewRecorder()
	reqMetrics := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	engine.ServeHTTP(wMetrics, reqMetrics)

	if wMetrics.Code != http.StatusOK {
		t.Errorf("expected status 200 for /metrics, got %d", wMetrics.Code)
	}
	body := wMetrics.Body.String()
	if body == "" {
		t.Error("expected non-empty metrics output")
	}
}
