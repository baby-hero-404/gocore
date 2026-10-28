package stdres

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
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
	if env.Message != "internal server error" {
		t.Errorf("unclassified error text must not reach the client, got %s", env.Message)
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

func TestLogUnclassified(t *testing.T) {
	var buf bytes.Buffer
	prev := stdlog.Get()
	stdlog.SetDefault(stdlog.New(stdlog.WithOutput(&buf)))
	t.Cleanup(func() { stdlog.SetDefault(prev) })

	ctx := stdlog.ContextWithRequestID(context.Background(), "rid-42")
	tests := []struct {
		name    string
		err     error
		wantLog bool
	}{
		{"plain error is logged", errors.New("pq: password authentication failed for user admin"), true},
		{"wrapped plain error is logged", fmt.Errorf("repo: %w", errors.New("boom")), true},
		{"classified error is not logged", stderr.NewInternal("db down"), false},
		{"wrapped classified error is not logged", fmt.Errorf("svc: %w", stderr.NewNotFound("x")), false},
		{"nil is ignored", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			LogUnclassified(ctx, tt.err)
			if got := buf.Len() > 0; got != tt.wantLog {
				t.Fatalf("logged = %v, want %v (%s)", got, tt.wantLog, buf.String())
			}
			if tt.wantLog && !strings.Contains(buf.String(), "rid-42") {
				t.Errorf("log must carry the request id: %s", buf.String())
			}
		})
	}
}

func TestFromError_NeverLeaksUnclassifiedText(t *testing.T) {
	secret := errors.New("dial tcp 10.0.0.5:5432: password=hunter2")
	env, status := FromError(context.Background(), fmt.Errorf("query failed: %w", secret))
	if status != http.StatusInternalServerError || strings.Contains(env.Message, "hunter2") || strings.Contains(env.Message, "10.0.0.5") {
		t.Fatalf("status=%d message=%q must be generic", status, env.Message)
	}
}

func TestValidationError_NilErrorDoesNotPanic(t *testing.T) {
	got := ValidationError(struct{}{}, nil)
	if got == nil || got.Code != stderr.CodeBadRequest {
		t.Fatalf("got %+v, want a bad request error", got)
	}
}
