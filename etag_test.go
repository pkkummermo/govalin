package govalin_test

import (
	"net/http"
	"testing"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/govalintest"
	"github.com/stretchr/testify/assert"
)

// revalidate sends a conditional request with the given If-None-Match header.
func revalidate(t *testing.T, client *govalintest.Client, method string, path string, ifNoneMatch string) *http.Response {
	t.Helper()

	request, err := http.NewRequest(method, client.Host+path, nil)
	if err != nil {
		t.Fatalf("failed to build a conditional request: %v", err)
	}

	if ifNoneMatch != "" {
		request.Header.Set("If-None-Match", ifNoneMatch)
	}

	return client.Do(request)
}

// etagApp serves one resource whose validator never changes, and whose handler
// always writes a body.
func etagApp(t *testing.T, tag string) *govalin.App {
	t.Helper()

	app := newTestApp()
	app.Get("/resource", func(call *govalin.Call) {
		if call.NotModified(tag) {
			return
		}

		call.Text("the representation")
	})

	return app
}

func TestNotModifiedSendsValidator(t *testing.T) {
	govalintest.Test(t, etagApp(t, "v3"), func(client *govalintest.Client) {
		response := client.GetResponse("/resource")

		assert.Equal(t, http.StatusOK, response.StatusCode, "Should serve the representation")
		assert.Equal(t, `"v3"`, response.Header.Get("ETag"), "Should quote a bare tag")
		assert.Equal(t, "the representation", readBody(t, response), "Should write the body")
	})
}

func TestNotModifiedAnswersRevalidation(t *testing.T) {
	govalintest.Test(t, etagApp(t, "v3"), func(client *govalintest.Client) {
		response := revalidate(t, client, http.MethodGet, "/resource", `"v3"`)

		assert.Equal(t, http.StatusNotModified, response.StatusCode, "Should answer 304 on a match")
		assert.Empty(t, readBody(t, response), "Should send no body")
		assert.Empty(t, response.Header.Get("Content-Type"), "Should describe no representation")
		assert.Equal(t, `"v3"`, response.Header.Get("ETag"), "Should still name the version")
	})
}

func TestNotModifiedServesAChangedResource(t *testing.T) {
	govalintest.Test(t, etagApp(t, "v3"), func(client *govalintest.Client) {
		response := revalidate(t, client, http.MethodGet, "/resource", `"v2"`)

		assert.Equal(t, http.StatusOK, response.StatusCode, "Should serve a version the client lacks")
		assert.Equal(t, "the representation", readBody(t, response), "Should write the body")
	})
}

// TestNotModifiedComparesWeakly covers the comparison RFC 9110 requires of
// If-None-Match: the weak form names the same version as the strong one, the
// header may list several, and * names whatever the server has.
func TestNotModifiedComparesWeakly(t *testing.T) {
	govalintest.Test(t, etagApp(t, "v3"), func(client *govalintest.Client) {
		for _, ifNoneMatch := range []string{`W/"v3"`, `"v1", "v3"`, `"v1",W/"v3"`, "*"} {
			response := revalidate(t, client, http.MethodGet, "/resource", ifNoneMatch)

			assert.Equal(
				t,
				http.StatusNotModified,
				response.StatusCode,
				"Should match the tag named by If-None-Match: %s", ifNoneMatch,
			)
		}
	})
}

// TestNotModifiedMatchesATagContainingAComma covers the separator being a comma
// outside a quoted tag rather than every comma: a caller's version string is
// quoted as it stands, and splitting one in half would silently stop it matching.
func TestNotModifiedMatchesATagContainingAComma(t *testing.T) {
	govalintest.Test(t, etagApp(t, "sha,42"), func(client *govalintest.Client) {
		served := client.GetResponse("/resource")
		assert.Equal(t, `"sha,42"`, served.Header.Get("ETag"), "Should quote the tag as it stands")

		for _, ifNoneMatch := range []string{`"sha,42"`, `"v1", "sha,42"`} {
			response := revalidate(t, client, http.MethodGet, "/resource", ifNoneMatch)

			assert.Equal(
				t,
				http.StatusNotModified,
				response.StatusCode,
				"Should match the tag named by If-None-Match: %s", ifNoneMatch,
			)
		}

		unrelated := revalidate(t, client, http.MethodGet, "/resource", `"sha", "42"`)
		assert.Equal(
			t,
			http.StatusOK,
			unrelated.StatusCode,
			"Should not match the halves of the tag listed separately",
		)
	})
}

func TestNotModifiedAnswersHead(t *testing.T) {
	govalintest.Test(t, etagApp(t, "v3"), func(client *govalintest.Client) {
		response := revalidate(t, client, http.MethodHead, "/resource", `"v3"`)

		assert.Equal(t, http.StatusNotModified, response.StatusCode, "Should revalidate a HEAD")
	})
}

// TestNotModifiedLeavesUnsafeMethodsAlone covers the boundary of the feature:
// on a write, If-None-Match is a precondition asking a different question, and
// answering it 304 would silently drop the write.
func TestNotModifiedLeavesUnsafeMethodsAlone(t *testing.T) {
	app := newTestApp()
	app.Post("/resource", func(call *govalin.Call) {
		if call.NotModified("v3") {
			return
		}

		call.Text("written")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := revalidate(t, client, http.MethodPost, "/resource", `"v3"`)

		assert.Equal(t, http.StatusOK, response.StatusCode, "Should not 304 a write")
		assert.Equal(t, "written", readBody(t, response), "Should perform the write")
		assert.Empty(
			t,
			response.Header.Get("ETag"),
			"Should not offer a validator a write could be conditioned on",
		)
	})
}

// TestNotModifiedWithoutTheEarlyReturn is the whole point of buffering the 304:
// a handler that ignores the return value still sends a correct response, and
// pays only in wasted work.
func TestNotModifiedWithoutTheEarlyReturn(t *testing.T) {
	app := newTestApp()
	app.Get("/resource", func(call *govalin.Call) {
		call.NotModified("v3")
		call.Text("the representation")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := revalidate(t, client, http.MethodGet, "/resource", `"v3"`)

		assert.Equal(t, http.StatusNotModified, response.StatusCode, "Should answer 304 regardless")
		assert.Empty(t, readBody(t, response), "Should refuse the body the handler wrote")
		assert.Empty(t, response.Header.Get("Content-Type"), "Should describe no representation")
	})
}

// TestNotModifiedInBeforeHandler covers applying one validator to a group of
// routes, which is the app-wide idiom: the short-circuit flushes the buffered
// 304 and the not-found fallback leaves it alone.
func TestNotModifiedInBeforeHandler(t *testing.T) {
	app := newTestApp()
	app.Before("/api/*", func(call *govalin.Call) bool {
		return !call.NotModified("v3")
	})
	app.Get("/api/resource", func(call *govalin.Call) {
		call.Text("the representation")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := revalidate(t, client, http.MethodGet, "/api/resource", `"v3"`)

		assert.Equal(t, http.StatusNotModified, response.StatusCode, "Should short-circuit with a 304")
		assert.Empty(t, readBody(t, response), "Should send no body")
	})
}

func TestNotModifiedKeepsAnAlreadyValidTag(t *testing.T) {
	tags := map[string]string{
		`"v3"`:   `"v3"`,
		`W/"v3"`: `W/"v3"`,
		`v3`:     `"v3"`,
	}

	for tag, expected := range tags {
		govalintest.Test(t, etagApp(t, tag), func(client *govalintest.Client) {
			response := client.GetResponse("/resource")

			assert.Equal(t, expected, response.Header.Get("ETag"), "Should send a valid entity-tag for %s", tag)
		})
	}
}
