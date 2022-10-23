package validation

import (
	"fmt"
)

// Error is an error who occured when trying to validate an entity
// It contains an ErrorResponse which can be used to write an http error message.
type Error struct {
	ErrorResponse *ErrorResponse
}

func (validationError *Error) Error() string {
	return fmt.Sprintf(
		"%d - %s - %s",
		validationError.ErrorResponse.Status,
		validationError.ErrorResponse.Title,
		validationError.ErrorResponse.Details,
	)
}

// NewError returns an error based on a validation error response.
func NewError(errorResponse *ErrorResponse) *Error {
	return &Error{
		ErrorResponse: errorResponse,
	}
}
