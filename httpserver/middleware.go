package httpserver

import (
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdlog"
	"github.com/baby-hero-404/gocore/stdres/stdgin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HeaderRequestID is the HTTP header carrying the request ID.
const HeaderRequestID = "X-Request-ID"

// RequestIDKey is the typed context key for retrieving the request ID from context.Context.
// It is an alias for stdlog.RequestIDCtxKey to ensure stdlog.WithContext() reads it automatically.
var RequestIDKey = stdlog.RequestIDCtxKey

// maxRequestIDLen bounds a client-supplied request ID so it cannot bloat logs and headers.
const maxRequestIDLen = 128

// validRequestID reports whether a client-supplied request ID is safe to echo and log: non-empty,
// at most maxRequestIDLen characters, from a conservative character set (no spaces or control
// characters that could forge log lines).
func validRequestID(id string) bool {
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

// RequestIDMiddleware uses a valid incoming X-Request-ID or generates a UUID, attaching it to the
// request context and the response header. Invalid values (too long, unsafe characters) are replaced.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader(HeaderRequestID)
		if !validRequestID(reqID) {
			reqID = uuid.NewString()
		}

		c.Header(HeaderRequestID, reqID)

		// Attach to http.Request Context with typed key (avoids collision)
		c.Request = c.Request.WithContext(stdlog.ContextWithRequestID(c.Request.Context(), reqID))

		// Also set in Gin context for Gin-based consumers
		c.Set("request_id", reqID)

		c.Next()
	}
}

// RecoveryMiddleware recovers from panics, logs stack trace with stdlog, and returns a 500 error envelope.
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				internalErr := stderr.FromPanic(r)

				stdlog.WithContext(c.Request.Context()).Error("unhandled panic recovered",
					"error", internalErr.Err.Error(),
					"stack", stack,
				)

				stdgin.Error(c, internalErr)
			}
		}()
		c.Next()
	}
}

// RequestLoggerMiddleware logs request start, completion status, latency, and client IP.
func RequestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		if raw != "" {
			path = path + "?" + raw
		}

		logger := stdlog.WithContext(c.Request.Context()).With(
			"method", method,
			"path", path,
			"status", statusCode,
			"latency", latency.String(),
			"client_ip", clientIP,
		)

		if statusCode >= 500 {
			logger.Error("http request failed")
		} else if statusCode >= 400 {
			logger.Warn("http client error")
		} else {
			logger.Info("http request completed")
		}
	}
}

// CORSMiddleware configures Cross-Origin Resource Sharing based on Config.
//
// Only requests carrying an Origin header get CORS headers. A request whose origin is listed
// explicitly in AllowOrigins is answered with that origin (plus Vary: Origin, and
// Access-Control-Allow-Credentials when AllowCredentials is set). A "*" entry allows any other
// origin with the literal "*" and never with credentials, so a wildcard cannot be combined with
// credentialed cross-origin access. Origins that match nothing receive no CORS headers, which makes
// browsers block the call.
func CORSMiddleware(cfg Config) gin.HandlerFunc {
	listed := make(map[string]struct{}, len(cfg.AllowOrigins))
	allowAny := false
	for _, o := range cfg.AllowOrigins {
		if o == "*" {
			allowAny = true
			continue
		}
		listed[o] = struct{}{}
	}
	if allowAny && cfg.AllowCredentials {
		stdlog.Warn("cors: AllowCredentials is ignored for the wildcard origin; list the origins explicitly to allow credentials")
	}
	methods := strings.Join(cfg.AllowMethods, ", ")
	headers := strings.Join(cfg.AllowHeaders, ", ")

	return func(c *gin.Context) {
		if origin := c.GetHeader("Origin"); origin != "" {
			if len(listed) > 0 {
				c.Writer.Header().Add("Vary", "Origin")
			}
			_, isListed := listed[origin]
			if isListed || allowAny {
				if isListed {
					c.Header("Access-Control-Allow-Origin", origin)
					if cfg.AllowCredentials {
						c.Header("Access-Control-Allow-Credentials", "true")
					}
				} else {
					c.Header("Access-Control-Allow-Origin", "*")
				}
				c.Header("Access-Control-Allow-Methods", methods)
				c.Header("Access-Control-Allow-Headers", headers)
				c.Header("Access-Control-Expose-Headers", HeaderRequestID)
			}
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
