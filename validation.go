package govalin

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/pkkummermo/govalin/internal/validation"
)

// StringValidator provides a curryable string validation interface.
type StringValidator struct {
	call  *Call
	key   string
	value string
	rules []func(string, string) error
}

// IntValidator provides a curryable integer validation interface.
type IntValidator struct {
	call  *Call
	key   string
	value string
	rules []func(int, string) error
}

// BodyValidator provides validation for request body.
type BodyValidator[T any] struct {
	call   *Call
	target *T
	rules  []func(*T) error
}

// BodyFieldValidator allows chaining validation rules for a specific field.
type BodyFieldValidator[T any] struct {
	bodyValidator *BodyValidator[T]
	fieldName     string
}

// String validation rule methods

// Required adds a required validation rule.
func (v *StringValidator) Required() *StringValidator {
	v.rules = append(v.rules, func(value, fieldName string) error {
		if strings.TrimSpace(value) == "" {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(fieldName, "This field is required"),
			))
		}
		return nil
	})
	return v
}

// MinLength adds a minimum length validation rule.
func (v *StringValidator) MinLength(minimum int) *StringValidator {
	v.rules = append(v.rules, func(value, fieldName string) error {
		if utf8.RuneCountInString(value) < minimum {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(fieldName, fmt.Sprintf("Must be at least %d characters long", minimum)),
			))
		}
		return nil
	})
	return v
}

// MaxLength adds a maximum length validation rule.
func (v *StringValidator) MaxLength(maximum int) *StringValidator {
	v.rules = append(v.rules, func(value, fieldName string) error {
		if utf8.RuneCountInString(value) > maximum {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(fieldName, fmt.Sprintf("Must be at most %d characters long", maximum)),
			))
		}
		return nil
	})
	return v
}

// Email adds an email validation rule.
func (v *StringValidator) Email() *StringValidator {
	v.rules = append(v.rules, func(value, fieldName string) error {
		if value != "" && !strings.Contains(value, "@") {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(fieldName, "Must be a valid email address"),
			))
		}
		return nil
	})
	return v
}

// Custom adds a custom validation rule for strings.
func (v *StringValidator) Custom(fn func(string) bool, message string) *StringValidator {
	v.rules = append(v.rules, func(value, fieldName string) error {
		if !fn(value) {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(fieldName, message),
			))
		}
		return nil
	})
	return v
}

// Get validates the string and returns it if valid.
func (v *StringValidator) Get() (string, error) {
	for _, rule := range v.rules {
		if err := rule(v.value, v.key); err != nil {
			return "", err
		}
	}
	return v.value, nil
}

// Integer validation rule methods

// Min adds a minimum value validation rule for integers.
func (v *IntValidator) Min(minimum int) *IntValidator {
	v.rules = append(v.rules, func(value int, fieldName string) error {
		if value < minimum {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(fieldName, fmt.Sprintf("Must be at least %d", minimum)),
			))
		}
		return nil
	})
	return v
}

// Max adds a maximum value validation rule for integers.
func (v *IntValidator) Max(maximum int) *IntValidator {
	v.rules = append(v.rules, func(value int, fieldName string) error {
		if value > maximum {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(fieldName, fmt.Sprintf("Must be at most %d", maximum)),
			))
		}
		return nil
	})
	return v
}

// Range adds a range validation rule for integers.
func (v *IntValidator) Range(minimum, maximum int) *IntValidator {
	v.rules = append(v.rules, func(value int, fieldName string) error {
		if value < minimum || value > maximum {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(fieldName, fmt.Sprintf("Must be between %d and %d", minimum, maximum)),
			))
		}
		return nil
	})
	return v
}

// Custom adds a custom validation rule for integers.
func (v *IntValidator) Custom(fn func(int) bool, message string) *IntValidator {
	v.rules = append(v.rules, func(value int, fieldName string) error {
		if !fn(value) {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(fieldName, message),
			))
		}
		return nil
	})
	return v
}

// Get validates the integer and returns it if valid.
func (v *IntValidator) Get() (int, error) {
	intVal, err := strconv.Atoi(v.value)
	if err != nil {
		return 0, validation.NewError(validation.NewErrorResponse(
			http.StatusBadRequest,
			validation.NewParameterErrorDetail(v.key, "Must be a valid integer"),
		))
	}

	for _, rule := range v.rules {
		if errRule := rule(intVal, v.key); errRule != nil {
			return 0, errRule
		}
	}
	return intVal, nil
}

// Body validation methods

// Custom adds a custom validation rule for the entire body. The rule receives
// the unmarshalled body and attributes a failure to "body".
func (v *BodyValidator[T]) Custom(validatorFn func(T) bool, message string) *BodyValidator[T] {
	v.rules = append(v.rules, func(data *T) error {
		if !validatorFn(*data) {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail("body", message),
			))
		}
		return nil
	})
	return v
}

// Get validates the body and returns error if invalid.
func (v *BodyValidator[T]) Get() error {
	if err := v.call.BodyAs(v.target); err != nil {
		return err
	}

	for _, rule := range v.rules {
		if err := rule(v.target); err != nil {
			return err
		}
	}
	return nil
}

// ValidateField sets the current field for validation and returns a BodyFieldValidator.
func (v *BodyValidator[T]) ValidateField(fieldName string) *BodyFieldValidator[T] {
	return &BodyFieldValidator[T]{
		bodyValidator: v,
		fieldName:     fieldName,
	}
}

// Required adds a required validation rule for the current field.
func (f *BodyFieldValidator[T]) Required() *BodyFieldValidator[T] {
	f.bodyValidator.rules = append(f.bodyValidator.rules, func(data *T) error {
		val := reflect.ValueOf(data).Elem()
		field := val.FieldByName(f.fieldName)
		if !field.IsValid() {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(f.fieldName, "Field does not exist"),
			))
		}

		if field.Kind() == reflect.String && strings.TrimSpace(field.String()) == "" {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(f.fieldName, "This field is required"),
			))
		}
		return nil
	})
	return f
}

// MinLength adds a minimum length validation rule for string fields.
func (f *BodyFieldValidator[T]) MinLength(minimum int) *BodyFieldValidator[T] {
	f.bodyValidator.rules = append(f.bodyValidator.rules, func(data *T) error {
		val := reflect.ValueOf(data).Elem()
		field := val.FieldByName(f.fieldName)
		if !field.IsValid() {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(f.fieldName, "Field does not exist"),
			))
		}

		if field.Kind() == reflect.String && len(field.String()) < minimum {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(f.fieldName, fmt.Sprintf("Must be at least %d characters long", minimum)),
			))
		}
		return nil
	})
	return f
}

// MaxLength adds a maximum length validation rule for string fields.
func (f *BodyFieldValidator[T]) MaxLength(maximum int) *BodyFieldValidator[T] {
	f.bodyValidator.rules = append(f.bodyValidator.rules, func(data *T) error {
		val := reflect.ValueOf(data).Elem()
		field := val.FieldByName(f.fieldName)
		if !field.IsValid() {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(f.fieldName, "Field does not exist"),
			))
		}

		if field.Kind() == reflect.String && len(field.String()) > maximum {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(f.fieldName, fmt.Sprintf("Must be at most %d characters long", maximum)),
			))
		}
		return nil
	})
	return f
}

// Email adds an email validation rule for string fields.
func (f *BodyFieldValidator[T]) Email() *BodyFieldValidator[T] {
	f.bodyValidator.rules = append(f.bodyValidator.rules, func(data *T) error {
		val := reflect.ValueOf(data).Elem()
		field := val.FieldByName(f.fieldName)
		if !field.IsValid() {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(f.fieldName, "Field does not exist"),
			))
		}

		if field.Kind() == reflect.String {
			email := field.String()
			if email != "" && !strings.Contains(email, "@") {
				return validation.NewError(validation.NewErrorResponse(
					http.StatusBadRequest,
					validation.NewParameterErrorDetail(f.fieldName, "Must be a valid email address"),
				))
			}
		}
		return nil
	})
	return f
}

// Min adds a minimum value validation rule for integer fields.
func (f *BodyFieldValidator[T]) Min(minimum int) *BodyFieldValidator[T] {
	f.bodyValidator.rules = append(f.bodyValidator.rules, func(data *T) error {
		val := reflect.ValueOf(data).Elem()
		field := val.FieldByName(f.fieldName)
		if !field.IsValid() {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(f.fieldName, "Field does not exist"),
			))
		}

		if field.Kind() == reflect.Int && int(field.Int()) < minimum {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(f.fieldName, fmt.Sprintf("Must be at least %d", minimum)),
			))
		}
		return nil
	})
	return f
}

// Max adds a maximum value validation rule for integer fields.
func (f *BodyFieldValidator[T]) Max(maximum int) *BodyFieldValidator[T] {
	f.bodyValidator.rules = append(f.bodyValidator.rules, func(data *T) error {
		val := reflect.ValueOf(data).Elem()
		field := val.FieldByName(f.fieldName)
		if !field.IsValid() {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(f.fieldName, "Field does not exist"),
			))
		}

		if field.Kind() == reflect.Int && int(field.Int()) > maximum {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(f.fieldName, fmt.Sprintf("Must be at most %d", maximum)),
			))
		}
		return nil
	})
	return f
}

// Custom adds a custom validation rule that attributes its failure to the current
// field. The rule receives a pointer to the whole body, not the field value.
func (f *BodyFieldValidator[T]) Custom(validatorFn func(interface{}) bool, message string) *BodyFieldValidator[T] {
	f.bodyValidator.rules = append(f.bodyValidator.rules, func(data *T) error {
		if !validatorFn(data) {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(f.fieldName, message),
			))
		}
		return nil
	})
	return f
}

// Get completes the field validation and returns the body validator for more fields or final validation.
func (f *BodyFieldValidator[T]) Get() *BodyValidator[T] {
	return f.bodyValidator
}
