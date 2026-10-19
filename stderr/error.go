package stderr

import (
	"errors"
	"fmt"
)

// FieldError represents a field-level error detail (e.g. invalid form field or missing JSON property).
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Tag     string `json:"tag,omitempty"`
}

// Error is a structured, typed error used across gocore boundaries.
type Error struct {
	Code    Code         `json:"code"`
	Message string       `json:"message"`
	Key     string       `json:"-"` // Optional i18n translation key
	Args    []any        `json:"-"` // Optional arguments for translation
	Details []FieldError `json:"details,omitempty"`
	Err     error        `json:"-"` // Wrapped root-cause error
}

// Error implements the standard error interface.
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Err != nil {
		return fmt.Sprintf("[%s:%d] %s (cause: %v)", e.Code.String(), e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s:%d] %s", e.Code.String(), e.Code, e.Message)
}

// Unwrap exposes the underlying error for errors.Is and errors.As.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Is implements error comparison. It returns true if target has the same Code.
func (e *Error) Is(target error) bool {
	if e == nil {
		return target == nil
	}
	var t *Error
	if errors.As(target, &t) {
		return e.Code == t.Code
	}
	return false
}

// HTTPStatus returns the mapped HTTP status code for this error.
func (e *Error) HTTPStatus() int {
	if e == nil {
		return CodeOK.HTTPStatus()
	}
	return e.Code.HTTPStatus()
}

// WithDetails attaches field errors to the Error and returns the pointer for chaining.
func (e *Error) WithDetails(details ...FieldError) *Error {
	if e == nil {
		return nil
	}
	e.Details = append(e.Details, details...)
	return e
}

// WithKey sets the i18n key and optional format arguments.
func (e *Error) WithKey(key string, args ...any) *Error {
	if e == nil {
		return nil
	}
	e.Key = key
	e.Args = args
	return e
}

// WithCause attaches an underlying root cause error to the Error.
func (e *Error) WithCause(err error) *Error {
	if e == nil {
		return nil
	}
	e.Err = err
	return e
}

// New creates a new typed Error with a code and message.
func New(code Code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Newf creates a new Error with a formatted message.
func Newf(code Code, format string, args ...any) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// Wrap wraps an existing error with a classified Code and optional message override.
// Returns nil if err is nil.
func Wrap(code Code, err error, message ...string) *Error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}
	return &Error{
		Code:    code,
		Message: msg,
		Err:     err,
	}
}

// Convenience constructors

// NewBadRequest creates a 400 Bad Request error.
func NewBadRequest(message string) *Error {
	return New(CodeBadRequest, message)
}

// NewValidation creates a 422 Validation error with optional field details.
func NewValidation(message string, details ...FieldError) *Error {
	return &Error{
		Code:    CodeValidationFailed,
		Message: message,
		Details: details,
	}
}

// NewNotFound creates a 404 Not Found error.
func NewNotFound(message string) *Error {
	return New(CodeNotFound, message)
}

// NewUnauthorized creates a 401 Unauthorized error.
func NewUnauthorized(message string) *Error {
	return New(CodeUnauthorized, message)
}

// NewForbidden creates a 403 Forbidden error.
func NewForbidden(message string) *Error {
	return New(CodeForbidden, message)
}

// NewConflict creates a 409 Conflict error.
func NewConflict(message string) *Error {
	return New(CodeConflict, message)
}

// NewInternal creates a 500 Internal Server Error with an optional root cause.
func NewInternal(message string, cause ...error) *Error {
	e := New(CodeInternal, message)
	if len(cause) > 0 && cause[0] != nil {
		e.Err = cause[0]
	}
	return e
}

// NewUnavailable creates a 503 Service Unavailable error.
func NewUnavailable(message string, cause ...error) *Error {
	e := New(CodeUnavailable, message)
	if len(cause) > 0 && cause[0] != nil {
		e.Err = cause[0]
	}
	return e
}
