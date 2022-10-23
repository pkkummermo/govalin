package govalin

import "fmt"

type govalinErrorType string

type govalinError struct {
	errorType     govalinErrorType
	originalError error
}

func (err *govalinError) Error() string {
	return fmt.Sprintf("Error type %s", err.errorType)
}

const (
	serverError govalinErrorType = "Server error"
	userError   govalinErrorType = "User error"
)

func newErrorFromType(errorType govalinErrorType, err error) error {
	return &govalinError{
		errorType:     errorType,
		originalError: err,
	}
}
