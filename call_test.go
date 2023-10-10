package govalin_test

import (
	"net/http"
	"testing"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/internal/govalintesting"
	"github.com/stretchr/testify/assert"
)

func TestQueryParam(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Get("/query", func(call *govalin.Call) {
			call.Text(call.QueryParam("foo"))
		})
		app.Get("/default", func(call *govalin.Call) {
			call.Text(call.QueryParamOrDefault("foo", "notGovalin"))
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		assert.Equal(
			t,
			"govalin",
			http.Get("/query?foo=govalin"),
			"Should retrieve query param",
		)
		assert.Equal(
			t,
			"govalin",
			http.Get("/default?foo=govalin"),
			"Should retrieve query param if present using default",
		)
		assert.Equal(
			t,
			"notGovalin",
			http.Get("/default"),
			"Should retrieve default if query param not present",
		)
	})
}

func TestPathParams(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Get("/simple/{org}", func(call *govalin.Call) {
			call.Text(call.PathParam("org"))
		}).Get("/multiple/{org}/{repo}", func(call *govalin.Call) {
			call.Text(call.PathParam("org") + call.PathParam("repo"))
		}).Get("/wildcard/*/{repo}", func(call *govalin.Call) {
			call.Text(call.PathParam("repo"))
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		assert.Equal(
			t,
			http.Get("/simple/govalin"),
			"govalin",
			"Should correctly parse simple path params",
		)
		assert.Equal(
			t,
			http.Get("/multiple/govalin/govalin"),
			"govalingovalin",
			"Should correctly parse multiple path params",
		)
		assert.Equal(
			t,
			http.Get("/wildcard/whatever/govalin"),
			"govalin",
			"Should correctly parse wildcard path with params",
		)
	})
}

func TestHeaders(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Get("/headers", func(call *govalin.Call) {
			call.Text(call.Header("test-header"))
		})
		return app
	}, func(http govalintesting.GovalinHTTP) {
		response, _ := http.Raw().WithHeader("test-header", "govalin").Get(http.Host + "/headers")
		body, _ := response.ToString()

		assert.Equal(
			t,
			body,
			"govalin",
			"Should parse and return given non-canonical header",
		)

		response, _ = http.Raw().WithHeader("Test-Header", "govalin").Get(http.Host + "/headers")
		body, _ = response.ToString()

		assert.Equal(
			t,
			body,
			"govalin",
			"Should parse and return given header when given canonical header",
		)
	})

	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Get("/headers", func(call *govalin.Call) {
			call.Header("test-header", "govalin")
		})
		return app
	}, func(http govalintesting.GovalinHTTP) {
		assert.Equal(
			t,
			http.GetResponse("/headers").Header.Get("Test-Header"),
			"govalin",
			"Should parse and return given header when given canonical header",
		)
	})
}

func TestCookies(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Get("/cookies", func(call *govalin.Call) {
			govalinCookie, err := call.Cookie("govalin")

			if err != nil {
				call.Text(err.Error())
				return
			}

			call.Text(govalinCookie.Name)
		})
		return app
	}, func(http govalintesting.GovalinHTTP) {
		response, _ := http.Raw().WithHeader("Cookie", "govalin=govalin").Get(http.Host + "/cookies")
		body, _ := response.ToString()

		assert.Equal(
			t,
			body,
			"govalin",
			"Should parse and return given cookie value",
		)
	})

	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Get("/setcookies", func(call *govalin.Call) {
			_, err := call.Cookie("govalin", &http.Cookie{Value: "govalin"})

			if err != nil {
				call.Text(err.Error())
				return
			}

			call.Status(204)
		})
		return app
	}, func(http govalintesting.GovalinHTTP) {
		response, _ := http.Raw().Get(http.Host + "/setcookies")
		setCookieHeader := response.Header.Get("Set-Cookie")

		assert.Equal(
			t,
			setCookieHeader,
			"govalin=govalin",
			"Should set correct header when setting cookie",
		)
	})
}

func TestRequestID(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Get("/govalin", func(call *govalin.Call) {
			call.Text(call.ID())
		})
		return app
	}, func(http govalintesting.GovalinHTTP) {
		body := http.Get("/govalin")

		assert.NotEmpty(
			t,
			body,
			"Should generate a unique request ID",
		)
	})

	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Get("/govalin", func(call *govalin.Call) {
			call.Text(call.ID())
		})
		return app
	}, func(http govalintesting.GovalinHTTP) {
		response, _ := http.Raw().WithHeader("X-Govalin-Id", "govalin").Get(http.Host + "/govalin")
		govalinID, _ := response.ToString()

		assert.Equal(
			t,
			govalinID,
			"govalin",
			"Should reuse given ID",
		)
	})
}
