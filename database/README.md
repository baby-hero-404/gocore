# Package `database`

`database` wraps GORM and PostgreSQL with robust connection pooling, exponential backoff retries, health checking, and automatic error translation into `stderr` codes.

## Features

- **Interface-Driven & Function-File Layout**: Exposes `database.Database` interface with dedicated function files (`spec.go`, `new.go`, `option.go`, `ping.go`, `close.go`, `mock_database.go`).
- **Built-in Resiliency**: Bounded exponential backoff retries when connecting, avoiding crashloops on temporary database blips.
- **Connection Pooling**: Conservative, production-ready connection pool defaults (`MaxOpenConns: 25`, `MaxIdleConns: 10`, `ConnMaxLifetime: 5m`).
- **Standardized Error Mapping**: `database.WrapError(err)` maps `gorm.ErrRecordNotFound` to `stderr.CodeNotFound` (404), `gorm.ErrDuplicatedKey` to `stderr.CodeConflict` (409), etc.
- **Downstream Unit Testing**: Provides `databasemock.Database` powered by `testify/mock`.

## Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/baby-hero-404/gocore/database"
)

func main() {
    cfg := database.Config{
        DSN: "host=localhost user=postgres password=secret dbname=mydb sslmode=disable",
        MaxOpenConns: 30,
        RetryAttempts: 5,
        RetryBackoff: 500 * time.Millisecond,
    }

    db, err := database.New(context.Background(), cfg)
    if err != nil {
        log.Fatalf("failed to connect to db: %v", err)
    }
    defer db.Close()

    // Health checking
    if err := db.Ping(context.Background()); err != nil {
        log.Printf("db unhealthy: %v", err)
    }

    // Access underlying *gorm.DB or scoped session
    session := db.NewDb(context.Background())
    _ = session
}
```

## Unit Testing in Downstream Services

```go
func TestService(t *testing.T) {
    mockDB := databasemock.NewDatabase(t)
    mockDB.On("Ping", mock.Anything).Return(nil)
}
```
