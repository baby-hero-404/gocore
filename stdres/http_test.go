package stdres

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdlog"
	"github.com/go-playground/validator/v10"
)

func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) Envelope {
	t.Helper()
	var env Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	return env
}

func TestWriteHelpers(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	tests := []struct {
		name       string
		write      func(w http.ResponseWriter)
		wantStatus int
		wantMsg    string
	}{
		{"ok", func(w http.ResponseWriter) { WriteOK(w, r, map[string]string{"id": "1"}) }, http.StatusOK, "success"},
		{"created", func(w http.ResponseWriter) { WriteCreated(w, r, "x") }, http.StatusCreated, "created"},
		{"no content", func(w http.ResponseWriter) { WriteNoContent(w) }, http.StatusNoContent, ""},
		{"error", func(w http.ResponseWriter) { WriteError(w, r, stderr.NewForbidden("nope")) }, http.StatusForbidden, "nope"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tt.write(w)
			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantMsg != "" && decodeEnvelope(t, w).Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", decodeEnvelope(t, w).Message, tt.wantMsg)
			}
		})
	}
}

func TestWriteError_NilAndRequestID(t *testing.T) {
	w := httptest.NewRecorder()
	WriteError(w, httptest.NewRequest(http.MethodGet, "/", nil), nil)
	if w.Body.Len() != 0 {
		t.Errorf("nil error must write nothing, got %q", w.Body.String())
	}

	ctx := context.WithValue(context.Background(), stdlog.RequestIDCtxKey, "rid-1")
	r := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	w = httptest.NewRecorder()
	WriteError(w, r, errors.New("boom"))
	env := decodeEnvelope(t, w)
	if w.Code != http.StatusInternalServerError || env.RequestID != "rid-1" {
		t.Errorf("got status=%d request_id=%q", w.Code, env.RequestID)
	}
}

type form struct {
	Email string `json:"user_email" validate:"required,email"`
}

func TestValidationError(t *testing.T) {
	valErr := validator.New().Struct(form{Email: "bad"})
	typeErr := &json.UnmarshalTypeError{Field: "age", Value: "string", Type: reflect.TypeOf(0)}

	tests := []struct {
		name       string
		err        error
		wantCode   stderr.Code
		wantKey    string
		wantField  string
		wantStatus int
	}{
		{"validator", valErr, stderr.CodeValidationFailed, "invalid_request_body", "user_email", http.StatusUnprocessableEntity},
		{"type mismatch", typeErr, stderr.CodeBadRequest, "invalid_request_format", "age", http.StatusBadRequest},
		{"generic", errors.New("unexpected EOF"), stderr.CodeBadRequest, "invalid_request_format", "body", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ValidationError(form{}, tt.err)
			if e.Code != tt.wantCode || e.Key != tt.wantKey || e.HTTPStatus() != tt.wantStatus {
				t.Fatalf("got code=%v key=%q status=%d", e.Code, e.Key, e.HTTPStatus())
			}
			if len(e.Details) != 1 || e.Details[0].Field != tt.wantField {
				t.Errorf("details = %+v, want field %q", e.Details, tt.wantField)
			}
		})
	}
}

func TestWriteValidationError_And_JSONFieldName(t *testing.T) {
	w := httptest.NewRecorder()
	WriteValidationError(w, httptest.NewRequest(http.MethodPost, "/", nil), form{}, errors.New("bad json"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}

	tests := []struct {
		name  string
		obj   any
		field string
		want  string
	}{
		{"nil target", nil, "Email", "Email"},
		{"non struct", 5, "Email", "Email"},
		{"pointer struct", &form{}, "Email", "user_email"},
		{"unknown field", form{}, "Nope", "Nope"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := jsonFieldName(tt.obj, tt.field); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
