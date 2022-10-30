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

func TestPut(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Put("/put", func(call *govalin.Call) {
			call.Text("putgovalin")
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		assert.Equal(
			t,
			"putgovalin",
			http.Put("/put", nil),
			"Should create put endpoint",
		)
	})
}

func TestPatch(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Patch("/patch", func(call *govalin.Call) {
			call.Text("patchgovalin")
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		assert.Equal(
			t,
			"patchgovalin",
			http.Patch("/patch", nil),
			"Should create patch endpoint",
		)
	})
}

func TestOptions(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Options("/options", func(call *govalin.Call) {
			call.Text("optionsgovalin")
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		assert.Equal(
			t,
			"optionsgovalin",
			http.Options("/options"),
			"Should create options endpoint",
		)
	})
}

func TestHead(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Head("/head", func(call *govalin.Call) {
			call.Header("govalin-header", "govalin")
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		assert.Equal(
			t,
			"govalin",
			http.HeadResponse("/head").Header.Get("govalin-header"),
			"Should create head endpoint",
		)
	})
}

func TestDelete(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Delete("/delete", func(call *govalin.Call) {
			call.Text("deletegovalin")
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		assert.Equal(
			t,
			"deletegovalin",
			http.Delete("/delete"),
			"Should create delete endpoint",
		)
	})
}

func TestBefore(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Before("/*", func(call *govalin.Call) bool {
			call.Text("before")
			return true
		})
		app.Get("/test", func(call *govalin.Call) {
			call.Text("govalin")
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		assert.Equal(
			t,
			"beforegovalin",
			http.Get("/test"),
			"Should trigger before and then endpoint",
		)
	})

	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Before("/*", func(call *govalin.Call) bool {
			call.Text("before")
			return false
		})
		app.Get("/test", func(call *govalin.Call) {
			call.Text("govalin")
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		assert.Equal(
			t,
			"before",
			http.Get("/test"),
			"Should trigger before and short circuit",
		)
	})

	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Before("/*", func(call *govalin.Call) bool {
			call.Text("before")
			return true
		})
		app.Before("/test", func(call *govalin.Call) bool {
			call.Text("before2")
			return true
		})
		app.Get("/test", func(call *govalin.Call) {
			call.Text("govalin")
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		assert.Equal(
			t,
			"beforebefore2govalin",
			http.Get("/test"),
			"Should trigger multiple before and endpoint",
		)
	})
}
