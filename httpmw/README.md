# Package `httpmw`

Framework-neutral `net/http` middleware: request ID, panic recovery, request logging and CORS.
A `Middleware` is `func(http.Handler) http.Handler`, so it works with **Chi** (`r.Use`),
`http.ServeMux` (wrap the mux) and any router built on `net/http`. The Gin stack in `httpserver`
shares the CORS and request-ID rules with this package, so both behave the same.

```go
r := chi.NewRouter()
r.Use(
    httpmw.RequestID(),
    httpmw.Recovery(),
    httpmw.RequestLogger(),
    httpmw.CORS(httpmw.CORSConfig{
        AllowOrigins:     []string{"https://app.example.com"},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Authorization", "Content-Type"},
        AllowCredentials: true,
    }),
)
```

| Middleware | Behaviour |
|---|---|
| `RequestID()` | Keeps a valid `X-Request-ID` (≤128 chars from `A-Za-z0-9-_.:/+=@`), otherwise generates a UUID; sets the response header and the request context (`stdlog.RequestIDFromContext`). |
| `Recovery()` | A panic becomes a 500 `stdres` envelope (`"internal server error"`); the panic and stack are logged once with the request ID. Nothing is written if the response already started. |
| `RequestLogger()` | One line per request (method, path+query, status, latency, client IP) at INFO/WARN/ERROR by status. Keeps `Flush`, `Hijack` (WebSocket) and `http.ResponseController` working. |
| `CORS(cfg)` | Listed origins are echoed (with `Vary: Origin`, credentials if enabled); `*` answers `*` and **never** carries credentials; unlisted origins and requests without `Origin` get no CORS headers; `OPTIONS` answers 204. |

Notes:
* Order: put `RequestID` first so later middleware and handlers see the ID; `Recovery` before `RequestLogger` so a panicking request is still logged with its 500.
* The logged client IP is the connection's remote address (the proxy, when you run behind one). To log the real client, resolve it from forwarding headers **only from trusted proxies** and pass it in: `httpmw.RequestLogger(httpmw.WithClientIP(func(r *http.Request) string { return chimw.GetClientIP(r.Context()) }))` after `r.Use(chimw.ClientIPFromXFF("10.0.0.0/8"))`. Do not use Chi's `middleware.RealIP` for this: it is deprecated because it copies `X-Forwarded-For`/`X-Real-IP` into `RemoteAddr` from any caller, so clients can choose the address that is logged and rate limited.
* `http.ErrAbortHandler` (what `httputil.ReverseProxy` panics with when an upstream body breaks) is honoured without panicking: `Recovery` closes the connection so clients see a truncated response as an error. Where the connection cannot be taken over (HTTP/2) it logs a warning and answers 500 if nothing was written.
