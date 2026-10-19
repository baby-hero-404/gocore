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
