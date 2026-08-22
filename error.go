package govalin

import "fmt"

// userError is a failure the caller of the API caused, answered with 400 by
// Call.Error.
type userError struct {
	originalError error
}

func (err *userError) Error() string {
	return fmt.Sprintf("user error: %v", err.originalError)
}

func newUserError(err error) error {
	return &userError{originalError: err}
}
