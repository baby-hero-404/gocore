# Design: golib-adoption

## 1. Pagination — `database/pagination`

### 1.1 Why go-library's cursor is wrong with 2+ columns
go-library's `WithCursorPagination` does:
```go
for column, param := range criteria.Cursor {        // Cursor is map[string]CursorParam
    db.Where(fmt.Sprintf("%s %s ?", column, param.Operator), param.Value)
}
```
1. **Wrong predicate.** Sort `created_at DESC, id DESC`, last row `(t5, 42)` produces `created_at < t5 AND id < 42`. A row `(t4, 99)` belongs on the next page but is skipped because `99 > 42`. The correct keyset predicate is lexicographic: `created_at < t5 OR (created_at = t5 AND id < 42)`.
2. **Unordered.** `map` iteration is random, so column order (which matters for the predicate) is lost.
3. **No `ORDER BY` coupling.** The library never applies the sort, so cursor and sort can disagree.
4. **Lossy values.** Values go through `interface{}` and JSON: `int64` comes back as `float64` (precision loss above 2^53), `time.Time` as a string in whatever format, which breaks comparisons on SQLite.
5. **Unsafe failure.** `ToCursorPaginationCriteria` calls `log.Fatal` on a bad cursor, and `column`/`operator` are interpolated with `fmt.Sprintf`.
6. **Two request params** (`cursor` + `signature`), and the signature is optional when `cursor` is absent.

### 1.2 API
```go
package pagination

// Sort is one ORDER BY term. Columns come from the caller's code (or a whitelist), never raw from clients.
type Sort struct {
    Column string
    Desc   bool
}

// Codec signs and verifies cursor tokens with HMAC-SHA256.
type Codec struct{ key []byte }
func NewCodec(key []byte) (*Codec, error) // key ≥ 32 bytes, else stderr.CodeValidationFailed

// CursorRequest binds from query/JSON.
type CursorRequest struct {
    Limit  int    `form:"limit" json:"limit"`
    Cursor string `form:"cursor" json:"cursor,omitempty"`
}

type CursorPage[T any] struct {
    Items      []T    `json:"items"`
    NextCursor string `json:"next_cursor,omitempty"`
    HasMore    bool   `json:"has_more"`
}

// CursorQuery configures a keyset query.
type CursorQuery[T any] struct {
    Sorts        []Sort          // last term must be unique (e.g. primary key)
    Key          func(T) []any   // values of Sorts columns for one row, same order
    DefaultLimit int             // used when Limit <= 0 (default 20)
    MaxLimit     int             // hard cap (default 100)
}

func Cursor[T any](ctx context.Context, db *gorm.DB, codec *Codec, q CursorQuery[T], req CursorRequest) (CursorPage[T], error)

// Offset pagination.
type OffsetRequest struct {
    Page     int `form:"page" json:"page"`           // 1-based
    PageSize int `form:"page_size" json:"page_size"`
}
type OffsetPage[T any] struct {
    Items      []T   `json:"items"`
    Page       int   `json:"page"`
    PageSize   int   `json:"page_size"`
    Total      int64 `json:"total"`
    TotalPages int   `json:"total_pages"`
}
type OffsetQuery struct {
    Sorts           []Sort
    DefaultPageSize int // default 20
    MaxPageSize     int // default 100
}
func Offset[T any](ctx context.Context, db *gorm.DB, q OffsetQuery, req OffsetRequest) (OffsetPage[T], error)
```
Final names may be adjusted to fit the existing gocore naming rules (no stutter: `pagination.Cursor`, not `pagination.CursorPagination`).

### 1.3 Keyset predicate
For sorts `c0..cn` with cursor values `v0..vn`, build:
```
(c0 op0 v0)
OR (c0 = v0 AND c1 op1 v1)
OR ...
OR (c0 = v0 AND ... AND c(n-1) = v(n-1) AND cn opn vn)
```
`opi` is `>` for ASC and `<` for DESC, so mixed directions work. A row-value form `(a,b) > (x,y)` is not used because it is only correct when all directions match and SQLite/MySQL planners differ. Columns are emitted with `clause.Column` (GORM quotes them), and values as bound parameters through `clause.Expr`. No `fmt.Sprintf` into SQL.

`ORDER BY` is always applied from `Sorts`, and the query fetches `limit + 1` rows to compute `HasMore` without a count.

### 1.4 Cursor token
```
token   = base64url(payload) "." base64url(HMAC-SHA256(key, payload))
payload = JSON {"v":1, "s":"<sort fingerprint>", "k":[{"t":"i","v":"42"}, {"t":"t","v":"2026-10-28T10:00:00.123456789Z"}]}
```
- **Typed values** (`t`): `i` int64, `u` uint64, `f` float64, `s` string, `b` bool, `t` time.Time (RFC3339Nano UTC). Integers are encoded as strings so they keep full precision. Unsupported types and `nil` return `stderr.CodeInternal` when encoding: NULL sort columns are not supported, and callers must use `COALESCE` or a non-null column.
- **Sort fingerprint** (`s`): hash of the `Sorts` list. A cursor issued for a different sort is rejected (`stderr.CodeInvalidParam`).
- **Version** (`v`): allows a future format change or backward cursors without silent misreads.
- Verification uses `hmac.Equal` (constant time). Any decode, signature, version or fingerprint failure returns `stderr.CodeInvalidParam` with key `pagination.invalid_cursor`. It never panics or exits.

### 1.5 Dependencies and rank
- Lives at `database/pagination`, so it inherits rank 3 from `database`. It imports `gorm.io` (add `database/pagination` to the `gorm.io` owner list in archtest), `stderr` and the standard library (`crypto/hmac`, `crypto/sha256`, `encoding/base64`, `encoding/json`).
- No new third-party dependency.

### 1.6 Tests
Table-driven tests against in-memory SQLite (the driver already exists in `database/sqlite.go`):
- 1, 2 and 3 columns, all ASC, all DESC, mixed, with many duplicate values in leading columns. Walk every page and assert the union equals `SELECT ... ORDER BY` exactly (no gaps, no duplicates).
- Limit defaults, cap, `HasMore`/empty last page.
- Tampered payload, tampered signature, wrong key, wrong sort fingerprint, bad base64, wrong version, all returning `CodeInvalidParam`.
- int64 above 2^53 and `time.Time` with nanoseconds round-trip exactly.
- Offset: page 1, middle and last, page beyond total, page size cap, `TotalPages` math.

## 2. Mockery

- `go get -tool github.com/vektra/mockery/v3@latest`; run as `go tool mockery`. The vendor directory includes it after `go mod vendor`.
- `.mockery.yml` (v3 schema, testify template):
  - `dir: "{{.InterfaceDir}}/{{.SrcPackageName}}mock"`, `pkgname: "{{.SrcPackageName}}mock"`, `structname: "{{.InterfaceName}}"`. This keeps the `<pkg>/<pkg>mock` convention and `testify` out of production packages (TEST-2 archtest still holds).
  - Interfaces are listed explicitly per package, not `all: true`, so only public contracts get mocks.
- `make mocks` regenerates. `make verify` runs `mocks-check`: regenerate, then `git diff --exit-code -- '*mock/'` so stale mocks fail.
- Existing hand-written mocks are replaced. Helper functions not derivable from interfaces (if any) are kept in a separate `helpers.go` in the same mock package. Each replacement is checked against the current mock tests before deletion.
- Exact template variable names are verified against the installed mockery v3 version during implementation.

## 3. gRPC-ready architecture (no server yet)

| Prep | Detail |
|---|---|
| Rank slot | archtest `rank["grpcserver"] = 4` (same tier as `httpserver`). An unused key is harmless and documents intent. |
| Vendor owners | `google.golang.org/grpc` and `connectrpc.com` are owned only by `grpcserver`, so nobody adds them elsewhere first. |
| Transport-neutral helpers | Extract the logic now embedded in Gin middleware into plain functions a gRPC interceptor can reuse: request-ID resolution (`stdlog` context helpers) and panic-to-`stderr` conversion. Gin middleware becomes a thin adapter over them. No public API removed. |
| Error mapping | `stderr.Code` keeps `HTTPStatus()`. The gRPC code mapping is written later inside `grpcserver`, so `stderr` stays dependency-free. |
| Lifecycle shape | Rule: every server package exposes `Start(ctx) error` and `Shutdown(ctx) error`, so a future app runner can run HTTP and gRPC together under `errgroup`. `httpserver.Server.Start(ctx)` was aligned with this rule on 2026-10-28 (it previously had no `ctx`). |
| ADR | `docs/ai/adr/…-grpc-ready-layout.md` records the decision and the rejected alternative (putting gRPC inside `httpserver`). |

## 4. Quality gates

### 4.1 Makefile
```
verify: fmt-check lint archtest mocks-check test cover-check
fmt-check:   test -z "$$(gofmt -l $$(git ls-files '*.go'))"
lint:        go vet ./... && golangci-lint run
mocks:       go tool mockery
mocks-check: mocks + git diff --exit-code on mock dirs
test:        go test ./... -race -coverprofile
cover-check: go run ./internal/tools/covercheck   (compares to coverage-baseline.txt)
```
`golangci-lint` is installed as a binary (its project discourages `go tool` because of dependency conflicts). CI uses the official action, and local runs need it on `PATH`; the Makefile prints install instructions if it is missing.

### 4.2 `.golangci.yml` (v2 schema)
Linters: `errcheck`, `govet`, `staticcheck`, `ineffassign`, `unused`, `revive`, `noctx`, `contextcheck`, `errorlint`, `gosec`, `forbidigo` (forbids `panic`, `log.Fatal*`, `os.Exit`, `time.Sleep` outside tests), `gofumpt` formatter. Mock packages and `vendor/` are excluded.

### 4.3 archtest additions
| Test | Rule |
|---|---|
| `TestSameRankImportsWhitelisted` | ARCH-5: same-rank cross imports only for listed pairs (`lock → cache`); `iam` must not import `database`/`cache`/`queue`. |
| `TestNoPanicOrExit` | CS-3: AST scan of non-test files for `panic`, `log.Fatal*`, `os.Exit`. Backs up `forbidigo` so the rule holds without golangci-lint. |
| `TestNoMainInLibrary` | EXT-6: no `package main` outside `internal/tools` and `examples/`. |

### 4.4 Coverage baseline
`internal/tools/covercheck` (standard library only) parses the coverprofile per package and compares it to `coverage-baseline.txt` (`<pkg> <percent>`):
- below baseline (beyond 0.5 points of tolerance): fail;
- new package not in baseline: must reach 80%;
- `-update` rewrites the baseline after an intentional raise.

### 4.5 CI — `.github/workflows/verify.yml`
Push and PR to `master`: `actions/setup-go` (version from `go.mod`), `golangci/golangci-lint-action`, `make verify`, `go mod tidy` diff check, `govulncheck`, `gitleaks`. Actions are pinned by commit SHA at their latest releases.

### 4.6 Fix found during review
`database/connect.go:87` calls `time.Sleep(backoff)` inside the retry loop, which ignores `ctx` cancellation (CS-5), and the backoff doubles without a cap. Replace it with `select { case <-ctx.Done(): …; case <-time.After(backoff): }` and cap the backoff (`MaxBackoff`, default 30s). This file has uncommitted user changes, so it is edited only after they are committed or with the user's go-ahead.
