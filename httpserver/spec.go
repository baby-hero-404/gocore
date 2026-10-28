package httpserver

import (
	"context"

	"github.com/gin-gonic/gin"
)

// Server defines the HTTP server lifecycle interface.
type Server interface {
	// Engine exposes the underlying Gin engine for registering routes.
	Engine() *gin.Engine

	// Start serves HTTP on the configured port and blocks until the server fails or ctx is
	// cancelled. On cancellation it shuts down gracefully, giving in-flight requests up to
	// Config.ShutdownTimeout, and returns nil. Request contexts inherit ctx's values (not its
	// cancellation), so a graceful shutdown does not abort requests already running.
	Start(ctx context.Context) error

	// Shutdown gracefully shuts down the server.
	Shutdown(ctx context.Context) error
}
