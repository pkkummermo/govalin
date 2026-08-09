package govalin_test

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

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

// TestStaticDerivesValidators covers both mount kinds getting a validator
// without being asked: a disk mount from modification time and size, an
// embedded one — which reports no modification time at all — from its content.
func TestStaticDerivesValidators(t *testing.T) {
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
			served := client.GetResponse(mount + "/index.html")
			etag := served.Header.Get("ETag")

			assert.NotEmpty(t, etag, "A file served from %s should carry a validator", mount)
			assert.Contains(t, readBody(t, served), "Hello world", "%s should serve the file", mount)

			revalidated := revalidate(t, client, http.MethodGet, mount+"/index.html", etag)
			assert.Equal(
				t,
				http.StatusNotModified,
				revalidated.StatusCode,
				"Revalidating a file on %s should be answered with a 304",
				mount,
			)
			assert.Empty(t, readBody(t, revalidated), "A 304 from %s should carry no body", mount)
		}
	})
}

// TestStaticDerivesValidatorsForDirectoryIndexes covers the URL a static site is
// asked for most: the mount root, and any directory holding an index.html. These
// used to be handed to http.FileServer, which sends no validator, so the SPA
// shell on an embedded mount was the one file that could not be revalidated.
func TestStaticDerivesValidatorsForDirectoryIndexes(t *testing.T) {
	bundle := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("shell")}}

	app := newTestApp()
	app.Static("/fs", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
		staticConfig.WithFS(bundle)
	})
	app.Static("/static", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
		staticConfig.WithStaticPath("internal/testdata/static")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		for _, mountRoot := range []string{"/fs/", "/static/"} {
			served := client.GetResponse(mountRoot)
			etag := served.Header.Get("ETag")

			assert.NotEmpty(t, etag, "%s should carry a validator", mountRoot)

			revalidated := revalidate(t, client, http.MethodGet, mountRoot, etag)
			assert.Equal(
				t,
				http.StatusNotModified,
				revalidated.StatusCode,
				"Revalidating %s should be answered with a 304",
				mountRoot,
			)
		}
	})
}

// TestStaticLeavesTheFileServerItsWork pins the other half of that split: a
// directory with no index.html is still the file server's to list, and a
// directory reached without its trailing slash is still its to redirect.
func TestStaticLeavesTheFileServerItsWork(t *testing.T) {
	indexless := fstest.MapFS{"notes/readme.txt": &fstest.MapFile{Data: []byte("notes")}}

	app := newTestApp()
	app.Static("/listing", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
		staticConfig.WithFS(indexless)
	})
	app.Static("/static", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
		staticConfig.WithStaticPath("internal/testdata/static")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		client.HTTP().CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}

		listing := client.GetResponse("/listing/")
		assert.Equal(t, 200, listing.StatusCode, "A mount with no index should still be listed")
		assert.Contains(t, readBody(t, listing), "notes", "The listing should name the directory's entries")
		assert.Empty(t, listing.Header.Get("ETag"), "A generated listing has no validator to derive")

		redirect := client.GetResponse("/static/sub")
		assert.Equal(
			t,
			http.StatusMovedPermanently,
			redirect.StatusCode,
			"A directory without its trailing slash should still redirect",
		)
	})
}

// TestStaticValidatorFollowsContentWithoutAClock is the reason an embedded file
// is hashed rather than measured. Two builds of a bundle can differ in content
// at identical size — a version bump of the same length — and with no
// modification time to fall back on, a size-derived tag would keep every client
// on the old copy indefinitely.
func TestStaticValidatorFollowsContentWithoutAClock(t *testing.T) {
	etagFor := func(t *testing.T, content string) string {
		t.Helper()

		bundle := fstest.MapFS{"app.js": &fstest.MapFile{Data: []byte(content)}}

		app := newTestApp()
		app.Static("/assets", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
			staticConfig.WithFS(bundle)
		})

		var etag string

		govalintest.Test(t, app, func(client *govalintest.Client) {
			etag = client.GetResponse("/assets/app.js").Header.Get("ETag")
		})

		return etag
	}

	before := etagFor(t, `const version = "1.0.9"`)
	after := etagFor(t, `const version = "1.1.0"`)

	assert.NotEmpty(t, before, "An embedded file should carry a validator despite having no modification time")
	assert.NotEqual(t, before, after, "A same-length change should change the validator")
}

// TestStaticValidatorKeepsRangesResumable pins the reason derived validators are
// strong: net/http requires a strong match for If-Range, so a weak tag would
// turn every resumed download back into a full one.
func TestStaticValidatorKeepsRangesResumable(t *testing.T) {
	app := newTestApp()
	app.Static("/static", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
		staticConfig.WithStaticPath("internal/testdata/static")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		full := readBody(t, client.GetResponse("/static/index.html"))
		etag := client.GetResponse("/static/index.html").Header.Get("ETag")

		request, err := http.NewRequest(http.MethodGet, client.Host+"/static/index.html", nil)
		if err != nil {
			t.Fatalf("failed to build a ranged request: %v", err)
		}

		request.Header.Set("Range", "bytes=0-4")
		request.Header.Set("If-Range", etag)

		partial := client.Do(request)
		assert.Equal(t, 206, partial.StatusCode, "A resume against the served validator should stay ranged")
		assert.Equal(t, full[:5], readBody(t, partial), "A 206 should carry exactly the requested bytes")
	})
}

// TestStaticServesIndexFileDirectly pins a behaviour change: http.FileServer
// used to answer an explicit index.html request with a 301 to the directory,
// costing a second round trip. Serving through Call, the file is simply served.
func TestStaticServesIndexFileDirectly(t *testing.T) {
	app := newTestApp()
	app.Static("/static", func(_ *govalin.Call, staticConfig *govalin.StaticConfig) {
		staticConfig.WithStaticPath("internal/testdata/static")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		client.HTTP().CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}

		response := client.GetResponse("/static/index.html")

		assert.Equal(
			t,
			200,
			response.StatusCode,
			"An explicit index.html request should be served, not redirected",
		)
		assert.Contains(t, readBody(t, response), "Hello world", "The index file should be the body")
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
