# Package `cache`

`cache` wraps Redis client (`go-redis/v9`) with an interface-driven contract, connection pool defaults, timeout configs, connectivity checks, and automatic error translation into `stderr` error codes.

## Features

- **Interface-Driven & Function-File Layout**: Exposes `cache.Cache` interface with dedicated function files (`spec.go`, `new.go`, `option.go`, `get.go`, `set.go`, `del.go`, `exists.go`, `expire.go`, `ping.go`, `close.go`, `mock_cache.go`).
- **Standardized Error Translation**: Maps `redis.Nil` to `stderr.CodeNotFound` (404), `redis.TxFailedErr` to `stderr.CodeConflict` (409), timeouts to `stderr.CodeTimeout` (504).
- **Resilient Defaults**: Sensible timeouts and pool sizing (`PoolSize: 20`, `DialTimeout: 5s`).
- **Fail-Fast Initialization**: Checks Redis connectivity at startup (`Ping`).
- **Downstream Unit Testing**: `cachemock.Cache` powered by `testify/mock`.

## Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/baby-hero-404/gocore/cache"
)

func main() {
    ctx := context.Background()
    cfg := cache.Config{
        Addr: "localhost:6379",
        DB:   0,
    }

    c, err := cache.New(ctx, cfg)
    if err != nil {
        log.Fatalf("failed to connect to redis: %v", err)
    }
    defer c.Close()

    // Get / Set / Expire / Del
    _ = c.Set(ctx, "session:123", "user_data", 15*time.Minute)
    val, _ := c.Get(ctx, "session:123")
    _ = val
}
```

## Unit Testing in Downstream Services

```go
func TestService(t *testing.T) {
    mockCache := cachemock.NewCache(t)
    mockCache.On("Get", mock.Anything, "key").Return("value", nil)
}
```
