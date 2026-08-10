package cors_test

import (
	"net/http"
	"testing"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/govalintest"
	"github.com/pkkummermo/govalin/internal/govalintesting"
	"github.com/pkkummermo/govalin/internal/http/headers"
	"github.com/pkkummermo/govalin/plugins/cors"
	"github.com/stretchr/testify/assert"
)

func TestSimpleAllowAllCorsOrigins(t *testing.T) {
	app := govalin.New(func(config *govalin.Config) {
		config.Plugin(cors.New().AllowAllOrigins())
	}).Get("/govalin", func(call *govalin.Call) { call.Text("govalin") })

	govalintest.Test(t, app, func(client *govalintest.Client) {
		req, err := http.NewRequest(http.MethodOptions, "/govalin", nil)
		assert.Nil(t, err)
		req.Header.Set(headers.Origin, "http://govalin.io")
		response := client.Do(req)
		defer func() { _ = response.Body.Close() }()

		assert.Equal(t, "http://govalin.io", response.Header.Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, 200, response.StatusCode)
	})
}

func TestSimpleAllowSingleOrigin(t *testing.T) {
	app := govalin.New(func(config *govalin.Config) {
		config.Plugin(cors.New().AllowOrigins("http://govalin.io"))
	}).Get("/govalin", func(call *govalin.Call) { call.Text("govalin") })

	govalintest.Test(t, app, func(client *govalintest.Client) {
		req, err := http.NewRequest(http.MethodOptions, "/govalin", nil)
		assert.Nil(t, err)
		req.Header.Set(headers.Origin, "http://govalin.io")
		response := client.Do(req)
		defer func() { _ = response.Body.Close() }()
		assert.Equal(t, "http://govalin.io", response.Header.Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, 200, response.StatusCode)

		req, err = http.NewRequest(http.MethodOptions, "/govalin", nil)
		assert.Nil(t, err)
		req.Header.Set(headers.Origin, "http://nogovalin.io")
		response = client.Do(req)
		defer func() { _ = response.Body.Close() }()
		assert.Equal(t, "", response.Header.Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, 200, response.StatusCode)
	})
}

func TestAllowCredentials(t *testing.T) {
	app := govalin.New(func(config *govalin.Config) {
		config.Plugin(cors.New().AllowOrigins("http://govalin.io").AllowCredentials(true))
	}).Get("/govalin", func(call *govalin.Call) { call.Text("govalin") })

	govalintest.Test(t, app, func(client *govalintest.Client) {
		req, err := http.NewRequest(http.MethodOptions, "/govalin", nil)
		assert.Nil(t, err)
		req.Header.Set(headers.Origin, "http://govalin.io")
		response := client.Do(req)
		defer func() { _ = response.Body.Close() }()
		assert.Equal(t, "http://govalin.io", response.Header.Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, "true", response.Header.Get(headers.AccessControlAllowCredentials))
		assert.Equal(t, 200, response.StatusCode)
	})
}

func TestAllowCredentialsNotPresentUnlessConfigured(t *testing.T) {
	app := govalin.New(func(config *govalin.Config) {
		config.Plugin(cors.New().AllowOrigins("http://govalin.io"))
	}).Get("/govalin", func(call *govalin.Call) { call.Text("govalin") })

	govalintest.Test(t, app, func(client *govalintest.Client) {
		req, err := http.NewRequest(http.MethodOptions, "/govalin", nil)
		assert.Nil(t, err)
		req.Header.Set(headers.Origin, "http://govalin.io")
		response := client.Do(req)
		defer func() { _ = response.Body.Close() }()
		assert.Equal(t, "http://govalin.io", response.Header.Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, "", response.Header.Get(headers.AccessControlAllowCredentials))
		assert.Equal(t, 200, response.StatusCode)
	})
}

func TestAllowHeaders(t *testing.T) {
	app := govalin.New(func(config *govalin.Config) {
		config.Plugin(cors.New().AllowAllOrigins().AllowHeaders("my-special-header"))
	}).Get("/govalin", func(call *govalin.Call) { call.Text("govalin") })

	govalintest.Test(t, app, func(client *govalintest.Client) {
		req, err := http.NewRequest(http.MethodOptions, "/govalin", nil)
		assert.Nil(t, err)
		req.Header.Set(headers.Origin, "http://govalin.io")
		response := client.Do(req)
		defer func() { _ = response.Body.Close() }()
		assert.Equal(t, "http://govalin.io", response.Header.Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, "my-special-header", response.Header.Get(headers.AccessControlAllowHeaders))
		assert.Equal(t, "", response.Header.Get(headers.AccessControlAllowCredentials))
		assert.Equal(t, 200, response.StatusCode)
	})
}

func TestShouldHaveDefaultsEnabledForSimpleConfiguration(t *testing.T) {
	app := govalin.New(func(config *govalin.Config) {
		config.Plugin(cors.New().AllowAllOrigins())
	}).Get("/govalin", func(call *govalin.Call) { call.Text("govalin") })

	govalintest.Test(t, app, func(client *govalintest.Client) {
		req, err := http.NewRequest(http.MethodOptions, "/govalin", nil)
		assert.Nil(t, err)
		req.Header.Set(headers.Origin, "http://govalin.io")
		response := client.Do(req)
		defer func() { _ = response.Body.Close() }()
		assert.Equal(t, "http://govalin.io", response.Header.Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, "*", response.Header.Get(headers.AccessControlAllowHeaders))
		assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", response.Header.Get(headers.AccessControlAllowMethods))
		assert.Equal(t, 200, response.StatusCode)
	})
}

func TestVaryOnOriginOnPreflightAndActualRequest(t *testing.T) {
	app := govalin.New(func(config *govalin.Config) {
		config.Plugin(cors.New().AllowAllOrigins())
	}).Get("/govalin", func(call *govalin.Call) { call.Text("govalin") })

	govalintest.Test(t, app, func(client *govalintest.Client) {
		preflight, err := http.NewRequest(http.MethodOptions, "/govalin", nil)
		assert.Nil(t, err)
		preflight.Header.Set(headers.Origin, "http://govalin.io")
		preflight.Header.Set(headers.AccessControlRequestMethod, http.MethodGet)
		preflight.Header.Set(headers.AccessControlRequestHeaders, "my-special-header")
		response := client.Do(preflight)
		defer func() { _ = response.Body.Close() }()

		assert.Equal(t, []string{headers.Origin}, response.Header.Values(headers.Vary))

		actual, err := http.NewRequest(http.MethodGet, "/govalin", nil)
		assert.Nil(t, err)
		actual.Header.Set(headers.Origin, "http://govalin.io")
		response = client.Do(actual)
		defer func() { _ = response.Body.Close() }()

		assert.Equal(t, []string{headers.Origin}, response.Header.Values(headers.Vary))
	})
}

func TestVaryOnOriginWhenOriginIsNotAllowed(t *testing.T) {
	app := govalin.New(func(config *govalin.Config) {
		config.Plugin(cors.New().AllowOrigins("http://govalin.io"))
	}).Get("/govalin", func(call *govalin.Call) { call.Text("govalin") })

	govalintest.Test(t, app, func(client *govalintest.Client) {
		req, err := http.NewRequest(http.MethodGet, "/govalin", nil)
		assert.Nil(t, err)
		req.Header.Set(headers.Origin, "http://nogovalin.io")
		response := client.Do(req)
		defer func() { _ = response.Body.Close() }()

		assert.Equal(t, "", response.Header.Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, []string{headers.Origin}, response.Header.Values(headers.Vary))
	})
}

func TestVaryOnOriginWhenRequestCarriesNoOrigin(t *testing.T) {
	app := govalin.New(func(config *govalin.Config) {
		config.Plugin(cors.New().AllowAllOrigins())
	}).Get("/govalin", func(call *govalin.Call) { call.Text("govalin") })

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/govalin")
		defer func() { _ = response.Body.Close() }()

		assert.Equal(t, []string{headers.Origin}, response.Header.Values(headers.Vary))
	})
}

// TestVaryOnOriginJoinsAHandlersOwnSelectingHeader covers the layering the
// plugin sits in: the route below it negotiates on a header of its own, and
// neither claim on the response may cost the other.
func TestVaryOnOriginJoinsAHandlersOwnSelectingHeader(t *testing.T) {
	app := govalin.New(func(config *govalin.Config) {
		config.Plugin(cors.New().AllowAllOrigins())
	}).Get("/govalin", func(call *govalin.Call) {
		call.VaryOn(headers.AcceptLanguage)
		call.Text("govalin")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/govalin")
		defer func() { _ = response.Body.Close() }()

		assert.Equal(t, []string{"Origin, Accept-Language"}, response.Header.Values(headers.Vary))
	})
}

func TestNoCorsHeadersWhenRequestCarriesNoOrigin(t *testing.T) {
	app := govalin.New(func(config *govalin.Config) {
		config.Plugin(cors.New().AllowAllOrigins())
	}).Get("/govalin", func(call *govalin.Call) { call.Text("govalin") })

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/govalin")
		defer func() { _ = response.Body.Close() }()

		assert.Empty(t, response.Header.Values(headers.AccessControlAllowOrigin))
		assert.Empty(t, response.Header.Values(headers.AccessControlAllowHeaders))
		assert.Empty(t, response.Header.Values(headers.AccessControlAllowMethods))
	})
}

func TestNotFoundHandlerStillTriggers(t *testing.T) {
	app := govalin.New(func(config *govalin.Config) {
		config.Plugin(cors.New().AllowAllOrigins())
	}).Get("/govalin", func(call *govalin.Call) { call.Text("govalin") })

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/nonexisting")
		defer func() { _ = response.Body.Close() }()
		assert.Equal(t, 404, response.StatusCode)
	})
}

func TestShouldExitOnAddNullOrigin(t *testing.T) {
	exitCode, output := govalintesting.TestExit(
		t,
		func() {
			govalin.New(func(config *govalin.Config) {
				config.Plugin(
					cors.New().AllowOrigins("null"),
				)
			})
		},
	)

	assert.Contains(t, output, "You should never allow the null origin in your CORS config")
	assert.Equal(t, 1, exitCode)
}

func TestShouldExitOnAllowCredentialsAndWildcard(t *testing.T) {
	exitCode, output := govalintesting.TestExit(
		t,
		func() {
			govalin.New(func(config *govalin.Config) {
				config.Plugin(
					cors.New().
						AllowOrigins("*").
						AllowCredentials(true),
				)
			})
		},
	)

	assert.Contains(t, output, "CORS plugin has been configured to allow credentials while having a wildcard in allowed origins") //nolint:lll // test
	assert.Equal(t, 1, exitCode)
}

func TestShouldNotExitOnCorrectConfig(t *testing.T) {
	exitCode, _ := govalintesting.TestExit(
		t,
		func() {
			govalin.New(func(config *govalin.Config) {
				config.Plugin(
					cors.New().AllowAllOrigins(),
				)
			})
		},
	)

	assert.Equal(t, 0, exitCode)
}
