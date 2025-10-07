package validation_test

import (
	"net/http"
	"testing"

	"github.com/pkkummermo/govalin/internal/validation"
	"github.com/stretchr/testify/assert"
)

// Test NewValidator and Validate functions.
func TestNewValidator(t *testing.T) {
	validator := validation.NewValidator[string]()
	assert.NotNil(t, validator)
}

func TestValidate(t *testing.T) {
	validator := validation.Validate[string]()
	assert.NotNil(t, validator)
}

// Test Rule chaining.
func TestValidatorRuleChaining(t *testing.T) {
	validator := validation.NewValidator[string]().
		Rule(validation.Required()).
		Rule(validation.MinLength(3)).
		Rule(validation.MaxLength(10))

	// Test that chaining works by validating
	err := validator.Validate("test", "field")
	assert.Nil(t, err)
}

// Test string validation rules.
func TestRequired(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		shouldErr bool
	}{
		{"valid non-empty", "test", false},
		{"empty string", "", true},
		{"whitespace only", "   ", true},
		{"single space", " ", true},
		{"valid with spaces", "test value", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := validation.NewValidator[string]().Rule(validation.Required())
			err := validator.Validate(tt.value, "testField")

			if tt.shouldErr {
				assert.NotNil(t, err)
				assert.Equal(t, http.StatusBadRequest, err.ErrorResponse.Status)
				assert.Contains(t, err.ErrorResponse.Details[0].Reason, "This field is required")
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestMinLength(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		minLength int
		shouldErr bool
	}{
		{"exact minimum", "abc", 3, false},
		{"above minimum", "abcd", 3, false},
		{"below minimum", "ab", 3, true},
		{"empty string", "", 3, true},
		{"zero minimum", "a", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := validation.NewValidator[string]().Rule(validation.MinLength(tt.minLength))
			err := validator.Validate(tt.value, "testField")

			if tt.shouldErr {
				assert.NotNil(t, err)
				assert.Equal(t, http.StatusBadRequest, err.ErrorResponse.Status)
				assert.Contains(t, err.ErrorResponse.Details[0].Reason, "Must be at least")
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestMaxLength(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		maxLength int
		shouldErr bool
	}{
		{"exact maximum", "abc", 3, false},
		{"below maximum", "ab", 3, false},
		{"above maximum", "abcd", 3, true},
		{"empty string", "", 3, false},
		{"zero maximum", "", 0, false},
		{"zero maximum with content", "a", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := validation.NewValidator[string]().Rule(validation.MaxLength(tt.maxLength))
			err := validator.Validate(tt.value, "testField")

			if tt.shouldErr {
				assert.NotNil(t, err)
				assert.Equal(t, http.StatusBadRequest, err.ErrorResponse.Status)
				assert.Contains(t, err.ErrorResponse.Details[0].Reason, "Must be at most")
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestEmail(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		shouldErr bool
	}{
		{"valid email", "test@example.com", false},
		{"valid simple email", "a@b", false},
		{"invalid no @", "testexample.com", true},
		{"empty string", "", false}, // Email rule allows empty, use validation.Required() for that
		{"only @", "@", false},
		{"multiple @", "test@example@com", false}, // Simple validation, just checks for @
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := validation.NewValidator[string]().Rule(validation.Email())
			err := validator.Validate(tt.value, "testField")

			if tt.shouldErr {
				assert.NotNil(t, err)
				assert.Equal(t, http.StatusBadRequest, err.ErrorResponse.Status)
				assert.Contains(t, err.ErrorResponse.Details[0].Reason, "Must be a valid email address")
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

// Test integer validation rules.
func TestMin(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		min       int
		shouldErr bool
	}{
		{"exact minimum", 5, 5, false},
		{"above minimum", 10, 5, false},
		{"below minimum", 3, 5, true},
		{"negative values", -10, -5, true},
		{"negative minimum met", -3, -5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := validation.NewValidator[int]().Rule(validation.Min(tt.min))
			err := validator.Validate(tt.value, "testField")

			if tt.shouldErr {
				assert.NotNil(t, err)
				assert.Equal(t, http.StatusBadRequest, err.ErrorResponse.Status)
				assert.Contains(t, err.ErrorResponse.Details[0].Reason, "Must be at least")
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		max       int
		shouldErr bool
	}{
		{"exact maximum", 10, 10, false},
		{"below maximum", 5, 10, false},
		{"above maximum", 15, 10, true},
		{"negative values", -5, -10, true},
		{"negative maximum met", -15, -10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := validation.NewValidator[int]().Rule(validation.Max(tt.max))
			err := validator.Validate(tt.value, "testField")

			if tt.shouldErr {
				assert.NotNil(t, err)
				assert.Equal(t, http.StatusBadRequest, err.ErrorResponse.Status)
				assert.Contains(t, err.ErrorResponse.Details[0].Reason, "Must be at most")
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestRange(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		min       int
		max       int
		shouldErr bool
	}{
		{"within range", 5, 1, 10, false},
		{"exact minimum", 1, 1, 10, false},
		{"exact maximum", 10, 1, 10, false},
		{"below range", 0, 1, 10, true},
		{"above range", 11, 1, 10, true},
		{"negative range", -5, -10, -1, false},
		{"outside negative range low", -11, -10, -1, true},
		{"outside negative range high", 0, -10, -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := validation.NewValidator[int]().Rule(validation.Range(tt.min, tt.max))
			err := validator.Validate(tt.value, "testField")

			if tt.shouldErr {
				assert.NotNil(t, err)
				assert.Equal(t, http.StatusBadRequest, err.ErrorResponse.Status)
				assert.Contains(t, err.ErrorResponse.Details[0].Reason, "Must be between")
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

// Test custom validation.
func TestCustomValidation(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		fn        func(string) bool
		shouldErr bool
	}{
		{
			"custom passes",
			"test123",
			func(s string) bool { return len(s) > 5 },
			false,
		},
		{
			"custom fails",
			"test",
			func(s string) bool { return len(s) > 5 },
			true,
		},
		{
			"custom complex logic",
			"ValidString",
			func(s string) bool {
				// Must start with uppercase and contain no spaces
				if len(s) == 0 {
					return false
				}
				return s[0] >= 'A' && s[0] <= 'Z' && !containsSpace(s)
			},
			false,
		},
		{
			"custom complex logic fails",
			"invalid string",
			func(s string) bool {
				if len(s) == 0 {
					return false
				}
				return s[0] >= 'A' && s[0] <= 'Z' && !containsSpace(s)
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := validation.NewValidator[string]().Rule(validation.Custom(tt.fn, "Custom validation failed"))
			err := validator.Validate(tt.value, "testField")

			if tt.shouldErr {
				assert.NotNil(t, err)
				assert.Equal(t, http.StatusBadRequest, err.ErrorResponse.Status)
				assert.Contains(t, err.ErrorResponse.Details[0].Reason, "Custom validation failed")
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func containsSpace(s string) bool {
	for _, r := range s {
		if r == ' ' {
			return true
		}
	}
	return false
}

// Test custom validation with int.
func TestCustomValidationInt(t *testing.T) {
	validator := validation.NewValidator[int]().Rule(validation.Custom(func(i int) bool {
		return i%2 == 0 // Must be even
	}, "Must be an even number"))

	// Test even number (should pass)
	err := validator.Validate(10, "testField")
	assert.Nil(t, err)

	// Test odd number (should fail)
	err = validator.Validate(11, "testField")
	assert.NotNil(t, err)
	assert.Contains(t, err.ErrorResponse.Details[0].Reason, "Must be an even number")
}

// Test multiple rules chaining.
func TestMultipleRulesChaining(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		shouldErr bool
		errorMsg  string
	}{
		{"all rules pass", "test@example.com", false, ""},
		{"fails required", "", true, "This field is required"},
		{"fails min length", "a@", true, "Must be at least 5 characters long"},
		{"fails max length", "verylongemailaddress@example.com", true, "Must be at most 30 characters long"},
		{"fails email", "notanemail", true, "Must be a valid email address"},
		{"valid edge case", "a@b.c", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := validation.NewValidator[string]().
				Rule(validation.Required()).
				Rule(validation.MinLength(5)).
				Rule(validation.MaxLength(30)).
				Rule(validation.Email())

			err := validator.Validate(tt.value, "email")

			if tt.shouldErr {
				assert.NotNil(t, err)
				assert.Contains(t, err.ErrorResponse.Details[0].Reason, tt.errorMsg)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

// Test StructValidator.
func TestNewStructValidator(t *testing.T) {
	validator := validation.NewStructValidator()
	assert.NotNil(t, validator)
}

func TestValidateStruct(t *testing.T) {
	validator := validation.ValidateStruct()
	assert.NotNil(t, validator)
}

func TestStructValidatorField(t *testing.T) {
	validator := validation.NewStructValidator().
		Field("Name", func(v interface{}) *validation.Error {
			//nolint:errcheck // This is a test helper that returns errors
			return validation.Validate[string]().Rule(validation.Required()).
				Validate(v.(string), "Name")
		}).
		Field("Age", func(v interface{}) *validation.Error {
			//nolint:errcheck // This is a test helper that returns errors
			return validation.Validate[int]().Rule(validation.Min(18)).
				Validate(v.(int), "Age")
		})

	// Test that fields can be validated
	testData := &TestStruct{Name: "John", Age: 25, Email: "test@example.com"}
	err := validator.Validate(testData)
	assert.Nil(t, err)
}

type TestStruct struct {
	Name  string
	Age   int
	Email string
}

func TestStructValidatorValidate(t *testing.T) {
	tests := []struct {
		name      string
		data      interface{}
		shouldErr bool
		errorMsg  string
	}{
		{
			"valid struct",
			&TestStruct{Name: "John", Age: 25, Email: "john@example.com"},
			false,
			"",
		},
		{
			"invalid name",
			&TestStruct{Name: "", Age: 25, Email: "john@example.com"},
			true,
			"This field is required",
		},
		{
			"invalid age",
			&TestStruct{Name: "John", Age: 15, Email: "john@example.com"},
			true,
			"Must be at least 18",
		},
		{
			"invalid email",
			&TestStruct{Name: "John", Age: 25, Email: "notanemail"},
			true,
			"Must be a valid email address",
		},
		{
			"nil pointer",
			(*TestStruct)(nil),
			true,
			"Data cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := validation.ValidateStruct().
				Field("Name", func(v interface{}) *validation.Error {
					//nolint:errcheck // This is a test helper that returns errors
					return validation.Validate[string]().Rule(validation.Required()).
						Validate(v.(string), "Name")
				}).
				Field("Age", func(v interface{}) *validation.Error {
					//nolint:errcheck // This is a test helper that returns errors
					return validation.Validate[int]().Rule(validation.Min(18)).
						Validate(v.(int), "Age")
				}).
				Field("Email", func(v interface{}) *validation.Error {
					//nolint:errcheck // This is a test helper that returns errors
					return validation.Validate[string]().Rule(validation.Email()).
						Validate(v.(string), "Email")
				})

			err := validator.Validate(tt.data)

			if tt.shouldErr {
				assert.NotNil(t, err)
				assert.Contains(t, err.ErrorResponse.Details[0].Reason, tt.errorMsg)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestStructValidatorValidateNonStruct(t *testing.T) {
	validator := validation.ValidateStruct()

	// Test with non-struct value
	err := validator.Validate("not a struct")
	assert.NotNil(t, err)
	assert.Contains(t, err.ErrorResponse.Details[0].Reason, "Data must be a struct")

	// Test with int
	err = validator.Validate(123)
	assert.NotNil(t, err)
	assert.Contains(t, err.ErrorResponse.Details[0].Reason, "Data must be a struct")
}

func TestStructValidatorValidateStructValue(t *testing.T) {
	// Test with struct value (not pointer)
	validator := validation.ValidateStruct().
		Field("Name", func(v interface{}) *validation.Error {
			//nolint:errcheck // This is a test helper that returns errors
			return validation.Validate[string]().Rule(validation.Required()).
				Validate(v.(string), "Name")
		})

	data := TestStruct{Name: "John", Age: 25}
	err := validator.Validate(data)
	assert.Nil(t, err)
}

// Test ValidateStringAsInt.
func TestValidateStringAsInt(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		validator *validation.Validator[int]
		shouldErr bool
		errorMsg  string
	}{
		{
			"valid int string",
			"25",
			validation.NewValidator[int]().Rule(validation.Min(18)).Rule(validation.Max(100)),
			false,
			"",
		},
		{
			"invalid int string",
			"not a number",
			validation.NewValidator[int]().Rule(validation.Min(18)),
			true,
			"Must be a valid integer",
		},
		{
			"empty string",
			"",
			validation.NewValidator[int]().Rule(validation.Min(18)),
			false,
			"", // Empty strings are allowed
		},
		{
			"int fails validation",
			"15",
			validation.NewValidator[int]().Rule(validation.Min(18)),
			true,
			"Must be at least 18",
		},
		{
			"negative int",
			"-10",
			validation.NewValidator[int]().Rule(validation.Min(0)),
			true,
			"Must be at least 0",
		},
		{
			"valid negative int",
			"-5",
			validation.NewValidator[int]().Rule(validation.Min(-10)).Rule(validation.Max(0)),
			false,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateStringAsInt(tt.value, "testField", tt.validator)

			if tt.shouldErr {
				assert.NotNil(t, err)
				assert.Contains(t, err.ErrorResponse.Details[0].Reason, tt.errorMsg)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

// Test edge cases.
func TestValidatorEdgeCases(t *testing.T) {
	t.Run("validator with no rules", func(t *testing.T) {
		validator := validation.NewValidator[string]()
		err := validator.Validate("any value", "field")
		assert.Nil(t, err) // Should pass with no rules
	})

	t.Run("field name in error message", func(t *testing.T) {
		validator := validation.NewValidator[string]().Rule(validation.Required())
		err := validator.Validate("", "customFieldName")
		assert.NotNil(t, err)
		assert.Equal(t, "customFieldName", err.ErrorResponse.Details[0].Field)
	})

	t.Run("multiple validation errors stop at first", func(t *testing.T) {
		validator := validation.NewValidator[string]().
			Rule(validation.Required()).
			Rule(validation.MinLength(10)) // This won't be checked if Required fails

		err := validator.Validate("", "field")
		assert.NotNil(t, err)
		assert.Contains(t, err.ErrorResponse.Details[0].Reason, "This field is required")
		// Should not contain MinLength error
	})

	t.Run("struct validator with no fields", func(t *testing.T) {
		validator := validation.ValidateStruct()
		err := validator.Validate(&TestStruct{Name: "test", Age: 20})
		assert.Nil(t, err) // Should pass with no field validators
	})
}
