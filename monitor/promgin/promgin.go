// Package promgin adapts the prom package to Gin.
package promgin

import (
	"time"

	"github.com/baby-hero-404/gocore/monitor/prom"
	"github.com/gin-gonic/gin"
)

// Handler exposes the Prometheus /metrics endpoint for Gin.
func Handler() gin.HandlerFunc {
	h := prom.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// Middleware records HTTP latency and status for Prometheus.
// If recorder is nil, metrics are silently skipped.
func Middleware(recorder *prom.Recorder) gin.HandlerFunc {
	return func(c *gin.Context) {
		if recorder == nil {
			c.Next()
			return
		}

		start := time.Now()
		recorder.IncrementInFlight()
		defer recorder.DecrementInFlight()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		recorder.RecordDuration(c.Request.Method, route, c.Writer.Status(), time.Since(start))
	}
}
