package httpmw

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/baby-hero-404/gocore/stdlog"
	"github.com/baby-hero-404/gocore/stdres"
)

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := stdlog.Get()
	stdlog.SetDefault(stdlog.New(stdlog.WithOutput(&buf), stdlog.WithLevel(slog.LevelDebug)))
	t.Cleanup(func() { stdlog.SetDefault(prev) })
	return &buf
}

func serve(h http.Handler, method, target string, hdr map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestValidRequestID(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"3f2b8c1e-7d44-4a51-9a0e-1c2d3e4f5a6b", true}, {"svc:abc_123.def/45+x=@y", true}, {strings.Repeat("a", 128), true},
		{strings.Repeat("a", 129), false}, {"", false}, {"has space", false}, {"line\nbreak", false}, {"tên", false},
	}
	for _, tt := range tests {
		if got := ValidRequestID(tt.id); got != tt.want {
			t.Errorf("ValidRequestID(%q) = %v, want %v", tt.id, got, tt.want)
		}
	}
	if id := NewRequestID(); !ValidRequestID(id) {
		t.Errorf("generated id %q must itself be valid", id)
	}
}

func TestRequestID(t *testing.T) {
	h := RequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(stdlog.RequestIDFromContext(r.Context())))
	}))
	tests := []struct {
		name     string
		incoming string
		keep     bool
	}{
		{"valid id is kept", "abc-123", true},
		{"missing id is generated", "", false},
		{"unsafe id is replaced", "bad id\r\nX-Evil: 1", false},
		{"too long id is replaced", strings.Repeat("z", 200), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hdr := map[string]string{}
			if tt.incoming != "" {
				hdr[HeaderRequestID] = tt.incoming
			}
			w := serve(h, http.MethodGet, "/", hdr)
			got := w.Header().Get(HeaderRequestID)
			if (got == tt.incoming) != tt.keep || got == "" || w.Body.String() != got {
				t.Errorf("response id %q, context id %q, kept=%v want %v", got, w.Body.String(), got == tt.incoming, tt.keep)
			}
		})
	}
}

func TestRecovery(t *testing.T) {
	logs := captureLogs(t)
	panicking := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic("db password=hunter2 leaked") })
	h := RequestID()(Recovery()(panicking))

	w := serve(h, http.MethodGet, "/boom", nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	var env stdres.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil || env.Message != "internal server error" || env.RequestID == "" {
		t.Fatalf("envelope = %+v (%v)", env, err)
	}
	if strings.Contains(w.Body.String(), "hunter2") {
		t.Error("panic text must not reach the client")
	}
	out := logs.String()
	if !strings.Contains(out, "hunter2") || !strings.Contains(out, "stack") || !strings.Contains(out, env.RequestID) {
		t.Errorf("panic, stack and request id must be logged: %s", out)
	}
	if strings.Count(out, "unhandled panic recovered") != 1 {
		t.Errorf("panic must be logged exactly once: %s", out)
	}
}

func TestRecovery_AfterPartialResponseDoesNotWriteASecondBody(t *testing.T) {
	captureLogs(t)
	h := Recovery()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("partial"))
		panic("late")
	}))
	w := serve(h, http.MethodGet, "/", nil)
	if w.Code != http.StatusAccepted || w.Body.String() != "partial" {
		t.Errorf("got %d %q, want the partial response untouched", w.Code, w.Body.String())
	}
}

func TestRecovery_SwallowsAbortHandler(t *testing.T) {
	captureLogs(t)
	h := Recovery()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic(http.ErrAbortHandler) }))
	if w := serve(h, http.MethodGet, "/", nil); w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

func TestRequestLogger_LevelByStatus(t *testing.T) {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		target       string
		wantMsg      string
		wantLvl      string
		wantInOutput string
	}{
		{"implicit 200", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) }, "/a", "http request completed", "INFO", `"status":200`},
		{"client error", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) }, "/b", "http client error", "WARN", `"status":404`},
		{"server error with query", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }, "/c?id=7", "http request failed", "ERROR", "/c?id=7"},
		{"no write at all counts as 200", func(w http.ResponseWriter, r *http.Request) {}, "/d", "http request completed", "INFO", `"status":200`},
		{"first WriteHeader wins", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			w.WriteHeader(http.StatusInternalServerError)
		}, "/e", "http request completed", "INFO", `"status":201`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := captureLogs(t)
			serve(RequestLogger()(tt.handler), http.MethodGet, tt.target, nil)
			out := logs.String()
			for _, want := range []string{tt.wantMsg, tt.wantLvl, tt.wantInOutput, "client_ip", "latency"} {
				if !strings.Contains(out, want) {
					t.Errorf("log %q does not contain %q", out, want)
				}
			}
		})
	}
}

func TestClientIP(t *testing.T) {
	logs := captureLogs(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.9:51234"
	RequestLogger()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(httptest.NewRecorder(), req)
	if !strings.Contains(logs.String(), `"client_ip":"203.0.113.9"`) {
		t.Errorf("client ip must be the host without port: %s", logs.String())
	}
	req.RemoteAddr = "weird"
	if got := clientIP(req); got != "weird" {
		t.Errorf("unparsable remote addr is returned as is, got %q", got)
	}
}

func TestStatusWriter_KeepsStreamingFeatures(t *testing.T) {
	t.Run("flush is forwarded and counts as a 200", func(t *testing.T) {
		logs := captureLogs(t)
		rec := httptest.NewRecorder()
		h := RequestLogger()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			f, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("wrapped writer must implement http.Flusher")
			}
			f.Flush()
		}))
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/stream", nil))
		if !rec.Flushed || !strings.Contains(logs.String(), `"status":200`) {
			t.Errorf("flushed=%v logs=%s", rec.Flushed, logs.String())
		}
	})

	t.Run("http.ResponseController reaches the real writer through Unwrap", func(t *testing.T) {
		captureLogs(t)
		rec := httptest.NewRecorder()
		h := RequestLogger()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := http.NewResponseController(w).Flush(); err != nil {
				t.Errorf("ResponseController.Flush through the wrapper: %v", err)
			}
		}))
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if !rec.Flushed {
			t.Error("flush did not reach the underlying writer")
		}
	})

	t.Run("hijack works end to end (websocket upgrade path)", func(t *testing.T) {
		captureLogs(t)
		srv := httptest.NewServer(RequestLogger()(Recovery()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, rw, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Errorf("hijack through logger and recovery: %v", err)
				return
			}
			defer func() { _ = conn.Close() }()
			_, _ = rw.WriteString("HTTP/1.1 101 Switching Protocols\r\n\r\n")
			_ = rw.Flush()
		}))))
		defer srv.Close()

		conn, err := net.Dial("tcp", strings.TrimPrefix(srv.URL, "http://"))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = conn.Close() }()
		_, _ = conn.Write([]byte("GET / HTTP/1.1\r\nHost: x\r\n\r\n"))
		buf := make([]byte, 64)
		n, _ := conn.Read(buf)
		if !strings.HasPrefix(string(buf[:n]), "HTTP/1.1 101") {
			t.Errorf("upgrade response = %q", buf[:n])
		}
	})

	t.Run("hijack on a writer that cannot hijack returns an error", func(t *testing.T) {
		captureLogs(t)
		var hijackErr error
		h := RequestLogger()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _, hijackErr = w.(http.Hijacker).Hijack()
		}))
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
		if hijackErr == nil {
			t.Error("expected an error from a recorder that does not support hijacking")
		}
	})
}

func TestCORS(t *testing.T) {
	tests := []struct {
		name       string
		cfg        CORSConfig
		method     string
		origin     string
		wantOrigin string
		wantCreds  string
		wantVary   bool
		wantStatus int
	}{
		{"wildcard answers *", CORSConfig{AllowOrigins: []string{"*"}}, http.MethodGet, "https://a.example", "*", "", false, http.StatusOK},
		{"wildcard never carries credentials", CORSConfig{AllowOrigins: []string{"*"}, AllowCredentials: true}, http.MethodGet, "https://a.example", "*", "", false, http.StatusOK},
		{"listed origin echoed", CORSConfig{AllowOrigins: []string{"https://a.example", "https://b.example"}}, http.MethodGet, "https://b.example", "https://b.example", "", true, http.StatusOK},
		{"listed origin with credentials", CORSConfig{AllowOrigins: []string{"https://a.example"}, AllowCredentials: true}, http.MethodGet, "https://a.example", "https://a.example", "true", true, http.StatusOK},
		{"listed beats wildcard", CORSConfig{AllowOrigins: []string{"*", "https://a.example"}, AllowCredentials: true}, http.MethodGet, "https://a.example", "https://a.example", "true", true, http.StatusOK},
		{"unlisted origin gets nothing", CORSConfig{AllowOrigins: []string{"https://a.example"}}, http.MethodGet, "https://evil.example", "", "", true, http.StatusOK},
		{"no Origin header gets nothing", CORSConfig{AllowOrigins: []string{"https://a.example"}}, http.MethodGet, "", "", "", false, http.StatusOK},
		{"preflight listed", CORSConfig{AllowOrigins: []string{"https://a.example"}}, http.MethodOptions, "https://a.example", "https://a.example", "", true, http.StatusNoContent},
		{"preflight unlisted", CORSConfig{AllowOrigins: []string{"https://a.example"}}, http.MethodOptions, "https://evil.example", "", "", true, http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			captureLogs(t)
			tt.cfg.AllowMethods = []string{"GET"}
			tt.cfg.AllowHeaders = []string{"X-Test"}
			called := false
			h := CORS(tt.cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; _, _ = w.Write([]byte("ok")) }))

			hdr := map[string]string{}
			if tt.origin != "" {
				hdr["Origin"] = tt.origin
			}
			w := serve(h, tt.method, "/", hdr)

			got := w.Header()
			if got.Get("Access-Control-Allow-Origin") != tt.wantOrigin || got.Get("Access-Control-Allow-Credentials") != tt.wantCreds || (got.Get("Vary") == "Origin") != tt.wantVary {
				t.Errorf("headers = %v", got)
			}
			static := got.Get("Access-Control-Allow-Methods") == "GET" && got.Get("Access-Control-Allow-Headers") == "X-Test" && got.Get("Access-Control-Expose-Headers") == HeaderRequestID
			if static != (tt.wantOrigin != "") {
				t.Errorf("static CORS headers present = %v, want %v", static, tt.wantOrigin != "")
			}
			if w.Code != tt.wantStatus || called == (tt.method == http.MethodOptions) {
				t.Errorf("status = %d, handler called = %v", w.Code, called)
			}
		})
	}
}

func TestCORS_WildcardWithCredentialsWarns(t *testing.T) {
	logs := captureLogs(t)
	NewCORSPolicy(CORSConfig{AllowOrigins: []string{"*"}, AllowCredentials: true})
	if !strings.Contains(logs.String(), "AllowCredentials is ignored") {
		t.Errorf("expected a warning, got %q", logs.String())
	}
	logs.Reset()
	NewCORSPolicy(CORSConfig{AllowOrigins: []string{"https://a.example"}, AllowCredentials: true})
	if logs.Len() != 0 {
		t.Errorf("no warning expected for explicit origins, got %q", logs.String())
	}
}

func TestMiddlewareChainWithServeMux(t *testing.T) {
	captureLogs(t)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items/{id}", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("item " + r.PathValue("id")))
	})
	var h http.Handler = mux
	for _, m := range []Middleware{CORS(CORSConfig{AllowOrigins: []string{"*"}}), RequestLogger(), Recovery(), RequestID()} {
		h = m(h)
	}
	w := serve(h, http.MethodGet, "/items/42", map[string]string{"Origin": "https://x.example"})
	if w.Code != http.StatusOK || w.Body.String() != "item 42" || w.Header().Get(HeaderRequestID) == "" || w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("got %d %q %v", w.Code, w.Body.String(), w.Header())
	}
}
