# Package `httpserver`

`httpserver` wraps Gin with production-ready base middlewares: panic recovery (mapping into `stdres.Envelope`), request tracking (`X-Request-ID`), structured latency logging with `stdlog`, CORS handling, and graceful shutdown.

## Features

- **Interface-Driven & Function-File Layout**: Exposes `httpserver.Server` interface with dedicated function files (`spec.go`, `new.go`, `option.go`, `start.go`, `shutdown.go`, `config.go`, `middleware.go`, `mock_server.go`).
- **Standardized Panic Recovery**: Intercepts panics, logs stack trace securely via `stdlog`, and outputs standard HTTP 500 error envelope (`stdres.Envelope`) without terminating the process.
- **Trace Propagation**: Attaches unique `X-Request-ID` to request context and headers.
- **Structured Latency Logging**: Logs HTTP method, path, status, latency, and client IP on every request.
- **Graceful Shutdown**: Wraps `http.Server.Shutdown(ctx)` to drain in-flight connections cleanly.
- **Downstream Unit Testing**: `httpservermock.Server` powered by `testify/mock`.

## Usage

```go
package main

import (
    "context"
    "log"
    "net/http"
    "time"

    "github.com/baby-hero-404/gocore/httpserver"
    "github.com/baby-hero-404/gocore/stdres"
    "github.com/gin-gonic/gin"
)

func main() {
    srv := httpserver.New(httpserver.Config{
        Port: 8080,
    })

    // Register routes
    srv.Engine().GET("/health", func(c *gin.Context) {
        stdgin.OK(c, map[string]string{"status": "ok"})
    })

    // Start server in goroutine
    go func() {
        if err := srv.Start(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server error: %v", err)
        }
    }()

    // Graceful shutdown on termination signal
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    _ = srv.Shutdown(ctx)
}
```

> Gin's run mode is process-wide, so `httpserver` does not set it. Set `GIN_MODE=release` or call `gin.SetMode(gin.ReleaseMode)` in `main`.
