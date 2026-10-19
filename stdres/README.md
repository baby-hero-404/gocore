# Package `stdres`

`stdres` standardizes JSON API response envelopes, error code mapping, field validation error responses, and optional i18n translation for Go microservices.

## Features

- **Unified Envelope**: All API responses conform to `{ "code": ..., "message": "...", "key": "...", "args": [...], "data": ..., "details": ..., "request_id": "..." }`. `key`/`args` let clients translate by code; `request_id` matches the `X-Request-ID` header and `stdlog` output.`.
- **Automatic Error & Status Mapping**: `FromError(ctx, err)` automatically converts any error (including `stderr.Error`) into the proper envelope and HTTP status code.
- **i18n Translation Support**: Plug in any `Translator` implementation (`Translate(ctx, key, args...)`) to translate error and success messages dynamically based on locale.
- **Validation Error Handling**: `ValidationError(target, err)` (framework-agnostic) and `stdgin.HandleValidationErrors(c, target, err)` translate validator errors (e.g. `go-playground/validator`) and JSON unmarshal errors into clean field-level error details matching struct `json` tags.
- **net/http Helpers** (chi, stdlib): `stdres.WriteOK`, `WriteCreated`, `WriteNoContent`, `WriteJSON`, `WriteError`, `WriteValidationError`. No Gin dependency.
- **Gin Helpers** (`stdres/stdgin`): `stdgin.OK`, `Created`, `NoContent`, `JSON`, `Error`, `HandleValidationErrors`. Only importing `stdgin` pulls in Gin.

## Usage

```go
package main

import (
    "github.com/baby-hero-404/gocore/stderr"
    "github.com/baby-hero-404/gocore/stdres/stdgin"
    "github.com/gin-gonic/gin"
)

type CreateUserRequest struct {
    Email string `json:"email" binding:"required,email"`
}

func createUser(c *gin.Context) {
    var req CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        stdgin.HandleValidationErrors(c, req, err)
        return
    }

    user, err := userService.Create(c.Request.Context(), req.Email)
    if err != nil {
        stdgin.Error(c, err)
        return
    }

    stdgin.Created(c, user)
}
```

### net/http (chi)

```go
func createUser(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        stdres.WriteValidationError(w, r, req, err)
        return
    }
    user, err := userService.Create(r.Context(), req.Email)
    if err != nil {
        stdres.WriteError(w, r, err)
        return
    }
    stdres.WriteCreated(w, r, user)
}
```
