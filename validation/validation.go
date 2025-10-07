// Package validation provides advanced validation constructors and rules for complex validation scenarios.
// It offers type-safe validation with generic support and integration with the govalin validation system.
package validation

import (
	"net/http"

	"github.com/pkkummermo/govalin/internal/validation"
)

// Validation types exposed without problematic generic type aliases

// NewStringValidator provides type-safe string validation.
func NewStringValidator() *validation.Validator[string] {
	return validation.Validate[string]()
}

// NewIntValidator provides type-safe integer validation.
func NewIntValidator() *validation.Validator[int] {
	return validation.Validate[int]()
}

// NewStructValidator provides validation for struct fields.
func NewStructValidator() *validation.StructValidator {
	return validation.ValidateStruct()
}

// Validation rule constructors

// Required validates that a string is not empty.
func Required() validation.Rule[string] {
	return validation.Required()
}

// MinLength validates minimum string length.
func MinLength(minimum int) validation.Rule[string] {
	return validation.MinLength(minimum)
}

// MaxLength validates maximum string length.
func MaxLength(maximum int) validation.Rule[string] {
	return validation.MaxLength(maximum)
}

// Email validates email format (simple validation).
func Email() validation.Rule[string] {
	return validation.Email()
}

// Min validates minimum integer value.
func Min(minimum int) validation.Rule[int] {
	return validation.Min(minimum)
}

// Max validates maximum integer value.
func Max(maximum int) validation.Rule[int] {
	return validation.Max(maximum)
}

// Range validates integer is within range.
func Range(minimum, maximum int) validation.Rule[int] {
	return validation.Range(minimum, maximum)
}

// CustomString allows defining custom validation logic for strings.
func CustomString(fn func(string) bool, message string) validation.Rule[string] {
	return validation.Custom(fn, message)
}

// CustomInt allows defining custom validation logic for integers.
func CustomInt(fn func(int) bool, message string) validation.Rule[int] {
	return validation.Custom(fn, message)
}

// Validate creates a validator for any type T - use with caution as it may cause build issues on older Go versions.
func Validate[T any]() *validation.Validator[T] {
	return validation.Validate[T]()
}

// Custom creates a custom validation rule for any type T - use with caution.
func Custom[T any](fn func(T) bool, message string) validation.Rule[T] {
	return validation.Custom(fn, message)
}

// TypedValidator provides a curryable typed validator.
type TypedValidator[T any] struct {
	validator interface {
		AddRule(func(interface{}) error)
		Get() error
	}
	rules []func(T) (bool, string)
}

// WithTyped creates a curryable typed validator for type-safe custom validation.
func WithTyped[T any, V interface {
	AddRule(func(interface{}) error)
	Get() error
}](v V) *TypedValidator[T] {
	return &TypedValidator[T]{
		validator: v,
		rules:     make([]func(T) (bool, string), 0),
	}
}

// Custom adds a type-safe custom validation rule that can be chained.
func (tv *TypedValidator[T]) Custom(validatorFn func(T) bool, message string) *TypedValidator[T] {
	tv.rules = append(tv.rules, func(data T) (bool, string) {
		return validatorFn(data), message
	})
	return tv
}

// Get executes all validation rules and returns any validation error.
func (tv *TypedValidator[T]) Get() error {
	// Add all accumulated rules to the underlying validator
	for _, rule := range tv.rules {
		validatorFn := rule // Capture the rule in closure
		tv.validator.AddRule(func(data interface{}) error {
			typedData, ok := data.(*T)
			if !ok {
				return validation.NewError(validation.NewErrorResponse(
					http.StatusBadRequest,
					validation.NewParameterErrorDetail("body", "Type assertion failed"),
				))
			}
			if valid, message := validatorFn(*typedData); !valid {
				return validation.NewError(validation.NewErrorResponse(
					http.StatusBadRequest,
					validation.NewParameterErrorDetail("body", message),
				))
			}
			return nil
		})
	}

	// Execute the underlying validator
	return tv.validator.Get()
}

// WithTypedCustom adds a type-safe custom validation rule for the entire body using a helper function
// This function works with any type that has an AddRule method
// Deprecated: Use WithTyped().Custom(...).Get() for curryable validation.
func WithTypedCustom[T any, V interface{ AddRule(func(interface{}) error) }](
	v V,
	validatorFn func(T) bool,
	message string,
) V {
	v.AddRule(func(data interface{}) error {
		typedData, ok := data.(*T)
		if !ok {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail("body", "Type assertion failed"),
			))
		}
		if !validatorFn(*typedData) {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail("body", message),
			))
		}
		return nil
	})
	return v
}
