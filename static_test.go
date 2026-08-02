package govalin_test

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
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

// TestStaticServesRanges covers static files going through Call rather than
// past it: a large file can be resumed and seeked into, on both mount kinds.
func TestStaticServesRanges(t *testing.T) {
	staticRoot, _ := fs.Sub(staticTestFiles, "internal/testdata/static")

	app := newTestApp()
	app.Static("/fs", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
		staticConfig.WithFS(staticRoot)
	})
	app.Static("/static", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
		staticConfig.WithStaticPath("internal/testdata/static")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		for _, mount := range []string{"/fs", "/static"} {
			full := readBody(t, client.GetResponse(mount+"/index.html"))

			partial := client.GetRange(mount+"/index.html", 0, 4)
			assert.Equal(
				t,
				206,
				partial.StatusCode,
				"A ranged request for a file on %s should be answered with a 206",
				mount,
			)
			assert.Equal(
				t,
				full[:5],
				readBody(t, partial),
				"A 206 from %s should carry exactly the requested bytes",
				mount,
			)
		}
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

func TestStaticFolderSymlinkEscape(t *testing.T) {
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	assert.Nil(t, os.WriteFile(secret, []byte("Govalin secret"), 0o600))

	staticRoot := t.TempDir()
	assert.Nil(t, os.WriteFile(
		filepath.Join(staticRoot, "index.html"),
		[]byte("Hello world"),
		0o600,
	))

	// A symlink is contained by the static root as a *path*, so path arithmetic
	// alone accepts it; only resolving it against the root shows it leaves.
	assert.Nil(t, os.Symlink(secret, filepath.Join(staticRoot, "escape.txt")))
	assert.Nil(t, os.Symlink(outside, filepath.Join(staticRoot, "escape-dir")))

	app := newTestApp()
	app.Static("/static", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
		staticConfig.WithStaticPath(staticRoot)
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		escapes := []string{
			"/static/escape.txt",
			"/static/escape-dir/secret.txt",
		}

		for _, attempt := range escapes {
			response := client.GetResponse(attempt)
			body := readBody(t, response)

			assert.Equal(
				t,
				404,
				response.StatusCode,
				"Symlink %q pointing outside the static root should not resolve",
				attempt,
			)
			assert.NotContains(
				t,
				body,
				"Govalin secret",
				"Symlink %q must not leak contents of files outside the static root",
				attempt,
			)
		}

		assert.Contains(
			t,
			client.Get("/static/index.html"),
			"Hello world",
			"Files genuinely inside the static root should still be served",
		)
	})
}

func TestStaticFolderMissingRoot(t *testing.T) {
	app := newTestApp()
	app.Static("/static", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
		staticConfig.WithStaticPath(filepath.Join(t.TempDir(), "does-not-exist"))
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		assert.Equal(
			t,
			500,
			client.GetStatus("/static/index.html"),
			"A static mount whose folder is missing is a misconfiguration, not a 404",
		)
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
