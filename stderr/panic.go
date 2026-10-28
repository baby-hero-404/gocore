package stderr

import (
	"errors"
	"fmt"
)

// FromPanic converts a recovered panic value into an internal *Error, keeping the panic as the
// cause. Transport-level recovery middleware (HTTP, gRPC, ...) shares this conversion.
func FromPanic(recovered any) *Error {
	var cause error
	if err, ok := recovered.(error); ok {
		cause = err
	} else {
		cause = errors.New(fmt.Sprint(recovered))
	}
	return NewInternal("internal server error", cause)
}
