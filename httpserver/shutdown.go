package httpserver

import (
	"context"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdlog"
)

// Shutdown gracefully shuts down the server.
func (s *serverImpl) Shutdown(ctx context.Context) error {
	stdlog.WithContext(ctx).Info("http server shutting down")
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return stderr.Wrap(stderr.CodeInternal, err, "http server graceful shutdown failed")
	}
	return nil
}
