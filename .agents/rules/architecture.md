# Architecture & Package Guidelines

This document specifies the architectural design and module boundaries for `gocore`.

---

## 1. Layout Structure: Flat Top-Level Packages

Unlike typical end-user applications using nested `internal/` or `pkg/` directories, `gocore` is an open foundation toolkit (chassis library) designed with a flat top-level layout:

```text
gocore/
├── stderr/       # Single source of typed error codes & error wrapping
├── stdlog/       # Context-aware structured logger (slog/zap)
├── stdres/       # API response envelope (JSON success/error mapper)
├── env/          # Viper-based configuration loader
├── database/     # Postgres/MySQL wrapper with connection pool & metrics
├── cache/        # Redis wrapper (standalone & cluster support)
├── lock/         # Distributed lock (Redis-backed via redsync)
├── queue/        # Kafka message queue wrapper
├── httpserver/   # Gin wrapper, base middlewares (tracing, recovery, cors)
├── iam/          # Auth toolkit (JWT, OAuth2, Firebase, Casbin)
└── monitor/      # Prometheus, Sentry, Graylog integrations
```

**Key Advantages**:
* Short, readable import paths: `import "github.com/baby-hero-404/gocore/database"`.
* Granular adoption: Consuming services only import what they need without pulling in unrelated transitive dependencies (e.g. a service using only `database` never pulls in `casbin` or `kafka`).

---

## 2. Dependency Hierarchy

The dependency tree is strictly **Bottom-Up** and **Acyclic** (zero circular dependencies permitted):

```text
       [ stderr ]  (Zero internal or third-party dependencies)
        ▲      ▲
        │      └─────────────────┐
     [ stdlog ]              [ stdres ]
        ▲                        ▲
        │                        │
  [ Infra Packages ]       [ httpserver ]
(database, cache, queue)         ▲
                                 │
                              [ iam ]
```

### Invariants:
1. **`stderr` is at the absolute bottom**: Does not import any other package in `gocore`.
2. **`iam` isolation**: No infra package imports `iam`. `iam` does not directly import `database` or `cache`; it depends on abstract consumer interfaces (e.g., `TokenStore`, `UserFinder`).
3. **Independent infra packages**: `database`, `cache`, and `queue` do not import one another. `lock` may depend on `cache`.

---

## 3. Configuration Separation

* **No monolithic shared config struct**: Do not maintain a single massive struct holding configuration for every package.
* **Per-package Config Ownership**: Each infrastructure package owns its own dedicated `Config` struct:
  * `database.Config`
  * `cache.Config`
  * `queue.Config`
* The `env` package provides helpers to unmarshal configurations from YAML/ENV files via Viper into their respective structs.

---

## 4. Built-in Infrastructure Resiliency

Every infrastructure connector wrapper must provide:
1. **Connection Pooling**: Sensible defaults for `MaxOpenConns`, `MaxIdleConns`, `ConnMaxLifetime`.
2. **Health Checks**: Standard `Ping(ctx)` or `Health(ctx) error` method.
3. **Graceful Shutdown**: Safe cleanup via `Close() error` when the application shuts down.

---

## 5. Package Layout Rules (enforced by `internal/archtest`)

1. **Core packages never import a web framework or heavy vendor SDK.** Adapters live in a sub-package named after what they adapt (`stdres/stdgin`, `iam/iamgin`, `iam/firebase`, `monitor/prom`, `monitor/promgin`, `monitor/sentry`, `monitor/graylog`). New vendor imports must be registered in `vendorOwners` in `internal/archtest/archtest_test.go`.
2. **Mocks live in `<pkg>/<pkg>mock`** (e.g. `cache/cachemock`). `testify` is only importable from `*mock` packages.
3. **Dependencies point down** by rank: `stderr` < `stdlog` < `stdres`/`env`/`monitor` < `database`/`cache`/`queue` < `lock`/`httpserver` < `iam`. A new top-level package must be given a rank.
4. **No `init()` and no process-global mutation** (e.g. `gin.SetMode`) in library code.
5. **Every package has a package doc comment** (`doc.go`).
6. **Constructors:** `New(cfg Config, opts ...Option)` for a package's main type (add `ctx` only if construction does I/O); `NewX` when a package has several types.
