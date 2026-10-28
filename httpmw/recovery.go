package httpmw

import (
	"errors"
	"net/http"
	"runtime/debug"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdlog"
	"github.com/baby-hero-404/gocore/stdres"
)

// Recovery turns a panic in a downstream handler into a 500 standard error envelope. The panic value
// and stack are logged once with the request ID and never sent to the client. If the handler had
// already started the response, only the log is written.
//
// A handler that panics with http.ErrAbortHandler (as httputil.ReverseProxy does when an upstream
// body breaks mid-copy) asks for the connection to be aborted. Library code may not re-panic, so
// Recovery closes the connection itself (HTTP/1.x), which makes the client see a truncated response
// as an error instead of a clean end. When the connection cannot be taken over (HTTP/2, test
// recorders) it logs a warning and answers 500 if nothing was written yet.
func Recovery() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			defer func() {
				if rec := recover(); rec != nil {
					respondToPanic(w, r, tw.wroteHeader, rec)
				}
			}()
			next.ServeHTTP(tw, r)
		})
	}
}

// respondToPanic logs the recovered panic and, if the response has not started, answers with a 500
// envelope that does not contain the panic value.
func respondToPanic(w http.ResponseWriter, r *http.Request, responseStarted bool, recovered any) {
	if err, ok := recovered.(error); ok && errors.Is(err, http.ErrAbortHandler) {
		abortConnection(w, r, responseStarted)
		return
	}
	internalErr := stderr.FromPanic(recovered)
	stdlog.WithContext(r.Context()).Error("unhandled panic recovered",
		"error", internalErr.Err.Error(),
		"stack", string(debug.Stack()),
	)
	if !responseStarted {
		stdres.WriteError(w, r, internalErr)
	}
}

// abortConnection honours http.ErrAbortHandler without panicking: it takes over the connection and
// closes it so the client cannot mistake a truncated body for a complete one.
func abortConnection(w http.ResponseWriter, r *http.Request, responseStarted bool) {
	conn, _, err := http.NewResponseController(w).Hijack()
	if err == nil {
		_ = conn.Close()
		stdlog.WithContext(r.Context()).Debug("request aborted by handler; connection closed", "path", r.URL.Path)
		return
	}
	stdlog.WithContext(r.Context()).Warn("request aborted by handler but the connection cannot be closed",
		"path", r.URL.Path, "error", err.Error())
	if !responseStarted {
		stdres.WriteError(w, r, stderr.NewInternal("request aborted"))
	}
}
