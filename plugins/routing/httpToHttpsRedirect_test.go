package routing_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/govalintest"
	"github.com/pkkummermo/govalin/internal/http/headers"
	"github.com/pkkummermo/govalin/plugins/routing"
	"github.com/stretchr/testify/assert"
)

// readBody reads and closes the response body, failing the test on error.
func readBody(t *testing.T, response *http.Response) string {
	t.Helper()

	defer func() { _ = response.Body.Close() }()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	return string(data)
}

func TestLocalhostRedirectToHTTPS(t *testing.T) {
	app := govalin.New(func(config *govalin.Config) {
		config.Plugin(routing.NewHTTPtoHTTPS().RedirectLocalHost(true))
	}).Get("/govalin", func(call *govalin.Call) {
		call.Text("govalin")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		client.HTTP().CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

		response := client.GetResponse("/govalin")
		defer func() { _ = response.Body.Close() }()

		assert.Equal(t, 302, response.StatusCode)
		assert.Equal(t, "https://"+strings.TrimPrefix(client.Host, "http://")+"/govalin", response.Header.Get(headers.Location))
	})
}

func TestDefaultsDoesNotRedirectLocalhost(t *testing.T) {
	app := govalin.New(func(config *govalin.Config) {
		config.Plugin(routing.NewHTTPtoHTTPS())
	}).Get("/govalin", func(call *govalin.Call) {
		call.Text("govalin")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		client.HTTP().CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

		response := client.GetResponse("/govalin")
		body := readBody(t, response)

		assert.Equal(t, 200, response.StatusCode)
		assert.Equal(t, "govalin", body)
	})
}

func TestRedirectOnExternalHost(t *testing.T) {
	app := govalin.New(func(config *govalin.Config) {
		config.Plugin(routing.NewHTTPtoHTTPS())
	}).Get("/govalin", func(call *govalin.Call) {
		call.Text("govalin")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		client.HTTP().CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

		req, err := http.NewRequest(http.MethodGet, "/govalin", nil)
		assert.Nil(t, err)
		req.Host = "govalin.io"
		response := client.Do(req)
		defer func() { _ = response.Body.Close() }()

		assert.Equal(t, 301, response.StatusCode)
		assert.Equal(t, "https://govalin.io/govalin", response.Header.Get(headers.Location))
	})
}
