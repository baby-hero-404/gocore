# Project Profile

## Identity
- **Project Name**: gocore (`github.com/baby-hero-404/gocore`)
- **Type**: Go Foundation Toolkit / Microservice Chassis (single module, flat packages)
- **Primary Language**: Go 1.27.3

## Architecture
- **Core Pillars**:
  1. *Standardization*: Unified typed error handling (`stderr`), API response envelope (`stdres`), context-aware structured logging (`stdlog`), and IAM/Casbin authorization.
  2. *Infrastructure Wrappers*: Pre-wired connectors for Postgres, Redis, Kafka, Distributed Lock, and Gin HTTP server with built-in connection pooling, retries, timeouts, and graceful shutdown.
  3. *Plug & Play*: Flat top-level layout, zero heavyweight DI container, and independent per-package configurations.
- **Key Components**:
  - `stderr`: Single source of truth for typed error codes & error wrapping. Zero internal or external dependencies.
  - `stdlog`: Context-aware structured logger built on standard library `log/slog`.
  - `stdres`: Standard API response envelope (JSON success/error mapper) with validation error unmarshaling.
  - `env`: Viper-based configuration loader with environment variable unmarshaling.
  - `database`: PostgreSQL/MySQL wrapper via GORM with connection pooling and health checks.
  - `cache`: Redis wrapper (standalone and cluster) via `go-redis/v9`.
  - `lock`: Distributed lock wrapper backed by Redis via `redsync/v4`.
  - `queue`: Message queue wrapper backed by Kafka via `segmentio/kafka-go`.
  - `httpserver`: Gin HTTP server wrapper, core middlewares (recovery, tracing, CORS), and graceful shutdown.
  - `iam`: Auth toolkit (JWT, OAuth2, Firebase, Casbin), strictly isolated from direct infra package imports.
  - `monitor`: Observability connectors (Prometheus, Sentry, Graylog).
  - `database/pagination`: Cursor (HMAC-signed, multi-column keyset) and offset pagination for GORM.
  - `internal/archtest`: Architectural tests verifying package ranks, vendor isolation, doc comments, and no `init()`.

## Tech Stack
- **Framework & Libraries**: Gin (`github.com/gin-gonic/gin`), GORM (`gorm.io/gorm`), go-redis (`github.com/redis/go-redis/v9`), kafka-go (`github.com/segmentio/kafka-go`), redsync (`github.com/go-redsync/redsync/v4`), casbin (`github.com/casbin/casbin/v3`), jwt (`github.com/golang-jwt/jwt/v5`), validator (`github.com/go-playground/validator/v10`), viper (`github.com/spf13/viper`).
- **Testing**: Go standard `testing`, `testify` (strictly restricted to `*mock` sub-packages).
- **Tooling**: `mockery` v3 (pinned as a `go tool` in `go.mod`) generates mocks; `golangci-lint`, `gitleaks` and `govulncheck` run via `go run <module>@<pinned version>` (Makefile / CI); `internal/tools/covercheck` enforces the coverage ratchet.

## Repository Map
- Flat top-level packages: `stderr/`, `stdlog/`, `stdres/`, `env/`, `database/`, `cache/`, `lock/`, `queue/`, `httpserver/`, `iam/`, `monitor/`
- Sub-packages: `database/pagination`, driver packages `database/{postgres,mysql,sqlite}` and `database/anydriver`
- Sub-package adapters: `stdres/stdgin`, `iam/iamgin`, `iam/firebase`, `monitor/prom`, `monitor/promgin`, `monitor/sentry`, `monitor/graylog`
- Mock packages: `<pkg>/<pkg>mock` (e.g. `cache/cachemock`): generated `mock_<Interface>.go` (never hand-edit) plus a hand-written `doc.go` holding compile-time `var _ Iface = (*Mock)(nil)` assertions
- `docs/openspecs/gocore-foundation/`: Active architecture OpenSpec (proposal, design, specs, tasks)
- `docs/ai/`: Persistent AI memory (`PROFILE.md`, `DOMAIN.md`, `manifest.json`, `adr/`, `archive/`)
- `.agents/rules/`: Foundation rules (`coding-standards.md`, `architecture.md`, `dependency-management.md`, `testing.md`, `git-workflow.md`, `extension.md`, `agent-workflow.md`)
- `.mockery.yml`: mock generation config (explicit interface list); `.gitleaks.toml`: secret-scan config; `.golangci.yml`: lint config; `coverage-baseline.txt`: per-package coverage floor; `.github/workflows/verify.yml`: CI
- `docs/openspecs/golib-adoption/`: OpenSpec for adopting go-library ideas (pagination, quality gates, gRPC-ready layout); see `tasks.md` for progress
- `Makefile`: Targets `verify` (`lint` + `mocks-check` + `secrets` + `test` + `cover-check`), `test`, `lint`, `tidy`, `mod`, `mocks`, `mocks-check`, `secrets`, `cover-check`, `cover-baseline`
