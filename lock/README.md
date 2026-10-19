# Package `lock`

`lock` implements distributed locking built on Redis using the Redlock algorithm via `go-redsync/redsync/v4`.

## Features

- **Interface-Driven & Function-File Layout**: Exposes `lock.Locker` and `lock.Lock` interfaces with dedicated function files (`spec.go`, `new.go`, `option.go`, `acquire.go`, `release.go`, `extend.go`, `mock_lock.go`).
- **Standard Redlock Algorithm**: Prevents split-brain and concurrent execution across microservice replicas.
- **Contention Handling**: When a lock cannot be acquired, returns `stderr.CodeLockHeld` (HTTP 409) for deterministic client responses.
- **Auto-Expiration**: Always requires a positive TTL so crashes never cause permanent deadlocks.
- **Extension Support**: `l.Extend(ctx)` allows long-running jobs to refresh their lock safely.
- **Downstream Unit Testing**: `lockmock.Locker` and `lockmock.Lock` powered by `testify/mock`.

## Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/baby-hero-404/gocore/cache"
    "github.com/baby-hero-404/gocore/lock"
)

func main() {
    ctx := context.Background()
    rdb, _ := cache.NewRedis(ctx, cache.Config{Addr: "localhost:6379"})

    locker := lock.New(rdb)

    // Acquire lock for 10 seconds
    l, err := locker.Acquire(ctx, "lock:order:12345", 10*time.Second)
    if err != nil {
        log.Fatalf("could not acquire lock: %v", err)
    }
    defer l.Release(ctx)

    // Critical section
    log.Println("Lock acquired! Performing atomic task...")
}
```

## Unit Testing in Downstream Services

```go
func TestOrderProcessing(t *testing.T) {
    mockLocker := lockmock.NewLocker(t)
    mockLock := lockmock.NewLock(t)
    mockLocker.On("Acquire", mock.Anything, "order:1", mock.Anything).Return(mockLock, nil)
}
```
