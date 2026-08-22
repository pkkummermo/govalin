// Package validation provides advanced validation constructors and rules for complex validation scenarios.
// It offers type-safe validation with generic support and integration with the govalin validation system.
package validation

import (
	"github.com/pkkummermo/govalin/internal/validation"
)

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
