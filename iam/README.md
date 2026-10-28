# Package `iam`

`iam` provides unified authentication and authorization primitives for Go services, composing JWT (`jwt/v5`), Casbin RBAC/ABAC (`casbin/v3`), OAuth2, and Firebase ID token verification.

## Features

- **Decoupled Architecture**: `TokenValidator` interface abstracts token verification across JWT, OAuth2, and Firebase.
- **Context-Aware Identity**: `Identity` struct (`ID`, `Email`, `Roles`, `Metadata`) automatically injected into both `gin.Context` and Go standard `context.Context`.
- **Casbin RBAC/ABAC Middleware**: Enforces policies at the route boundary, automatically returning HTTP 403 on denial.
- **Fail-Fast & Secure**: Rejects missing or expired tokens with standard HTTP 401 error envelopes before reaching application handlers.

## Usage

### 1. JWT Authentication

```go
package main

import (
    "github.com/baby-hero-404/gocore/iam"
    "github.com/baby-hero-404/gocore/iam/iamgin"
    "github.com/gin-gonic/gin"
)

func main() {
    validator := iam.NewJWTValidator(iam.JWTConfig{Secret: "supersecretkey"})

    r := gin.Default()
    protected := r.Group("/api", iamgin.JWTMiddleware(validator))
    protected.GET("/me", func(c *gin.Context) {
        id, _ := iamgin.IdentityFromGin(c)
        c.JSON(200, id)
    })
}
```

**Issuing tokens.** `iam.NewJWTSigner(cfg)` issues HMAC tokens that a `JWTValidator` with the same `JWTConfig` accepts. Extra claims (e.g. a session ID) end up in `Identity.Metadata`:

```go
signer, _ := iam.NewJWTSigner(iam.JWTConfig{Secret: "supersecretkey"})
token, _ := signer.Sign(userID, 30*24*time.Hour, map[string]any{"sid": sessionID})
```

**Token from a cookie.** For web apps that keep the token in an HttpOnly cookie, `iamgin.WithCookie(name)` reads the cookie first and falls back to the `Authorization: Bearer` header:

```go
protected := r.Group("/api", iamgin.JWTMiddleware(validator, iamgin.WithCookie("session")))
```

> ⚠️ **Security Note (CSRF Protection):** `iamgin` extracts and verifies authentication tokens from cookies, but does **not** provide CSRF (Cross-Site Request Forgery) protection. Consuming services that authenticate via cookies must implement their own CSRF mitigation (e.g., `SameSite=Lax` or `SameSite=Strict` cookie policies, anti-CSRF synchronizer tokens / headers, or `Origin`/`Sec-Fetch-Site` request verification).

### 2. Casbin RBAC Authorization

```go
enforcer, _ := casbin.NewEnforcer("model.conf", "policy.csv")

// Apply Casbin policy enforcement on authenticated routes
adminGroup := protected.Group("/admin", iamgin.CasbinMiddleware(enforcer))
adminGroup.GET("/users", listUsersHandler)
```

> `iam` is framework-free (`BearerToken`, `Authorize`, validators). Gin glue is in `iam/iamgin`; Firebase is in `iam/firebase` so the Firebase SDK is only pulled in when used.

## Custom claims and token types

`JWTConfig` can adapt to an existing token format and keep different kinds of token apart:

```go
access := iam.JWTConfig{Secret: secret, IDClaim: "userId", RolesClaim: "role", TokenType: "access"}
refresh := iam.JWTConfig{Secret: secret, IDClaim: "userId", RolesClaim: "role", TokenType: "refresh"}
```

* `IDClaim` / `RolesClaim` choose the claim names; empty keeps the default lookup (`sub` then `id`, `roles` then `role`).
* `TokenType` makes `JWTSigner` write a `typ` claim and `JWTValidator` reject tokens with another `typ`. **If a service issues more than one kind of token, set it on every signer and validator**; without it an access token and a refresh token signed with the same secret are interchangeable.
