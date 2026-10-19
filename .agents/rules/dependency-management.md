# Dependency Management Rules & Best Practices

This document defines dependency management rules for the `gocore` repository. All AI agents and developers adding or updating dependencies **must** comply with these principles.

---

## 1. Core Principles

### 1.1. Always Use Latest Stable Versions
* **Mandatory**: When adding any new dependency to the project, always install the latest stable version:
  ```bash
  go get <package>@latest
  ```
* **No Deprecated / Outdated Packages**: Never copy-paste outdated versions from older repositories or rely on unmaintained libraries.
* **Routine Verification**: Periodically check for available dependency updates:
  ```bash
  go list -u -m all
  ```

### 1.2. Standard Library First
* Modern Go provides a powerful, high-performance standard library:
  * **Structured Logging**: Prefer `log/slog` over third-party loggers where possible.
  * **Error Handling**: Use `errors` (`errors.Is`, `errors.As`, `errors.Join`, `fmt.Errorf("%w")`).
  * **Concurrency**: Utilize `context.Context`, `sync`, and `golang.org/x/sync/errgroup`.
* Only introduce third-party libraries when specialized infrastructure connectivity is required (GORM/pgx, go-redis, Kafka, Gin, Viper, Casbin).

### 1.3. Zero Dependency Bloat & Acyclic Boundaries
* **Zero Bloat**: Never pull in a massive multi-module dependency just for a trivial helper function (write lightweight in-tree utilities instead).
* **Module Boundary Isolation**: Independent packages must not leak or drag each other's dependencies:
  * `database` and `cache` must not pull in `iam` or `casbin`.
  * `stderr` must remain completely dependency-free (zero external dependencies).

---

## 2. Dependency Addition Workflow

When adding a dependency to a package (e.g., GORM for `database`):

1. **Fetch Latest Version**:
   ```bash
   go get gorm.io/gorm@latest
   go get gorm.io/driver/postgres@latest
   ```
2. **Tidy and Prune Modules**:
   ```bash
   make tidy   # equivalent to: go mod tidy
   ```
3. **Verify Linting & Tests**:
   ```bash
   make lint
   make test
   ```
4. **Inspect `go.mod` and `go.sum`**: Ensure no extraneous packages or version conflicts were introduced.

---

## 3. Approved Baseline Libraries

| Purpose | Standardized Library | Rationale |
|---|---|---|
| HTTP Framework | `github.com/gin-gonic/gin@latest` | Fast, battle-tested, compatible with downstream services |
| Validation | `github.com/go-playground/validator/v10@latest` | De-facto Go standard, built-in Gin integration |
| Configuration | `github.com/spf13/viper@latest` | Multi-source configuration (file, env, mapstructure) |
| Structured Logging | `log/slog` (Standard Library) | Zero external dependencies, context-aware, high throughput |
| Postgres ORM | `gorm.io/gorm@latest`, `gorm.io/driver/postgres@latest` | Common standard ORM, robust connection pool |
| Redis Cache | `github.com/redis/go-redis/v9@latest` | Full standalone/cluster support, context-aware |
| Distributed Lock | `github.com/go-redsync/redsync/v4@latest` | Standard Redlock implementation over Redis |
| Kafka Queue | `github.com/segmentio/kafka-go@latest` | Pure Go (no CGO required), idiomatic streaming API |
| Authorization | `github.com/casbin/casbin/v3@latest` | Flexible RBAC/ABAC engine |
| JWT Auth | `github.com/golang-jwt/jwt/v5@latest` | Latest secure version, fixes vulnerabilities in v4 |
