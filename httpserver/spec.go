package httpserver

import (
	"context"

	"github.com/gin-gonic/gin"
)

// Server defines the HTTP server lifecycle interface.
type Server interface {
	// Engine exposes the underlying Gin engine for registering routes.
	Engine() *gin.Engine

	// Start launches the HTTP server listening on the configured port.
	Start() error

	// Shutdown gracefully shuts down the server.
	Shutdown(ctx context.Context) error
}
