package govalin_test

import (
	"embed"
	"io/fs"
	"testing"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/govalintest"
	"github.com/stretchr/testify/assert"
)

//go:embed internal/testdata/static
var staticTestFiles embed.FS

func TestStaticFS(t *testing.T) {
	staticRoot, _ := fs.Sub(staticTestFiles, "internal/testdata/static")

	app := newTestApp()
	app.Static("/fs", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
		staticConfig.WithFS(staticRoot)
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		assert.Contains(
			t,
			client.Get("/fs/index.html"),
			"Hello world",
			"Should serve index.html from embedded files",
		)
		assert.Contains(
			t,
			client.Get("/fs/"),
			"Hello world",
			"Should serve index.html from embedded files on /",
		)
		assert.Contains(
			t,
			client.Get("/fs/sub/test.html"),
			"Sub hello world",
			"Should serve subfolder html files from embedded files",
		)
		notFoundResponse := client.GetResponse("/fs/non-existing-path")
		notFoundBody := readBody(t, notFoundResponse)
		assert.Contains(
			t,
			notFoundBody,
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

	app := newTestApp()
	app.Static("/fsspa", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
		staticConfig.
			WithFS(staticRoot).
			EnableSPAMode(true)
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		assert.Contains(
			t,
			client.Get("/fsspa/index.html"),
			"Hello world",
			"Should serve index.html from embedded files",
		)
		assert.Contains(
			t,
			client.Get("/fsspa/non/existing/path"),
			"Hello world",
			"Should serve index.html with SPA mode",
		)
		assert.Contains(
			t,
			client.Get("/fsspa/sub/test.html"),
			"Sub hello world",
			"Should serve files if they exist ahead of SPA index.html",
		)
	})
}

func TestStaticFolder(t *testing.T) {
	app := newTestApp()
	app.Static("/static", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
		staticConfig.WithStaticPath("internal/testdata/static")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		assert.Contains(
			t,
			client.Get("/static/index.html"),
			"Hello world",
			"Should serve index.html from embedded files",
		)
		assert.Contains(
			t,
			client.Get("/static/"),
			"Hello world",
			"Should serve index.html from embedded files on /",
		)
		assert.Contains(
			t,
			client.Get("/static/sub/test.html"),
			"Sub hello world",
			"Should serve subfolder html files from embedded files",
		)
		notFoundResponse := client.GetResponse("/static/non-existing-path")
		notFoundBody := readBody(t, notFoundResponse)
		assert.Contains(
			t,
			notFoundBody,
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

func TestStaticFolderPathTraversal(t *testing.T) {
	app := newTestApp()
	app.Static("/static", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
		staticConfig.WithStaticPath("internal/testdata/static")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		// Encoded "../" segments survive client-side URL normalization and reach
		// the handler as ".." in the request path. They must not be allowed to
		// escape the configured static root and probe/serve files such as the
		// repository's README.md, three levels above internal/testdata/static.
		traversals := []string{
			"/static/%2e%2e/%2e%2e/%2e%2e/README.md",
			"/static/..%2f..%2f..%2fREADME.md",
			"/static/%2e%2e/%2e%2e/%2e%2e/static.go",
		}

		for _, attempt := range traversals {
			response := client.GetResponse(attempt)
			body := readBody(t, response)

			assert.Equal(
				t,
				404,
				response.StatusCode,
				"Traversal attempt %q should not resolve to a file outside the static root",
				attempt,
			)
			assert.NotContains(
				t,
				body,
				"Govalin",
				"Traversal attempt %q must not leak contents of files outside the static root",
				attempt,
			)
		}
	})
}

func TestStaticFolderSPAMode(t *testing.T) {
	app := newTestApp()
	app.Static("/staticspa", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
		staticConfig.
			WithStaticPath("internal/testdata/static").
			EnableSPAMode(true)
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		assert.Contains(
			t,
			client.Get("/staticspa/index.html"),
			"Hello world",
			"Should serve index.html from embedded files",
		)
		assert.Contains(
			t,
			client.Get("/staticspa/non/existing/path"),
			"Hello world",
			"Should serve index.html with SPA mode",
		)
		assert.Contains(
			t,
			client.Get("/staticspa/sub/test.html"),
			"Sub hello world",
			"Should serve files if they exist ahead of SPA index.html",
		)
	})
}
