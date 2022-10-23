package govalin_test

import (
	"testing"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/internal/govalintesting"
	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Get("/get", func(call *govalin.Call) {
			call.Text("getgovalin")
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		assert.Equal(
			t,
			"getgovalin",
			http.Get("/get"),
			"Should create get endpoint",
		)
	})
}

func TestPost(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Post("/post", func(call *govalin.Call) {
			call.Text("postgovalin")
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		assert.Equal(
			t,
			"postgovalin",
			http.Post("/post", map[string]string{}),
			"Should create post endpoint",
		)
	})
}

func TestPut(t *testing.T) {}

func TestDelete(t *testing.T) {}
