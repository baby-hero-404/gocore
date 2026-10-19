package httpserver

import (
	"errors"
	"net/http"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdlog"
)

// Start launches the HTTP server listening on the configured port.
func (s *serverImpl) Start() error {
	stdlog.Info("http server starting", "addr", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return stderr.Wrap(stderr.CodeInternal, err, "http server failed to start")
	}
	return nil
}
