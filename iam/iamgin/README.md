# Package `iamgin`

`iamgin` adapts the `iam` authentication and authorization primitives to the [Gin](https://github.com/gin-gonic/gin) web framework.

## Features

- **`JWTMiddleware`**: Validates JWT/bearer tokens and injects `iam.Identity` into `gin.Context` and `http.Request.Context()`.
- **`WithCookie(name)`**: Option to extract the token from a specified cookie name (with fallback to `Authorization: Bearer`).
- **`CasbinMiddleware`**: Enforces RBAC/ABAC authorization rules with flexible `CasbinResolver`.
- **`IdentityFromGin`**: Helper to extract `iam.Identity` from `gin.Context`.

## Usage

### JWT Authentication

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

    // 1. Bearer token authentication (default)
    api := r.Group("/api", iamgin.JWTMiddleware(validator))
    api.GET("/profile", func(c *gin.Context) {
        id, ok := iamgin.IdentityFromGin(c)
        c.JSON(200, id)
    })

    // 2. Cookie-based authentication
    web := r.Group("/web", iamgin.JWTMiddleware(validator, iamgin.WithCookie("session_token")))
    web.GET("/dashboard", dashboardHandler)
}
```

---

## ⚠️ Security Notice: Cookie Authentication & CSRF

When using `iamgin.WithCookie(name)`, the middleware reads the authentication token from incoming HTTP cookies.

### The CSRF Risk
Browsers automatically send cookies with cross-site requests (e.g. via `<form>`, `<img>`, or `fetch` from malicious origins). Because `gocore` is an infrastructure chassis and does not manage session cookies or render frontend forms, **it does not include built-in CSRF mitigation**.

### Consuming Service Requirements
Any microservice or web application using cookie-based authentication **MUST** implement CSRF defenses:

1. **Cookie Attributes**:
   - Set `SameSite=Lax` (minimum) or `SameSite=Strict` (recommended for state-changing endpoints).
   - Always set `HttpOnly` and `Secure`.
2. **Anti-CSRF Tokens**:
   - Implement Double Submit Cookie or Synchronizer Token Pattern (require custom header such as `X-CSRF-Token` on state-changing `POST`, `PUT`, `DELETE` requests).
3. **Origin / Referer Verification**:
   - Validate that the `Origin` or `Referer` matches the application's trusted domain, or enforce modern `Sec-Fetch-Site: same-origin` checks.
