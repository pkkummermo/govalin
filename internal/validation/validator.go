package validation

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

// ValidationRule represents a single validation rule.
type ValidationRule[T any] func(value T, fieldName string) *Error

// Validator provides type-safe validation for various data types.
type Validator[T any] struct {
	rules []ValidationRule[T]
}

// NewValidator creates a new type-safe validator.
func NewValidator[T any]() *Validator[T] {
	return &Validator[T]{
		rules: make([]ValidationRule[T], 0),
	}
}

// Validate creates a new type-safe validator for the specified type T
// Example usage:
//
//	validator := validation.Validate[string]().Rule(validation.Required()).Rule(validation.MinLength(3))
//	if err := validator.Validate(userInput, "username"); err != nil { ... }
func Validate[T any]() *Validator[T] {
	return NewValidator[T]()
}

// Rule adds a validation rule to the validator (currying/chaining).
func (v *Validator[T]) Rule(rule ValidationRule[T]) *Validator[T] {
	v.rules = append(v.rules, rule)
	return v
}

// Validate validates a value against all rules.
func (v *Validator[T]) Validate(value T, fieldName string) *Error {
	for _, rule := range v.rules {
		if err := rule(value, fieldName); err != nil {
			return err
		}
	}
	return nil
}

// String validation rules

// Required validates that a string is not empty.
func Required() ValidationRule[string] {
	return func(value string, fieldName string) *Error {
		if strings.TrimSpace(value) == "" {
			return NewError(NewErrorResponse(
				http.StatusBadRequest,
				NewParameterErrorDetail(fieldName, "This field is required"),
			))
		}
		return nil
	}
}

// MinLength validates minimum string length.
func MinLength(min int) ValidationRule[string] {
	return func(value string, fieldName string) *Error {
		if len(value) < min {
			return NewError(NewErrorResponse(
				http.StatusBadRequest,
				NewParameterErrorDetail(fieldName, fmt.Sprintf("Must be at least %d characters long", min)),
			))
		}
		return nil
	}
}

// MaxLength validates maximum string length.
func MaxLength(max int) ValidationRule[string] {
	return func(value string, fieldName string) *Error {
		if len(value) > max {
			return NewError(NewErrorResponse(
				http.StatusBadRequest,
				NewParameterErrorDetail(fieldName, fmt.Sprintf("Must be at most %d characters long", max)),
			))
		}
		return nil
	}
}

// Email validates email format (simple validation).
func Email() ValidationRule[string] {
	return func(value string, fieldName string) *Error {
		if value != "" && !strings.Contains(value, "@") {
			return NewError(NewErrorResponse(
				http.StatusBadRequest,
				NewParameterErrorDetail(fieldName, "Must be a valid email address"),
			))
		}
		return nil
	}
}

// Integer validation rules

// Min validates minimum integer value.
func Min(min int) ValidationRule[int] {
	return func(value int, fieldName string) *Error {
		if value < min {
			return NewError(NewErrorResponse(
				http.StatusBadRequest,
				NewParameterErrorDetail(fieldName, fmt.Sprintf("Must be at least %d", min)),
			))
		}
		return nil
	}
}

// Max validates maximum integer value.
func Max(max int) ValidationRule[int] {
	return func(value int, fieldName string) *Error {
		if value > max {
			return NewError(NewErrorResponse(
				http.StatusBadRequest,
				NewParameterErrorDetail(fieldName, fmt.Sprintf("Must be at most %d", max)),
			))
		}
		return nil
	}
}

// Range validates integer is within range.
func Range(min, max int) ValidationRule[int] {
	return func(value int, fieldName string) *Error {
		if value < min || value > max {
			return NewError(NewErrorResponse(
				http.StatusBadRequest,
				NewParameterErrorDetail(fieldName, fmt.Sprintf("Must be between %d and %d", min, max)),
			))
		}
		return nil
	}
}

// Generic validation rules

// Custom allows defining custom validation logic.
func Custom[T any](fn func(T) bool, message string) ValidationRule[T] {
	return func(value T, fieldName string) *Error {
		if !fn(value) {
			return NewError(NewErrorResponse(
				http.StatusBadRequest,
				NewParameterErrorDetail(fieldName, message),
			))
		}
		return nil
	}
}

// StructValidator provides validation for struct fields.
type StructValidator struct {
	fields map[string]func(interface{}) *Error
}

// NewStructValidator creates a new struct validator.
func NewStructValidator() *StructValidator {
	return &StructValidator{
		fields: make(map[string]func(interface{}) *Error),
	}
}

// ValidateStruct creates a new struct validator
// Example usage:
//
//	validator := validation.ValidateStruct().Field("Name", func(v interface{}) *validation.Error {
//	    return validation.Validate[string]().Rule(validation.Required()).Validate(v.(string), "Name")
//	})
func ValidateStruct() *StructValidator {
	return NewStructValidator()
}

// Field adds a field validator.
func (sv *StructValidator) Field(fieldName string, validator func(interface{}) *Error) *StructValidator {
	sv.fields[fieldName] = validator
	return sv
}

// Validate validates a struct.
func (sv *StructValidator) Validate(data interface{}) *Error {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return NewError(NewErrorResponse(
				http.StatusBadRequest,
				NewParameterErrorDetail("data", "Data cannot be nil"),
			))
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return NewError(NewErrorResponse(
			http.StatusBadRequest,
			NewParameterErrorDetail("data", "Data must be a struct"),
		))
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		if validator, exists := sv.fields[field.Name]; exists {
			if err := validator(fieldValue.Interface()); err != nil {
				return err
			}
		}
	}

	return nil
}

// Helper functions for common type conversions and validations

// ValidateStringAsInt validates a string can be converted to int and applies int validation.
func ValidateStringAsInt(value string, fieldName string, validator *Validator[int]) *Error {
	if value == "" {
		return nil // Let Required() handle empty strings
	}

	intVal, err := strconv.Atoi(value)
	if err != nil {
		return NewError(NewErrorResponse(
			http.StatusBadRequest,
			NewParameterErrorDetail(fieldName, "Must be a valid integer"),
		))
	}

	return validator.Validate(intVal, fieldName)
}
