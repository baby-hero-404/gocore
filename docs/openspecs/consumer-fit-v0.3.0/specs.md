# Expected Behavior: gocore v0.3.0 consumer fit

## Scenario: A Postgres-only service carries no other driver
**When:**
- A service imports `database` and `database/postgres` only, then runs `go mod vendor`

**Then:**
- `vendor/modules.txt` lists no package from `gorm.io/driver/mysql`, `gorm.io/driver/sqlite` or `github.com/go-sql-driver/mysql`, and (unless the service imports it itself) none from `github.com/mattn/go-sqlite3`
- `go build` succeeds with `CGO_ENABLED=0`

## Scenario: Connect through a driver package
**When:**
- `postgres.New(ctx, cfg)` (or `mysql.New`, `sqlite.New`) is called with a reachable database

**Then:**
- It returns a `database.Database` with pool limits from `cfg`, after a successful ping
- Retries, backoff cap and context cancellation behave exactly as in v0.2.0

## Scenario: Runtime driver selection
**When:**
- `anydriver.New(ctx, cfg)` is called with `cfg.Driver` set to `postgres`, `""`, `mysql` or `sqlite`

**Then:**
- It connects with the matching driver

## Failure Scenario: Unknown driver
**When:**
- `anydriver.New` receives `cfg.Driver = "oracle"`

**Then:**
- It returns `*stderr.Error` with `CodeValidationFailed` and a `FieldError` on `driver`; no connection is attempted

## Failure Scenario: Empty DSN
**When:**
- Any driver package receives an empty `cfg.DSN`

**Then:**
- It returns `CodeValidationFailed` with a `FieldError` on `dsn` before opening anything

## Scenario: Custom id and roles claims
**When:**
- `JWTConfig{IDClaim: "userId", RolesClaim: "role"}` validates a token with `{"userId": "u1", "role": "admin"}`

**Then:**
- `Identity.ID == "u1"` and `Identity.Roles == ["admin"]`

## Scenario: Defaults keep v0.2.0 claim behaviour
**When:**
- `JWTConfig` sets none of `IDClaim`, `RolesClaim`, `TokenType`

**Then:**
- Identity is read from `sub` then `id`, roles from `roles` then `role`, and `typ` is ignored, exactly as in v0.2.0

## Scenario: Token type is written and checked
**When:**
- A signer with `TokenType: "access"` signs a token and a validator with `TokenType: "access"` validates it

**Then:**
- The token carries `"typ": "access"` and validates successfully, even if `extra` tried to set another `typ`

## Failure Scenario: Refresh token used as access token
**When:**
- A token signed with `TokenType: "refresh"` (or with no `typ`) is presented to a validator with `TokenType: "access"`

**Then:**
- Validation fails with `CodeUnauthorized` and key `KeyUnauthenticated`; `iamgin.JWTMiddleware` responds 401

## Scenario: zero-note protected route
**When:**
- A request with a valid access token calls a protected zero-note route

**Then:**
- The handler/use case obtains the user id from `iam.IdentityFromContext(ctx)` using the request context
- No code path converts `context.Context` to `*gin.Context`

## Failure Scenario: zero-note refresh token on a protected route
**When:**
- A request presents a zero-note refresh token as `Authorization: Bearer <refresh>` on a protected route

**Then:**
- The response is 401

## Scenario: Services build from a published tag
**When:**
- A fresh clone of zero-note, faster-learning or auto_code_os runs its normal build (Make target, Dockerfile, CI) without a `go.work` file

**Then:**
- The build fetches `github.com/baby-hero-404/gocore@v0.3.0` (and `aicli` at its tag) and succeeds

## Scenario: Local development against an unreleased gocore
**When:**
- A developer creates a `go.work` at the repository root with `use ./<module>` and `replace github.com/baby-hero-404/gocore => ../gocore`

**Then:**
- The service builds against the local gocore checkout; `go.work` is git-ignored and never committed

## Invariants
- The core `database` package never imports a `gorm.io/driver/*` module (enforced by archtest vendor owners).
- No `init()` or package-level mutable registry is introduced for driver selection.
- A `JWTConfig` without the new fields behaves identically to v0.2.0.
- No committed `go.mod` in the three services contains a `replace` that points outside its own repository.

## Rules
- Each breaking change in v0.3.0 has a CHANGELOG entry with a concrete migration step.
- Services that issue more than one kind of JWT must set `TokenType` on every signer and validator.

## Constraints
- Go 1.27.3 toolchain everywhere, including Dockerfiles.
- Latest stable versions of `gorm.io/driver/*` at implementation time (DEP-1).
- gocore `make verify` stays green; new packages reach at least 80% coverage (TEST-3).
