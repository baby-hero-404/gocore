package stderr

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestCode_HTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		code     Code
		expected int
	}{
		{"OK", CodeOK, http.StatusOK},
		{"InvalidParam", CodeInvalidParam, http.StatusBadRequest},
		{"ValidationFailed", CodeValidationFailed, http.StatusUnprocessableEntity},
		{"MalformedBody", CodeMalformedBody, http.StatusBadRequest},
		{"BadRequest", CodeBadRequest, http.StatusBadRequest},
		{"Unauthorized", CodeUnauthorized, http.StatusUnauthorized},
		{"Forbidden", CodeForbidden, http.StatusForbidden},
		{"TooManyRequests", CodeTooManyRequests, http.StatusTooManyRequests},
		{"NotFound", CodeNotFound, http.StatusNotFound},
		{"Conflict", CodeConflict, http.StatusConflict},
		{"AlreadyExists", CodeAlreadyExists, http.StatusConflict},
		{"LockHeld", CodeLockHeld, http.StatusConflict},
		{"Internal", CodeInternal, http.StatusInternalServerError},
		{"DatabaseError", CodeDatabaseError, http.StatusInternalServerError},
		{"CacheError", CodeCacheError, http.StatusInternalServerError},
		{"Unavailable", CodeUnavailable, http.StatusServiceUnavailable},
		{"Timeout", CodeTimeout, http.StatusGatewayTimeout},
		{"UnknownCode", Code(9999), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.code.HTTPStatus(); got != tt.expected {
				t.Errorf("Code.HTTPStatus() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestError_WrapAndUnwrap(t *testing.T) {
	rootErr := errors.New("underlying db connection failed")
	wrapped := Wrap(CodeDatabaseError, rootErr, "could not query user")

	if wrapped == nil {
		t.Fatal("expected non-nil wrapped error")
	}

	if wrapped.Code != CodeDatabaseError {
		t.Errorf("expected code %v, got %v", CodeDatabaseError, wrapped.Code)
	}

	if wrapped.HTTPStatus() != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, wrapped.HTTPStatus())
	}

	// errors.Unwrap
	if !errors.Is(wrapped, rootErr) {
		t.Errorf("expected errors.Is(wrapped, rootErr) to be true")
	}

	var customErr *Error
	if !errors.As(wrapped, &customErr) {
		t.Errorf("expected errors.As(wrapped, &customErr) to be true")
	}

	// Test nil wrap
	if Wrap(CodeInternal, nil) != nil {
		t.Errorf("expected Wrap(code, nil) to return nil")
	}
}

func TestError_Is(t *testing.T) {
	err1 := New(CodeNotFound, "user not found")
	err2 := New(CodeNotFound, "order not found")
	err3 := New(CodeUnauthorized, "unauthorized")

	if !errors.Is(err1, err2) {
		t.Errorf("expected err1 and err2 with same code to match with errors.Is")
	}

	if errors.Is(err1, err3) {
		t.Errorf("expected err1 and err3 with different codes not to match")
	}
}

func TestError_ChainingAndDetails(t *testing.T) {
	fieldErr := FieldError{
		Field:   "email",
		Message: "email is invalid",
		Tag:     "email",
	}

	err := NewValidation("validation error").
		WithDetails(fieldErr).
		WithKey("invalid_input", "user_form").
		WithCause(errors.New("syntax error"))

	if len(err.Details) != 1 {
		t.Fatalf("expected 1 detail, got %d", len(err.Details))
	}
	if err.Details[0].Field != "email" {
		t.Errorf("expected field email, got %s", err.Details[0].Field)
	}
	if err.Key != "invalid_input" {
		t.Errorf("expected key invalid_input, got %s", err.Key)
	}
	if len(err.Args) != 1 || err.Args[0] != "user_form" {
		t.Errorf("expected args [user_form], got %v", err.Args)
	}
	if err.Err == nil || err.Err.Error() != "syntax error" {
		t.Errorf("expected cause syntax error, got %v", err.Err)
	}
}

func TestError_ConvenienceConstructors(t *testing.T) {
	tests := []struct {
		err        *Error
		wantCode   Code
		wantStatus int
	}{
		{NewBadRequest("bad"), CodeBadRequest, http.StatusBadRequest},
		{NewNotFound("not found"), CodeNotFound, http.StatusNotFound},
		{NewUnauthorized("unauth"), CodeUnauthorized, http.StatusUnauthorized},
		{NewForbidden("forbidden"), CodeForbidden, http.StatusForbidden},
		{NewConflict("conflict"), CodeConflict, http.StatusConflict},
		{NewInternal("internal"), CodeInternal, http.StatusInternalServerError},
		{NewUnavailable("down"), CodeUnavailable, http.StatusServiceUnavailable},
		{NewValidation("invalid"), CodeValidationFailed, http.StatusUnprocessableEntity},
	}

	for _, tt := range tests {
		if tt.err.Code != tt.wantCode {
			t.Errorf("got code %v, want %v", tt.err.Code, tt.wantCode)
		}
		if tt.err.HTTPStatus() != tt.wantStatus {
			t.Errorf("got status %d, want %d", tt.err.HTTPStatus(), tt.wantStatus)
		}
	}
}

func TestError_StringFormatting(t *testing.T) {
	err := New(CodeNotFound, "item not found")
	expected := fmt.Sprintf("[%s:%d] item not found", CodeNotFound.String(), CodeNotFound)
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}

	cause := errors.New("db record missing")
	errWithCause := Wrap(CodeNotFound, cause, "item not found")
	expectedWithCause := fmt.Sprintf("[%s:%d] item not found (cause: db record missing)", CodeNotFound.String(), CodeNotFound)
	if errWithCause.Error() != expectedWithCause {
		t.Errorf("expected %q, got %q", expectedWithCause, errWithCause.Error())
	}
}

func TestError_Interface(t *testing.T) {
	err := New(CodeBadRequest, "bad request")
	var iface ErrorInterface = err

	if iface.HTTPStatus() != 400 {
		t.Errorf("expected status 400, got %d", iface.HTTPStatus())
	}
	if iface.StatusCode() != CodeBadRequest {
		t.Errorf("expected StatusCode CodeBadRequest, got %v", iface.StatusCode())
	}
	if iface.ErrorMessage() != "bad request" {
		t.Errorf("expected ErrorMessage 'bad request', got %q", iface.ErrorMessage())
	}
}
