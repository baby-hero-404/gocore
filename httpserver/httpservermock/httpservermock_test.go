package httpservermock

import (
	"context"
	"github.com/gin-gonic/gin"
	"testing"
)

func TestMockServer(t *testing.T) {
	mockSrv := NewServer(t)
	ctx := context.Background()

	mockSrv.On("Engine").Return((*gin.Engine)(nil))
	mockSrv.On("Start").Return(nil)
	mockSrv.On("Shutdown", ctx).Return(nil)

	if e := mockSrv.Engine(); e != nil {
		t.Errorf("expected nil engine")
	}
	if err := mockSrv.Start(); err != nil {
		t.Errorf("start error: %v", err)
	}
	if err := mockSrv.Shutdown(ctx); err != nil {
		t.Errorf("shutdown error: %v", err)
	}
}
