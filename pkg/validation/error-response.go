package validation

import (
	"encoding/json"
	"net/http"
	"strconv"
)

var defaultErrorMessages = map[int]string{
	400: "Bad request",
	401: "Unauthorized",
	403: "Forbidden",
	404: "Not found",
	405: "Method not allowed",
	409: "Conflict",
	500: "Server error",
	501: "Not implemented",
	502: "Bad gateway",
	503: "Service unavailable",
	504: "Gateway timeout",
}

// ErrorResponse is a generic response type for errors in HTTP requests.
type ErrorResponse struct {
	Title   string        `json:"title"`
	Detail  string        `json:"detail,omitempty"`
	Status  int           `json:"status"`
	Type    string        `json:"type"`
	Details []ErrorDetail `json:"details,omitempty"`
}

// MarshalJSON marshals a JSON string from the ErrorResponse.
func (errorResponse *ErrorResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(*errorResponse)
}

// WriteHTTPError writes error to a writer as JSON.
func (errorResponse *ErrorResponse) WriteHTTPError(writer http.ResponseWriter) {
	jsonBytes, _ := errorResponse.MarshalJSON()
	if writer.Header().Get("Content-Type") != "application/json" {
		writer.Header().Add("Content-Type", "application/json")
	}
	writer.WriteHeader(errorResponse.Status)
	_, _ = writer.Write(jsonBytes)
}

func getErrorTypeURLForStatusCode(statusCode int) string {
	return "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/" + strconv.Itoa(statusCode)
}

// NewErrorResponse returns a new ErrorResponse based on given status code and details.
func NewErrorResponse(statusCode int, details ...ErrorDetail) *ErrorResponse {
	errorTitle := "Unknown error"

	if val, ok := defaultErrorMessages[statusCode]; ok {
		errorTitle = val
	}

	return &ErrorResponse{
		Title:   errorTitle,
		Status:  statusCode,
		Type:    getErrorTypeURLForStatusCode(statusCode),
		Details: details,
	}
}
