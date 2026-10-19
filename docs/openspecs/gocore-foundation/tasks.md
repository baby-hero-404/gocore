# Implementation Map: gocore Foundation

**Goal:** Build the `gocore` Go module (chassis) covering standardization, infra wrappers, and IAM, per proposal.md/design.md.
**Tech Stack:** Go 1.27+, Gin, GORM (Postgres), go-redis, Kafka client (segmentio/kafka-go or confluent), spf13/viper, Casbin, golang-jwt, golang.org/x/oauth2, Firebase Admin SDK, Prometheus client.

---

## Phase 0: Module scaffolding

### Initialize Go module and repo layout

**Why:**
Nothing else can be built without a module definition and baseline tooling (lint, test, Makefile) in place.

**Files:**
- `go.mod`, `go.sum`
- `.gitignore`
- `Makefile`
- `README.md`

**Changes:**
- [x] `go mod init github.com/baby-hero-404/gocore`
- [x] Add `.gitignore` (vendor/, .env, bin/, coverage files)
- [x] Add `Makefile` with `test`, `lint`, `tidy` targets
- [x] Add root `README.md` with module purpose, install instructions, package index (filled in as packages land)

**Verify:**
- [x] `go build ./...` succeeds on an empty module
- [x] `make lint` runs (even with nothing to lint yet)

---

## Phase 1: Standardization core (`stderr`, `stdlog`, `stdres`, `env`)

### `stderr` — typed error codes & validation details

**Why:**
Foundation for every other package; must exist before `stdlog`/`stdres`/infra wrappers can depend on it. Supports classified error codes, field-level validation causes, and i18n keys for seamless migration of `kdc_bike_BE`.

**Depends on:**
- Phase 0

**Files:**
- `stderr/code.go`
- `stderr/error.go`
- `stderr/error_test.go`

**Changes:**
- [x] Define `Code` type with classed ranges (Validation, NotFound, Conflict, Unauthorized, Forbidden, Internal, Unavailable, Timeout)
- [x] Implement `FieldError` struct (`Field`, `Message`, `Tag`)
- [x] Implement `Error` struct with `Code`, `Message`, `Key`, `Args`, `Details []FieldError`, `Err`
- [x] Implement `New`, `Wrap`, `WithDetails`, `WithKey`, `WithCause`, `HTTPStatus()`, `Error()`, `Unwrap()`
- [x] Implement `errors.Is` / `errors.As` compatibility

**Verify:**
- [x] Unit tests: each `Code` class maps to the correct `HTTPStatus()`
- [x] Unit tests: `Wrap` and `Unwrap` preserve the original error for `errors.Is`/`errors.As`
- [x] Unit tests: `WithDetails` attaches field validation errors without losing code or message

### `env` — config loading

**Why:**
Every infra wrapper needs a consistent way to decode its `Config` struct from environment/files; built early so Phase 2 wrappers can use it immediately.

**Depends on:**
- Phase 0

**Files:**
- `env/loader.go`
- `env/loader_test.go`

**Changes:**
- [x] Viper-backed config loader: `Load(prefix string, target any, opts ...Option) error`
- [x] Support env vars, automatic uppercase env mapping, optional `.env` or YAML file loading
- [x] Support fail-fast validation for required struct tags or custom validators

**Verify:**
- [x] Unit test: struct with required fields fails to load with a clear `stderr` error when a field is missing
- [x] Unit test: env var overrides file value
- [x] Unit test: prefixing and automatic nested env mapping works properly

### `stdlog` — context-aware structured logging (`log/slog`)

**Why:**
Needed by `stdres` (to log mapped errors) and every infra wrapper (retry/failure logging), per design.md dependency order. Built on Go standard library `log/slog`.

**Depends on:**
- `stderr`

**Files:**
- `stdlog/logger.go`
- `stdlog/level.go`
- `stdlog/redact.go`
- `stdlog/logger_test.go`

**Changes:**
- [x] Structured logger wrapping `log/slog.Logger` with `WithContext(ctx)` extracting trace/request ID
- [x] Level mapping helper: `LevelFor(err error) slog.Level` using `stderr.Code` class
- [x] Redaction guard: automatic masking/redaction of sensitive fields (password, token, secret, dsn)
- [x] Support custom handlers and output destinations (JSONHandler, TextHandler)

**Verify:**
- [x] Unit test: `LevelFor` returns error/warn/info appropriately per `stderr.Code` class
- [x] Unit test: logging fields with sensitive keys (password, token) redacts the values
- [x] Unit test: context with request/trace ID automatically attaches to log records

### `stdres` — response envelope & helpers

**Why:**
Converts `stderr` errors and success data into a standard consistent API shape, compatible with `kdc_bike_BE`'s clients.

**Depends on:**
- `stderr`, `stdlog`

**Files:**
- `stdres/response.go`
- `stdres/translator.go`
- `stdres/gin.go`
- `stdres/response_test.go`

**Changes:**
- [x] Standard response envelope: `Envelope{Code: int, Message: string, Data: any, Details: any}`
- [x] `Translator` interface for optional i18n translation of error messages and success keys
- [x] `FromError(ctx context.Context, err error, translator Translator) (Envelope, int)` unwrapping `stderr.Error`
- [x] Gin response helpers: `OK`, `Created`, `NoContent`, `JSON`, `Error`, `HandleValidationErrors`

**Verify:**
- [x] Unit test: `FromError` on a `stderr.Error` returns matching code, HTTP status, and field details
- [x] Unit test: `FromError` on unknown `errors.New(...)` defaults to internal 500 without panicking
- [x] Unit test: Gin helpers produce valid JSON envelopes and correct HTTP status codes
- [x] Unit test: `HandleValidationErrors` translates validator errors into `FieldError` details

---

## Phase 2: Infra wrappers (`database`, `cache`, `queue`, `lock`)

### `database` — Postgres/GORM wrapper

**Why:**
First infra wrapper; validates the `stderr`+`env`+retry pattern that `cache`/`queue` will replicate.

**Depends on:**
- `stderr`, `stdlog`, `env`

**Files:**
- `database/postgres.go`
- `database/config.go`
- `database/postgres_test.go`

**Changes:**
- [x] `Config` struct (DSN, pool limits, retry attempts/backoff)
- [x] `NewPostgres(cfg Config) (*gorm.DB, error)` with bounded retry on connect
- [x] Wrap GORM errors into `stderr` at the boundary (e.g. `ErrRecordNotFound` -> `stderr` not-found code)

**Verify:**
- [x] Unit test: invalid config (empty DSN) fails fast with a `stderr` validation error, no retry attempted
- [x] Unit test: simulated unreachable DB exhausts retries and returns typed `stderr` error, not a panic
- [x] Ping and Close helpers

### `cache` — Redis wrapper

**Why:**
Second infra wrapper; also the dependency `lock` builds on.

**Depends on:**
- `stderr`, `stdlog`, `env`

**Files:**
- `cache/redis.go`
- `cache/config.go`
- `cache/redis_test.go`

**Changes:**
- [x] `Config` struct (addr, password, db index, timeouts)
- [x] `NewRedis(cfg Config) (*redis.Client, error)`
- [x] Wrap client errors into `stderr` at operation boundaries used by gocore itself (e.g. `lock`)

**Verify:**
- [x] Unit test: invalid config fails fast
- [x] Unit test: Ping and Close helpers

### `lock` — distributed lock (Redis-backed)

**Why:**
Directly needed per specs.md Scenario "Service acquires a distributed lock"; depends on `cache`.

**Depends on:**
- `cache`, `stderr`

**Files:**
- `lock/lock.go`
- `lock/lock_test.go`

**Changes:**
- [x] `Acquire(ctx, client, key, ttl) (*Lock, error)` using `SET NX PX` / Redlock
- [x] `Release(ctx) error` and `Extend(ctx) error`
- [x] Typed `stderr` "lock held" error on contention

**Verify:**
- [x] Unit test: validation on key/ttl/client
- [x] Unit test: nil safety and options

### `queue` — Kafka wrapper

**Why:**
Completes the infra-wrapper trio named in the proposal; same pattern as `database`/`cache`.

**Depends on:**
- `stderr`, `stdlog`, `env`

**Files:**
- `queue/kafka.go`
- `queue/config.go`
- `queue/kafka_test.go`

**Changes:**
- [x] `Config` struct (brokers, topic, consumer group, timeouts)
- [x] `NewProducer(cfg Config) (*Producer, error)`, `NewConsumer(cfg Config) (*Consumer, error)`
- [x] Wrap client errors into `stderr`

**Verify:**
- [x] Unit test: invalid broker config fails fast
- [x] Unit test: Producer and Consumer nil safety and defaults

---

## Phase 3: `httpserver` (Gin wrapper)

### Gin bootstrap + standard middleware

**Why:**
Ties `stdres`/`stdlog` into an actual HTTP server so a service gets working request/response/error handling out of the box (Scenario: new service bootstraps with minimal boilerplate).

**Depends on:**
- `stdres`, `stdlog`, `stderr`

**Files:**
- `httpserver/server.go`
- `httpserver/middleware.go`
- `httpserver/middleware_test.go`

**Changes:**
- [x] `New(cfg Config) *Server` with recovery + request logging + error-to-`stdres` middleware wired by default
- [x] Panic recovery middleware converts panics into a `stderr` internal error response instead of crashing the process
- [x] Request logging middleware logs method/path/status/duration via `stdlog`

**Verify:**
- [x] Unit test: handler returning a `stderr.Error` produces the expected `stdres` JSON body + status code
- [x] Unit test: handler panic is recovered and returns a 500 `stdres` envelope, server stays up

---

## Phase 4: `iam` (auth/authz)

### JWT authentication middleware

**Why:**
Minimum viable auth path (Scenario: authenticate via IAM); Casbin/OAuth/Firebase build on this same middleware shape.

**Depends on:**
- `httpserver`, `stdres`, `stderr`, `stdlog`

**Files:**
- `iam/jwt.go`
- `iam/identity.go`
- `iam/jwt_test.go`

**Changes:**
- [x] `TokenValidator` interface + `JWTValidator` implementation (golang-jwt)
- [x] `JWTMiddleware(v TokenValidator) gin.HandlerFunc` populating identity in context on success
- [x] Standardized 401 `stderr`/`stdres` response on invalid/expired/malformed token

**Verify:**
- [x] Unit test: valid token populates identity in context, request proceeds
- [x] Unit test: expired token rejected with 401, no identity set
- [x] Unit test: malformed token rejected with 401, no panic

### Casbin authorization middleware

**Why:**
Adds policy-based authorization on top of authentication, per specs.md Scenario.

**Depends on:**
- JWT authentication middleware (identity must exist in context first)

**Files:**
- `iam/casbin.go`
- `iam/casbin_test.go`

**Changes:**
- [x] `CasbinMiddleware(enforcer *casbin.Enforcer) gin.HandlerFunc` reading identity from context, enforcing policy
- [x] Standardized 403 `stderr`/`stdres` response on policy denial

**Verify:**
- [x] Unit test: identity with permitted policy passes through to handler
- [x] Unit test: identity without permission gets 403, handler never invoked

### OAuth2 / Firebase token validators

**Why:**
Rounds out `TokenValidator` implementations named in the proposal, reusing the same middleware from the JWT task.

**Depends on:**
- JWT authentication middleware (same `TokenValidator` interface)

**Files:**
- `iam/oauth2.go`
- `iam/firebase.go`
- `iam/oauth2_test.go`
- `iam/firebase_test.go`

**Changes:**
- [x] `OAuth2Validator` implementing `TokenValidator` (validates against configured provider)
- [x] `FirebaseValidator` implementing `TokenValidator` (Firebase Admin SDK token verification)

**Verify:**
- [x] Unit test (mocked provider/SDK): valid token -> identity populated
- [x] Unit test: invalid/expired token -> `stderr` unauthorized error, no panic

---

## Phase 5: Monitoring (`monitor`) and docs

### Prometheus/Sentry/Graylog integration

**Why:**
Proposal names monitoring as part of standardization; wires the metrics named in design.md's Observability section.

**Depends on:**
- `stdlog`, `httpserver` (for request-duration metric hook)

**Files:**
- `monitor/prometheus.go`
- `monitor/sentry.go`
- `monitor/graylog.go`

**Changes:**
- [x] Prometheus registry + the metrics listed in design.md (`gocore.database.*`, `gocore.cache.*`, `gocore.lock.*`, `gocore.http.*`)
- [x] Sentry error-reporting hook triggered from `stdlog` on ERROR-level logs
- [x] Graylog (GELF) log sink option for `stdlog`

**Verify:**
- [x] Unit test: a simulated ERROR log triggers the Sentry hook (mocked transport)
- [x] Manual/integration check: `/metrics` endpoint exposes the named counters/histograms after exercising `database`/`httpserver`

### Root README + package-level docs

**Why:**
Doc Sync — every package added above needs a discoverable entry point per design/proposal decisions.

**Files:**
- `README.md`
- `stderr/README.md`, `database/README.md`, `iam/README.md` (one per package, short)

**Changes:**
- [x] Root README: module purpose, package index table, quickstart (`stdlog`+`stdres`+`database` bootstrap example)
- [x] Per-package README: purpose, `Config` fields, one usage example

**Verify:**
- [x] `grep -L "README" */` returns no package directories missing a README
- [x] Quickstart example in root README compiles as a runnable snippet (or is covered by an example test)

---

## Rollback Plan

Not applicable — `gocore` is a new, unpublished module with no existing consumers or deployed infrastructure (per proposal.md Backward Compatibility: N/A). No feature flags, DB migrations, or production traffic are involved in this task.
