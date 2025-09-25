package govalin_test

import (
	"encoding/json"
	"testing"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/internal/govalintesting"
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
			validator := govalin.Validate[string]().
				Rule(govalin.Required()).
				Rule(govalin.MinLength(3)).
				Rule(govalin.MaxLength(20))
			
			name, err := call.ValidatedQueryParam("name", validator)
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
			validator := govalin.Validate[string]().
				Rule(govalin.Required()).
				Rule(govalin.MinLength(3)).
				Rule(govalin.MaxLength(15)).
				Rule(govalin.Custom(func(s string) bool {
					// Username should only contain alphanumeric characters
					for _, r := range s {
						if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
							return false
						}
					}
					return true
				}, "Username must contain only alphanumeric characters"))
			
			username, err := call.ValidatedPathParam("username", validator)
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
			validator := govalin.Validate[int]().
				Rule(govalin.Min(18)).
				Rule(govalin.Max(100))
			
			age, err := call.ValidatedQueryParamAsInt("age", validator)
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

func TestValidatedBody(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Post("/validate-body", func(call *govalin.Call) {
			var user TestUser
			
			validator := govalin.ValidateStruct().
				Field("Name", func(v interface{}) *govalin.ValidationError {
					return govalin.Validate[string]().
						Rule(govalin.Required()).
						Rule(govalin.MinLength(2)).
						Validate(v.(string), "Name")
				}).
				Field("Email", func(v interface{}) *govalin.ValidationError {
					return govalin.Validate[string]().
						Rule(govalin.Required()).
						Rule(govalin.Email()).
						Validate(v.(string), "Email")
				}).
				Field("Age", func(v interface{}) *govalin.ValidationError {
					return govalin.Validate[int]().
						Rule(govalin.Min(18)).
						Rule(govalin.Max(100)).
						Validate(v.(int), "Age")
				})
			
			if err := call.ValidatedBody(&user, validator); err != nil {
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

func TestValidatedFormParam(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Post("/validate-form", func(call *govalin.Call) {
			validator := govalin.Validate[string]().
				Rule(govalin.Required()).
				Rule(govalin.Email())
			
			email, err := call.ValidatedFormParam("email", validator)
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

// Keep one example showing the old API for comparison
func TestLegacyStandaloneValidation(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Post("/validate-legacy", func(call *govalin.Call) {
			name := call.QueryParam("name")
			
			validator := govalin.Validate[string]().
				Rule(govalin.Required()).
				Rule(govalin.MinLength(3)).
				Rule(govalin.MaxLength(20))
			
			if err := validator.Validate(name, "name"); err != nil {
				call.Error(err)
				return
			}
			
			call.JSON(map[string]string{"message": "Valid name", "name": name})
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		// Test valid input
		response := http.Post("/validate-legacy?name=John", map[string]string{})
		assert.Contains(t, response, "Valid name")
		assert.Contains(t, response, "John")
	})
}