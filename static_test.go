package govalin_test

import (
	"embed"
	"io"
	"io/fs"
	"testing"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/internal/govalintesting"
	"github.com/stretchr/testify/assert"
)

//go:embed internal/testdata/static
var staticTestFiles embed.FS

func TestStaticFS(t *testing.T) {
	staticRoot, _ := fs.Sub(staticTestFiles, "internal/testdata/static")

	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Static("/fs", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
			staticConfig.WithFS(staticRoot)
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		assert.Contains(
			t,
			http.Get("/fs/index.html"),
			"Hello world",
			"Should serve index.html from embedded files",
		)
		assert.Contains(
			t,
			http.Get("/fs/"),
			"Hello world",
			"Should serve index.html from embedded files on /",
		)
		assert.Contains(
			t,
			http.Get("/fs/sub/test.html"),
			"Sub hello world",
			"Should serve subfolder html files from embedded files",
		)
		notFoundResponse := http.GetResponse("/fs/non-existing-path")
		notFoundBody, _ := io.ReadAll(notFoundResponse.Body)
		assert.Contains(
			t,
			string(notFoundBody),
			"page not found",
			"Should contain not found",
		)
		assert.Equal(
			t,
			404,
			notFoundResponse.StatusCode,
			"Should return 404",
		)
	})
}

func TestStaticFSSPAMode(t *testing.T) {
	staticRoot, _ := fs.Sub(staticTestFiles, "internal/testdata/static")

	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Static("/fsspa", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
			staticConfig.
				WithFS(staticRoot).
				EnableSPAMode(true)
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		assert.Contains(
			t,
			http.Get("/fsspa/index.html"),
			"Hello world",
			"Should serve index.html from embedded files",
		)
		assert.Contains(
			t,
			http.Get("/fsspa/non/existing/path"),
			"Hello world",
			"Should serve index.html with SPA mode",
		)
		assert.Contains(
			t,
			http.Get("/fsspa/sub/test.html"),
			"Sub hello world",
			"Should serve files if they exist ahead of SPA index.html",
		)
	})
}

func TestStaticFolder(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Static("/static", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
			staticConfig.WithStaticPath("internal/testdata/static")
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		assert.Contains(
			t,
			http.Get("/static/index.html"),
			"Hello world",
			"Should serve index.html from embedded files",
		)
		assert.Contains(
			t,
			http.Get("/static/"),
			"Hello world",
			"Should serve index.html from embedded files on /",
		)
		assert.Contains(
			t,
			http.Get("/static/sub/test.html"),
			"Sub hello world",
			"Should serve subfolder html files from embedded files",
		)
		notFoundResponse := http.GetResponse("/static/non-existing-path")
		notFoundBody, _ := io.ReadAll(notFoundResponse.Body)
		assert.Contains(
			t,
			string(notFoundBody),
			"page not found",
			"Should contain not found",
		)
		assert.Equal(
			t,
			404,
			notFoundResponse.StatusCode,
			"Should return 404",
		)
	})
}

func TestStaticFolderSPAMode(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.Static("/staticspa", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
			staticConfig.
				WithStaticPath("internal/testdata/static").
				EnableSPAMode(true)
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		assert.Contains(
			t,
			http.Get("/staticspa/index.html"),
			"Hello world",
			"Should serve index.html from embedded files",
		)
		assert.Contains(
			t,
			http.Get("/staticspa/non/existing/path"),
			"Hello world",
			"Should serve index.html with SPA mode",
		)
		assert.Contains(
			t,
			http.Get("/staticspa/sub/test.html"),
			"Sub hello world",
			"Should serve files if they exist ahead of SPA index.html",
		)
	})
}
