package stdres

import (
	"encoding/json"
	"net/http"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdlog"
)

// WriteJSON writes v as JSON with the given HTTP status.
func WriteJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		stdlog.WithContext(r.Context()).Error("write json response", "error", err.Error())
	}
}

// WriteOK responds with HTTP 200 and the standard success envelope.
func WriteOK(w http.ResponseWriter, r *http.Request, data any) {
	WriteJSON(w, r, http.StatusOK, SuccessCtx(r.Context(), data))
}

// WriteCreated responds with HTTP 201 and the created envelope.
func WriteCreated(w http.ResponseWriter, r *http.Request, data any) {
	WriteJSON(w, r, http.StatusCreated, Envelope{
		Code:      int(stderr.CodeOK),
		Message:   "created",
		Data:      data,
		RequestID: requestIDFromContext(r.Context()),
	})
}

// WriteNoContent responds with HTTP 204.
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// WriteError responds with the standard error envelope for err.
// A translator set on the request context via WithTranslator is applied.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}
	LogUnclassified(r.Context(), err)
	env, status := FromError(r.Context(), err)
	WriteJSON(w, r, status, env)
}

// WriteValidationError responds with the envelope for a request binding/validation error.
func WriteValidationError(w http.ResponseWriter, r *http.Request, target any, err error) {
	WriteError(w, r, ValidationError(target, err))
}
