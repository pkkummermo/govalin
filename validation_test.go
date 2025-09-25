package govalin_test

import (
	"encoding/json"
	"testing"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/internal/govalintesting"
	"github.com/pkkummermo/govalin/validation"
	"github.com/stretchr/testify/assert"
)

type TestUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func TestValidatedQueryParam(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
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

		return app
	}, func(http govalintesting.GovalinHTTP) {
		// Test valid input
		response := http.Post("/validate-query?name=John", map[string]string{})
		assert.Contains(t, response, "Valid name")
		assert.Contains(t, response, "John")

		// Test empty string (should fail Required)
		response = http.Post("/validate-query?name=", map[string]string{})
		assert.Contains(t, response, "This field is required")

		// Test too short (should fail MinLength)
		response = http.Post("/validate-query?name=Jo", map[string]string{})
		assert.Contains(t, response, "Must be at least 3 characters long")

		// Test too long (should fail MaxLength)
		response = http.Post("/validate-query?name=ThisNameIsTooLongForOurValidation", map[string]string{})
		assert.Contains(t, response, "Must be at most 20 characters long")
	})
}

func TestValidatedPathParam(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Post("/validate-path/{username}", func(call *govalin.Call) {
			username, err := call.ValidatedPathParam("username").
				Required().
				MinLength(3).
				MaxLength(15).
				Custom(func(s string) bool {
					// Username should only contain alphanumeric characters
					for _, r := range s {
						if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
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

		return app
	}, func(http govalintesting.GovalinHTTP) {
		// Test valid username
		response := http.Post("/validate-path/john123", map[string]string{})
		assert.Contains(t, response, "Valid username")
		assert.Contains(t, response, "john123")

		// Test username with special characters
		response = http.Post("/validate-path/john@123", map[string]string{})
		assert.Contains(t, response, "Username must contain only alphanumeric characters")
	})
}

func TestValidatedQueryParamAsInt(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
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

		return app
	}, func(http govalintesting.GovalinHTTP) {
		// Test valid input
		response := http.Post("/validate-int?age=25", map[string]string{})
		assert.Contains(t, response, "Valid age")

		// Test invalid integer
		response = http.Post("/validate-int?age=notanumber", map[string]string{})
		assert.Contains(t, response, "Must be a valid integer")

		// Test too low
		response = http.Post("/validate-int?age=15", map[string]string{})
		assert.Contains(t, response, "Must be at least 18")

		// Test too high
		response = http.Post("/validate-int?age=150", map[string]string{})
		assert.Contains(t, response, "Must be at most 100")
	})
}

func TestValidatedFormParam(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
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

		return app
	}, func(http govalintesting.GovalinHTTP) {
		// Test valid email
		response := http.Post("/validate-form", map[string]string{"email": "test@example.com"})
		assert.Contains(t, response, "Valid email")

		// Test invalid email
		response = http.Post("/validate-form", map[string]string{"email": "invalidemail"})
		assert.Contains(t, response, "Must be a valid email address")

		// Test empty email
		response = http.Post("/validate-form", map[string]string{"email": ""})
		assert.Contains(t, response, "This field is required")
	})
}

func TestValidatedBody(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Post("/validate-body", func(call *govalin.Call) {
			var user TestUser
			
			// Parse the body first
			if err := call.ValidatedBody(&user).Get(); err != nil {
				call.Error(err)
				return
			}
			
			// Now validate the parsed user data using proper validation
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
			
			call.JSON(map[string]interface{}{"message": "Valid user data", "user": user})
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		// Test valid user
		validUser := TestUser{Name: "John Doe", Email: "john@example.com", Age: 25}
		validUserJSON, _ := json.Marshal(validUser)
		response := http.Post("/validate-body", string(validUserJSON))
		assert.Contains(t, response, "Valid user data")

		// Test invalid name (too short)
		invalidUser := TestUser{Name: "J", Email: "john@example.com", Age: 25}
		invalidUserJSON, _ := json.Marshal(invalidUser)
		response = http.Post("/validate-body", string(invalidUserJSON))
		assert.Contains(t, response, "Must be at least 2 characters long")

		// Test invalid email
		invalidUser = TestUser{Name: "John Doe", Email: "invalidemail", Age: 25}
		invalidUserJSON, _ = json.Marshal(invalidUser)
		response = http.Post("/validate-body", string(invalidUserJSON))
		assert.Contains(t, response, "Must be a valid email address")

		// Test invalid age
		invalidUser = TestUser{Name: "John Doe", Email: "john@example.com", Age: 15}
		invalidUserJSON, _ = json.Marshal(invalidUser)
		response = http.Post("/validate-body", string(invalidUserJSON))
		assert.Contains(t, response, "Must be at least 18")
	})
}

func TestChainingValidation(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Post("/validate-chain", func(call *govalin.Call) {
			// Demonstrate complex chaining
			username, err := call.ValidatedQueryParam("username").
				Required().
				MinLength(3).
				MaxLength(15).
				Custom(func(s string) bool {
					// Username should only contain alphanumeric characters
					for _, r := range s {
						if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
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
				"message": "Valid data", 
				"username": username,
				"age": age,
			})
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		// Test valid input
		response := http.Post("/validate-chain?username=john123&age=25", map[string]string{})
		assert.Contains(t, response, "Valid data")
		assert.Contains(t, response, "john123")

		// Test custom validation failure
		response = http.Post("/validate-chain?username=john123&age=42", map[string]string{})
		assert.Contains(t, response, "Age cannot be 42")
	})
}

func TestPublicValidationAPI(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
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

		return app
	}, func(http govalintesting.GovalinHTTP) {
		// Test valid input
		response := http.Post("/validate-public?name=John", map[string]string{})
		assert.Contains(t, response, "Valid name")

		// Test custom validation failure
		response = http.Post("/validate-public?name=john", map[string]string{})
		assert.Contains(t, response, "Name must start with an uppercase letter")
	})
}