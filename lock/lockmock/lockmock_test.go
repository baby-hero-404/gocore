package lockmock

import (
	"context"
	"testing"
	"time"
)

func TestMockLockAndLocker(t *testing.T) {
	mockLocker := NewLocker(t)
	mockLock := NewLock(t)
	ctx := context.Background()

	mockLock.On("Key").Return("order:123")
	mockLock.On("Release", ctx).Return(nil)
	mockLock.On("Extend", ctx).Return(nil)

	mockLocker.On("Acquire", ctx, "order:123", 5*time.Second).Return(mockLock, nil)

	l, err := mockLocker.Acquire(ctx, "order:123", 5*time.Second)
	if err != nil || l == nil {
		t.Fatalf("expected acquired lock, got %v", err)
	}
	if l.Key() != "order:123" {
		t.Errorf("expected key order:123, got %s", l.Key())
	}
	if err := l.Extend(ctx); err != nil {
		t.Errorf("extend error: %v", err)
	}
	if err := l.Release(ctx); err != nil {
		t.Errorf("release error: %v", err)
	}
}
