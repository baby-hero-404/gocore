package stdres

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdlog"
)

type mockTranslator struct{}

func (m *mockTranslator) Translate(ctx context.Context, key string, args ...any) string {
	if key == "user_not_found" {
		if len(args) > 0 {
			return fmt.Sprintf("User with ID %v was not found", args[0])
		}
		return "User not found"
	}
	return key
}

func TestFromError_StderrError(t *testing.T) {
	err := stderr.NewNotFound("resource missing").
		WithDetails(stderr.FieldError{Field: "id", Message: "cannot be empty"})

	env, status := FromError(context.Background(), err)

	if status != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", status)
	}
	if env.Code != int(stderr.CodeNotFound) {
		t.Errorf("expected code %d, got %d", stderr.CodeNotFound, env.Code)
	}
	if env.Message != "resource missing" {
		t.Errorf("expected message 'resource missing', got %s", env.Message)
	}

	details, ok := env.Details.([]stderr.FieldError)
	if !ok || len(details) != 1 {
		t.Fatalf("expected 1 field error detail, got %v", env.Details)
	}
	if details[0].Field != "id" {
		t.Errorf("expected detail field 'id', got %s", details[0].Field)
	}
}

func TestFromError_WithTranslator(t *testing.T) {
	tr := &mockTranslator{}
	err := stderr.NewNotFound("fallback message").WithKey("user_not_found", 42)

	// Translator passed explicitly
	env, status := FromError(context.Background(), err, tr)
	if status != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", status)
	}
	if env.Message != "User with ID 42 was not found" {
		t.Errorf("expected translated message, got %s", env.Message)
	}

	// Translator passed via context
	ctx := WithTranslator(context.Background(), tr)
	env2, _ := FromError(ctx, err)
	if env2.Message != "User with ID 42 was not found" {
		t.Errorf("expected translated message from ctx, got %s", env2.Message)
	}
}

func TestFromError_StandardGoError(t *testing.T) {
	plainErr := errors.New("raw unhandled system failure")
	env, status := FromError(context.Background(), plainErr)

	if status != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", status)
	}
	if env.Code != int(stderr.CodeInternal) {
		t.Errorf("expected code %d, got %d", stderr.CodeInternal, env.Code)
	}
	if env.Message != "raw unhandled system failure" {
		t.Errorf("expected error message to match, got %s", env.Message)
	}
}

func TestFromError_NilError(t *testing.T) {
	env, status := FromError(context.Background(), nil)
	if status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}
	if env.Code != 0 || env.Message != "success" {
		t.Errorf("expected success envelope, got %v", env)
	}
}

func TestFromError_KeyArgsAndRequestID(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		err       error
		wantKey   string
		wantArgs  int
		wantReqID string
	}{
		{
			name:      "stderr error with key, args and request id",
			ctx:       context.WithValue(context.Background(), stdlog.RequestIDCtxKey, "req-123"),
			err:       stderr.NewNotFound("missing").WithKey("user_not_found", 42),
			wantKey:   "user_not_found",
			wantArgs:  1,
			wantReqID: "req-123",
		},
		{
			name:      "stderr error without request id",
			ctx:       context.Background(),
			err:       stderr.NewNotFound("missing"),
			wantKey:   "",
			wantArgs:  0,
			wantReqID: "",
		},
		{
			name:      "plain error still carries request id",
			ctx:       context.WithValue(context.Background(), stdlog.RequestIDCtxKey, "req-9"),
			err:       errors.New("boom"),
			wantReqID: "req-9",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, _ := FromError(tt.ctx, tt.err)
			if env.Key != tt.wantKey || len(env.Args) != tt.wantArgs || env.RequestID != tt.wantReqID {
				t.Errorf("got key=%q args=%d request_id=%q", env.Key, len(env.Args), env.RequestID)
			}
		})
	}
}
