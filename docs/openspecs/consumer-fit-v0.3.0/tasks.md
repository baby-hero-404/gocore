# Implementation Map: gocore v0.3.0 consumer fit

**Goal:** Ship gocore v0.3.0 (driver split, configurable JWT claims and token type, CHANGELOG) and move zero-note, faster-learning and auto_code_os onto tagged releases with zero-note's auth fixed.
**Tech Stack:** Go 1.27.3, GORM + `gorm.io/driver/{postgres,mysql,sqlite}`, `golang-jwt/jwt/v5`, Gin, Chi (auto_code_os), `go.work`.

---

## Phase 1: gocore — database driver split

### Core connects through a dialector

**Why:**
Driver imports in the core package reach every consumer (design.md §Approach).

**Depends on:**
- None

**Files:**
- `database/new.go`, `database/connect.go`, `database/doc.go`
- Delete `database/mysql.go`, `database/sqlite.go`

**Changes:**
- [x] Export `Open(ctx, dialector, cfg, opts...) (Database, error)` and `Connect(ctx, dialector, cfg, opts...) (*gorm.DB, error)` built on the existing `connect` loop
- [x] Remove `New`, `NewPostgres`, `NewMySQL`, `NewSQLite` and the `gorm.io/driver/*` imports from `database`
- [x] Update `doc.go` and `Config.Driver` comment (read only by `anydriver`)

**Verify:**
- [x] `go list -deps ./database | grep gorm.io/driver` prints nothing
- [x] Existing retry, cancel, pool-leak and logger tests pass against a dialector passed in

**Cleanup:**
- [x] `grep -rn "NewPostgres\|NewMySQL\|NewSQLite\|database.New(" --include=*.go .` returns only the new sub-packages

### Driver sub-packages and anydriver

**Why:**
Each service imports only the driver it uses; zero-note keeps runtime choice explicitly.

**Depends on:**
- Core connects through a dialector

**Files:**
- `database/postgres/{doc.go,postgres.go,postgres_test.go}`
- `database/mysql/{doc.go,mysql.go,mysql_test.go}`
- `database/sqlite/{doc.go,sqlite.go,sqlite_test.go}`
- `database/anydriver/{doc.go,anydriver.go,anydriver_test.go}`
- `internal/archtest/archtest_test.go`, `database/pagination/pagination_test.go`, `database/driver_test.go`, `database/postgres_test.go`, `database/database_ops_test.go`

**Changes:**
- [x] `New` and `Open` in each driver package
- [x] `anydriver.New` with the switch moved from `database.New`
- [x] archtest: `gorm.io/driver/postgres` owned by `database/postgres` (+ `anydriver`), same for mysql and sqlite; `gorm.io/gorm` stays with `database`, `database/pagination`, `databasemock`
- [x] Move driver-specific tests to their packages; pagination tests import `database/sqlite`

**Verify:**
- [x] specs.md: runtime selection, unknown driver, empty DSN scenarios have table-driven tests
- [x] Each new package ≥ 80% coverage; `make verify` green
- [x] Throwaway module importing only `database/postgres`: `go mod vendor` has no mysql/sqlite packages and builds with `CGO_ENABLED=0`

## Phase 2: gocore — iam claims and token type

### Configurable claims and token type

**Why:**
Lets zero-note adopt `iam` without changing token format, and closes the access/refresh confusion in the library.

**Depends on:**
- None (independent of Phase 1)

**Files:**
- `iam/jwt.go`, `iam/signer.go`, `iam/README.md`, `iam/iam_extra_test.go`, `iam/signer_test.go`

**Changes:**
- [x] Add `IDClaim`, `RolesClaim`, `TokenType` to `JWTConfig`
- [x] Validator reads configured claims; falls back to v0.2.0 lookup when unset
- [x] Signer writes `typ` when `TokenType` is set (overrides `extra["typ"]`); validator enforces it
- [x] README: services issuing more than one token kind must set `TokenType`

**Verify:**
- [x] specs.md: custom claims, defaults unchanged, token type written, refresh-as-access rejected
- [x] Untyped legacy token rejected by a typed validator; accepted by an untyped one

## Phase 3: gocore — documentation and release

### CHANGELOG and docs

**Why:**
Behaviour changes in v0.2.0 and v0.3.0 are invisible without notes.

**Depends on:**
- Phases 1 and 2

**Files:**
- `CHANGELOG.md`, `README.md`, `database/README.md`, `docs/ai/PROFILE.md`, `docs/ai/DOMAIN.md`, `.mockery.yml` (if interfaces moved)

**Changes:**
- [x] `CHANGELOG.md` with v0.2.0 (retroactive) and v0.3.0 entries, each with a Migration list
- [x] README package index: driver packages and `anydriver`
- [x] Refresh `docs/ai` and run `memory_stale.py --update`

**Verify:**
- [x] Every Public API removal in proposal.md has a migration line in CHANGELOG
- [x] `make verify` and `make vuln` green
- [x] Tag `v0.3.0` pushed (only after user approval)

## Phase 4: zero-note

### Fix refresh-token acceptance and adopt iam

**Why:**
A refresh token currently opens every protected route; use-case code depends on `*gin.Context`.

**Depends on:**
- Phase 2 (and v0.3.0 tag for CI; `go.work` locally before that)

**Files:**
- `backend/internal/infrastructure/auth/jwt.go`
- `backend/internal/usecase/auth_usecase.go`
- `backend/internal/interface/middleware/auth_middleware.go` (delete), `identity_middleware.go`
- `backend/cmd/server/main.go`
- the 19 `GetUserID` call sites

**Changes:**
- [x] Token generator built on `iam.JWTSigner` / `iam.JWTValidator` with access and refresh configs (`TokenType`, `IDClaim: "userId"`, `RolesClaim: "role"`)
- [x] Protected group uses `iamgin.JWTMiddleware(accessValidator)`
- [x] Replace `GetUserID(ctx)` with a helper reading `iam.IdentityFromContext(ctx)`; handlers pass `c.Request.Context()`
- [x] `RefreshToken` uses the refresh validator

**Verify:**
- [x] Test: refresh token on a protected route → 401; access token → 200
- [x] Test: refresh endpoint rejects an access token
- [x] `grep -rn "\.(\*gin.Context)" backend/internal` returns nothing
- [x] zero-note's own test suite passes

**Cleanup:**
- [x] Remove `auth_middleware.go`, `GetUserID`, unused `CustomClaims` parsing code

### Database wiring and build

**Why:**
`coredb.New` is removed in v0.3.0; the dev Dockerfile cannot build.

**Depends on:**
- Phase 1

**Files:**
- `backend/internal/infrastructure/database/connection.go`, `backend/Dockerfile`, `backend/go.mod`, `backend/go.sum`, `.gitignore`

**Changes:**
- [x] Use `anydriver.New` (zero-note selects Postgres or SQLite from config)
- [x] Dockerfile: `golang:1.27-alpine`, `CGO_ENABLED=0` unless SQLite is needed in the image
- [x] Remove `replace`; `go get github.com/baby-hero-404/gocore@v0.3.0`; add `go.work` and `go.work.sum` to `.gitignore`

**Verify:**
- [x] `docker build backend` succeeds from a clean clone
- [ ] App starts against Postgres and SQLite configs

## Phase 5: faster-learning

### Tagged gocore and driver package

**Why:**
Build currently depends on a local checkout through `replace` + vendor.

**Depends on:**
- Phase 1, v0.3.0 tag

**Files:**
- `api/internal/platform/store/store.go`, `api/go.mod`, `api/go.sum`, `.gitignore`, `Dockerfile.api`, `Makefile`

**Changes:**
- [x] `database.New` → `postgres.New`
- [x] Remove gocore and aicli `replace`; require tags; ignore `go.work`
- [x] Update the `Dockerfile.api` comment and `make vendor` (vendor now comes from tags)

**Verify:**
- [x] `make vendor && docker build -f Dockerfile.api .` succeeds from a clean clone
- [x] `internal/archtest` and full test suite pass

## Phase 6: auto_code_os

### Postgres-only import and smaller vendor

**Why:**
Removes 11 MB SQLite/CGO and MySQL code from `vendor/`.

**Depends on:**
- Phase 1, v0.3.0 tag

**Files:**
- `server/internal/database/database.go`, `server/go.mod`, `server/go.sum`, `server/vendor/`, `.gitignore`

**Changes:**
- [x] `gcdb.NewPostgres` → `postgres.Open`
- [x] Remove gocore and aicli `replace`; require tags; ignore `go.work`; re-vendor

**Verify:**
- [x] `vendor/` has no `gorm.io/driver/{mysql,sqlite}` or `go-sql-driver/mysql` (`go-sqlite3` stays: auto_code_os imports it itself)
- [x] `server` builds and its tests pass; sandbox Dockerfile still builds (`go-tree-sitter` CGO unaffected)

---

## Rollback Plan

### Safe Deploy
- [ ] Release gocore v0.3.0 first; services stay on v0.2.0 until their phase is done (no forced upgrade)
- [ ] Deploy zero-note after announcing a one-time logout; roll back by redeploying the previous image (old tokens still valid there)
- [ ] faster-learning and auto_code_os: rollback = revert the go.mod/go.sum commit to v0.2.0

---

## Implementation notes (2026-10-28)
- **Pending, needs the user's approval:** commit, push and tag `v0.3.0` of gocore. Until then the
  three services build only through the git-ignored `go.work` at each repository root, and these
  steps stay open: `go mod tidy` / `go mod vendor` (faster-learning, auto_code_os, run with
  `GOWORK=off`), `docker build` checks, and the "fresh clone builds from the tag" scenario.
- `go.work` lives at each repository root (not in the module directory) so Docker contexts that
  copy `backend/` or `api/` never see it, and it uses `replace` (not only `use`): a `use`d module
  still fails to resolve when `go.mod` requires a version that is not published yet.
- aicli: the local checkout is one line away from the published `v0.1.2`, not `v0.1.1`, so the
  services now require `aicli v0.1.2`.
- Correction to proposal.md: `auto_code_os` imports `github.com/mattn/go-sqlite3` itself
  (`internal/context/source/cache.go`), so that 11 MB is not gocore's doing. The split removes only
  gocore's GORM SQLite and MySQL drivers from it; faster-learning no longer links any of the three
  beyond Postgres.
- Found while migrating zero-note: 15 handlers called `GetUserID(c.Request.Context())`, which can
  never work with the old implementation (it asserted `*gin.Context`), so notes, folders, tasks and
  shared notes ran with an empty user id. Reading the identity from the request context fixes them
  and is covered by the new middleware tests.
- zero-note: access and refresh tokens now carry `typ`; sessions issued before the deploy are
  rejected once and users sign in again.

## Post-release verification (v0.3.0 tagged at ce7695f)
- `GOWORK=off go mod tidy` + build + tests pass in zero-note, faster-learning and auto_code_os against the published tag.
- `docker build` succeeds for zero-note (`backend/Dockerfile`) and faster-learning (`Dockerfile.api`); the auto_code_os sandbox image was not built.
- faster-learning `vendor/` contains only `gorm.io/driver/postgres`; auto_code_os only that driver plus its own `go-sqlite3`.
- **Footgun found:** `go mod vendor` drops header-only directories, so auto_code_os's CGO build of `smacker/go-tree-sitter` (`php/tree_sitter`, `sql/tree_sitter`) fails after re-vendoring. They were restored from the module cache; the Makefile vendor step should do this automatically.
- Not done: zero-note handlers still put `err.Error()` into client messages, and its refresh cookie has no `Secure`/`SameSite` (see review notes).
