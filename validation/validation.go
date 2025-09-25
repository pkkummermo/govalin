package validation

import "github.com/pkkummermo/govalin/internal/validation"

// Validation types exposed without problematic generic type aliases

// NewStringValidator provides type-safe string validation
func NewStringValidator() *validation.Validator[string] {
	return validation.Validate[string]()
}

// NewIntValidator provides type-safe integer validation
func NewIntValidator() *validation.Validator[int] {
	return validation.Validate[int]()
}

// NewStructValidator provides validation for struct fields
func NewStructValidator() *validation.StructValidator {
	return validation.ValidateStruct()
}

// Validation rule constructors

// Required validates that a string is not empty
func Required() validation.ValidationRule[string] {
	return validation.Required()
}

// MinLength validates minimum string length
func MinLength(min int) validation.ValidationRule[string] {
	return validation.MinLength(min)
}

// MaxLength validates maximum string length
func MaxLength(max int) validation.ValidationRule[string] {
	return validation.MaxLength(max)
}

// Email validates email format (simple validation)
func Email() validation.ValidationRule[string] {
	return validation.Email()
}

// Min validates minimum integer value
func Min(min int) validation.ValidationRule[int] {
	return validation.Min(min)
}

// Max validates maximum integer value
func Max(max int) validation.ValidationRule[int] {
	return validation.Max(max)
}

// Range validates integer is within range
func Range(min, max int) validation.ValidationRule[int] {
	return validation.Range(min, max)
}

// CustomString allows defining custom validation logic for strings
func CustomString(fn func(string) bool, message string) validation.ValidationRule[string] {
	return validation.Custom(fn, message)
}

// CustomInt allows defining custom validation logic for integers
func CustomInt(fn func(int) bool, message string) validation.ValidationRule[int] {
	return validation.Custom(fn, message)
}

// Validate creates a validator for any type T - use with caution as it may cause build issues on older Go versions
func Validate[T any]() *validation.Validator[T] {
	return validation.Validate[T]()
}

// Custom creates a custom validation rule for any type T - use with caution
func Custom[T any](fn func(T) bool, message string) validation.ValidationRule[T] {
	return validation.Custom(fn, message)
}
