# Expected Behavior: golib-adoption

## Scenario: Cursor pagination over two or more sort columns
**When:**
- A service calls `pagination.Cursor` with `Sorts = [{created_at, Desc}, {id, Desc}]` on a table where many rows share `created_at`
- and walks pages by passing `NextCursor` back until `HasMore == false`

**Then:**
- The concatenation of all pages equals `SELECT * ORDER BY created_at DESC, id DESC` exactly: no row skipped, no row repeated
- The same holds for 1 or 3 columns and for mixed ASC/DESC directions

## Scenario: Client tampers with a cursor
**When:**
- The cursor payload or signature is modified, signed with another key, issued for a different sort, has an unknown version, or is not valid base64

**Then:**
- `pagination.Cursor` returns a `*stderr.Error` with `CodeInvalidParam` and key `pagination.invalid_cursor`
- No query is executed, nothing panics, the process does not exit

## Scenario: Cursor values keep their type
**When:**
- The sort includes an `int64` above 2^53 or a `time.Time` with nanoseconds

**Then:**
- The decoded cursor value equals the original exactly, and the next page starts right after that row (tested on SQLite; the predicate uses only standard SQL comparisons and bound parameters, so Postgres and MySQL behave the same, but they are not covered by tests here)

## Scenario: Limits are bounded
**When:**
- `Limit <= 0`, or `Limit > MaxLimit` (same for `PageSize` in offset pagination)

**Then:**
- The default is used, or the value is capped at the maximum. A client can never request an unbounded page.

## Scenario: Offset pagination
**When:**
- A service calls `pagination.Offset` with `Page = 3, PageSize = 10` on 45 rows

**Then:**
- It returns rows 21..30 in `Sorts` order, `Total = 45`, `TotalPages = 5`
- `Page` beyond the last page returns empty `Items` with correct `Total`; `Page <= 0` is treated as 1

## Scenario: Mocks drift from interfaces
**When:**
- A developer or agent changes an interface method but does not regenerate mocks

**Then:**
- `make verify` fails at `mocks-check` with a diff of the mock package

## Scenario: Agent violates a mechanical rule
**When:**
- Code adds `panic`, `log.Fatal`, `os.Exit` or `time.Sleep` in library code, an unowned vendor import, a same-rank import not on the whitelist, `package main` in the library, unformatted code, or lowers a package's coverage below baseline

**Then:**
- `make verify` fails locally (and so does the Stop hook gate), and the GitHub Actions `verify` job fails on the PR, naming the file and rule ID

## Scenario: Adding a gRPC server later
**When:**
- A `grpcserver` package is created importing `google.golang.org/grpc` or `connectrpc.com`

**Then:**
- archtest accepts it at rank 4 with no archtest edit beyond adding its sub-packages
- It reuses the request-ID and panic-recovery helpers without importing `httpserver` or Gin
- No existing package's public API changes
