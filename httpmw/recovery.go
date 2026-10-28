package httpmw

import (
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
// http.ErrAbortHandler is swallowed like any other panic; library code may not re-panic, so the
// connection is not aborted.
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
	internalErr := stderr.FromPanic(recovered)
	stdlog.WithContext(r.Context()).Error("unhandled panic recovered",
		"error", internalErr.Err.Error(),
		"stack", string(debug.Stack()),
	)
	if !responseStarted {
		stdres.WriteError(w, r, internalErr)
	}
}
