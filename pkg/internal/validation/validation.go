package validation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"
)

// GetUnmarshalError returns a validation Error based on given UnmarshalTypeError.
func GetUnmarshalError(err *json.UnmarshalTypeError) *Error {
	// If the type is nested we still only want the type
	expectedTypeChunk := strings.Split(err.Type.String(), ".")
	expectedType := expectedTypeChunk[len(expectedTypeChunk)-1]

	return NewError(
		NewErrorResponse(
			http.StatusBadRequest,
			NewParameterErrorDetail(
				lowerFirst(err.Struct)+"."+err.Field,
				fmt.Sprintf("Incorrect type. '%s' is not of type '%s'",
					err.Value,
					expectedType,
				),
			),
		),
	)
}

func lowerFirst(s string) string {
	if s == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToLower(r)) + s[n:]
}
