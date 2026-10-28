package httpserver

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/baby-hero-404/gocore/httpmw"
	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdlog"
	"github.com/baby-hero-404/gocore/stdres/stdgin"
	"github.com/gin-gonic/gin"
)

// HeaderRequestID is the HTTP header carrying the request ID.
const HeaderRequestID = httpmw.HeaderRequestID

// RequestIDKey is the typed context key for retrieving the request ID from context.Context.
// It is an alias for stdlog.RequestIDCtxKey to ensure stdlog.WithContext() reads it automatically.
var RequestIDKey = stdlog.RequestIDCtxKey

// RequestIDMiddleware uses a valid incoming X-Request-ID or generates a UUID, attaching it to the
// request context and the response header. Invalid values (too long, unsafe characters) are replaced.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader(HeaderRequestID)
		if !httpmw.ValidRequestID(reqID) {
			reqID = httpmw.NewRequestID()
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

// RequestLoggerMiddleware logs request completion status, latency, and client IP (shared log shape
// with httpmw.RequestLogger).
func RequestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			path += "?" + raw
		}

		c.Next()

		httpmw.LogRequest(c.Request.Context(), c.Request.Method, path, c.Writer.Status(), time.Since(start), c.ClientIP())
	}
}

// CORSMiddleware configures Cross-Origin Resource Sharing based on Config. The decision rules live
// in httpmw.CORSPolicy (see there): listed origins are echoed (with credentials if enabled), "*"
// answers "*" and never carries credentials, anything else gets no CORS headers.
func CORSMiddleware(cfg Config) gin.HandlerFunc {
	policy := httpmw.NewCORSPolicy(httpmw.CORSConfig{
		AllowOrigins:     cfg.AllowOrigins,
		AllowMethods:     cfg.AllowMethods,
		AllowHeaders:     cfg.AllowHeaders,
		AllowCredentials: cfg.AllowCredentials,
	})

	return func(c *gin.Context) {
		policy.Apply(c.Writer.Header(), c.GetHeader("Origin"))

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
