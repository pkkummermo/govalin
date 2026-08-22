package govalin

import (
	"net/http"
	"strconv"

	"github.com/pkkummermo/govalin/internal/validation"
)

// StringValidator provides a curryable string validation interface.
type StringValidator struct {
	key   string
	value string
	err   error
	rules []validation.Rule[string]
}

// IntValidator provides a curryable integer validation interface.
type IntValidator struct {
	key   string
	value string
	err   error
	rules []validation.Rule[int]
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
	v.rules = append(v.rules, validation.Required())
	return v
}

// MinLength adds a minimum length validation rule, counted in runes.
func (v *StringValidator) MinLength(minimum int) *StringValidator {
	v.rules = append(v.rules, validation.MinLength(minimum))
	return v
}

// MaxLength adds a maximum length validation rule, counted in runes.
func (v *StringValidator) MaxLength(maximum int) *StringValidator {
	v.rules = append(v.rules, validation.MaxLength(maximum))
	return v
}

// Email adds an email validation rule. An empty value passes, so combine it with
// Required to reject one.
func (v *StringValidator) Email() *StringValidator {
	v.rules = append(v.rules, validation.Email())
	return v
}

// Custom adds a custom validation rule for strings.
func (v *StringValidator) Custom(fn func(string) bool, message string) *StringValidator {
	v.rules = append(v.rules, validation.Custom(fn, message))
	return v
}

// Get validates the string and returns it if valid.
func (v *StringValidator) Get() (string, error) {
	if v.err != nil {
		return "", v.err
	}

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
	v.rules = append(v.rules, validation.Min(minimum))
	return v
}

// Max adds a maximum value validation rule for integers.
func (v *IntValidator) Max(maximum int) *IntValidator {
	v.rules = append(v.rules, validation.Max(maximum))
	return v
}

// Range adds a range validation rule for integers.
func (v *IntValidator) Range(minimum, maximum int) *IntValidator {
	v.rules = append(v.rules, validation.Range(minimum, maximum))
	return v
}

// Custom adds a custom validation rule for integers.
func (v *IntValidator) Custom(fn func(int) bool, message string) *IntValidator {
	v.rules = append(v.rules, validation.Custom(fn, message))
	return v
}

// Get validates the integer and returns it if valid.
func (v *IntValidator) Get() (int, error) {
	if v.err != nil {
		return 0, v.err
	}

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
	addBodyRule(v, "body", validation.Custom(validatorFn, message))
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

func addBodyRule[T any](v *BodyValidator[T], name string, rule validation.Rule[T]) {
	v.rules = append(v.rules, func(data *T) error {
		if err := rule(*data, name); err != nil {
			return err
		}
		return nil
	})
}

// onField lifts a rule on a field's value to one on the whole body.
func onField[T, V any](accessor func(T) V, rule validation.Rule[V]) validation.Rule[T] {
	return func(body T, fieldName string) *validation.Error {
		return rule(accessor(body), fieldName)
	}
}

// String field validation rule methods

// Required adds a required validation rule for the field.
func (v *StringFieldValidator[T]) Required() *StringFieldValidator[T] {
	addBodyRule(v.BodyValidator, v.name, onField(v.accessor, validation.Required()))
	return v
}

// MinLength adds a minimum length validation rule for the field, counted in runes.
func (v *StringFieldValidator[T]) MinLength(minimum int) *StringFieldValidator[T] {
	addBodyRule(v.BodyValidator, v.name, onField(v.accessor, validation.MinLength(minimum)))
	return v
}

// MaxLength adds a maximum length validation rule for the field, counted in runes.
func (v *StringFieldValidator[T]) MaxLength(maximum int) *StringFieldValidator[T] {
	addBodyRule(v.BodyValidator, v.name, onField(v.accessor, validation.MaxLength(maximum)))
	return v
}

// Email adds an email validation rule for the field. An empty field passes, so
// combine it with Required to reject one.
func (v *StringFieldValidator[T]) Email() *StringFieldValidator[T] {
	addBodyRule(v.BodyValidator, v.name, onField(v.accessor, validation.Email()))
	return v
}

// Custom adds a custom validation rule that receives the whole body, so a check can
// depend on other fields, and attributes a failure to this field. It shadows the
// body-level Custom: one meant for the body must come before the first field.
func (v *StringFieldValidator[T]) Custom(validatorFn func(T) bool, message string) *StringFieldValidator[T] {
	addBodyRule(v.BodyValidator, v.name, validation.Custom(validatorFn, message))
	return v
}

// Integer field validation rule methods

// Min adds a minimum value validation rule for the field.
func (v *IntFieldValidator[T]) Min(minimum int) *IntFieldValidator[T] {
	addBodyRule(v.BodyValidator, v.name, onField(v.accessor, validation.Min(minimum)))
	return v
}

// Max adds a maximum value validation rule for the field.
func (v *IntFieldValidator[T]) Max(maximum int) *IntFieldValidator[T] {
	addBodyRule(v.BodyValidator, v.name, onField(v.accessor, validation.Max(maximum)))
	return v
}

// Custom adds a custom validation rule that receives the whole body, so a check can
// depend on other fields, and attributes a failure to this field. It shadows the
// body-level Custom: one meant for the body must come before the first field.
func (v *IntFieldValidator[T]) Custom(validatorFn func(T) bool, message string) *IntFieldValidator[T] {
	addBodyRule(v.BodyValidator, v.name, validation.Custom(validatorFn, message))
	return v
}
