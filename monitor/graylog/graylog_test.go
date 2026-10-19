package graylog

import (
	"errors"
	"testing"

	"github.com/baby-hero-404/gocore/stderr"
)

func TestGraylog_EmptyAddrFailsFast(t *testing.T) {
	_, err := NewWriter(Config{Addr: ""})
	if err == nil {
		t.Fatal("expected validation error for empty Addr")
	}
	var stdErr *stderr.Error
	if !errors.As(err, &stdErr) || stdErr.Code != stderr.CodeValidationFailed {
		t.Errorf("expected CodeValidationFailed, got %v", err)
	}
}
