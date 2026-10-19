# Proposal: gocore Foundation (Backend Chassis Toolkit)

## Problem
Every new Go microservice (e.g. `kdc_bike_BE`) reimplements the same infrastructure boilerplate: error codes, API response shape, logging, DB/cache/queue clients with retry/pool config, and auth/authorization wiring. This duplication causes inconsistent error formats, ad-hoc logging, and repeated setup cost per service, and makes cross-service observability (Prometheus/Sentry/Graylog) inconsistent.

## Goal
Build `gocore`, a reusable Go module ("microservice chassis") that a service imports to get: standardized error codes/response/logging/monitoring, pre-wired infra clients (Postgres, Redis, Kafka, Gin, WebSocket) with built-in resiliency (pool, retry, distributed lock, timeout), and standardized auth (IAM/Casbin/OAuth/Firebase) — so a new service can be bootstrapped in minutes and `kdc_bike_BE` can be incrementally migrated onto it.

## Success
- A new Go service can import `gocore` and get working error handling, structured responses, logging, and at least one DB + one cache connection with sane defaults, without writing infra glue code.
- `kdc_bike_BE`'s `pkg/httperrors`, `pkg/response`, `pkg/logger` can be swapped for `gocore` equivalents with no behavior change visible to API consumers.
- Adding a new infra dependency (DB/cache/queue) to a service is a config entry + a few lines of wiring, not hand-rolled client setup.

## Assumptions
- Go 1.27+ toolchain is available in all target environments (exact floor set in design.md).
- Postgres is the primary relational DB target for v1 (MySQL wrapper deferred, see Non-goals).
- Gin remains the HTTP framework for v1 (matches `kdc_bike_BE`); ConnectRPC/HTTP3 support is architected for but not required to ship in v1.
- Package boundaries and APIs prioritize clean separation, standard Go idioms, and maintainability.
- Auth backends (Casbin, OAuth providers, Firebase) are optional, pluggable dependencies — a service that doesn't need IAM must not be forced to pull them in (module boundaries enforce this, see design.md).

## Decisions
- **Module path**: `github.com/baby-hero-404/gocore` (matches go.mod repository path; path is stable/importable).
- **Package layout**: flat top-level packages (`stderr/`, `stdres/`, `stdlog/`, `database/`, `cache/`, `queue/`, `httpserver/`, `iam/`, ...), rather than `kdc_bike_BE`'s `pkg/`-nested style. Flat layout keeps import paths short (`gocore/stderr` vs `gocore/pkg/httperrors`) and each package is independently versionable/testable.
- **Scope for v1**: full chassis — all 3 pillars (standardization, infra wrappers, plug-and-play bootstrap) — rather than a narrow slice, so the design settles module boundaries and shared conventions (error codes flowing into stdres, context propagation, config loading) once, instead of retrofitting them later. Implementation is still sequenced pillar-by-pillar in tasks.md so value ships incrementally.
- **Web framework**: Gin only for v1's HTTP wrapper. ConnectRPC/HTTP3 is a Non-goal for v1 since `kdc_bike_BE` only needs Gin today; the `httpserver` package is designed so a ConnectRPC adapter can be added later without breaking the Gin adapter's API.
- **Config**: use `spf13/viper` (already a `kdc_bike_BE` dependency) for config loading, to minimize new dependencies migrating services must learn.
- **Auth**: build `iam` as a thin package composing Casbin (authorization/RBAC) + JWT/OAuth2/Firebase (authentication), each behind small interfaces so a service can use only what it needs (e.g. JWT without Casbin).

## Trade-offs
- Flat top-level packages mean more top-level names in the module (vs. nesting under `pkg/`) — accepted because it keeps import paths ergonomic and dependencies decoupled.
- Building all 3 pillars up front is higher total design effort than shipping just `stderr`/`stdres`/`stdlog` first, but avoids a breaking redesign once `database`/`cache` need to consume `stderr` error codes and `stdlog` context propagation.
- Choosing Gin-only for v1 means the network wrapper is not framework-agnostic yet; a future ConnectRPC migration will require a second adapter, but that cost is deferred until actually needed.
- Depending on Casbin/Firebase/OAuth2 pulls in heavier third-party deps into `go.mod`; mitigated by keeping `iam` decomposed so importing `gocore/database` alone doesn't pull auth deps (enforced via separate packages, not build tags).

## Non-goals
- MySQL wrapper — Postgres only for v1; add MySQL later if/when a service needs it.
- ConnectRPC / HTTP3 / WebSocket server wrapper — architected for extensibility, not implemented in v1.
- Blockchain, payment, email, crypto, socket packages — out of scope; gocore focuses on the chassis, not domain-specific integrations.
- Automatic code generation/scaffolding CLI ("plug & play" service generator) — v1 delivers the library; a `gocore new service` CLI is a possible future task, not part of this proposal.

## Out of Scope
- Migrating `kdc_bike_BE` itself onto `gocore` — that is a follow-up task once `gocore` v1 is published and stable, tracked separately.
- CI/CD pipeline setup for `gocore` as a publishable module (versioning/release automation).
- Multi-tenant or multi-region specific infra concerns.

## Impact

### Components
New standalone Go module `gocore`, containing (v1): `stderr`, `stdres`, `stdlog`, `monitor` (Prometheus/Sentry/Graylog), `env`/`config`, `database` (Postgres/GORM), `cache` (Redis), `queue` (Kafka), `httpserver` (Gin), `lock` (distributed lock), `iam` (Casbin/OAuth/JWT/Firebase).

### Files
All new — this is a new repository/module at `/home/ubuntu/my_projects/gocore`. No existing service files are modified by this proposal (kdc_bike_BE migration is out of scope, see above).

### Public API
New module — this proposal defines the public API surface (see `specs.md` for expected behavior per package, `design.md` for exact interfaces).

### Migration
None for this task — `gocore` is a new, standalone artifact not yet consumed by anything.

### Backward Compatibility
N/A — greenfield module, no existing consumers.
