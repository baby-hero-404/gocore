package monitormock

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMockMonitor(t *testing.T) {
	mockRec := NewMetricsRecorder(t)
	mockRec.On("RecordDuration", "GET", "/api/v1/bikes", 200, 100*time.Millisecond).Return()
	mockRec.On("IncrementInFlight").Return()
	mockRec.On("DecrementInFlight").Return()

	mockRec.IncrementInFlight()
	mockRec.RecordDuration("GET", "/api/v1/bikes", 200, 100*time.Millisecond)
	mockRec.DecrementInFlight()

	mockTracker := NewErrorTracker(t)
	ctx := context.Background()
	mockTracker.On("CaptureException", errors.New("boom"), map[string]string{"env": "staging"}).Return()
	mockTracker.On("Flush", ctx, 5*time.Second).Return(true)

	mockTracker.CaptureException(errors.New("boom"), map[string]string{"env": "staging"})
	if ok := mockTracker.Flush(ctx, 5*time.Second); !ok {
		t.Error("expected flush ok")
	}
}
