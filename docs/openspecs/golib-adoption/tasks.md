# Implementation Map: golib-adoption

**Goal:** Adopt the useful ideas from go-library into gocore with mechanical enforcement. See proposal.md (scope), design.md (how), specs.md (expected behavior).

---

## Phase 1: Rules (text)

**Files:** `.agents/rules/coding-standards.md`, `.agents/rules/architecture.md`, `.agents/rules/testing.md`, `.agents/rules/dependency-management.md`, `.agents/rules/git-workflow.md`, `.agents/rules/extension.md` (new), `.agents/rules/agent-workflow.md` (new), `AGENTS.md`, `docs/ai/DOMAIN.md`

- [x] Add rule IDs and enforcement tags to every existing rule file
- [x] New `extension.md`: EXT-1..EXT-6 (new package checklist, public API stability, vendor owner entry, scope gate, mock generation, no `main` in library)
- [x] New `agent-workflow.md`: AGT-1..AGT-7 (read rules and OpenSpec first, OpenSpec for non-trivial work, `make verify` before reporting done, surgical edits, ask before deps/CI/breaking changes, keep docs in sync)
- [x] Add SEC-1 (no secrets in repo) and ARCH-5/ARCH-6 (same-rank whitelist, transport-neutral core)
- [x] Update AGENTS.md rules index and DOMAIN.md invariants (≤13 lines)

## Phase 2: Quality gates

**Files:** `Makefile`, `.golangci.yml`, `internal/archtest/archtest_test.go`, `internal/tools/covercheck/`, `coverage-baseline.txt`, `.github/workflows/verify.yml`

- [x] `.golangci.yml` v2 with the linters in design.md §4.2; fix or justify every finding
- [x] archtest: `TestSameRankImportsWhitelisted`, `TestNoPanicOrExit`, `TestNoMainInLibrary`; `grpcserver` rank slot and vendor owners
- [x] `internal/tools/covercheck` with success and failure tests; generate `coverage-baseline.txt`
- [x] Makefile: `fmt-check`, `lint`, `mocks`, `mocks-check`, `cover-check`; `verify` chains them
- [x] GitHub Actions workflow (actions pinned by SHA, latest releases)

**Verify:**
- [x] Each gate fails on a deliberately broken sample, then passes on clean code (checked: gofmt, panic archtest; others covered by unit tests)
- [x] `make verify` green

## Phase 3: Mockery

**Files:** `go.mod`, `go.sum`, `vendor/`, `.mockery.yml`, all `*mock/` packages

- [x] `go get -tool github.com/vektra/mockery/v3@latest`, `go mod tidy`, `go mod vendor`
- [x] `.mockery.yml` listing every public interface; output into `<pkg>/<pkg>mock`
- [x] Generate; delete hand-written mocks and their self-tests (they tested the old hand-written code); compile-time assertions live in each mock package's `doc.go`
- [x] R2 check: `git status` for unexpected AI-instruction files after install

**Verify:**
- [x] `make mocks-check` clean; archtest TEST-2 passes

## Phase 4: Pagination

**Files:** `database/pagination/{doc.go,sort.go,codec.go,cursor.go,offset.go,*_test.go,README.md}`, `internal/archtest/archtest_test.go`, `README.md`

- [x] `Codec` (HMAC-SHA256, typed values, version, sort fingerprint)
- [x] `Cursor` with expanded keyset predicate, `ORDER BY`, `limit + 1`
- [x] `Offset` with count on a cloned session and page-size cap
- [x] Tests per design.md §1.6 (SQLite, table-driven, ≥80% coverage)
- [x] README and package index entry

**Verify:**
- [x] All specs.md pagination scenarios covered by tests

## Phase 5: gRPC-ready prep

**Files:** `stdlog/with.go`, `httpserver/middleware.go`, `.agents/rules/architecture.md`, `docs/ai/adr/`

- [x] Extract request-ID and panic-to-`stderr` helpers into transport-neutral functions (`stdlog.ContextWithRequestID`, `stderr.FromPanic`); Gin middleware calls them
- [x] ADR for the gRPC-ready layout

## Phase 6: Fix

**Files:** `database/connect.go`, `database/config.go`

- [x] Replace `time.Sleep` with context-aware wait and cap backoff (needs the user's uncommitted `database/*` changes settled first)

## Phase 7: Close out

- [x] `make verify` green; update README, PROFILE.md, DOMAIN.md; `memory_stale.py --update`

---

## Implementation notes (2026-10-28)
- `golangci-lint`, `gitleaks` and `govulncheck` run via `go run <module>@<pinned version>` (Makefile / CI), so nothing needs installing and `go.mod` stays clean. Only `mockery` is a `go tool` dependency, as decided.
- `TestInfraPackagesStayIndependent` replaces the planned `TestSameRankImportsWhitelisted`; `TestNoPanicOrExit` skips `package main` (only `internal/tools` may have one, per EXT-6).
- Invalid cursors return `stderr.CodeInvalidParam` (HTTP 400), not `CodeValidation`.
- `database/connect.go` retry now waits with `select` on `ctx.Done()` and caps the backoff at 30s.
- Fixed while enabling lint: ignored `Close` errors, `err.(T)` assertion in `env`, unused `stdlog.keyToString`, missing doc comments on exported symbols.
