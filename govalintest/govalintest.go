// Package govalintest provides a test harness for testing govalin
// applications over real HTTP.
//
// The harness takes your already constructed App — built by your own
// application constructor, with your config, plugins and routes — starts it
// on an OS-assigned port, waits for it to be ready, and hands your test a
// Client bound to it. The app is shut down automatically via t.Cleanup.
//
//	func TestMyAPI(t *testing.T) {
//		govalintest.Test(t, myapp.New(), func(client *govalintest.Client) {
//			if body := client.Get("/health"); body != "ok" {
//				t.Errorf("expected ok, got %q", body)
//			}
//		})
//	}
//
// Failures in the harness or client fail the calling test; they never abort
// the test process. Client methods must be called from the test goroutine.
package govalintest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkkummermo/govalin"
)

const defaultStartupTimeout = 5 * time.Second

// Options configures the test harness. The zero value means sane defaults.
type Options struct {
	// StartupTimeout is the maximum time to wait for the app to start.
	// Zero means the default of 5 seconds.
	StartupTimeout time.Duration
}

// Client makes HTTP requests against the app under test and fails the test
// on transport errors.
//
// The body-returning verbs (Get, Post, ...) return the response body
// regardless of status code, so tests can assert on error handler responses;
// use the *Response variants to assert on status codes and headers. The
// caller is responsible for closing the body of a returned *http.Response.
type Client struct {
	// Host is the base URL of the app under test, e.g. "http://localhost:54321".
	Host string

	t    *testing.T
	http *http.Client
}

// Test starts the given app on an OS-assigned port, waits for it to be
// ready, runs the test function with a Client bound to it, and shuts the
// app down when the test finishes.
func Test(t *testing.T, app *govalin.App, fn func(client *Client)) {
	t.Helper()
	TestWithOptions(t, app, Options{}, fn)
}

// TestWithOptions is Test with explicit harness options.
func TestWithOptions(t *testing.T, app *govalin.App, opts Options, fn func(client *Client)) {
	t.Helper()

	startupTimeout := opts.StartupTimeout
	if startupTimeout == 0 {
		startupTimeout = defaultStartupTimeout
	}

	ready := make(chan struct{})
	startErr := make(chan error, 1)

	app.Events(func(serverEvents *govalin.ServerEvents) {
		serverEvents.AddOnServerStartup(func() {
			close(ready)
		})
	})

	go func() {
		if err := app.Start(0); err != nil {
			startErr <- err
		}
	}()

	t.Cleanup(func() {
		if err := app.Shutdown(); err != nil {
			t.Errorf("govalintest: failed to shut down app: %v", err)
		}
	})

	timeout := time.NewTimer(startupTimeout)
	defer timeout.Stop()

	select {
	case <-ready:
	case err := <-startErr:
		t.Fatalf("govalintest: failed to start app: %v", err)
	case <-timeout.C:
		t.Fatalf("govalintest: app did not start within %s", startupTimeout)
	}

	fn(&Client{
		Host: fmt.Sprintf("http://localhost:%d", app.Port()),
		t:    t,
		http: &http.Client{},
	})
}

// Get performs a GET request and returns the response body.
func (client *Client) Get(path string) string {
	client.t.Helper()
	return client.bodyString(client.request(http.MethodGet, path, nil))
}

// GetResponse performs a GET request and returns the full response.
func (client *Client) GetResponse(path string) *http.Response {
	client.t.Helper()
	return client.request(http.MethodGet, path, nil)
}

// Head performs a HEAD request and returns the response body.
func (client *Client) Head(path string) string {
	client.t.Helper()
	return client.bodyString(client.request(http.MethodHead, path, nil))
}

// HeadResponse performs a HEAD request and returns the full response.
func (client *Client) HeadResponse(path string) *http.Response {
	client.t.Helper()
	return client.request(http.MethodHead, path, nil)
}

// Options performs an OPTIONS request and returns the response body.
func (client *Client) Options(path string) string {
	client.t.Helper()
	return client.bodyString(client.request(http.MethodOptions, path, nil))
}

// OptionsResponse performs an OPTIONS request and returns the full response.
func (client *Client) OptionsResponse(path string) *http.Response {
	client.t.Helper()
	return client.request(http.MethodOptions, path, nil)
}

// Delete performs a DELETE request and returns the response body.
func (client *Client) Delete(path string) string {
	client.t.Helper()
	return client.bodyString(client.request(http.MethodDelete, path, nil))
}

// DeleteResponse performs a DELETE request and returns the full response.
func (client *Client) DeleteResponse(path string) *http.Response {
	client.t.Helper()
	return client.request(http.MethodDelete, path, nil)
}

// Post performs a POST request and returns the response body.
//
// A string, []byte or io.Reader body is sent as-is; any other value is
// JSON-encoded and sent with Content-Type: application/json.
func (client *Client) Post(path string, body any) string {
	client.t.Helper()
	return client.bodyString(client.request(http.MethodPost, path, body))
}

// PostResponse performs a POST request and returns the full response. See
// Post for body semantics.
func (client *Client) PostResponse(path string, body any) *http.Response {
	client.t.Helper()
	return client.request(http.MethodPost, path, body)
}

// Put performs a PUT request and returns the response body. See Post for
// body semantics.
func (client *Client) Put(path string, body any) string {
	client.t.Helper()
	return client.bodyString(client.request(http.MethodPut, path, body))
}

// PutResponse performs a PUT request and returns the full response. See Post
// for body semantics.
func (client *Client) PutResponse(path string, body any) *http.Response {
	client.t.Helper()
	return client.request(http.MethodPut, path, body)
}

// Patch performs a PATCH request and returns the response body. See Post for
// body semantics.
func (client *Client) Patch(path string, body any) string {
	client.t.Helper()
	return client.bodyString(client.request(http.MethodPatch, path, body))
}

// PatchResponse performs a PATCH request and returns the full response. See
// Post for body semantics.
func (client *Client) PatchResponse(path string, body any) *http.Response {
	client.t.Helper()
	return client.request(http.MethodPatch, path, body)
}

// Do sends a custom request and returns the response, failing the test on
// transport errors. A request with a relative URL (no scheme/host) is
// resolved against the app under test, so requests built with
// http.NewRequest("GET", "/path", nil) work directly.
func (client *Client) Do(req *http.Request) *http.Response {
	client.t.Helper()

	if req == nil || req.URL == nil {
		client.t.Fatalf("govalintest: Do requires a non-nil request with a URL; build it with http.NewRequest")
	}

	if req.Header == nil {
		req.Header = http.Header{}
	}

	if req.URL.Scheme == "" {
		base, err := url.Parse(client.Host)
		if err != nil {
			client.t.Fatalf("govalintest: invalid host %q: %v", client.Host, err)
		}
		req.URL.Scheme = base.Scheme
		req.URL.Host = base.Host
	}

	if _, hasUserAgent := req.Header["User-Agent"]; !hasUserAgent {
		req.Header.Set("User-Agent", "govalintest")
	}

	response, err := client.http.Do(req)
	if err != nil {
		client.t.Fatalf("govalintest: %s %s failed: %v", req.Method, req.URL, err)
	}

	return response
}

// HTTP returns the underlying http.Client for customization, e.g. setting a
// timeout or a cookie jar.
func (client *Client) HTTP() *http.Client {
	return client.http
}

// Websocket connects a websocket to the given path on the app under test.
func (client *Client) Websocket(path string) *websocket.Conn {
	client.t.Helper()

	wsURL := strings.Replace(client.Host, "http", "ws", 1) + path
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil) //nolint:bodyclose // on success the response body is owned by the connection
	if err != nil {
		client.t.Fatalf("govalintest: failed to connect websocket at %s: %v", wsURL, err)
	}

	return conn
}

func (client *Client) request(method string, path string, body any) *http.Response {
	client.t.Helper()

	var reader io.Reader
	var contentType string

	switch typedBody := body.(type) {
	case nil:
	case string:
		reader = strings.NewReader(typedBody)
	case []byte:
		reader = bytes.NewReader(typedBody)
	case io.Reader:
		reader = typedBody
	default:
		data, err := json.Marshal(typedBody)
		if err != nil {
			client.t.Fatalf("govalintest: failed to JSON-encode %s body for %s: %v", method, path, err)
		}
		reader = bytes.NewReader(data)
		contentType = "application/json"
	}

	req, err := http.NewRequest(method, client.Host+path, reader)
	if err != nil {
		client.t.Fatalf("govalintest: failed to create %s request for %s: %v", method, path, err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	return client.Do(req)
}

func (client *Client) bodyString(response *http.Response) string {
	client.t.Helper()

	defer func() {
		_ = response.Body.Close()
	}()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		client.t.Fatalf("govalintest: failed to read response body: %v", err)
	}

	return string(data)
}
