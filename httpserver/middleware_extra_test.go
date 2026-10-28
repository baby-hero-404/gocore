package httpserver

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/baby-hero-404/gocore/stdlog"
	"github.com/gin-gonic/gin"
)

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := stdlog.Get()
	stdlog.SetDefault(stdlog.New(stdlog.WithOutput(&buf), stdlog.WithLevel(slog.LevelDebug)))
	t.Cleanup(func() { stdlog.SetDefault(prev) })
	return &buf
}

func TestRequestLoggerMiddleware_LevelByStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name    string
		status  int
		path    string
		wantMsg string
		wantLvl string
	}{
		{"success", http.StatusOK, "/r", "http request completed", "INFO"},
		{"client error", http.StatusBadRequest, "/r", "http client error", "WARN"},
		{"server error keeps the query string", http.StatusInternalServerError, "/r?id=7", "http request failed", "ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := captureLogs(t)
			engine := gin.New()
			engine.Use(RequestLoggerMiddleware())
			engine.GET("/r", func(c *gin.Context) { c.Status(tt.status) })

			engine.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, tt.path, nil))

			out := buf.String()
			for _, want := range []string{tt.wantMsg, tt.wantLvl, tt.path} {
				if !strings.Contains(out, want) {
					t.Errorf("log output %q does not contain %q", out, want)
				}
			}
		})
	}
}

func TestCORSMiddleware_OriginHandling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		cfg        Config
		method     string
		origin     string
		wantOrigin string // "" means the header must be absent
		wantCreds  string
		wantVary   bool
		wantStatus int
	}{
		{"wildcard answers with *", Config{AllowOrigins: []string{"*"}}, http.MethodGet, "https://a.example", "*", "", false, http.StatusOK},
		{"wildcard never carries credentials", Config{AllowOrigins: []string{"*"}, AllowCredentials: true}, http.MethodGet, "https://a.example", "*", "", false, http.StatusOK},
		{"listed origin is echoed", Config{AllowOrigins: []string{"https://a.example", "https://b.example"}}, http.MethodGet, "https://b.example", "https://b.example", "", true, http.StatusOK},
		{"listed origin gets credentials when enabled", Config{AllowOrigins: []string{"https://a.example"}, AllowCredentials: true}, http.MethodGet, "https://a.example", "https://a.example", "true", true, http.StatusOK},
		{"listed origin beats the wildcard for credentials", Config{AllowOrigins: []string{"*", "https://a.example"}, AllowCredentials: true}, http.MethodGet, "https://a.example", "https://a.example", "true", true, http.StatusOK},
		{"unlisted origin gets no CORS headers", Config{AllowOrigins: []string{"https://a.example"}}, http.MethodGet, "https://evil.example", "", "", true, http.StatusOK},
		{"request without Origin gets no CORS headers", Config{AllowOrigins: []string{"https://a.example"}}, http.MethodGet, "", "", "", false, http.StatusOK},
		{"preflight from a listed origin", Config{AllowOrigins: []string{"https://a.example"}}, http.MethodOptions, "https://a.example", "https://a.example", "", true, http.StatusNoContent},
		{"preflight from an unlisted origin is answered without CORS headers", Config{AllowOrigins: []string{"https://a.example"}}, http.MethodOptions, "https://evil.example", "", "", true, http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cfg.AllowMethods = []string{"GET"}
			tt.cfg.AllowHeaders = []string{"X-Test"}
			engine := gin.New()
			engine.Use(CORSMiddleware(tt.cfg))
			engine.GET("/c", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

			req := httptest.NewRequest(tt.method, "/c", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			h := w.Header()
			if got := h.Get("Access-Control-Allow-Origin"); got != tt.wantOrigin {
				t.Errorf("Allow-Origin = %q, want %q", got, tt.wantOrigin)
			}
			if got := h.Get("Access-Control-Allow-Credentials"); got != tt.wantCreds {
				t.Errorf("Allow-Credentials = %q, want %q", got, tt.wantCreds)
			}
			if got := h.Get("Vary") == "Origin"; got != tt.wantVary {
				t.Errorf("Vary: Origin present = %v, want %v", got, tt.wantVary)
			}
			hasStatic := h.Get("Access-Control-Allow-Methods") == "GET" && h.Get("Access-Control-Allow-Headers") == "X-Test" && h.Get("Access-Control-Expose-Headers") == HeaderRequestID
			if hasStatic != (tt.wantOrigin != "") {
				t.Errorf("static CORS headers present = %v, want %v (%v)", hasStatic, tt.wantOrigin != "", h)
			}
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestRequestIDMiddleware_ValidatesClientValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name     string
		incoming string
		keep     bool
	}{
		{"uuid is kept", "3f2b8c1e-7d44-4a51-9a0e-1c2d3e4f5a6b", true},
		{"trace style id is kept", "svc:abc_123.def/45+x=@y", true},
		{"exactly max length is kept", strings.Repeat("a", maxRequestIDLen), true},
		{"too long is replaced", strings.Repeat("a", maxRequestIDLen+1), false},
		{"spaces are replaced", "has space", false},
		{"newline is replaced", "line\nbreak", false},
		{"non ascii is replaced", "tên-yêu-cầu", false},
		{"blank is replaced", "   ", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := gin.New()
			engine.Use(RequestIDMiddleware())
			engine.GET("/", func(c *gin.Context) { c.String(http.StatusOK, stdlog.RequestIDFromContext(c.Request.Context())) })

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(HeaderRequestID, tt.incoming)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			got := w.Header().Get(HeaderRequestID)
			if (got == tt.incoming) != tt.keep {
				t.Errorf("response id %q, kept=%v want kept=%v", got, got == tt.incoming, tt.keep)
			}
			if got == "" || w.Body.String() != got {
				t.Errorf("context id %q must equal response id %q", w.Body.String(), got)
			}
		})
	}
}
