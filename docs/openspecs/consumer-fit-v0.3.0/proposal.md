# Proposal: gocore v0.3.0 consumer fit

## Problem
gocore v0.2.0 is now used by three services: `zero-note`, `faster-learning` and `auto_code_os`. A review of how they use it (2026-10-28) found friction that comes from gocore itself, plus problems in the services:

1. **Driver bloat.** `gocore/database` imports the Postgres, MySQL and SQLite drivers. `auto_code_os` uses only Postgres through gocore, yet its `vendor/` carries gocore's GORM SQLite driver and the MySQL driver (the `mattn/go-sqlite3` library itself is also imported by auto_code_os directly, so that part stays; see tasks.md notes). Every driver gocore adds lands in every consumer.
2. **zero-note cannot adopt `iam`.** Its tokens carry the user id in a `userId` claim. `iam.JWTValidator` only reads `sub` or `id`, so `Identity.ID` would be empty. zero-note therefore keeps a hand-written auth middleware and reads the user with `ctx.(*gin.Context)`, which ties use-case code to Gin.
3. **zero-note accepts refresh tokens as access tokens.** Both tokens have the same claims and secret and one `ValidateToken` checks both, so a 7-day refresh token works on every protected route. gocore's `iam` has the same blind spot: it has no notion of token type.
4. **Builds depend on the local checkout.** All three `go.mod` files contain `replace github.com/baby-hero-404/gocore => ../../gocore` (two also replace `aicli`). Builds therefore use whatever gocore is on disk, not the `v0.2.0` tag, and `zero-note/backend/Dockerfile` cannot build at all (no `../../gocore` in the image, and it uses Go 1.23 while `go.mod` needs 1.27.3).
5. **No changelog.** v0.2.0 changed behaviour in ways a consumer cannot see from compiler errors alone (500 responses hide raw error text, CORS no longer sends credentials for `*`, `lock.Acquire` error codes, `JWTValidator` enforcing issuer/audience). Nothing tells consumers.

## Goal
Remove the friction on the gocore side (v0.3.0), and bring the three services onto tagged gocore releases with the zero-note auth problems fixed.

## Success
- A service that uses only Postgres compiles and vendors no MySQL or SQLite code from gocore.
- zero-note authenticates every protected route through `iam`/`iamgin`, reads the user from `context.Context`, and rejects refresh tokens on those routes.
- All three services build in CI and Docker from a published gocore tag, with no `replace` pointing outside the repository.
- A consumer upgrading gocore can read in `CHANGELOG.md` what changed in behaviour and what to edit.

## Assumptions
- gocore is pre-1.0, so a breaking change ships as a minor bump (v0.3.0) with migration notes. All three consumers are owned by the same person and are migrated in this task.
- zero-note keeps its current token format (HS256, `userId`, `role` claims) for this task; existing tokens of logged-in users may be invalidated once (see Backward Compatibility).
- `aicli` follows the same tagging practice; its `replace` is handled the same way as gocore's.
- Postgres is the production database for all three services; SQLite is used by zero-note for local/dev only.

## Decisions
- **Split drivers into sub-packages** (`database/postgres`, `database/mysql`, `database/sqlite`). The core `database` package keeps config, pooling, retries, ping, close and error mapping, and connects through a `gorm.Dialector` it receives. A service imports only the driver it needs. Details in design.md.
- **Keep runtime driver selection opt-in** in a separate package (`database/anydriver`), for services such as zero-note that choose the driver from configuration. Importing it is an explicit choice to carry every driver.
- **Remove `database.New`, `NewPostgres`, `NewMySQL`, `NewSQLite` from the core package** instead of deprecating them. Keeping them would keep the driver imports in the core package, which is the problem being solved.
- **Configurable claim mapping and token type in `iam.JWTConfig`** (`IDClaim`, `RolesClaim`, `TokenType`). The signer writes `typ` when `TokenType` is set; the validator rejects tokens whose `typ` differs. This fixes zero-note's bug class in the library, not only in zero-note.
- **`go.work` for local development, tags for everything else.** `replace` directives pointing outside the repo are removed from all three `go.mod` files; developers who work on gocore and a service together use a git-ignored `go.work`.
- **Add `CHANGELOG.md`** (Keep a Changelog format) with entries for v0.2.0 (written retroactively) and v0.3.0, each with a "Migration" list.

## Trade-offs
- Driver split: consumers pay one import change per call site, and gocore gains three small packages. In return, the dependency cost of each driver is paid only by services that use it.
- `anydriver` reintroduces the "carry everything" cost for whoever imports it; acceptable because it is opt-in and named for what it does.
- `TokenType` adds a claim to signed tokens. Tokens signed before the change have no `typ`; a validator with `TokenType` set rejects them, which logs users out once.
- `go.work` means a fresh clone builds against the published tag, so a gocore change must be tagged before a service can use it in CI. This is the intended discipline.

## Non-goals
- Do not move `auto_code_os` from Chi to Gin. It uses gocore only for `database` and `stdlog`, which are router-independent.
- Do not split gocore into multiple Go modules. Package-level splitting already removes the measured bloat; multi-module releases would multiply tagging work.
- Do not change zero-note's token signing algorithm, lifetimes, or refresh flow beyond the token-type check.

## Out of Scope
- A `net/http` auth middleware in gocore (`iam/iamhttp`). No consumer needs it today: `auto_code_os` does not use gocore auth. Add it when a Chi or `net/http` service needs `iam`.
- Fixing `faster-learning` or `auto_code_os` application logic unrelated to gocore.
- The Kafka consumer commit behaviour (tracked separately, deferred by the user).

## Impact

### Components
- gocore: `database`, new `database/postgres`, `database/mysql`, `database/sqlite`, `database/anydriver`; `iam`; `internal/archtest`; docs.
- zero-note: `backend/` auth, database wiring, Dockerfile, `go.mod`.
- faster-learning: `api/` database wiring, `go.mod`, Makefile/Dockerfile vendor step.
- auto_code_os: `server/` database wiring, `go.mod`, vendor.

### Files
gocore:
- Modify: `database/new.go`, `database/connect.go`, `database/doc.go`, `database/README.md`, `database/driver_test.go`, `database/database_ops_test.go`, `database/postgres_test.go`
- Delete: `database/mysql.go`, `database/sqlite.go`
- Create: `database/postgres/{doc.go,postgres.go,postgres_test.go}`, `database/mysql/{doc.go,mysql.go,mysql_test.go}`, `database/sqlite/{doc.go,sqlite.go,sqlite_test.go}`, `database/anydriver/{doc.go,anydriver.go,anydriver_test.go}`
- Modify: `iam/jwt.go`, `iam/signer.go`, `iam/README.md`, `iam/*_test.go`
- Modify: `internal/archtest/archtest_test.go` (vendor owners for driver modules), `database/pagination/pagination_test.go` (SQLite import path)
- Create: `CHANGELOG.md`; modify `README.md`, `docs/ai/PROFILE.md`, `docs/ai/DOMAIN.md`

zero-note (`/home/ubuntu/my_projects/zero-note`):
- Modify: `backend/internal/infrastructure/auth/jwt.go`, `backend/internal/infrastructure/database/connection.go`, `backend/cmd/server/main.go`, `backend/go.mod`, `backend/go.sum`, `backend/Dockerfile`, `.gitignore`
- Delete: `backend/internal/interface/middleware/auth_middleware.go` (replaced by `iamgin.JWTMiddleware`); update the 19 `GetUserID` call sites and `identity_middleware.go`

faster-learning (`/home/ubuntu/my_projects/faster-learning`):
- Modify: `api/internal/platform/store/store.go`, `api/go.mod`, `api/go.sum`, `.gitignore`, `Dockerfile.api` comment, Makefile `vendor` target

auto_code_os (`/home/ubuntu/my_projects/auto_code_os`):
- Modify: `server/internal/database/database.go`, `server/go.mod`, `server/go.sum`, `server/vendor/`, `.gitignore`

### Public API
- Removed from `database`: `New`, `NewPostgres`, `NewMySQL`, `NewSQLite`, and the `Config.Driver` field's meaning moves to `anydriver`.
- Added to `database`: `Open(ctx, dialector, cfg, opts...) (Database, error)` and `Connect(ctx, dialector, cfg, opts...) (*gorm.DB, error)`.
- Added packages: `database/postgres`, `database/mysql`, `database/sqlite` (each `New(ctx, cfg, opts...) (database.Database, error)` and `Open(ctx, cfg, opts...) (*gorm.DB, error)`), `database/anydriver` (`New(ctx, cfg, opts...)` switching on `Config.Driver`).
- Added to `iam.JWTConfig`: `IDClaim`, `RolesClaim`, `TokenType`. Defaults keep v0.2.0 behaviour.
- New error: `iam` returns `CodeUnauthorized` with key `KeyUnauthenticated` for a token-type mismatch (same as other invalid tokens).

### Migration
- Consumers: replace `database.New(ctx, cfg)` with `postgres.New(ctx, cfg)` (or `anydriver.New` for runtime selection); replace `database.NewPostgres` with `postgres.Open`.
- zero-note: set `TokenType` on access/refresh configs; users are logged out once after deploy.
- All services: drop `replace` lines, run `go get github.com/baby-hero-404/gocore@v0.3.0`, `go mod tidy`, re-vendor where `vendor/` is used.

### Backward Compatibility
Breaking for Go code that calls the removed `database` constructors (compile errors with clear fixes, listed in CHANGELOG). `iam` defaults are unchanged: a config without the new fields behaves exactly as v0.2.0. zero-note session tokens issued before the deploy stop working once `TokenType` is enforced; users sign in again.
