package httpmw

import (
	"net/http"
	"strings"

	"github.com/baby-hero-404/gocore/stdlog"
	"github.com/google/uuid"
)

// Middleware wraps a handler. It is an alias, so values are assignable to chi.Router.Use and to any
// router expecting func(http.Handler) http.Handler.
type Middleware = func(http.Handler) http.Handler

// HeaderRequestID is the HTTP header carrying the request ID.
const HeaderRequestID = "X-Request-ID"

// maxRequestIDLen bounds a client-supplied request ID so it cannot bloat logs and headers.
const maxRequestIDLen = 128

// ValidRequestID reports whether a client-supplied request ID is safe to echo and log: non-empty,
// at most 128 characters, from a conservative character set (no spaces or control characters that
// could forge log lines).
func ValidRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLen {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case strings.ContainsRune("-_.:/+=@", r):
		default:
			return false
		}
	}
	return true
}

// NewRequestID returns a fresh request ID.
func NewRequestID() string { return uuid.NewString() }

// RequestID reuses a valid incoming X-Request-ID or generates a UUID, and sets it on the response
// header and on the request context (read it with stdlog.RequestIDFromContext; stdlog.WithContext
// attaches it to log lines).
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(HeaderRequestID)
			if !ValidRequestID(id) {
				id = NewRequestID()
			}
			w.Header().Set(HeaderRequestID, id)
			next.ServeHTTP(w, r.WithContext(stdlog.ContextWithRequestID(r.Context(), id)))
		})
	}
}
