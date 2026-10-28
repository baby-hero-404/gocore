package httpserver

import (
	"context"
	"errors"
	"net"
	"net/http"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdlog"
)

// Start serves HTTP until the server fails or ctx is cancelled, then shuts down gracefully.
func (s *serverImpl) Start(ctx context.Context) error {
	// Requests inherit ctx's values (request IDs, loggers) but not its cancellation, so
	// cancelling ctx starts a graceful shutdown instead of aborting in-flight requests.
	s.httpServer.BaseContext = func(net.Listener) context.Context { return context.WithoutCancel(ctx) }

	stdlog.WithContext(ctx).Info("http server starting", "addr", s.httpServer.Addr)

	// The goroutine is owned by this function: every return path below waits for it.
	serveErr := make(chan error, 1)
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		serveErr <- s.httpServer.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return stderr.Wrap(stderr.CodeInternal, err, "http server failed to start")
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.cfg.ShutdownTimeout)
		defer cancel()
		err := s.Shutdown(shutdownCtx)
		<-stopped // ListenAndServe has returned (after Shutdown it is always http.ErrServerClosed)
		return err
	}
}
