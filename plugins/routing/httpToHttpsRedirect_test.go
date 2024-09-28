package routing_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/ddliu/go-httpclient"
	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/internal/govalintesting"
	"github.com/pkkummermo/govalin/internal/http/headers"
	"github.com/pkkummermo/govalin/plugins/routing"
	"github.com/stretchr/testify/assert"
)

func TestLocalhostRedirectToHTTPS(t *testing.T) {
	govalintesting.HTTPTestUtil(func(_ *govalin.App) *govalin.App {
		return govalin.New(func(config *govalin.Config) {
			config.Plugin(routing.NewHTTPtoHTTPS().RedirectLocalHost(true))
		}).Get("/govalin", func(call *govalin.Call) {
			call.Text("govalin")
		})
	}, func(http govalintesting.GovalinHTTP) {
		response, err := http.
			Raw().
			Begin().
			WithOption(httpclient.OPT_FOLLOWLOCATION, false).
			Get(http.Host + "/govalin")

		assert.Equal(t, true, httpclient.IsRedirectError(err), fmt.Sprintf("Request was not redirect. Error: %s", err))
		assert.Equal(t, 302, response.StatusCode)
		assert.Equal(t, "https://"+strings.TrimPrefix(http.Host, "http://")+"/govalin", response.Header.Get(headers.Location))
	})
}

func TestDefaultsDoesNotRedirectLocalhost(t *testing.T) {
	govalintesting.HTTPTestUtil(func(_ *govalin.App) *govalin.App {
		return govalin.New(func(config *govalin.Config) {
			config.Plugin(routing.NewHTTPtoHTTPS())
		}).Get("/govalin", func(call *govalin.Call) {
			call.Text("govalin")
		})
	}, func(http govalintesting.GovalinHTTP) {
		response, err := http.
			Raw().
			Begin().
			WithOption(httpclient.OPT_FOLLOWLOCATION, false).
			Get(http.Host + "/govalin")
		body, _ := response.ToString()

		assert.NoError(t, err, fmt.Sprintf("Request errored. Error: %s", err))
		assert.Equal(t, 200, response.StatusCode)
		assert.Equal(t, "govalin", body)
	})
}

func TestRedirectOnExternalHost(t *testing.T) {
	govalintesting.HTTPTestUtil(func(_ *govalin.App) *govalin.App {
		return govalin.New(func(config *govalin.Config) {
			config.Plugin(routing.NewHTTPtoHTTPS())
		}).Get("/govalin", func(call *govalin.Call) {
			call.Text("govalin")
		})
	}, func(govalinHttp govalintesting.GovalinHTTP) {
		response, err := govalinHttp.
			Raw().
			Begin().
			WithOption(httpclient.OPT_FOLLOWLOCATION, false).
			WithOption(httpclient.OPT_BEFORE_REQUEST_FUNC, func(_ *http.Client, req *http.Request) {
				req.Host = "govalin.io"
			}).
			Get(govalinHttp.Host + "/govalin")

		assert.Equal(t, true, httpclient.IsRedirectError(err), fmt.Sprintf("Request was not redirect. Error: %s", err))
		assert.Equal(t, 301, response.StatusCode)
		assert.Equal(t, "https://govalin.io/govalin", response.Header.Get(headers.Location))
	})
}
