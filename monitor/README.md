# Package `monitor`

`monitor` provides pre-configured observability instrumentation for Prometheus metrics, Sentry error tracking, and Graylog GELF log forwarding.

## Features

- **Standardized Prometheus Metrics**:
  - `gocore_database_connect_duration_seconds`
  - `gocore_database_query_errors_total`
  - `gocore_cache_op_errors_total`
  - `gocore_lock_acquire_failures_total`
  - `gocore_http_request_duration_seconds`
- **Gin Integrations**: `promgin.Handler()` for the `/metrics` endpoint and `promgin.Middleware()` for HTTP request latency tracking.
- **Sentry Integration**: Centralized error tracking with `sentry.CaptureError(ctx, err)` (`monitor/sentry`) propagating `request_id`.
- **Graylog Integration**: Fast UDP GELF writer (`monitor/graylog`) for log stream forwarding.

## Usage

```go
package main

import (
    "github.com/baby-hero-404/gocore/monitor/prom"
    "github.com/baby-hero-404/gocore/monitor/promgin"
    "github.com/gin-gonic/gin"
)

func main() {
    r := gin.New()

    // Create Prometheus recorder
    recorder := prom.NewRecorder(nil)

    // Attach metrics middleware
    r.Use(promgin.Middleware(recorder))

    // Expose /metrics
    r.GET("/metrics", promgin.Handler())
}
```
