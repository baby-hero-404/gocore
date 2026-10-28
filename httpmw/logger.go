package httpmw

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/baby-hero-404/gocore/stdlog"
)

// LogRequest logs one completed request at a level derived from the status: ERROR for 5xx, WARN
// for 4xx, INFO otherwise. The Gin request logger uses it too, so both stacks log the same shape.
func LogRequest(ctx context.Context, method, path string, status int, latency time.Duration, clientIP string) {
	logger := stdlog.WithContext(ctx).With(
		"method", method,
		"path", path,
		"status", status,
		"latency", latency.String(),
		"client_ip", clientIP,
	)
	switch {
	case status >= 500:
		logger.Error("http request failed")
	case status >= 400:
		logger.Warn("http client error")
	default:
		logger.Info("http request completed")
	}
}

// statusWriter records the status code and bytes written. It keeps the optional interfaces that
// streaming and WebSocket handlers rely on: Flush, Hijack, and Unwrap (for http.ResponseController).
type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.status = code
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.status = http.StatusOK
		w.wroteHeader = true
	}
	return w.ResponseWriter.Write(b)
}

// Flush forwards to the underlying writer so SSE and chunked responses keep streaming.
func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		if !w.wroteHeader {
			w.status = http.StatusOK
			w.wroteHeader = true
		}
		f.Flush()
	}
}

// Hijack forwards to the underlying writer so WebSocket upgrades work through the logger.
func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("httpmw: underlying ResponseWriter does not support hijacking")
	}
	conn, rw, err := h.Hijack()
	if err == nil && !w.wroteHeader {
		w.status = http.StatusSwitchingProtocols
		w.wroteHeader = true
	}
	return conn, rw, err
}

// Unwrap exposes the underlying writer to http.ResponseController.
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// RequestLogger logs method, path (with query), status, latency and client IP once per request.
// The client IP is the connection's remote address; put a RealIP-style middleware (Chi's RealIP, or
// a reverse proxy rewriting RemoteAddr) before it when running behind a proxy.
func RequestLogger() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(sw, r)

			path := r.URL.Path
			if r.URL.RawQuery != "" {
				path += "?" + r.URL.RawQuery
			}
			LogRequest(r.Context(), r.Method, path, sw.status, time.Since(start), clientIP(r))
		})
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
