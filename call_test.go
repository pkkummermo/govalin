package govalin_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/ddliu/go-httpclient"
	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/internal/govalintesting"
	"github.com/pkkummermo/govalin/internal/http/headers"
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

func TestSession(t *testing.T) {
	govalintesting.HTTPTestUtil(func(_ *govalin.App) *govalin.App {
		return govalin.New(func(config *govalin.Config) {
			config.EnableSessions()
		}).Get("/govalin", func(call *govalin.Call) {
			call.Text("govalin")
		})
	}, func(http govalintesting.GovalinHTTP) {
		response := http.GetResponse("/govalin")
		setCookieHeader := response.Header.Get("Set-Cookie")

		assert.NotEmpty(
			t,
			setCookieHeader,
			"Should set session cookie if no session is set",
		)
	})

	govalintesting.HTTPTestUtil(func(_ *govalin.App) *govalin.App {
		return govalin.New(func(config *govalin.Config) {
			config.EnableSessions()
		}).Get("/govalin", func(call *govalin.Call) {
			call.Text("govalin")
		})
	}, func(govalinHttp govalintesting.GovalinHTTP) {
		response, err := govalinHttp.Raw().Begin().WithCookie(&http.Cookie{
			Name:  "govalin-session",
			Value: "non-existent",
		}).Get(govalinHttp.Host + "/govalin")

		setCookieHeader := response.Header.Get("Set-Cookie")

		assert.NoError(t, err, "Request errored")
		assert.NotEmpty(
			t,
			setCookieHeader,
			"Should set session cookie if a session is not found",
		)
	})

	govalintesting.HTTPTestUtil(func(_ *govalin.App) *govalin.App {
		return govalin.New(func(config *govalin.Config) {
			config.EnableSessions()
		}).
			Get("/get", func(call *govalin.Call) {
				call.Text(call.SessionAttrOrDefault("test", "notGovalin").(string))
			}).
			Get("/set", func(call *govalin.Call) {
				_, err := call.SessionAttr("test", "govalin")
				if err != nil {
					call.Error(err)
					return
				}

				call.Status(200)
			})
	}, func(govalinHttp govalintesting.GovalinHTTP) {
		response := govalinHttp.GetResponse("/get")
		body, _ := response.ToString()
		setCookies := response.Cookies()

		assert.Equal(t, 1, len(setCookies), "Should set one cookie")
		assert.Equal(t, 200, response.StatusCode, "Should set status to 200")
		assert.Equal(t, "notGovalin", body, "Should retrieve default value if no session attr is set")

		response = govalinHttp.GetResponse("/set")
		setCookies = response.Cookies()

		assert.Equal(t, 0, len(setCookies), "Should not set cookies when already received one")
		assert.Equal(t, 200, response.StatusCode, "Should set status to 200")

		response = govalinHttp.GetResponse("/get")
		body, _ = response.ToString()

		assert.Equal(t, 200, response.StatusCode, "Should set status to 200")
		assert.Equal(t, "govalin", body, "Should retrieve session attr")

		response, err := govalinHttp.Raw().Begin().WithCookie(&http.Cookie{
			Name:  "govalin-session",
			Value: "invalid-session",
		}).Get(govalinHttp.Host + "/get")

		body, _ = response.ToString()

		assert.NoError(t, err, "Request errored")
		assert.Equal(t, 200, response.StatusCode, "Should set status to 200")
		assert.Equal(t, "notGovalin", body, "Should retrieve default value if session attr is not found")
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

func TestRedirect(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Get("/govalin", func(call *govalin.Call) {
			call.Redirect("http://" + call.Host() + "/govalin2")
		})
		return app
	}, func(http govalintesting.GovalinHTTP) {
		response, err := http.
			Raw().
			Begin().
			WithOption(httpclient.OPT_FOLLOWLOCATION, false).
			Get(http.Host + "/govalin")

		assert.Equal(t, true, httpclient.IsRedirectError(err), fmt.Sprintf("Request was not redirect. Error: %s", err))
		assert.Equal(t, 302, response.StatusCode, "Should redirect with 302")
		assert.Equal(t, http.Host+"/govalin2", response.Header.Get(headers.Location))
	})

	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Get("/govalin", func(call *govalin.Call) {
			call.Redirect("http://"+call.Host()+"/govalin2", true)
		})
		return app
	}, func(http govalintesting.GovalinHTTP) {
		response, err := http.
			Raw().
			Begin().
			WithOption(httpclient.OPT_FOLLOWLOCATION, false).
			Get(http.Host + "/govalin")

		assert.Equal(t, true, httpclient.IsRedirectError(err), fmt.Sprintf("Request was not redirect. Error: %s", err))
		assert.Equal(t, 301, response.StatusCode, "Should redirect with 301")
		assert.Equal(t, http.Host+"/govalin2", response.Header.Get(headers.Location))
	})

	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Get("/govalin", func(call *govalin.Call) {
			call.Redirect("http://" + call.Host() + "/govalin2")
		})
		app.Get("/govalin2", func(call *govalin.Call) {
			call.Text("govalin2")
		})
		return app
	}, func(http govalintesting.GovalinHTTP) {
		response, err := http.
			Raw().
			Begin().
			Get(http.Host + "/govalin")
		body, _ := response.ToString()

		assert.NoError(t, err, fmt.Sprintf("Request errored. Error: %s", err))
		assert.Equal(t, 200, response.StatusCode)
		assert.Equal(t, "govalin2", body)
	})
}
