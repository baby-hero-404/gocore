package prom

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestRecorder_ImplementsAndRecords(t *testing.T) {
	reg := prometheus.NewRegistry()
	r := NewRecorder(reg)
	r.RecordDuration("GET", "/x", 200, 10*time.Millisecond)
	r.IncrementInFlight()
	r.DecrementInFlight()

	mfs, err := reg.Gather()
	if err != nil || len(mfs) == 0 {
		t.Fatalf("expected gathered metrics, got %d, err=%v", len(mfs), err)
	}

	w := httptest.NewRecorder()
	Handler().ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	if w.Code != 200 {
		t.Errorf("metrics handler status = %d", w.Code)
	}
}

func gatherValue(t *testing.T, reg *prometheus.Registry, name string, match map[string]string) (float64, bool) {
	t.Helper()
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			labels := map[string]string{}
			for _, l := range m.GetLabel() {
				labels[l.GetName()] = l.GetValue()
			}
			ok := true
			for k, v := range match {
				if labels[k] != v {
					ok = false
				}
			}
			if !ok {
				continue
			}
			switch {
			case m.Counter != nil:
				return m.GetCounter().GetValue(), true
			case m.Gauge != nil:
				return m.GetGauge().GetValue(), true
			case m.Histogram != nil:
				return float64(m.GetHistogram().GetSampleCount()), true
			}
		}
	}
	return 0, false
}

func TestRecorder_MetricAccessorsRecord(t *testing.T) {
	reg := prometheus.NewRegistry()
	r := NewRecorder(reg)

	r.DBConnectDuration().Observe(0.2)
	r.DBQueryErrors().WithLabelValues("not_found").Add(2)
	r.CacheOpErrors().WithLabelValues("get").Inc()
	r.LockAcquireFailures().Inc()
	r.RecordDuration("GET", "/x", 503, time.Millisecond)

	tests := []struct {
		name   string
		metric string
		labels map[string]string
		want   float64
	}{
		{"db connect histogram", "gocore_database_connect_duration_seconds", nil, 1},
		{"db query errors by class", "gocore_database_query_errors_total", map[string]string{"error_class": "not_found"}, 2},
		{"cache errors by op", "gocore_cache_op_errors_total", map[string]string{"op": "get"}, 1},
		{"lock failures", "gocore_lock_acquire_failures_total", nil, 1},
		{"http duration labelled by status", "gocore_http_request_duration_seconds", map[string]string{"method": "GET", "route": "/x", "status": "503"}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := gatherValue(t, reg, tt.metric, tt.labels)
			if !ok || got != tt.want {
				t.Errorf("%s = %v (found=%v), want %v", tt.metric, got, ok, tt.want)
			}
		})
	}
}

func TestRecorder_InFlightGauge(t *testing.T) {
	reg := prometheus.NewRegistry()
	r := NewRecorder(reg)

	r.IncrementInFlight()
	r.IncrementInFlight()
	r.DecrementInFlight()
	if got, ok := gatherValue(t, reg, "gocore_http_in_flight_requests", nil); !ok || got != 1 {
		t.Errorf("in flight = %v (found=%v), want 1", got, ok)
	}
}

func TestNewRecorder_NilRegistererUsesDefault(t *testing.T) {
	// A nil registerer must fall back to the default registry; registering twice there would
	// panic, so run it once and clean up.
	r := NewRecorder(nil)
	t.Cleanup(func() {
		prometheus.Unregister(r.dbConnectDuration)
		prometheus.Unregister(r.dbQueryErrors)
		prometheus.Unregister(r.cacheOpErrors)
		prometheus.Unregister(r.lockAcquireFailures)
		prometheus.Unregister(r.httpRequestDuration)
		prometheus.Unregister(r.httpInFlight)
	})
	if r == nil {
		t.Fatal("expected recorder")
	}
}
