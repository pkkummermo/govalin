package validation

import (
	"encoding/json"

	"github.com/pkkummermo/govalin/pkg/input"
)

// ErrorDetail details an error.
type ErrorDetail struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

// MarshalJSON marshals a JSON string from the ErrorResponse.
func (errorDetail *ErrorDetail) MarshalJSON() ([]byte, error) {
	return json.Marshal(*errorDetail)
}

// NewParameterErrorDetail returns an error struct containing which field and the reason behind the error.
func NewParameterErrorDetail(field, reason string) ErrorDetail {
	return ErrorDetail{
		Field:  field,
		Reason: input.NormalizeStringInput(reason),
	}
}
