# Domain Knowledge & Invariants

## Core Concepts
- **Microservice Chassis**: `gocore` is a foundation infrastructure toolkit. Downstream microservices import `gocore` to become production-ready without boilerplate setup.
- **Unified Error Protocol**: `stderr` is the single source of truth for errors. `stdres` and `stdlog` map typed error codes (`1xxx`, `4xxx`, `5xxx`) directly to HTTP status codes, log levels, and standard response envelopes.
- **Resilient Connectors**: Infrastructure wrappers (`database`, `cache`, `queue`, `lock`) provide built-in connection pooling, health checks (`Ping`/`Health`), retries, and graceful shutdown (`Close`).
- **IAM Isolation**: `iam` handles authn/authz (JWT, OAuth2, Firebase, Casbin) and depends only on consumer interfaces, never directly on `database` or `cache`.

## Invariant Rules
1. **CS-1** `[test]` No `init()` functions or process-global mutation in library code (enforced by `internal/archtest`).
2. **CS-2** `[review]` No package-level global variables (`var GlobalDB`). Explicit initialization via constructor functions `New*(cfg Config)`.
3. **CS-3/CS-6** `[test]` Zero `panic()`, `log.Fatal*`, `os.Exit` in library code (archtest `TestNoPanicOrExit`). Failures return `error`.
4. **CS-4** `[review]` Single source of truth for error handling: all errors wrapped/returned through `stderr`. Never leak raw database/driver errors.
5. **CS-5** `[review]` Context & Concurrency: first parameter of I/O functions must be `ctx context.Context`. Goroutines require explicit lifecycle owners (`errgroup`/`sync.WaitGroup`).
6. **ARCH-1/5** `[test]` Dependency hierarchy strictly bottom-up by rank (`stderr` < `stdlog` < `stdres`/`env`/`monitor` < `database`/`cache`/`queue` < `lock`/`httpserver`/`grpcserver` < `iam`). No cycles; infra packages independent (only `lock→cache`); `iam` never imports infra.
7. **ARCH-2/3** `[test]` Vendor SDK imports isolated to owner packages/adapters (e.g. `iam/firebase`, `stdres/stdgin`); every package has a `doc.go` package comment.
8. **ARCH-4** `[review]` Independent per-package configuration structs (`database.Config`, `cache.Config`, etc.); no monolithic shared configs.
9. **DEP-1** `[review]` Standard library first (`log/slog`, `errors`, `sync`), always latest stable versions (`@latest`), zero bloat.
10. **TEST-1/2** `[test]` Table-driven tests under `go test ./... -race -cover` (80% target on core packages); `testify` only in `*mock` sub-packages.
11. **EXT-4/6** `[test]` Mocks are generated (`make mocks`, `mocks-check`) and carry `var _ Iface = (*Mock)(nil)`; no `package main` or demos in the library.
12. **SEC-1** `[lint]` No secrets/tokens/credentials in the repo (`make secrets`, gitleaks).
13. **GIT-1** `[review]` Conventional Commits in lowercase imperative mood (see `.agents/rules/git-workflow.md`).

## Key Learnings / Quirks
- **Architectural Tests**: Run `go test ./internal/archtest/...` to catch boundary leaks, package ranking violations, or missing `doc.go` files early.
- **Adapters Layout**: Put web framework or vendor-specific glue in sub-packages (e.g., `stdres/stdgin` adapts `stdres` to Gin; `iam/iamgin` provides Gin auth middleware).
- **Domain Integrations Non-Goals**: Domain-specific logic (payments, crypto, emails) belongs in consumer microservices, not in `gocore`.
- **Quality Gate**: `make verify` = gofmt + vet + golangci-lint + mocks-check + gitleaks + `-race` tests + coverage ratchet. Tools run via pinned `go run module@version`; CI is `.github/workflows/verify.yml`. After raising coverage: `make test && make cover-baseline`.
- **Error Exposure**: `stdres.FromError` never sends the text of unclassified (non-`*stderr.Error`) errors to clients (generic "internal server error"); transports call `stdres.LogUnclassified` so the real cause is logged once with the request ID. `lock.Acquire` reports contention as `CodeLockHeld` (409) and backend failures as `CodeUnavailable`/`CodeTimeout`.
- **CORS**: wildcard origins never get credentials; credentials only for explicitly listed origins. Client `X-Request-ID` is validated (<=128 chars, safe charset) before being echoed or logged.
- **HTTP Stacks**: Gin (`httpserver`, `stdgin`, `iamgin`) and net/http (`httpmw`, `stdres`, `iam/iamhttp`) share one implementation of the CORS decision, request-ID validation and request-log shape in `httpmw`. Chi needs no adapter of its own: its middleware are plain `net/http`.
- **Database Drivers**: the `database` core imports no `gorm.io/driver/*`; each driver lives in `database/{postgres,mysql,sqlite}` (archtest owners). `anydriver` links all drivers and is opt-in. Services issuing several JWT kinds must set `iam.JWTConfig.TokenType`.
- **Test Doubles**: no third-party test deps. Redis uses an in-process fake RESP server (`cache/fakeredis_test.go`), Kafka an unexported `reader`/`writer` seam, GORM code an in-memory SQLite. Ask before adding `miniredis`/`sqlmock`.
- **JWT Validation**: `iam.JWTValidator` enforces `SigningMethod`, `Issuer`, `Audience` and `ClockSkew` when set (the signer writes `iss`/`aud`).
- **Pagination**: `database/pagination` cursors are HMAC-signed keyset tokens (multi-column, typed values); the last sort must be unique, sort values non-NULL.
- **Generated Mocks**: Change the interface or `.mockery.yml`, then run `make mocks`. `make mocks-check` (part of `make verify`) fails when mocks drift. New interfaces need an entry in `.mockery.yml` and an assertion line in the mock package's `doc.go`.
- **go-library Scope**: `/home/ubuntu/projects/go-library` is the predecessor; only ideas are adopted, never wholesale copies. Domain packages (blockchain, payment, email, sign-in, socket, http3) stay out.
