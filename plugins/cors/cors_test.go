package cors_test

import (
	"testing"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/internal/govalintesting"
	"github.com/pkkummermo/govalin/internal/http/headers"
	"github.com/pkkummermo/govalin/plugins/cors"
	"github.com/stretchr/testify/assert"
)

func TestSimpleAllowAllCorsOrigins(t *testing.T) {
	govalintesting.HTTPTestUtil(func(_ *govalin.App) *govalin.App {
		return govalin.New(func(config *govalin.Config) {
			config.Plugin(cors.New().AllowAllOrigins())
		}).Get("/govalin", func(call *govalin.Call) { call.Text("govalin") })
	}, func(http govalintesting.GovalinHTTP) {
		response, _ := http.Raw().Begin().WithHeader(
			headers.Origin,
			"http://govalin.io",
		).Options(http.Host + "/govalin")

		assert.Equal(t, "http://govalin.io", response.Header.Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, 200, response.StatusCode)
	})
}

func TestSimpleAllowSingleOrigin(t *testing.T) {
	govalintesting.HTTPTestUtil(func(_ *govalin.App) *govalin.App {
		return govalin.New(func(config *govalin.Config) {
			config.Plugin(cors.New().AllowOrigins("http://govalin.io"))
		}).Get("/govalin", func(call *govalin.Call) { call.Text("govalin") })
	}, func(http govalintesting.GovalinHTTP) {
		response, _ := http.Raw().Begin().WithHeader(
			headers.Origin,
			"http://govalin.io",
		).Options(http.Host + "/govalin")
		assert.Equal(t, "http://govalin.io", response.Header.Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, 200, response.StatusCode)

		response, _ = http.Raw().Begin().WithHeader(
			headers.Origin,
			"http://nogovalin.io",
		).Options(http.Host + "/govalin")
		assert.Equal(t, "", response.Header.Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, 200, response.StatusCode)
	})
}

func TestAllowCredentials(t *testing.T) {
	govalintesting.HTTPTestUtil(func(_ *govalin.App) *govalin.App {
		return govalin.New(func(config *govalin.Config) {
			config.Plugin(cors.New().AllowOrigins("http://govalin.io").AllowCredentials(true))
		}).Get("/govalin", func(call *govalin.Call) { call.Text("govalin") })
	}, func(http govalintesting.GovalinHTTP) {
		response, _ := http.Raw().Begin().WithHeader(
			headers.Origin,
			"http://govalin.io",
		).Options(http.Host + "/govalin")
		assert.Equal(t, "http://govalin.io", response.Header.Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, "true", response.Header.Get(headers.AccessControlAllowCredentials))
		assert.Equal(t, 200, response.StatusCode)
	})
}

func TestAllowCredentialsNotPresentUnlessConfigured(t *testing.T) {
	govalintesting.HTTPTestUtil(func(_ *govalin.App) *govalin.App {
		return govalin.New(func(config *govalin.Config) {
			config.Plugin(cors.New().AllowOrigins("http://govalin.io"))
		}).Get("/govalin", func(call *govalin.Call) { call.Text("govalin") })
	}, func(http govalintesting.GovalinHTTP) {
		response, _ := http.Raw().Begin().WithHeader(
			headers.Origin,
			"http://govalin.io",
		).Options(http.Host + "/govalin")
		assert.Equal(t, "http://govalin.io", response.Header.Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, "", response.Header.Get(headers.AccessControlAllowCredentials))
		assert.Equal(t, 200, response.StatusCode)
	})
}

func TestAllowHeaders(t *testing.T) {
	govalintesting.HTTPTestUtil(func(_ *govalin.App) *govalin.App {
		return govalin.New(func(config *govalin.Config) {
			config.Plugin(cors.New().AllowAllOrigins().AllowHeaders("my-special-header"))
		}).Get("/govalin", func(call *govalin.Call) { call.Text("govalin") })
	}, func(http govalintesting.GovalinHTTP) {
		response, _ := http.Raw().Begin().WithHeader(
			headers.Origin,
			"http://govalin.io",
		).Options(http.Host + "/govalin")
		assert.Equal(t, "http://govalin.io", response.Header.Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, "my-special-header", response.Header.Get(headers.AccessControlAllowHeaders))
		assert.Equal(t, "", response.Header.Get(headers.AccessControlAllowCredentials))
		assert.Equal(t, 200, response.StatusCode)
	})
}

func TestShouldHaveDefaultsEnabledForSimpleConfiguration(t *testing.T) {
	govalintesting.HTTPTestUtil(func(_ *govalin.App) *govalin.App {
		return govalin.New(func(config *govalin.Config) {
			config.Plugin(cors.New().AllowAllOrigins())
		}).Get("/govalin", func(call *govalin.Call) { call.Text("govalin") })
	}, func(http govalintesting.GovalinHTTP) {
		response, _ := http.Raw().Begin().WithHeader(
			headers.Origin,
			"http://govalin.io",
		).Options(http.Host + "/govalin")
		assert.Equal(t, "http://govalin.io", response.Header.Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, "*", response.Header.Get(headers.AccessControlAllowHeaders))
		assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", response.Header.Get(headers.AccessControlAllowMethods))
		assert.Equal(t, 200, response.StatusCode)
	})
}

func TestNotFoundHandlerStillTriggers(t *testing.T) {
	govalintesting.HTTPTestUtil(func(_ *govalin.App) *govalin.App {
		return govalin.New(func(config *govalin.Config) {
			config.Plugin(cors.New().AllowAllOrigins())
		}).Get("/govalin", func(call *govalin.Call) { call.Text("govalin") })
	}, func(http govalintesting.GovalinHTTP) {
		response, _ := http.Raw().Get(http.Host + "/nonexisting")
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
