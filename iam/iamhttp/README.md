# Package `iamhttp`

JWT authentication and Casbin authorization middleware for `net/http` routers (Chi, `http.ServeMux`, ...). It is the framework-neutral counterpart of `iamgin`; errors are written with the standard `stdres` envelope.

```go
cfg := iam.JWTConfig{Secret: secret, TokenType: "access"}

r := chi.NewRouter()
r.Group(func(r chi.Router) {
    r.Use(iamhttp.JWT(iam.NewJWTValidator(cfg)))     // 401 envelope when the token is missing/invalid
    r.Use(iamhttp.Casbin(enforcer))                  // 403 when the policy denies, 401 without identity

    r.Get("/projects/{id}", func(w http.ResponseWriter, req *http.Request) {
        id, _ := iamhttp.IdentityFromRequest(req)    // or iam.IdentityFromContext(req.Context())
        _ = id
    })
})
```

* `JWT(validator, opts...)`: reads `Authorization: Bearer <token>`; `WithCookie("sid")` makes that cookie take precedence (see the CSRF note below).
* `Casbin(enforcer, resolver...)`: the default resolver uses the identity's role (or id), the URL path and the method; pass a `Resolver` to map routes differently (for example from a Chi route pattern).
* The identity is stored in the request context, so use cases never need the HTTP types.

> **CSRF:** cookie authentication needs CSRF protection that gocore does not provide (SameSite cookies, anti-CSRF tokens or Origin/Sec-Fetch-Site checks).
