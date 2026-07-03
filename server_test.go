package govalin_test

import (
	"fmt"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/govalintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPortReflectsBoundPort(t *testing.T) {
	app := newTestApp()
	govalintest.Test(t, app, func(client *govalintest.Client) {
		hostURL, err := url.Parse(client.Host)
		require.NoError(t, err)

		assert.Equal(
			t,
			hostURL.Port(),
			fmt.Sprintf("%d", app.Port()),
			"Port() should report the port the server is bound to",
		)
	})
}

func TestPortReturnsOSAssignedPortWhenStartedOnZero(t *testing.T) {
	// This test covers the OS-assigned (Start(0)) path with a minimal bespoke
	// start/shutdown harness so the startup/port mechanics are exercised
	// directly, independent of the shared test harness.
	startup := make(chan bool, 1)
	app := govalin.New(func(config *govalin.Config) {
		config.EnableAccessLog(false)
		config.EnableStartupLog(false)
		config.Events(func(events *govalin.ServerEvents) {
			events.AddOnServerStartup(func() { startup <- true })
		})
	})

	go func() {
		if err := app.Start(0); err != nil {
			t.Errorf("failed to start server on port 0: %v", err)
		}
	}()

	select {
	case <-startup:
	case <-time.After(time.Second):
		t.Fatal("server startup timed out")
	}
	defer func() { _ = app.Shutdown() }()

	boundPort := app.Port()
	assert.NotZero(t, boundPort, "Port() should return the real OS-assigned port, not 0")

	// The reported port must be the one the server actually listens on.
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", boundPort), time.Second)
	require.NoError(t, err, "server should be reachable on the port reported by Port()")
	_ = conn.Close()
}

func TestGet(t *testing.T) {
	app := newTestApp()
	app.Get("/get", func(call *govalin.Call) {
		call.Text("getgovalin")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		assert.Equal(
			t,
			"getgovalin",
			client.Get("/get"),
			"Should create get endpoint",
		)
	})
}

func TestPost(t *testing.T) {
	app := newTestApp()
	app.Post("/post", func(call *govalin.Call) {
		call.Text("postgovalin")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		assert.Equal(
			t,
			"postgovalin",
			client.Post("/post", nil),
			"Should create post endpoint",
		)
	})
}

func TestPut(t *testing.T) {
	app := newTestApp()
	app.Put("/put", func(call *govalin.Call) {
		call.Text("putgovalin")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		assert.Equal(
			t,
			"putgovalin",
			client.Put("/put", nil),
			"Should create put endpoint",
		)
	})
}

func TestPatch(t *testing.T) {
	app := newTestApp()
	app.Patch("/patch", func(call *govalin.Call) {
		call.Text("patchgovalin")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		assert.Equal(
			t,
			"patchgovalin",
			client.Patch("/patch", nil),
			"Should create patch endpoint",
		)
	})
}

func TestOptions(t *testing.T) {
	app := newTestApp()
	app.Options("/options", func(call *govalin.Call) {
		call.Text("optionsgovalin")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		assert.Equal(
			t,
			"optionsgovalin",
			client.Options("/options"),
			"Should create options endpoint",
		)
	})
}

func TestHead(t *testing.T) {
	app := newTestApp()
	app.Head("/head", func(call *govalin.Call) {
		call.Header("govalin-header", "govalin")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.HeadResponse("/head")
		defer func() { _ = response.Body.Close() }()

		assert.Equal(
			t,
			"govalin",
			response.Header.Get("govalin-header"),
			"Should create head endpoint",
		)
	})
}

func TestDelete(t *testing.T) {
	app := newTestApp()
	app.Delete("/delete", func(call *govalin.Call) {
		call.Text("deletegovalin")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		assert.Equal(
			t,
			"deletegovalin",
			client.Delete("/delete"),
			"Should create delete endpoint",
		)
	})
}

func TestNotFoundHandler(t *testing.T) {
	app := newTestApp()
	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/nonExistingPath")
		defer func() { _ = response.Body.Close() }()

		assert.Equal(t, 404, response.StatusCode)
	})
}

func TestBefore(t *testing.T) {
	app := newTestApp()
	app.Before("/*", func(call *govalin.Call) bool {
		call.Text("before")
		return true
	})
	app.Get("/test", func(call *govalin.Call) {
		call.Text("govalin")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		assert.Equal(
			t,
			"beforegovalin",
			client.Get("/test"),
			"Should trigger before and then endpoint",
		)
	})

	shortCircuitApp := newTestApp()
	shortCircuitApp.Before("/*", func(call *govalin.Call) bool {
		call.Text("before")
		return false
	})
	shortCircuitApp.Get("/test", func(call *govalin.Call) {
		call.Text("govalin")
	})

	govalintest.Test(t, shortCircuitApp, func(client *govalintest.Client) {
		assert.Equal(
			t,
			"before",
			client.Get("/test"),
			"Should trigger before and short circuit",
		)
	})

	multipleBeforeApp := newTestApp()
	multipleBeforeApp.Before("/*", func(call *govalin.Call) bool {
		call.Text("before")
		return true
	})
	multipleBeforeApp.Before("/test", func(call *govalin.Call) bool {
		call.Text("before2")
		return true
	})
	multipleBeforeApp.Get("/test", func(call *govalin.Call) {
		call.Text("govalin")
	})

	govalintest.Test(t, multipleBeforeApp, func(client *govalintest.Client) {
		assert.Equal(
			t,
			"beforebefore2govalin",
			client.Get("/test"),
			"Should trigger multiple before and endpoint",
		)
	})
}

func TestAfter(t *testing.T) {
	app := newTestApp()
	app.Get("/test", func(call *govalin.Call) {
		call.Text("govalin")
	})
	app.After("/*", func(call *govalin.Call) {
		call.Text("after")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		assert.Equal(
			t,
			"govalinafter",
			client.Get("/test"),
			"Should trigger endpoint and after",
		)
	})

	multipleAfterApp := newTestApp()
	multipleAfterApp.Get("/test", func(call *govalin.Call) {
		call.Text("govalin")
	})
	multipleAfterApp.After("/test", func(call *govalin.Call) {
		call.Text("after")
	})
	multipleAfterApp.After("/*", func(call *govalin.Call) {
		call.Text("after2")
	})

	govalintest.Test(t, multipleAfterApp, func(client *govalintest.Client) {
		assert.Equal(
			t,
			"govalinafterafter2",
			client.Get("/test"),
			"Should trigger endpoint and multiple after",
		)
	})
}

func TestRoute(t *testing.T) {
	app := newTestApp()
	app.Route("/test", func() {
		app.Get("/get", func(call *govalin.Call) {
			call.Text("routegovalin")
		})
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		assert.Equal(
			t,
			"routegovalin",
			client.Get("/test/get"),
			"Should create endpoint within route",
		)
	})

	nestedApp := newTestApp()
	nestedApp.Route("/test", func() {
		nestedApp.Route("/subroute", func() {
			nestedApp.Get("/get", func(call *govalin.Call) {
				call.Text("subroutegovalin")
			})
		})
	})

	govalintest.Test(t, nestedApp, func(client *govalintest.Client) {
		assert.Equal(
			t,
			"subroutegovalin",
			client.Get("/test/subroute/get"),
			"Should create endpoint within nested routes",
		)
	})

	multiRouteApp := newTestApp()
	multiRouteApp.Route("/test", func() {
		multiRouteApp.Route("/subroute", func() {
			multiRouteApp.Get("/get", func(call *govalin.Call) {
				call.Text("subroutegovalin")
			})
		})
	})
	multiRouteApp.Route("/test2", func() {
		multiRouteApp.Route("/subroute2", func() {
			multiRouteApp.Get("/get", func(call *govalin.Call) {
				call.Text("subroutegovalin2")
			})
		})
	})

	govalintest.Test(t, multiRouteApp, func(client *govalintest.Client) {
		assert.Equal(
			t,
			"subroutegovalin",
			client.Get("/test/subroute/get"),
			"Should create endpoint within nested routes",
		)
		assert.Equal(
			t,
			"subroutegovalin2",
			client.Get("/test2/subroute2/get"),
			"Should create endpoint2 within nested routes on new route endpoint",
		)
	})
}
