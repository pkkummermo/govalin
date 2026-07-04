package govalintest_test

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/govalintest"
	"github.com/stretchr/testify/assert"
)

// newQuietApp creates an app configured the way an application constructor
// would, with logs disabled to keep test output clean.
func newQuietApp() *govalin.App {
	return govalin.New(func(config *govalin.Config) {
		config.EnableAccessLog(false)
		config.EnableStartupLog(false)
	})
}

func TestGet(t *testing.T) {
	app := newQuietApp().Get("/hello", func(call *govalin.Call) {
		call.Text("hello govalin")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		assert.Equal(t, "hello govalin", client.Get("/hello"), "Should return body from GET endpoint")
	})
}

func TestGetResponseExposesStatusAndHeaders(t *testing.T) {
	app := newQuietApp().Get("/created", func(call *govalin.Call) {
		call.Status(http.StatusCreated)
		call.Text("created")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/created")
		defer func() { _ = response.Body.Close() }()

		assert.Equal(t, http.StatusCreated, response.StatusCode, "Should expose status code")
		assert.Equal(t, "govalin", response.Header.Get("Server"), "Should expose headers")
	})
}

func TestStatusVerbsReturnStatusCode(t *testing.T) {
	app := newQuietApp().
		Get("/status", func(call *govalin.Call) { call.Status(http.StatusNoContent) }).
		Post("/status", func(call *govalin.Call) { call.Status(http.StatusCreated) }).
		Put("/status", func(call *govalin.Call) { call.Status(http.StatusAccepted) }).
		Patch("/status", func(call *govalin.Call) { call.Status(http.StatusOK) }).
		Delete("/status", func(call *govalin.Call) { call.Status(http.StatusGone) })

	govalintest.Test(t, app, func(client *govalintest.Client) {
		assert.Equal(t, http.StatusNoContent, client.GetStatus("/status"), "Should return GET status")
		assert.Equal(t, http.StatusCreated, client.PostStatus("/status", "body"), "Should return POST status")
		assert.Equal(t, http.StatusAccepted, client.PutStatus("/status", "body"), "Should return PUT status")
		assert.Equal(t, http.StatusOK, client.PatchStatus("/status", "body"), "Should return PATCH status")
		assert.Equal(t, http.StatusGone, client.DeleteStatus("/status"), "Should return DELETE status")
		assert.Equal(t, http.StatusNotFound, client.GetStatus("/no-such-path"), "Should return error status without failing")
	})
}

func TestBodyAccessIsStatusAgnostic(t *testing.T) {
	app := newQuietApp().Get("/teapot", func(call *govalin.Call) {
		call.Status(http.StatusTeapot)
		call.Text("short and stout")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		assert.Equal(
			t,
			"short and stout",
			client.Get("/teapot"),
			"Should return body regardless of error status",
		)
		assert.Contains(
			t,
			client.Get("/no-such-path"),
			"doesn't exist",
			"Should return 404 body without failing the test",
		)
	})
}

func TestPostJSONEncodesNonRawBodies(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	app := newQuietApp().Post("/users", func(call *govalin.Call) {
		var received payload
		assert.Nil(t, call.BodyAs(&received), "Should parse JSON body")
		assert.Equal(t, "application/json", call.Header("Content-Type"), "Should set JSON content type")
		call.Text("created " + received.Name)
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		assert.Equal(
			t,
			"created govalin",
			client.Post("/users", payload{Name: "govalin"}),
			"Should JSON-encode struct bodies",
		)
	})
}

func TestPostSendsRawBodiesAsIs(t *testing.T) {
	app := newQuietApp().Post("/echo", func(call *govalin.Call) {
		body, err := io.ReadAll(call.Raw.Req.Body)
		assert.Nil(t, err, "Should read raw body")
		call.Text(string(body))
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		assert.Equal(t, "raw string", client.Post("/echo", "raw string"), "Should send string as-is")
		assert.Equal(t, "raw bytes", client.Post("/echo", []byte("raw bytes")), "Should send bytes as-is")
		assert.Equal(
			t,
			"raw reader",
			client.Post("/echo", strings.NewReader("raw reader")),
			"Should send reader as-is",
		)
	})
}

func TestPutPatchDelete(t *testing.T) {
	app := newQuietApp().
		Put("/resource", func(call *govalin.Call) {
			call.Text("put")
		}).
		Patch("/resource", func(call *govalin.Call) {
			call.Text("patch")
		}).
		Delete("/resource", func(call *govalin.Call) {
			call.Text("delete")
		})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		assert.Equal(t, "put", client.Put("/resource", "body"), "Should PUT")
		assert.Equal(t, "patch", client.Patch("/resource", "body"), "Should PATCH")
		assert.Equal(t, "delete", client.Delete("/resource"), "Should DELETE")
	})
}

func TestDoResolvesRelativeRequests(t *testing.T) {
	app := newQuietApp().Get("/secured", func(call *govalin.Call) {
		call.Text("token: " + call.Header("Authorization"))
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		request, err := http.NewRequest(http.MethodGet, "/secured", nil)
		assert.Nil(t, err, "Should create relative request")
		request.Header.Set("Authorization", "Bearer test-token")

		response := client.Do(request)
		defer func() { _ = response.Body.Close() }()

		assert.Equal(t, http.StatusOK, response.StatusCode, "Should resolve relative URL against app under test")
	})
}

func TestWebsocket(t *testing.T) {
	app := newQuietApp()
	app.Ws("/ws", func(wsConfig *govalin.WsConfig) {
		wsConfig.OnOpen = func(wsConnection *govalin.WsConnection) {
			assert.Nil(t, wsConnection.SendText("hello ws"), "Should send text on open")
		}
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		ws := client.Websocket("/ws")
		defer func() { _ = ws.Close() }()

		_, message, err := ws.ReadMessage()
		assert.Nil(t, err, "Should read message from websocket")
		assert.Equal(t, "hello ws", string(message), "Should receive message sent on open")
	})
}

func TestWithOptionsCustomStartupTimeout(t *testing.T) {
	app := newQuietApp().Get("/hello", func(call *govalin.Call) {
		call.Text("hello")
	})

	govalintest.TestWithOptions(t, app, govalintest.Options{StartupTimeout: 10 * time.Second}, func(client *govalintest.Client) {
		assert.Equal(t, "hello", client.Get("/hello"), "Should work with custom startup timeout")
	})
}

func TestAppEventsFireOnShutdown(t *testing.T) {
	shutdownCalled := false

	t.Run("run app under test", func(t *testing.T) {
		app := newQuietApp().Get("/hello", func(call *govalin.Call) {
			call.Text("hello")
		})
		app.Events(func(serverEvents *govalin.ServerEvents) {
			serverEvents.AddOnServerShutdown(func() {
				shutdownCalled = true
			})
		})

		govalintest.Test(t, app, func(client *govalintest.Client) {
			assert.Equal(t, "hello", client.Get("/hello"), "Should serve while running")
		})
	})

	assert.True(t, shutdownCalled, "Should fire shutdown event registered via App.Events after the test finishes")
}
