# Go Coding Standards & Conventions

Go programming standards for the `gocore` repository. All AI agents and developers contributing code **must** strictly adhere to these conventions.

---

## 1. Core Philosophy

1. **Foundation Library (Microservice Chassis)**: `gocore` is a shared infrastructural toolkit across microservices. Code must be stable, simple, high-performance, and feature clean APIs.
2. **No Global State**: Never use package-level global variables (`var GlobalDB *gorm.DB`). Do not rely on `init()` functions for implicit resource registration. Always initialize explicitly via constructors: `New*(cfg Config) (*Client, error)`.
3. **Accept Interfaces, Return Structs**: Accept narrow interfaces in function signatures (when decoupling is required) and return concrete struct pointers for predictable caller usage.
4. **No Panic / No Exit (CS-3, CS-6 `[test]`)**: Never call `panic()`, `log.Fatal*` or `os.Exit` in library code. All operational errors must be returned as `error`; only the consuming service's `main` decides to terminate. Enforced by `archtest.TestNoPanicOrExit`.
5. **No Secrets (SEC-1 `[lint]`)**: Never commit secrets, tokens or credentials (compose files, config, tests). Test fixtures that look like secrets carry `// gitleaks:allow`. Enforced by `make secrets` (gitleaks) locally and in CI.

---

## 2. Error Handling (with `stderr`)

* **Single Source of Truth**: All packages in `gocore` returning errors must return or wrap through the `stderr` package (`github.com/baby-hero-404/gocore/stderr`).
* **No Raw Driver Errors**: Do not leak raw database/driver errors (e.g. `pgconn.PgError`, `redis.Nil`, `kafka.Error`) out to consuming services. Wrap them with context: `stderr.Wrap(code, err)`.
* **Error Code Ranges**:
  * `1xxx`: Client Validation / Bad Request (e.g., missing parameter, invalid payload).
  * `4xxx`: Client-side failures (Not Found, Unauthorized, Forbidden, Conflict).
  * `5xxx`: Internal / Infrastructure failures (DB unreachable, connection timeout, network partition).
* **Standard Go Compatibility**: Support standard Go error unwrapping (`errors.Is`, `errors.As`).

---

## 3. Context & Concurrency

* **Always Propagate `context.Context`**: The first parameter of every I/O function (Database, Redis, Kafka, HTTP) must be `ctx context.Context`.
* **Exemptions (CS-5)**: `Close() error` follows `io.Closer`; constructors and helpers that only do local, non-blocking work (config parsing, `env.Load` reading local files, in-memory policy checks, signing) take no `ctx`. Anything that talks to the network, a database or blocks on I/O takes `ctx` first.
* **Honor Cancellation & Timeouts**: Always check `ctx.Done()` in loops or background workers.
* **Goroutine Lifecycle Management**: Never spawn unmanaged goroutines. Ensure explicit termination mechanisms via `sync.WaitGroup`, `errgroup.Group`, or context cancellation to prevent goroutine leaks.

---

## 4. Naming Conventions

* **Package Names**: Short, lower-case, single-word, no underscores or camelCase (e.g., `stderr`, `stdlog`, `database`, `cache`).
* **Avoid Repetitive Identifiers**:
  * Good: `database.New()`, `cache.Client`, `stderr.New()`
  * Bad: `database.NewDatabase()`, `cache.CacheClient`, `stderr.NewError()`
* **Interface Names**: Suffix single-method interfaces with `-er` (e.g., `Reader`, `Closer`). Use descriptive domain roles for multi-method interfaces (e.g., `Locker`, `Publisher`).

---

## 5. Structs & Configuration

* **Independent Config Ownership**: Each infrastructure package owns its own dedicated `Config` struct.
  * Tag fields with `mapstructure` (for Viper) and `json`.
  * Provide sensible defaults via `DefaultConfig()` or validate during `New(cfg)`.
* **Functional Options Pattern**: Use functional options when constructors need clean optional configuration:
  ```go
  type Option func(*Options)
  func WithTimeout(d time.Duration) Option { ... }
  ```

---

## 6. Formatting & Static Analysis

Before committing, code must cleanly pass:
* `gofmt -l -w .`
* `go vet ./...`
* `golangci-lint run`
* `go test ./... -race` (zero race conditions permitted)
