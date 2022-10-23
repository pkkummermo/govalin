package govalin_test

import (
	"testing"

	"github.com/pkkummermo/govalin/pkg/govalin"
	"github.com/pkkummermo/govalin/pkg/internal/govalintesting"
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
