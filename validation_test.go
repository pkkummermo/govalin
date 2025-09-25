package govalin_test

import (
	"encoding/json"
	"strings"
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

func TestValidateString(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Post("/validate-string", func(call *govalin.Call) {
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
		response := http.Post("/validate-string?name=John", map[string]string{})
		assert.Contains(t, response, "Valid name")
		assert.Contains(t, response, "John")

		// Test empty string (should fail Required)
		response = http.Post("/validate-string?name=", map[string]string{})
		assert.Contains(t, response, "This field is required")

		// Test too short (should fail MinLength)
		response = http.Post("/validate-string?name=Jo", map[string]string{})
		assert.Contains(t, response, "Must be at least 3 characters long")

		// Test too long (should fail MaxLength)
		response = http.Post("/validate-string?name=ThisNameIsTooLongForOurValidation", map[string]string{})
		assert.Contains(t, response, "Must be at most 20 characters long")
	})
}

func TestValidateInteger(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Post("/validate-int", func(call *govalin.Call) {
			ageStr := call.QueryParam("age")
			
			// First validate it can be converted to int and apply int validations
			intValidator := govalin.Validate[int]().
				Rule(govalin.Min(18)).
				Rule(govalin.Max(100))
			
			if err := govalin.ValidateStringAsInt(ageStr, "age", intValidator); err != nil {
				call.Error(err)
				return
			}
			
			call.JSON(map[string]string{"message": "Valid age", "age": ageStr})
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

func TestValidateEmail(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Post("/validate-email", func(call *govalin.Call) {
			email := call.QueryParam("email")
			
			validator := govalin.Validate[string]().
				Rule(govalin.Required()).
				Rule(govalin.Email())
			
			if err := validator.Validate(email, "email"); err != nil {
				call.Error(err)
				return
			}
			
			call.JSON(map[string]string{"message": "Valid email", "email": email})
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		// Test valid email
		response := http.Post("/validate-email?email=test@example.com", map[string]string{})
		assert.Contains(t, response, "Valid email")

		// Test invalid email
		response = http.Post("/validate-email?email=invalidemail", map[string]string{})
		assert.Contains(t, response, "Must be a valid email address")

		// Test empty email
		response = http.Post("/validate-email?email=", map[string]string{})
		assert.Contains(t, response, "This field is required")
	})
}

func TestValidateStruct(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Post("/validate-struct", func(call *govalin.Call) {
			var user TestUser
			if err := call.BodyAs(&user); err != nil {
				call.Error(err)
				return
			}
			
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
			
			if err := validator.Validate(user); err != nil {
				call.Error(err)
				return
			}
			
			call.JSON(map[string]string{"message": "Valid user data"})
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		// Test valid user
		validUser := TestUser{Name: "John Doe", Email: "john@example.com", Age: 25}
		validUserJSON, _ := json.Marshal(validUser)
		response := http.Post("/validate-struct", string(validUserJSON))
		assert.Contains(t, response, "Valid user data")

		// Test invalid name (too short)
		invalidUser := TestUser{Name: "J", Email: "john@example.com", Age: 25}
		invalidUserJSON, _ := json.Marshal(invalidUser)
		response = http.Post("/validate-struct", string(invalidUserJSON))
		assert.Contains(t, response, "Must be at least 2 characters long")

		// Test invalid email
		invalidUser = TestUser{Name: "John Doe", Email: "invalidemail", Age: 25}
		invalidUserJSON, _ = json.Marshal(invalidUser)
		response = http.Post("/validate-struct", string(invalidUserJSON))
		assert.Contains(t, response, "Must be a valid email address")

		// Test invalid age
		invalidUser = TestUser{Name: "John Doe", Email: "john@example.com", Age: 15}
		invalidUserJSON, _ = json.Marshal(invalidUser)
		response = http.Post("/validate-struct", string(invalidUserJSON))
		assert.Contains(t, response, "Must be at least 18")
	})
}

func TestValidateCustom(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Post("/validate-custom", func(call *govalin.Call) {
			password := call.QueryParam("password")
			
			validator := govalin.Validate[string]().
				Rule(govalin.Required()).
				Rule(govalin.MinLength(8)).
				Rule(govalin.Custom(func(s string) bool {
					return strings.ContainsAny(s, "0123456789")
				}, "Password must contain at least one digit"))
			
			if err := validator.Validate(password, "password"); err != nil {
				call.Error(err)
				return
			}
			
			call.JSON(map[string]string{"message": "Valid password"})
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		// Test valid password
		response := http.Post("/validate-custom?password=mypassword123", map[string]string{})
		assert.Contains(t, response, "Valid password")

		// Test password without digit
		response = http.Post("/validate-custom?password=mypassword", map[string]string{})
		assert.Contains(t, response, "Password must contain at least one digit")

		// Test too short password
		response = http.Post("/validate-custom?password=pass1", map[string]string{})
		assert.Contains(t, response, "Must be at least 8 characters long")
	})
}

func TestValidateChaining(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Post("/validate-chain", func(call *govalin.Call) {
			username := call.QueryParam("username")
			
			// Demonstrate chaining multiple rules
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
			
			if err := validator.Validate(username, "username"); err != nil {
				call.Error(err)
				return
			}
			
			call.JSON(map[string]string{"message": "Valid username", "username": username})
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		// Test valid username
		response := http.Post("/validate-chain?username=john123", map[string]string{})
		assert.Contains(t, response, "Valid username")
		assert.Contains(t, response, "john123")

		// Test username with special characters
		response = http.Post("/validate-chain?username=john@123", map[string]string{})
		assert.Contains(t, response, "Username must contain only alphanumeric characters")
	})
}