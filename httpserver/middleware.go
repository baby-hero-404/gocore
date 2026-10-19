package httpserver

import (
	"context"
	"fmt"
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

const HeaderRequestID = "X-Request-ID"

// RequestIDKey is the typed context key for retrieving the request ID from context.Context.
// It is an alias for stdlog.RequestIDCtxKey to ensure stdlog.WithContext() reads it automatically.
var RequestIDKey = stdlog.RequestIDCtxKey

// RequestIDMiddleware extracts X-Request-ID or generates a UUID, attaching it to context and response.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader(HeaderRequestID)
		if strings.TrimSpace(reqID) == "" {
			reqID = uuid.NewString()
		}

		c.Header(HeaderRequestID, reqID)

		// Attach to http.Request Context with typed key (avoids collision)
		ctx := context.WithValue(c.Request.Context(), RequestIDKey, reqID)
		c.Request = c.Request.WithContext(ctx)

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
				err := fmt.Errorf("%v", r)

				stdlog.WithContext(c.Request.Context()).Error("unhandled panic recovered",
					"error", err.Error(),
					"stack", stack,
				)

				internalErr := stderr.NewInternal("internal server error", err)
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
func CORSMiddleware(cfg Config) gin.HandlerFunc {
	origins := strings.Join(cfg.AllowOrigins, ", ")
	methods := strings.Join(cfg.AllowMethods, ", ")
	headers := strings.Join(cfg.AllowHeaders, ", ")

	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", origins)
		c.Header("Access-Control-Allow-Methods", methods)
		c.Header("Access-Control-Allow-Headers", headers)
		c.Header("Access-Control-Expose-Headers", HeaderRequestID)

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
