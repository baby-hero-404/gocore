# Package `stderr`

`stderr` defines typed, classified error codes, field-level error details, and deterministic HTTP status mapping across all `gocore` boundaries.

## Features

- **Classified Error Codes (`Code`)**:
  - `1000 - 1999`: Validation & Bad Request errors (HTTP 400, 422)
  - `4000 - 4099`: Authentication & Authorization errors (HTTP 401, 403)
  - `4100 - 4199`: Resource state errors (HTTP 404, 409)
  - `5000 - 5099`: Internal Server errors (HTTP 500)
  - `5100 - 5199`: Infrastructure & Network errors (HTTP 502, 503, 504)
- **Field-level Details (`FieldError`)**: Supports form and request validation failures (`Field`, `Message`, `Tag`), compatible with `kdc_bike_BE`'s `Causes`.
- **i18n Translation Support**: `WithKey(key, args...)` allows error messages to be dynamically translated at the API boundary without polluting business logic.
- **Go 1.13+ standard wrapping**: Supports `Unwrap()`, `errors.Is()`, and `errors.As()`.

## Usage

```go
import "github.com/baby-hero-404/gocore/stderr"

// Simple error
err := stderr.NewNotFound("user not found")

// Validation error with field details
valErr := stderr.NewValidation("invalid request input").
    WithDetails(
        stderr.FieldError{Field: "email", Message: "must be a valid email address"},
        stderr.FieldError{Field: "age", Message: "must be at least 18"},
    )

// Wrapping underlying errors
if err := db.Find(&user).Error; err != nil {
    return stderr.Wrap(stderr.CodeDatabaseError, err, "database query failed")
}
```
