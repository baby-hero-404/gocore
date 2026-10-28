# Proposal: framework-neutral net/http adapters (Chi and friends)

## Problem
gocore's HTTP layer is Gin-only (`httpserver`, `stdres/stdgin`, `iam/iamgin`, `monitor/promgin`). `auto_code_os` routes with Chi, so it can use only `database` and `stdlog` (about 20% of gocore). Its handlers already speak `net/http`, and it carries its own request-ID, recovery, logging, CORS and JWT middleware.

## Goal
Let any `net/http` router (Chi, `http.ServeMux`, Gorilla, ...) get the same request ID, panic recovery, request logging, CORS and JWT/Casbin behaviour that Gin services get, with the same error envelope and the same security rules.

## Success
- A Chi service can replace its hand-written middleware with gocore's and returns byte-for-byte the same envelopes and headers as a Gin service for the same failures (401, 403, 500, CORS).
- The Gin and net/http middleware share one implementation of the security-sensitive rules (CORS origin decision, request-ID validation), so they cannot drift.
- gocore gains no new third-party dependency.

## Assumptions
- Chi middleware is `func(http.Handler) http.Handler`, so a `net/http` middleware is a Chi middleware. No Chi import is needed.
- `stdres` already has `net/http` helpers (`WriteOK`, `WriteError`, ...), so no `stdres/stdchi` is needed.

## Decisions
- **Name by capability, not by router:** `httpmw` (middleware) and `iam/iamhttp` (auth), not `stdchi`/`iamchi`. A Chi-named package would import Chi for nothing and exclude every other router.
- **`httpmw` is a new top-level package at rank 3** (above `stdres`, below the infrastructure packages), importing only `stderr`, `stdlog`, `stdres` and the standard library plus `google/uuid`.
- **Share, do not copy:** the CORS decision and request-ID validation move into `httpmw`; `httpserver` (Gin) becomes a thin adapter over them. Existing Gin tests guard the refactor.
- **`iam/iamhttp` mirrors `iamgin`** (JWT with optional cookie, Casbin with a resolver), writing errors through `stdres.WriteError`.
- **Response wrapper keeps streaming features:** the status-capturing writer forwards `Flush`, `Hijack` and `Unwrap`, because `auto_code_os` serves WebSocket and SSE.

## Trade-offs
- Moving shared logic into `httpmw` touches working Gin code; mitigated by the existing `httpserver` tests (CORS table, request-ID table).
- `RemoteAddr` is the client IP unless a RealIP middleware runs first (Chi's `RealIP`, or a reverse proxy setting it); Gin's trusted-proxy handling has no neutral equivalent. Documented.
- A panic in a handler is recovered and answered with a 500 envelope; `http.ErrAbortHandler` is swallowed instead of re-panicked because library code may not call `panic` (CS-3), so the connection is not aborted.

## Non-goals
- Migrating `auto_code_os` to these middleware in this task (done separately, after a release).
- Rate limiting, tracing and the project-specific `InjectClaimsFromJWT`/`RequireRole` of `auto_code_os`.
- A Chi route-pattern based Casbin resolver (callers pass a resolver).

## Out of Scope
- `monitor/prom` HTTP middleware for `net/http`.
- gRPC server.

## Impact
### Components
`httpmw` (new), `iam/iamhttp` (new), `httpserver` (internal refactor), `internal/archtest`, docs.
### Files
- Create: `httpmw/{doc.go,requestid.go,recovery.go,logger.go,cors.go,*_test.go,README.md}`, `iam/iamhttp/{doc.go,iamhttp.go,option.go,iamhttp_test.go,README.md}`
- Modify: `httpserver/middleware.go` (use `httpmw`), `internal/archtest/archtest_test.go` (rank for `httpmw`), `README.md`, `CHANGELOG.md`, `docs/ai/{PROFILE,DOMAIN}.md`
### Public API
Added only: `httpmw.{Middleware,RequestID,Recovery,RequestLogger,CORS,CORSConfig,ValidRequestID,NewRequestID}`, `iamhttp.{JWT,Casbin,WithCookie,Resolver,DefaultResolver,IdentityFromRequest}`. `httpserver` exports are unchanged.
### Migration
None for existing code.
### Backward Compatibility
No breaking change; ships as v0.4.0 (additive).
