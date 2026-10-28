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

func inFlight(t *testing.T, reg *prometheus.Registry) float64 {
	t.Helper()
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, mf := range mfs {
		if mf.GetName() == "gocore_http_in_flight_requests" {
			return mf.GetMetric()[0].GetGauge().GetValue()
		}
	}
	t.Fatal("in-flight gauge not registered")
	return 0
}

func TestMiddleware_TracksInFlightAndRoutes(t *testing.T) {
	reg := prometheus.NewRegistry()
	recorder := prom.NewRecorder(reg)

	engine := gin.New()
	engine.Use(Middleware(recorder))
	var during float64
	engine.GET("/items/:id", func(c *gin.Context) {
		during = inFlight(t, reg)
		c.Status(http.StatusNoContent)
	})

	for _, path := range []string{"/items/7", "/no-such-route"} {
		engine.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))
	}

	if during != 1 {
		t.Errorf("in flight while handling = %v, want 1", during)
	}
	if after := inFlight(t, reg); after != 0 {
		t.Errorf("in flight after requests = %v, want 0", after)
	}

	mfs, _ := reg.Gather()
	routes := map[string]bool{}
	for _, mf := range mfs {
		if mf.GetName() != "gocore_http_request_duration_seconds" {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, l := range m.GetLabel() {
				if l.GetName() == "route" {
					routes[l.GetValue()] = true
				}
			}
		}
	}
	if !routes["/items/:id"] || !routes["unmatched"] {
		t.Errorf("route labels = %v, want the route template and \"unmatched\"", routes)
	}
}

func TestMiddleware_NilRecorderPassesThrough(t *testing.T) {
	engine := gin.New()
	engine.Use(Middleware(nil))
	engine.GET("/ok", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ok", nil))
	if w.Code != http.StatusOK || w.Body.String() != "ok" {
		t.Errorf("got %d %q", w.Code, w.Body.String())
	}
}
