package stdresmock

import (
	"context"
	"testing"
)

func TestMockTranslator(t *testing.T) {
	mockTrans := NewTranslator(t)
	ctx := context.Background()
	mockTrans.On("Translate", ctx, "welcome", []any(nil)).Return("Chào mừng")

	result := mockTrans.Translate(ctx, "welcome")
	if result != "Chào mừng" {
		t.Errorf("expected 'Chào mừng', got %s", result)
	}
}
