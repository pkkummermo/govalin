package govalin_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/govalintest"
	"github.com/pkkummermo/govalin/validation"
	"github.com/stretchr/testify/assert"
)

type TestUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func TestValidatedQueryParam(t *testing.T) {
	app := newTestApp()
	app.Post("/validate-query", func(call *govalin.Call) {
		name, err := call.ValidatedQueryParam("name").
			Required().
			MinLength(3).
			MaxLength(20).
			Get()
		if err != nil {
			call.Error(err)
			return
		}

		call.JSON(map[string]string{"message": "Valid name", "name": name})
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		// Test valid input
		response := client.Post("/validate-query?name=John", nil)
		assert.Contains(t, response, "Valid name")
		assert.Contains(t, response, "John")

		// Test empty string (should fail Required)
		response = client.Post("/validate-query?name=", nil)
		assert.Contains(t, response, "This field is required")

		// Test too short (should fail MinLength)
		response = client.Post("/validate-query?name=Jo", nil)
		assert.Contains(t, response, "Must be at least 3 characters long")

		// Test too long (should fail MaxLength)
		response = client.Post("/validate-query?name=ThisNameIsTooLongForOurValidation", nil)
		assert.Contains(t, response, "Must be at most 20 characters long")
	})
}

// TestValidatedQueryParamRuneLength verifies that MinLength/MaxLength on the
// public StringValidator count runes rather than bytes, so multi-byte input
// (accented characters, emoji) is measured by character count.
func TestValidatedQueryParamRuneLength(t *testing.T) {
	app := newTestApp()
	// Requires at least 5 characters.
	app.Post("/validate-rune-min", func(call *govalin.Call) {
		name, err := call.ValidatedQueryParam("name").MinLength(5).Get()
		if err != nil {
			call.Error(err)
			return
		}
		call.JSON(map[string]string{"message": "Valid name", "name": name})
	})

	// Allows at most 4 characters.
	app.Post("/validate-rune-max", func(call *govalin.Call) {
		name, err := call.ValidatedQueryParam("name").MaxLength(4).Get()
		if err != nil {
			call.Error(err)
			return
		}
		call.JSON(map[string]string{"message": "Valid name", "name": name})
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		// "café" is 4 runes but 5 bytes. It must fail a minimum of 5 runes,
		// even though a byte-based check (len == 5) would have passed it.
		response := client.Post("/validate-rune-min?name=caf%C3%A9", nil)
		assert.Contains(t, response, "Must be at least 5 characters long")

		// "café" (4 runes / 5 bytes) must satisfy a maximum of 4 runes,
		// even though a byte-based check (len == 5) would have rejected it.
		response = client.Post("/validate-rune-max?name=caf%C3%A9", nil)
		assert.Contains(t, response, "Valid name")

		// "👍👍" is 2 runes but 8 bytes; it must satisfy a maximum of 4 runes.
		response = client.Post("/validate-rune-max?name=%F0%9F%91%8D%F0%9F%91%8D", nil)
		assert.Contains(t, response, "Valid name")

		// "héllo" is 5 runes; it must exceed a maximum of 4 runes.
		response = client.Post("/validate-rune-max?name=h%C3%A9llo", nil)
		assert.Contains(t, response, "Must be at most 4 characters long")
	})
}

func TestValidatedPathParam(t *testing.T) {
	app := newTestApp()
	app.Post("/validate-path/{username}", func(call *govalin.Call) {
		username, err := call.ValidatedPathParam("username").
			Required().
			MinLength(3).
			MaxLength(15).
			Custom(func(s string) bool {
				// Username should only contain alphanumeric characters
				for _, r := range s {
					if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
						return false
					}
				}
				return true
			}, "Username must contain only alphanumeric characters").
			Get()
		if err != nil {
			call.Error(err)
			return
		}

		call.JSON(map[string]string{"message": "Valid username", "username": username})
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		// Test valid username
		response := client.Post("/validate-path/john123", nil)
		assert.Contains(t, response, "Valid username")
		assert.Contains(t, response, "john123")

		// Test username with special characters
		response = client.Post("/validate-path/john@123", nil)
		assert.Contains(t, response, "Username must contain only alphanumeric characters")
	})
}

func TestValidatedQueryParamAsInt(t *testing.T) {
	app := newTestApp()
	app.Post("/validate-int", func(call *govalin.Call) {
		age, err := call.ValidatedQueryParamAsInt("age").
			Min(18).
			Max(100).
			Get()
		if err != nil {
			call.Error(err)
			return
		}

		call.JSON(map[string]interface{}{"message": "Valid age", "age": age})
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		// Test valid input
		response := client.Post("/validate-int?age=25", nil)
		assert.Contains(t, response, "Valid age")

		// Test invalid integer
		response = client.Post("/validate-int?age=notanumber", nil)
		assert.Contains(t, response, "Must be a valid integer")

		// Test too low
		response = client.Post("/validate-int?age=15", nil)
		assert.Contains(t, response, "Must be at least 18")

		// Test too high
		response = client.Post("/validate-int?age=150", nil)
		assert.Contains(t, response, "Must be at most 100")
	})
}

func TestValidatedFormParam(t *testing.T) {
	app := newTestApp()
	app.Post("/validate-form", func(call *govalin.Call) {
		email, err := call.ValidatedFormParam("email").
			Required().
			Email().
			Get()
		if err != nil {
			call.Error(err)
			return
		}

		call.JSON(map[string]string{"message": "Valid email", "email": email})
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		// Test valid email
		form := url.Values{"email": {"test@example.com"}}
		req, err := http.NewRequest(http.MethodPost, "/validate-form", strings.NewReader(form.Encode()))
		assert.Nil(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response := readBody(t, client.Do(req))
		assert.Contains(t, response, "Valid email")

		// Test invalid email
		form = url.Values{"email": {"invalidemail"}}
		req, err = http.NewRequest(http.MethodPost, "/validate-form", strings.NewReader(form.Encode()))
		assert.Nil(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response = readBody(t, client.Do(req))
		assert.Contains(t, response, "Must be a valid email address")

		// Test empty email
		form = url.Values{"email": {""}}
		req, err = http.NewRequest(http.MethodPost, "/validate-form", strings.NewReader(form.Encode()))
		assert.Nil(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response = readBody(t, client.Do(req))
		assert.Contains(t, response, "This field is required")
	})
}

func TestValidatedBody(t *testing.T) {
	app := newTestApp()
	app.Post("/validate-body", func(call *govalin.Call) {
		var user TestUser

		// Use generic validation methods on ValidatedBody - no validation package import needed!
		err := call.ValidatedBody(&user).
			ValidateField("Name").Required().MinLength(2).Get().
			ValidateField("Email").Required().Email().Get().
			ValidateField("Age").Min(18).Max(100).Get().
			Get()

		if err != nil {
			call.Error(err)
			return
		}

		call.JSON(map[string]interface{}{"message": "Valid user data", "user": user})
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		// Test valid user
		validUser := TestUser{Name: "John Doe", Email: "john@example.com", Age: 25}
		validUserJSON, _ := json.Marshal(validUser)
		response := client.Post("/validate-body", string(validUserJSON))
		assert.Contains(t, response, "Valid user data")

		// Test invalid name (too short)
		invalidUser := TestUser{Name: "J", Email: "john@example.com", Age: 25}
		invalidUserJSON, _ := json.Marshal(invalidUser)
		response = client.Post("/validate-body", string(invalidUserJSON))
		assert.Contains(t, response, "Must be at least 2 characters long")

		// Test invalid email
		invalidUser = TestUser{Name: "John Doe", Email: "invalidemail", Age: 25}
		invalidUserJSON, _ = json.Marshal(invalidUser)
		response = client.Post("/validate-body", string(invalidUserJSON))
		assert.Contains(t, response, "Must be a valid email address")

		// Test invalid age
		invalidUser = TestUser{Name: "John Doe", Email: "john@example.com", Age: 15}
		invalidUserJSON, _ = json.Marshal(invalidUser)
		response = client.Post("/validate-body", string(invalidUserJSON))
		assert.Contains(t, response, "Must be at least 18")
	})
}

func TestValidatedBodyWithPublicAPI(t *testing.T) {
	app := newTestApp()
	app.Post("/validate-body-custom", func(call *govalin.Call) {
		var user TestUser

		// Parse the body first
		if err := call.ValidatedBody(&user).Get(); err != nil {
			call.Error(err)
			return
		}

		// For complex validation that can't be done with built-in field validators,
		// still use the validation package
		nameValidator := validation.NewStringValidator().
			Rule(validation.Required()).
			Rule(validation.MinLength(2))

		if err := nameValidator.Validate(user.Name, "Name"); err != nil {
			call.Error(err)
			return
		}

		emailValidator := validation.NewStringValidator().
			Rule(validation.Required()).
			Rule(validation.Email())

		if err := emailValidator.Validate(user.Email, "Email"); err != nil {
			call.Error(err)
			return
		}

		ageValidator := validation.NewIntValidator().
			Rule(validation.Min(18)).
			Rule(validation.Max(100))

		if err := ageValidator.Validate(user.Age, "Age"); err != nil {
			call.Error(err)
			return
		}

		call.JSON(map[string]interface{}{"message": "Custom validation passed", "user": user})
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		// Test valid user with custom validation
		validUser := TestUser{Name: "Jane Smith", Email: "jane@example.com", Age: 30}
		validUserJSON, _ := json.Marshal(validUser)
		response := client.Post("/validate-body-custom", string(validUserJSON))
		assert.Contains(t, response, "Custom validation passed")
	})
}

func TestChainingValidation(t *testing.T) {
	app := newTestApp()
	app.Post("/validate-chain", func(call *govalin.Call) {
		// Demonstrate complex chaining
		username, err := call.ValidatedQueryParam("username").
			Required().
			MinLength(3).
			MaxLength(15).
			Custom(func(s string) bool {
				// Username should only contain alphanumeric characters
				for _, r := range s {
					if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
						return false
					}
				}
				return true
			}, "Username must contain only alphanumeric characters").
			Get()
		if err != nil {
			call.Error(err)
			return
		}

		age, err := call.ValidatedQueryParamAsInt("age").
			Min(13).
			Max(120).
			Custom(func(i int) bool {
				return i != 42 // No answer to universe allowed!
			}, "Age cannot be 42").
			Get()
		if err != nil {
			call.Error(err)
			return
		}

		call.JSON(map[string]interface{}{
			"message":  "Valid data",
			"username": username,
			"age":      age,
		})
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		// Test valid input
		response := client.Post("/validate-chain?username=john123&age=25", nil)
		assert.Contains(t, response, "Valid data")
		assert.Contains(t, response, "john123")

		// Test custom validation failure
		response = client.Post("/validate-chain?username=john123&age=42", nil)
		assert.Contains(t, response, "Age cannot be 42")
	})
}

func TestPublicValidationAPI(t *testing.T) {
	app := newTestApp()
	app.Post("/validate-public", func(call *govalin.Call) {
		name := call.QueryParam("name")

		// Demonstrate using public validation API for custom scenarios
		validator := validation.NewStringValidator().
			Rule(validation.Required()).
			Rule(validation.MinLength(3)).
			Rule(validation.CustomString(func(s string) bool {
				// Custom validation: name must start with uppercase
				return len(s) > 0 && s[0] >= 'A' && s[0] <= 'Z'
			}, "Name must start with an uppercase letter"))

		if err := validator.Validate(name, "name"); err != nil {
			call.Error(err)
			return
		}

		call.JSON(map[string]string{"message": "Valid name", "name": name})
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		// Test valid input
		response := client.Post("/validate-public?name=John", nil)
		assert.Contains(t, response, "Valid name")

		// Test custom validation failure
		response = client.Post("/validate-public?name=john", nil)
		assert.Contains(t, response, "Name must start with an uppercase letter")
	})
}

func TestBodyValidatorCustom(t *testing.T) {
	app := newTestApp()
	app.Post("/validate-body-custom-validator", func(call *govalin.Call) {
		var user TestUser

		// Use validation.WithTypedCustom for type-safe validation without manual type casting
		validator := call.ValidatedBody(&user)
		validator = validation.WithTypedCustom(validator, func(user TestUser) bool {
			// Type-safe custom validation on the entire body - no casting needed!
			return user.Name != "InvalidUser" && user.Age >= 18
		}, "User validation failed: invalid user or under 18")

		err := validator.
			ValidateField("Name").Required().MinLength(2).Get().
			ValidateField("Email").Required().Email().Get().
			Get()

		if err != nil {
			call.Error(err)
			return
		}

		call.JSON(map[string]interface{}{"message": "Body custom validation passed", "user": user})
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		// Test valid user with custom body validation
		validUser := TestUser{Name: "ValidUser", Email: "valid@example.com", Age: 25}
		validUserJSON, _ := json.Marshal(validUser)
		response := client.Post("/validate-body-custom-validator", string(validUserJSON))
		assert.Contains(t, response, "Body custom validation passed")

		// Test invalid user with custom body validation (InvalidUser name)
		invalidUser := TestUser{Name: "InvalidUser", Email: "invalid@example.com", Age: 25}
		invalidUserJSON, _ := json.Marshal(invalidUser)
		response = client.Post("/validate-body-custom-validator", string(invalidUserJSON))
		assert.Contains(t, response, "User validation failed: invalid user or under 18")

		// Test invalid user with custom body validation (under 18)
		invalidUser = TestUser{Name: "ValidUser", Email: "valid@example.com", Age: 16}
		invalidUserJSON, _ = json.Marshal(invalidUser)
		response = client.Post("/validate-body-custom-validator", string(invalidUserJSON))
		assert.Contains(t, response, "User validation failed: invalid user or under 18")
	})
}

func TestBodyValidatorWithTypedCustom(t *testing.T) {
	app := newTestApp()
	app.Post("/validate-typed-custom", func(call *govalin.Call) {
		var user TestUser

		// Use validation.WithTyped for curryable type-safe validation without manual type casting
		err := validation.WithTyped[TestUser](call.ValidatedBody(&user)).
			Custom(func(user TestUser) bool {
				// Type-safe custom validation - no casting needed!
				// Test complex business rule: name must not contain "banned"
				return user.Name != "banned"
			}, "Name cannot be 'banned'").
			Custom(func(user TestUser) bool {
				// Chain another validation: email domain rules
				if user.Email == "admin@test.com" && user.Age < 21 {
					return false // Admin emails require age 21+
				}
				return true
			}, "Admin emails require age 21 or higher").
			Custom(func(user TestUser) bool {
				// Chain another validation: general minimum age requirement
				return user.Age >= 13
			}, "Must be at least 13 years old").
			Get()

		if err != nil {
			call.Error(err)
			return
		}

		call.JSON(map[string]string{"message": "Curryable typed custom validation passed"})
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		// Test valid user
		validUser := TestUser{Name: "ValidUser", Email: "user@example.com", Age: 25}
		validUserJSON, _ := json.Marshal(validUser)
		response := client.Post("/validate-typed-custom", string(validUserJSON))
		assert.Contains(t, response, "Curryable typed custom validation passed")

		// Test banned user name
		bannedUser := TestUser{Name: "banned", Email: "banned@example.com", Age: 25}
		bannedUserJSON, _ := json.Marshal(bannedUser)
		response = client.Post("/validate-typed-custom", string(bannedUserJSON))
		assert.Contains(t, response, "Name cannot be 'banned'")

		// Test admin email with insufficient age
		adminUser := TestUser{Name: "AdminUser", Email: "admin@test.com", Age: 18}
		adminUserJSON, _ := json.Marshal(adminUser)
		response = client.Post("/validate-typed-custom", string(adminUserJSON))
		assert.Contains(t, response, "Admin emails require age 21 or higher")

		// Test admin email with sufficient age
		validAdminUser := TestUser{Name: "AdminUser", Email: "admin@test.com", Age: 22}
		validAdminUserJSON, _ := json.Marshal(validAdminUser)
		response = client.Post("/validate-typed-custom", string(validAdminUserJSON))
		assert.Contains(t, response, "Curryable typed custom validation passed")

		// Test under minimum age
		youngUser := TestUser{Name: "YoungUser", Email: "young@example.com", Age: 10}
		youngUserJSON, _ := json.Marshal(youngUser)
		response = client.Post("/validate-typed-custom", string(youngUserJSON))
		assert.Contains(t, response, "Must be at least 13 years old")
	})
}

func TestCurryableTypedCustomValidation(t *testing.T) {
	app := newTestApp()
	app.Post("/validate-curryable-typed", func(call *govalin.Call) {
		var user TestUser

		// Demonstrate the curryable API: WithTyped().Custom(...).Custom(...).Custom(...).Get()
		err := validation.WithTyped[TestUser](call.ValidatedBody(&user)).
			Custom(func(u TestUser) bool {
				return len(u.Name) >= 2
			}, "Name must be at least 2 characters").
			Custom(func(u TestUser) bool {
				return u.Age >= 18
			}, "Must be at least 18 years old").
			Custom(func(u TestUser) bool {
				return strings.Contains(u.Email, "@")
			}, "Email must contain @ symbol").
			Get()

		if err != nil {
			call.Error(err)
			return
		}

		call.JSON(map[string]string{"message": "All curryable validations passed"})
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		// Test all validations pass
		validUser := TestUser{Name: "John", Email: "john@example.com", Age: 25}
		validUserJSON, _ := json.Marshal(validUser)
		response := client.Post("/validate-curryable-typed", string(validUserJSON))
		assert.Contains(t, response, "All curryable validations passed")

		// Test first validation fails (name too short)
		invalidUser := TestUser{Name: "A", Email: "a@example.com", Age: 25}
		invalidUserJSON, _ := json.Marshal(invalidUser)
		response = client.Post("/validate-curryable-typed", string(invalidUserJSON))
		assert.Contains(t, response, "Name must be at least 2 characters")

		// Test second validation fails (age too low)
		invalidUser = TestUser{Name: "Alice", Email: "alice@example.com", Age: 16}
		invalidUserJSON, _ = json.Marshal(invalidUser)
		response = client.Post("/validate-curryable-typed", string(invalidUserJSON))
		assert.Contains(t, response, "Must be at least 18 years old")

		// Test third validation fails (no @ in email)
		invalidUser = TestUser{Name: "Bob", Email: "bobexample.com", Age: 25}
		invalidUserJSON, _ = json.Marshal(invalidUser)
		response = client.Post("/validate-curryable-typed", string(invalidUserJSON))
		assert.Contains(t, response, "Email must contain @ symbol")
	})
}
