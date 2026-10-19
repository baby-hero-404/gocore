# Expected Behavior: gocore Foundation

## Scenario: Service returns a standardized error
**When:**
- A handler/service returns a `stderr.Error` (created via `stderr.New(code, message)` or wrapping an underlying error)

**Then:**
- `stdres` middleware/helper converts it into a consistent JSON error response `{ "code": ..., "message": ..., "details": ... }`
- The error is logged via `stdlog` with severity mapped from the error's class (e.g. `Internal` -> error level, `NotFound` -> warn/info level)
- The HTTP status code is derived deterministically from the error code's class (e.g. 4xx for client errors, 5xx for internal errors)

## Scenario: Service returns a successful result
**When:**
- A handler returns data with no error

**Then:**
- `stdres` wraps it in the standard success envelope `{ "code": 0, "message": "success", "data": ... }` (exact shape finalized in design.md)

## Scenario: Service bootstraps a Postgres connection
**When:**
- A service calls `database.NewPostgres(cfg)` with connection config (host, credentials, pool settings)

**Then:**
- Returns a ready `*gorm.DB` (or equivalent handle) with connection pool limits applied from config
- Retries transient connection failures with backoff up to a configured max attempts before returning an error
- Connection/query errors surface as `stderr` typed errors, not raw driver errors

## Scenario: Service bootstraps a Redis cache client
**When:**
- A service calls `cache.NewRedis(cfg)`

**Then:**
- Returns a ready client with pool settings and timeout applied from config
- Cache operation failures surface as `stderr` typed errors

## Scenario: Service acquires a distributed lock
**When:**
- A service calls `lock.Acquire(ctx, key, ttl)` against the configured backend (Redis-based)

**Then:**
- Returns a lock handle on success, or a typed `stderr` "lock held" error if already locked
- Lock auto-expires after `ttl` even if never explicitly released (no permanent deadlock from a crashed holder)

## Scenario: Service authenticates a request via IAM
**When:**
- A request carries a bearer JWT and hits a route protected by `iam` middleware

**Then:**
- Middleware validates the token (signature, expiry) and populates request context with the authenticated identity
- If Casbin policy is configured for the route, the middleware also enforces the policy and rejects unauthorized identities with a standardized 403 `stderr` response

## Scenario: New service bootstraps with minimal boilerplate
**When:**
- A developer starts a new Go service and imports `gocore`

**Then:**
- Wiring `stdlog` + `stdres` + one infra client (e.g. `database`) requires config plus a small number of explicit calls (no hidden global state, no reflection-based magic) — see design.md for the exact bootstrap call sequence
- Each package is independently importable: importing `gocore/database` does not require configuring `gocore/iam`

## Failure Scenario: Invalid/malformed config at startup
**When:**
- Required config fields (e.g. DB DSN, Redis address) are missing or invalid

**Then:**
- The relevant `New*` constructor returns a clear `stderr` config-validation error at startup (fail fast), not a nil/zero-value client that fails mysteriously later

## Failure Scenario: Downstream infra unavailable after startup
**When:**
- Postgres/Redis/Kafka becomes unreachable after the service has started (network blip, restart)

**Then:**
- Wrapper retries per configured policy (bounded attempts/backoff) rather than retrying forever or failing immediately on first transient error
- After retries are exhausted, the caller receives a typed `stderr` error (e.g. `ErrUnavailable`), never a panic

## Failure Scenario: JWT invalid or expired
**When:**
- `iam` middleware receives a malformed, expired, or signature-invalid token

**Then:**
- Request is rejected with a standardized 401 `stderr`/`stdres` response before reaching the handler; no partial identity is set in context

## Invariants
- Every error returned across a package boundary in `gocore` is a `stderr`-typed error (or wraps one) — no raw driver/library errors leak to callers.
- Every HTTP response emitted through `stdres` follows one envelope shape for success and one for error, regardless of which package produced the underlying error.
- No `gocore` package other than `iam` imports auth-specific third-party deps (Casbin/Firebase/OAuth2) — importing `gocore/database` or `gocore/cache` alone must not pull those in.
- Config structs for each package are explicit Go structs (viper-decoded), never global mutable package state read implicitly.

## Rules
- Each infra wrapper package (`database`, `cache`, `queue`, `lock`) exposes its own `Config` struct and `New*` constructor — no shared "god config" struct across packages.
- Retry/backoff policy is configurable per wrapper but ships with a sane default so a service can omit it.
- `stdlog` is context-aware: log calls accept `context.Context` so trace/request IDs propagate automatically when present.

## Constraints
- v1 targets Postgres only for `database` (see proposal Non-goals) — MySQL is not implemented.
- v1 targets Gin only for `httpserver` — no ConnectRPC/HTTP3 adapter in this task.
- `iam` composes existing libraries (Casbin, golang-jwt, Firebase Admin SDK, golang.org/x/oauth2) rather than reimplementing auth protocols.
