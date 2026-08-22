package govalin

import (
	"fmt"
	"net/http"
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

// StringFieldValidator chains validation rules for one string field of the body.
// It embeds the body validator, so a chain continues into the next field without
// closing this one.
type StringFieldValidator[T any] struct {
	*BodyValidator[T]
	name     string
	accessor func(T) string
}

// IntFieldValidator chains validation rules for one int field of the body.
// It embeds the body validator, so a chain continues into the next field without
// closing this one.
type IntFieldValidator[T any] struct {
	*BodyValidator[T]
	name     string
	accessor func(T) int
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

// StringField begins validating the string field the accessor reads, reporting a
// failure under name. Rules chained after it apply to that field, and a following
// StringField, IntField or Get continues on the body.
func (v *BodyValidator[T]) StringField(name string, accessor func(T) string) *StringFieldValidator[T] {
	return &StringFieldValidator[T]{BodyValidator: v, name: name, accessor: accessor}
}

// IntField begins validating the int field the accessor reads, reporting a failure
// under name. Rules chained after it apply to that field, and a following
// StringField, IntField or Get continues on the body.
func (v *BodyValidator[T]) IntField(name string, accessor func(T) int) *IntFieldValidator[T] {
	return &IntFieldValidator[T]{BodyValidator: v, name: name, accessor: accessor}
}

func addFieldRule[T any](v *BodyValidator[T], name, message string, isValid func(T) bool) {
	v.rules = append(v.rules, func(data *T) error {
		if !isValid(*data) {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(name, message),
			))
		}
		return nil
	})
}

// Formatted where it is reported, not at chain build: a chain is built per request.
func addFieldLimitRule[T any](v *BodyValidator[T], name, format string, limit int, isValid func(T) bool) {
	v.rules = append(v.rules, func(data *T) error {
		if !isValid(*data) {
			return validation.NewError(validation.NewErrorResponse(
				http.StatusBadRequest,
				validation.NewParameterErrorDetail(name, fmt.Sprintf(format, limit)),
			))
		}
		return nil
	})
}

// String field validation rule methods

// Required adds a required validation rule for the field.
func (v *StringFieldValidator[T]) Required() *StringFieldValidator[T] {
	addFieldRule(v.BodyValidator, v.name, "This field is required", func(body T) bool {
		return strings.TrimSpace(v.accessor(body)) != ""
	})
	return v
}

// MinLength adds a minimum length validation rule for the field, counted in runes.
func (v *StringFieldValidator[T]) MinLength(minimum int) *StringFieldValidator[T] {
	addFieldLimitRule(v.BodyValidator, v.name, "Must be at least %d characters long", minimum, func(body T) bool {
		return utf8.RuneCountInString(v.accessor(body)) >= minimum
	})
	return v
}

// MaxLength adds a maximum length validation rule for the field, counted in runes.
func (v *StringFieldValidator[T]) MaxLength(maximum int) *StringFieldValidator[T] {
	addFieldLimitRule(v.BodyValidator, v.name, "Must be at most %d characters long", maximum, func(body T) bool {
		return utf8.RuneCountInString(v.accessor(body)) <= maximum
	})
	return v
}

// Email adds an email validation rule for the field. An empty field passes, so
// combine it with Required to reject one.
func (v *StringFieldValidator[T]) Email() *StringFieldValidator[T] {
	addFieldRule(v.BodyValidator, v.name, "Must be a valid email address", func(body T) bool {
		email := v.accessor(body)
		return email == "" || strings.Contains(email, "@")
	})
	return v
}

// Custom adds a custom validation rule that receives the whole body, so a check can
// depend on other fields, and attributes a failure to this field. It shadows the
// body-level Custom: one meant for the body must come before the first field.
func (v *StringFieldValidator[T]) Custom(validatorFn func(T) bool, message string) *StringFieldValidator[T] {
	addFieldRule(v.BodyValidator, v.name, message, validatorFn)
	return v
}

// Integer field validation rule methods

// Min adds a minimum value validation rule for the field.
func (v *IntFieldValidator[T]) Min(minimum int) *IntFieldValidator[T] {
	addFieldLimitRule(v.BodyValidator, v.name, "Must be at least %d", minimum, func(body T) bool {
		return v.accessor(body) >= minimum
	})
	return v
}

// Max adds a maximum value validation rule for the field.
func (v *IntFieldValidator[T]) Max(maximum int) *IntFieldValidator[T] {
	addFieldLimitRule(v.BodyValidator, v.name, "Must be at most %d", maximum, func(body T) bool {
		return v.accessor(body) <= maximum
	})
	return v
}

// Custom adds a custom validation rule that receives the whole body, so a check can
// depend on other fields, and attributes a failure to this field. It shadows the
// body-level Custom: one meant for the body must come before the first field.
func (v *IntFieldValidator[T]) Custom(validatorFn func(T) bool, message string) *IntFieldValidator[T] {
	addFieldRule(v.BodyValidator, v.name, message, validatorFn)
	return v
}
