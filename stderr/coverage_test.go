package stderr

import (
	"errors"
	"net/http"
	"testing"
)

func TestCode_String(t *testing.T) {
	tests := []struct {
		code Code
		want string
	}{
		{CodeOK, "OK"}, {CodeInvalidParam, "INVALID_PARAM"}, {CodeValidationFailed, "VALIDATION_FAILED"},
		{CodeMalformedBody, "MALFORMED_BODY"}, {CodeBadRequest, "BAD_REQUEST"}, {CodeUnauthorized, "UNAUTHORIZED"},
		{CodeForbidden, "FORBIDDEN"}, {CodeNotFound, "NOT_FOUND"}, {CodeConflict, "CONFLICT"},
		{CodeAlreadyExists, "ALREADY_EXISTS"}, {CodeInternal, "INTERNAL_SERVER_ERROR"},
		{CodeDatabaseError, "DATABASE_ERROR"}, {CodeCacheError, "CACHE_ERROR"},
		{CodeUnavailable, "SERVICE_UNAVAILABLE"}, {CodeTimeout, "GATEWAY_TIMEOUT"},
		{CodeLockHeld, "LOCK_HELD"}, {CodeTooManyRequests, "TOO_MANY_REQUESTS"}, {Code(9999), "UNKNOWN_ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.code.String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestError_NilReceiver(t *testing.T) {
	var e *Error
	if e.Error() != "<nil>" || e.Unwrap() != nil || !e.Is(nil) || e.Is(errors.New("x")) {
		t.Error("nil receiver must be safe")
	}
	if e.HTTPStatus() != http.StatusOK || e.StatusCode() != CodeOK || e.ErrorMessage() != "" {
		t.Error("nil receiver accessors must return zero values")
	}
	if e.WithDetails(FieldError{}) != nil || e.WithKey("k") != nil || e.WithCause(errors.New("x")) != nil {
		t.Error("chaining on nil must return nil")
	}
}

func TestError_Accessors(t *testing.T) {
	cause := errors.New("db down")
	tests := []struct {
		name       string
		err        *Error
		wantStatus int
		wantCode   Code
		wantMsg    string
		wantCause  error
	}{
		{"unavailable with cause", NewUnavailable("busy", cause), http.StatusServiceUnavailable, CodeUnavailable, "busy", cause},
		{"unavailable no cause", NewUnavailable("busy"), http.StatusServiceUnavailable, CodeUnavailable, "busy", nil},
		{"internal with nil cause", NewInternal("oops", nil), http.StatusInternalServerError, CodeInternal, "oops", nil},
		{"formatted", Newf(CodeNotFound, "user %d", 7), http.StatusNotFound, CodeNotFound, "user 7", nil},
		{"wrap with message override", Wrap(CodeDatabaseError, cause, "query failed"), http.StatusInternalServerError, CodeDatabaseError, "query failed", cause},
		{"wrap keeps cause text", Wrap(CodeTimeout, cause), http.StatusGatewayTimeout, CodeTimeout, "db down", cause},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.HTTPStatus() != tt.wantStatus || tt.err.StatusCode() != tt.wantCode ||
				tt.err.ErrorMessage() != tt.wantMsg || !errors.Is(tt.err.Unwrap(), tt.wantCause) {
				t.Errorf("unexpected accessors for %v", tt.err)
			}
			if tt.err.Error() == "" {
				t.Error("Error() must not be empty")
			}
		})
	}
	if Wrap(CodeInternal, nil) != nil {
		t.Error("Wrap(nil) must return nil")
	}
}

func TestError_Is_CodeMatching(t *testing.T) {
	if !errors.Is(NewNotFound("a"), NewNotFound("b")) {
		t.Error("same code must match")
	}
	if errors.Is(NewNotFound("a"), NewConflict("b")) {
		t.Error("different code must not match")
	}
	if NewNotFound("a").Is(errors.New("plain")) {
		t.Error("non-stderr target must not match")
	}
}
