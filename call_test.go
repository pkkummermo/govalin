package govalin_test

import (
	"net/http"
	"net/http/cookiejar"
	"testing"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/govalintest"
	"github.com/pkkummermo/govalin/internal/http/headers"
	"github.com/stretchr/testify/assert"
)

// findCookie returns the first cookie matching name, or nil if none is present.
func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}

	return nil
}

func TestQueryParam(t *testing.T) {
	app := newTestApp()
	app.Get("/query", func(call *govalin.Call) {
		call.Text(call.QueryParam("foo"))
	})
	app.Get("/default", func(call *govalin.Call) {
		call.Text(call.QueryParamOrDefault("foo", "notGovalin"))
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		assert.Equal(
			t,
			"govalin",
			client.Get("/query?foo=govalin"),
			"Should retrieve query param",
		)
		assert.Equal(
			t,
			"govalin",
			client.Get("/default?foo=govalin"),
			"Should retrieve query param if present using default",
		)
		assert.Equal(
			t,
			"notGovalin",
			client.Get("/default"),
			"Should retrieve default if query param not present",
		)
	})
}

func TestPathParams(t *testing.T) {
	app := newTestApp()
	app.Get("/simple/{org}", func(call *govalin.Call) {
		call.Text(call.PathParam("org"))
	}).Get("/multiple/{org}/{repo}", func(call *govalin.Call) {
		call.Text(call.PathParam("org") + call.PathParam("repo"))
	}).Get("/wildcard/*/{repo}", func(call *govalin.Call) {
		call.Text(call.PathParam("repo"))
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		assert.Equal(
			t,
			client.Get("/simple/govalin"),
			"govalin",
			"Should correctly parse simple path params",
		)
		assert.Equal(
			t,
			client.Get("/multiple/govalin/govalin"),
			"govalingovalin",
			"Should correctly parse multiple path params",
		)
		assert.Equal(
			t,
			client.Get("/wildcard/whatever/govalin"),
			"govalin",
			"Should correctly parse wildcard path with params",
		)
	})
}

func TestHeaders(t *testing.T) {
	app := newTestApp()
	app.Get("/headers", func(call *govalin.Call) {
		call.Text(call.Header("test-header"))
	})
	govalintest.Test(t, app, func(client *govalintest.Client) {
		req, err := http.NewRequest(http.MethodGet, "/headers", nil)
		assert.Nil(t, err)
		req.Header.Set("test-header", "govalin")
		body := readBody(t, client.Do(req))

		assert.Equal(
			t,
			body,
			"govalin",
			"Should parse and return given non-canonical header",
		)

		req, err = http.NewRequest(http.MethodGet, "/headers", nil)
		assert.Nil(t, err)
		req.Header.Set("Test-Header", "govalin")
		body = readBody(t, client.Do(req))

		assert.Equal(
			t,
			body,
			"govalin",
			"Should parse and return given header when given canonical header",
		)
	})

	app = newTestApp()
	app.Get("/headers", func(call *govalin.Call) {
		call.Header("test-header", "govalin")
	})
	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/headers")
		defer func() { _ = response.Body.Close() }()

		assert.Equal(
			t,
			response.Header.Get("Test-Header"),
			"govalin",
			"Should parse and return given header when given canonical header",
		)
	})
}

func TestCookies(t *testing.T) {
	app := newTestApp()
	app.Get("/cookies", func(call *govalin.Call) {
		govalinCookie, err := call.Cookie("govalin")
		if err != nil {
			call.Text(err.Error())
			return
		}

		call.Text(govalinCookie.Name)
	})
	govalintest.Test(t, app, func(client *govalintest.Client) {
		req, err := http.NewRequest(http.MethodGet, "/cookies", nil)
		assert.Nil(t, err)
		req.Header.Set("Cookie", "govalin=govalin")
		body := readBody(t, client.Do(req))

		assert.Equal(
			t,
			body,
			"govalin",
			"Should parse and return given cookie value",
		)
	})

	app = newTestApp()
	app.Get("/setcookies", func(call *govalin.Call) {
		_, err := call.Cookie("govalin", &http.Cookie{Value: "govalin"})
		if err != nil {
			call.Text(err.Error())
			return
		}

		call.Status(204)
	})
	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/setcookies")
		defer func() { _ = response.Body.Close() }()
		setCookieHeader := response.Header.Get("Set-Cookie")

		assert.Equal(
			t,
			setCookieHeader,
			"govalin=govalin",
			"Should set correct header when setting cookie",
		)
		assert.Equal(
			t,
			204,
			response.StatusCode,
			"Should send the status set by the handler even with no body",
		)
	})
}

func TestStatusOnlyResponse(t *testing.T) {
	app := newTestApp()
	app.Get("/nobody", func(call *govalin.Call) {
		call.Status(204)
	})
	app.Before("/guarded", func(call *govalin.Call) bool {
		call.Status(401)
		return false
	})
	app.Get("/guarded", func(call *govalin.Call) {
		call.Text("should never run")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		nobodyResponse := client.GetResponse("/nobody")
		defer func() { _ = nobodyResponse.Body.Close() }()
		assert.Equal(
			t,
			204,
			nobodyResponse.StatusCode,
			"A handler that sets a status but writes no body should send that status",
		)

		guardedResponse := client.GetResponse("/guarded")
		defer func() { _ = guardedResponse.Body.Close() }()
		assert.Equal(
			t,
			401,
			guardedResponse.StatusCode,
			"A before handler that short-circuits with a status but no body should send that status",
		)
	})
}

func TestSession(t *testing.T) {
	app := govalin.New(func(config *govalin.Config) {
		config.EnableSessions()
	}).Get("/govalin", func(call *govalin.Call) {
		call.Text("govalin")
	})
	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/govalin")
		defer func() { _ = response.Body.Close() }()
		setCookieHeader := response.Header.Get("Set-Cookie")

		assert.NotEmpty(
			t,
			setCookieHeader,
			"Should set session cookie if no session is set",
		)
	})

	app = govalin.New(func(config *govalin.Config) {
		config.EnableSessions()
	}).Get("/govalin", func(call *govalin.Call) {
		call.Text("govalin")
	})
	govalintest.Test(t, app, func(client *govalintest.Client) {
		req, err := http.NewRequest(http.MethodGet, "/govalin", nil)
		assert.Nil(t, err)
		req.AddCookie(&http.Cookie{
			Name:  "govalin-session",
			Value: "non-existent",
		})
		response := client.Do(req)
		defer func() { _ = response.Body.Close() }()

		setCookieHeader := response.Header.Get("Set-Cookie")

		assert.NotEmpty(
			t,
			setCookieHeader,
			"Should set session cookie if a session is not found",
		)
	})

	// Verify the session cookie is hardened: HttpOnly, SameSite=Lax and Path=/.
	// The Secure attribute must follow the request transport so the cookie works
	// over plain HTTP locally but is restricted to HTTPS when forwarded as such.
	// Presenting an unknown session id forces the server to issue a fresh cookie
	// regardless of any cookie persisted by the shared test client jar.
	app = govalin.New(func(config *govalin.Config) {
		config.EnableSessions()
	}).Get("/govalin", func(call *govalin.Call) {
		call.Text("govalin")
	})
	govalintest.Test(t, app, func(client *govalintest.Client) {
		unknownSession := &http.Cookie{Name: "govalin-session", Value: "non-existent"}

		req, err := http.NewRequest(http.MethodGet, "/govalin", nil)
		assert.Nil(t, err)
		req.AddCookie(unknownSession)
		response := client.Do(req)
		defer func() { _ = response.Body.Close() }()

		sessionCookie := findCookie(response.Cookies(), "govalin-session")
		assert.NotNil(t, sessionCookie, "Should set the session cookie")
		assert.True(t, sessionCookie.HttpOnly, "Session cookie should be HttpOnly")
		assert.Equal(t, http.SameSiteLaxMode, sessionCookie.SameSite, "Session cookie should be SameSite=Lax")
		assert.Equal(t, "/", sessionCookie.Path, "Session cookie should be scoped to root path")
		assert.False(t, sessionCookie.Secure, "Session cookie should not be Secure over plain HTTP")

		secureReq, err := http.NewRequest(http.MethodGet, "/govalin", nil)
		assert.Nil(t, err)
		secureReq.AddCookie(unknownSession)
		secureReq.Header.Set("X-Forwarded-Proto", "https")
		secureResponse := client.Do(secureReq)
		defer func() { _ = secureResponse.Body.Close() }()

		forwardedCookie := findCookie(secureResponse.Cookies(), "govalin-session")
		assert.NotNil(t, forwardedCookie, "Should set the session cookie")
		assert.True(
			t,
			forwardedCookie.Secure,
			"Session cookie should be Secure when the request is forwarded as HTTPS",
		)
	})

	app = govalin.New(func(config *govalin.Config) {
		config.EnableSessions()
	}).
		Get("/get", func(call *govalin.Call) {
			sessionVal, ok := call.SessionAttrOrDefault("test", "notGovalin").(string)
			if !ok {
				call.Text("notGovalin")
				return
			}
			call.Text(sessionVal)
		}).
		Get("/set", func(call *govalin.Call) {
			_, err := call.SessionAttr("test", "govalin")
			if err != nil {
				call.Error(err)
				return
			}

			call.Status(200)
		})
	govalintest.Test(t, app, func(client *govalintest.Client) {
		// Use a cookie jar so the session cookie is persisted across requests.
		jar, err := cookiejar.New(nil)
		assert.Nil(t, err)
		client.HTTP().Jar = jar

		response := client.GetResponse("/get")
		body := readBody(t, response)
		setCookies := response.Cookies()

		assert.Equal(t, 1, len(setCookies), "Should set one cookie")
		assert.Equal(t, 200, response.StatusCode, "Should set status to 200")
		assert.Equal(t, "notGovalin", body, "Should retrieve default value if no session attr is set")

		response = client.GetResponse("/set")
		setCookies = response.Cookies()
		_ = response.Body.Close()

		assert.Equal(t, 0, len(setCookies), "Should not set cookies when already received one")
		assert.Equal(t, 200, response.StatusCode, "Should set status to 200")

		response = client.GetResponse("/get")
		body = readBody(t, response)

		assert.Equal(t, 200, response.StatusCode, "Should set status to 200")
		assert.Equal(t, "govalin", body, "Should retrieve session attr")

		// Drop the jar so only the invalid session cookie is presented.
		client.HTTP().Jar = nil
		req, err := http.NewRequest(http.MethodGet, "/get", nil)
		assert.Nil(t, err)
		req.AddCookie(&http.Cookie{
			Name:  "govalin-session",
			Value: "invalid-session",
		})
		response = client.Do(req)
		body = readBody(t, response)

		assert.Equal(t, 200, response.StatusCode, "Should set status to 200")
		assert.Equal(t, "notGovalin", body, "Should retrieve default value if session attr is not found")
	})
}

func TestRequestID(t *testing.T) {
	app := newTestApp()
	app.Get("/govalin", func(call *govalin.Call) {
		call.Text(call.ID())
	})
	govalintest.Test(t, app, func(client *govalintest.Client) {
		body := client.Get("/govalin")

		assert.NotEmpty(
			t,
			body,
			"Should generate a unique request ID",
		)
	})

	app = newTestApp()
	app.Get("/govalin", func(call *govalin.Call) {
		call.Text(call.ID())
	})
	govalintest.Test(t, app, func(client *govalintest.Client) {
		req, err := http.NewRequest(http.MethodGet, "/govalin", nil)
		assert.Nil(t, err)
		req.Header.Set("X-Govalin-Id", "govalin")
		govalinID := readBody(t, client.Do(req))

		assert.Equal(
			t,
			govalinID,
			"govalin",
			"Should reuse given ID",
		)
	})
}

func TestRedirect(t *testing.T) {
	app := newTestApp()
	app.Get("/govalin", func(call *govalin.Call) {
		call.Redirect("http://" + call.Host() + "/govalin2")
	})
	govalintest.Test(t, app, func(client *govalintest.Client) {
		client.HTTP().CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		response := client.GetResponse("/govalin")
		defer func() { _ = response.Body.Close() }()

		assert.Equal(t, 302, response.StatusCode, "Should redirect with 302")
		assert.Equal(t, client.Host+"/govalin2", response.Header.Get(headers.Location))
	})

	app = newTestApp()
	app.Get("/govalin", func(call *govalin.Call) {
		call.Redirect("http://"+call.Host()+"/govalin2", true)
	})
	govalintest.Test(t, app, func(client *govalintest.Client) {
		client.HTTP().CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		response := client.GetResponse("/govalin")
		defer func() { _ = response.Body.Close() }()

		assert.Equal(t, 301, response.StatusCode, "Should redirect with 301")
		assert.Equal(t, client.Host+"/govalin2", response.Header.Get(headers.Location))
	})

	app = newTestApp()
	app.Get("/govalin", func(call *govalin.Call) {
		call.Redirect("http://" + call.Host() + "/govalin2")
	})
	app.Get("/govalin2", func(call *govalin.Call) {
		call.Text("govalin2")
	})
	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/govalin")
		body := readBody(t, response)

		assert.Equal(t, 200, response.StatusCode)
		assert.Equal(t, "govalin2", body)
	})
}
