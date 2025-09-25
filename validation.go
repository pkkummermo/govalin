package govalin

import "github.com/pkkummermo/govalin/internal/validation"

// Validator provides type-safe validation for various data types
type Validator[T any] = validation.Validator[T]

// ValidationRule represents a single validation rule
type ValidationRule[T any] = validation.ValidationRule[T]

// StructValidator provides validation for struct fields
type StructValidator = validation.StructValidator

// ValidationError represents a validation error
type ValidationError = validation.Error

// Validate creates a new type-safe validator for the specified type T
// 
// For simpler usage, consider using the Call validation methods:
//   - call.ValidatedQueryParam("name", validator) 
//   - call.ValidatedPathParam("id", validator)
//   - call.ValidatedFormParam("email", validator)
//   - call.ValidatedBody(&user, structValidator)
//
// Example usage:
//   validator := govalin.Validate[string]().Rule(govalin.Required()).Rule(govalin.MinLength(3))
//   if err := validator.Validate(userInput, "username"); err != nil { ... }
func Validate[T any]() *Validator[T] {
	return validation.Validate[T]()
}

// ValidateStruct creates a new struct validator
//
// For simpler usage, consider using call.ValidatedBody(&target, validator)
//
// Example usage:
//   validator := govalin.ValidateStruct().Field("Name", func(v interface{}) *govalin.ValidationError {
//       return govalin.Validate[string]().Rule(govalin.Required()).Validate(v.(string), "Name")
//   })
func ValidateStruct() *StructValidator {
	return validation.ValidateStruct()
}

// String validation rules

// Required validates that a string is not empty
func Required() ValidationRule[string] {
	return validation.Required()
}

// MinLength validates minimum string length
func MinLength(min int) ValidationRule[string] {
	return validation.MinLength(min)
}

// MaxLength validates maximum string length
func MaxLength(max int) ValidationRule[string] {
	return validation.MaxLength(max)
}

// Email validates email format (simple validation)
func Email() ValidationRule[string] {
	return validation.Email()
}

// Integer validation rules

// Min validates minimum integer value
func Min(min int) ValidationRule[int] {
	return validation.Min(min)
}

// Max validates maximum integer value
func Max(max int) ValidationRule[int] {
	return validation.Max(max)
}

// Range validates integer is within range
func Range(min, max int) ValidationRule[int] {
	return validation.Range(min, max)
}

// Generic validation rules

// Custom allows defining custom validation logic
func Custom[T any](fn func(T) bool, message string) ValidationRule[T] {
	return validation.Custom(fn, message)
}

// Helper functions

// ValidateStringAsInt validates a string can be converted to int and applies int validation
func ValidateStringAsInt(value string, fieldName string, validator *Validator[int]) *ValidationError {
	return validation.ValidateStringAsInt(value, fieldName, validator)
}