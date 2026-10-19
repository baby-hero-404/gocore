package stdlogmock

import (
	"context"
	"log/slog"
	"testing"
)

func TestMockLogger(t *testing.T) {
	mockL := NewLogger(t)
	ctx := context.Background()

	mockL.On("Info", "service starting", []any{"port", 8080}).Return()
	mockL.On("Debug", "debugging", []any(nil)).Return()
	mockL.On("Warn", "warning", []any(nil)).Return()
	mockL.On("Error", "error occurred", []any(nil)).Return()
	mockL.On("Log", ctx, slog.LevelInfo, "custom log", []any(nil)).Return()
	mockL.On("With", []any{"env", "test"}).Return(mockL)
	mockL.On("WithContext", ctx).Return(mockL)
	mockL.On("Slog").Return((*slog.Logger)(nil))

	mockL.Info("service starting", "port", 8080)
	mockL.Debug("debugging")
	mockL.Warn("warning")
	mockL.Error("error occurred")
	mockL.Log(ctx, slog.LevelInfo, "custom log")
	_ = mockL.With("env", "test")
	_ = mockL.WithContext(ctx)
	if sl := mockL.Slog(); sl != nil {
		t.Errorf("expected nil slog")
	}
}
