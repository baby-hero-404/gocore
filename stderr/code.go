package stderr

import "net/http"

// Code represents a standardized error code categorized into ranges.
// 1000 - 1999: Validation & Client Bad Request errors (HTTP 400, 422)
// 4000 - 4099: Authentication & Authorization errors (HTTP 401, 403)
// 4100 - 4199: Resource state errors (HTTP 404, 409)
// 5000 - 5099: Internal Server errors (HTTP 500)
// 5100 - 5199: Infrastructure, Upstream & Concurrency errors (HTTP 502, 503, 504)
type Code int

const (
	// CodeOK indicates success / no error.
	CodeOK Code = 0

	// Validation and Client Request errors (1000 - 1999)

	// CodeInvalidParam indicates a missing or invalid request parameter.
	CodeInvalidParam Code = 1000 // HTTP 400
	// CodeValidationFailed indicates the payload failed field-level validation.
	CodeValidationFailed Code = 1001 // HTTP 422
	// CodeMalformedBody indicates the request body could not be parsed.
	CodeMalformedBody Code = 1002 // HTTP 400
	// CodeBadRequest is a generic client error.
	CodeBadRequest Code = 1003 // HTTP 400

	// Authentication & Authorization errors (4000 - 4099)

	// CodeUnauthorized indicates missing or invalid credentials.
	CodeUnauthorized Code = 4001 // HTTP 401
	// CodeForbidden indicates the caller lacks permission.
	CodeForbidden Code = 4003 // HTTP 403
	// CodeTooManyRequests indicates the caller exceeded a rate limit (e.g. login attempts).
	CodeTooManyRequests Code = 4029 // HTTP 429

	// Resource errors (4100 - 4199)

	// CodeNotFound indicates the requested resource does not exist.
	CodeNotFound Code = 4104 // HTTP 404
	// CodeConflict indicates a state conflict with the current resource.
	CodeConflict Code = 4109 // HTTP 409
	// CodeAlreadyExists indicates the resource already exists.
	CodeAlreadyExists Code = 4110 // HTTP 409

	// Internal Server errors (5000 - 5099)

	// CodeInternal is an unexpected server failure.
	CodeInternal Code = 5000 // HTTP 500
	// CodeDatabaseError is a failed database operation.
	CodeDatabaseError Code = 5001 // HTTP 500
	// CodeCacheError is a failed cache operation.
	CodeCacheError Code = 5002 // HTTP 500

	// Infrastructure & Concurrency errors (5100 - 5199)

	// CodeUnavailable indicates a dependency is temporarily unavailable.
	CodeUnavailable Code = 5100 // HTTP 503
	// CodeTimeout indicates an operation exceeded its deadline.
	CodeTimeout Code = 5101 // HTTP 504
	// CodeLockHeld indicates a distributed lock is held by another owner.
	CodeLockHeld Code = 5102 // HTTP 409 (Conflict on lock acquisition)
)

// HTTPStatus maps the Code to an HTTP status code deterministically.
func (c Code) HTTPStatus() int {
	switch {
	case c == CodeOK:
		return http.StatusOK
	case c == CodeValidationFailed:
		return http.StatusUnprocessableEntity
	case c >= 1000 && c < 2000:
		return http.StatusBadRequest
	case c == CodeUnauthorized:
		return http.StatusUnauthorized
	case c == CodeForbidden:
		return http.StatusForbidden
	case c == CodeTooManyRequests:
		return http.StatusTooManyRequests
	case c == CodeNotFound:
		return http.StatusNotFound
	case c == CodeConflict || c == CodeAlreadyExists || c == CodeLockHeld:
		return http.StatusConflict
	case c == CodeUnavailable:
		return http.StatusServiceUnavailable
	case c == CodeTimeout:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

// String returns a human-readable identifier for the error code.
func (c Code) String() string {
	switch c {
	case CodeOK:
		return "OK"
	case CodeInvalidParam:
		return "INVALID_PARAM"
	case CodeValidationFailed:
		return "VALIDATION_FAILED"
	case CodeMalformedBody:
		return "MALFORMED_BODY"
	case CodeBadRequest:
		return "BAD_REQUEST"
	case CodeUnauthorized:
		return "UNAUTHORIZED"
	case CodeForbidden:
		return "FORBIDDEN"
	case CodeTooManyRequests:
		return "TOO_MANY_REQUESTS"
	case CodeNotFound:
		return "NOT_FOUND"
	case CodeConflict:
		return "CONFLICT"
	case CodeAlreadyExists:
		return "ALREADY_EXISTS"
	case CodeInternal:
		return "INTERNAL_SERVER_ERROR"
	case CodeDatabaseError:
		return "DATABASE_ERROR"
	case CodeCacheError:
		return "CACHE_ERROR"
	case CodeUnavailable:
		return "SERVICE_UNAVAILABLE"
	case CodeTimeout:
		return "GATEWAY_TIMEOUT"
	case CodeLockHeld:
		return "LOCK_HELD"
	default:
		return "UNKNOWN_ERROR"
	}
}
