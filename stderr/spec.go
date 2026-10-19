package stderr

// ErrorInterface defines the common methods provided by gocore errors.
type ErrorInterface interface {
	error
	HTTPStatus() int
	StatusCode() Code
	ErrorMessage() string
}

// StatusCode returns the classified Code for this error.
func (e *Error) StatusCode() Code {
	if e == nil {
		return CodeOK
	}
	return e.Code
}

// ErrorMessage returns the user-facing message.
func (e *Error) ErrorMessage() string {
	if e == nil {
		return ""
	}
	return e.Message
}
