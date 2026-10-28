# Expected Behavior: net/http adapters

## Scenario: Request ID
**When:** a request carries a valid `X-Request-ID`, or none, or an invalid one (too long, spaces, control characters)
**Then:** a valid ID is kept; otherwise a UUID replaces it; the response header and `stdlog.RequestIDFromContext(r.Context())` hold the same value

## Scenario: Panic recovery
**When:** a downstream handler panics
**Then:** the client gets a 500 `stdres` envelope with `"internal server error"` (no panic text); the panic and stack are logged once with the request ID

## Failure Scenario: nothing was written before the panic vs partial response
**When:** the handler panicked after writing part of the response
**Then:** recovery logs but does not write a second body (the status is already sent)

## Scenario: Request logging
**When:** a request completes with status 200, 404 or 503
**Then:** one log line at INFO, WARN or ERROR with method, path (+query), status, latency and client IP; the response behaves unchanged, including `Flush`, `Hijack` (WebSocket upgrade) and `http.ResponseController`

## Scenario: CORS decision
**When:** the request has an `Origin` header
**Then:** a listed origin is echoed with `Vary: Origin` (and credentials when enabled); `*` answers `*` and never credentials; an unlisted origin gets no CORS headers; a request without `Origin` gets none; `OPTIONS` is answered 204 without reaching the handler
**And:** the Gin `CORSMiddleware` produces the same headers for the same configuration (one shared implementation)

## Scenario: JWT middleware
**When:** a valid token is sent as `Authorization: Bearer` (or in the configured cookie)
**Then:** the handler sees the identity through `iam.IdentityFromContext(r.Context())` and `iamhttp.IdentityFromRequest(r)`

## Failure Scenario: JWT rejected
**When:** the token is missing, malformed, expired, of the wrong type or the validator errors
**Then:** 401 with the standard envelope and key `error.unauthenticated`; the handler is not called

## Scenario: Casbin middleware
**When:** an identity is present and the enforcer allows `(sub, obj, act)` from the resolver
**Then:** the handler runs

## Failure Scenario: Casbin rejected
**When:** no identity (401), enforcer nil (500), enforcer error (500), policy denies (403)
**Then:** the matching envelope, handler not called

## Scenario: Works with Chi
**When:** the middleware are installed with `chi.Router.Use` and the JWT middleware is scoped to a `Route`/`Group`
**Then:** requests behave as above and `chi.URLParam` still works in handlers

## Invariants
- `httpmw` and `iam/iamhttp` import no router and no Gin.
- Error bodies are produced only through `stdres`; unclassified errors never reach the client.
- Client-supplied request IDs are never echoed unvalidated.

## Constraints
- No new third-party dependency in gocore's `go.mod`.
- `make verify` stays green; new packages ≥ 80% coverage.
