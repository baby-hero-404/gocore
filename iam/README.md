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

### 2. Casbin RBAC Authorization

```go
enforcer, _ := casbin.NewEnforcer("model.conf", "policy.csv")

// Apply Casbin policy enforcement on authenticated routes
adminGroup := protected.Group("/admin", iamgin.CasbinMiddleware(enforcer))
adminGroup.GET("/users", listUsersHandler)
```

> `iam` is framework-free (`BearerToken`, `Authorize`, validators). Gin glue is in `iam/iamgin`; Firebase is in `iam/firebase` so the Firebase SDK is only pulled in when used.
