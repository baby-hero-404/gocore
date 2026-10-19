package stderrmock

import (
	"github.com/baby-hero-404/gocore/stderr"
	"testing"
)

func TestMockError(t *testing.T) {
	mockErr := NewError(t)
	mockErr.On("Error").Return("mock error")
	mockErr.On("HTTPStatus").Return(404)
	mockErr.On("StatusCode").Return(stderr.CodeNotFound)
	mockErr.On("ErrorMessage").Return("not found")

	if mockErr.Error() != "mock error" {
		t.Errorf("expected 'mock error', got %s", mockErr.Error())
	}
	if mockErr.HTTPStatus() != 404 {
		t.Errorf("expected 404, got %d", mockErr.HTTPStatus())
	}
	if mockErr.StatusCode() != stderr.CodeNotFound {
		t.Errorf("expected stderr.CodeNotFound, got %v", mockErr.StatusCode())
	}
	if mockErr.ErrorMessage() != "not found" {
		t.Errorf("expected 'not found', got %s", mockErr.ErrorMessage())
	}
}
