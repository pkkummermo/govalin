package validation

import (
	"fmt"
	"net/http"
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

// Status returns the HTTP status the error carries, falling back to 400 for an
// error without a response.
func (validationError *Error) Status() int {
	if validationError.ErrorResponse == nil {
		return http.StatusBadRequest
	}

	return validationError.ErrorResponse.Status
}

// NewError returns an error based on a validation error response.
func NewError(errorResponse *ErrorResponse) *Error {
	return &Error{
		ErrorResponse: errorResponse,
	}
}
