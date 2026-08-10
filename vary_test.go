package govalin_test

import (
	"net/http"
	"testing"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/govalintest"
	"github.com/stretchr/testify/assert"
)

func TestVaryOnDeclaresTheSelectingHeaders(t *testing.T) {
	app := newTestApp()
	app.Get("/greeting", func(call *govalin.Call) {
		call.VaryOn("Accept", "Accept-Language")
		call.Text("hei")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/greeting")

		assert.Equal(
			t,
			[]string{"Accept, Accept-Language"},
			response.Header.Values("Vary"),
			"Should name every header the response was selected from",
		)
	})
}

// TestVaryOnDeclaresNothingForNoNames covers a caller spreading a list that
// turned out to be empty: a header naming no selecting header is not one.
func TestVaryOnDeclaresNothingForNoNames(t *testing.T) {
	app := newTestApp()
	app.Get("/greeting", func(call *govalin.Call) {
		call.VaryOn()
		call.Text("hei")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/greeting")

		assert.Empty(t, response.Header.Values("Vary"), "Should leave the response alone")
	})
}

// TestVaryOnAddsToAnEarlierDeclaration covers the difference from freshness: two
// layers can each have a claim on the same response, so the second declaration
// joins the first rather than replacing it.
func TestVaryOnAddsToAnEarlierDeclaration(t *testing.T) {
	app := newTestApp()
	app.Before("/greeting", func(call *govalin.Call) bool {
		call.VaryOn("Origin")

		return true
	})
	app.Get("/greeting", func(call *govalin.Call) {
		call.VaryOn("Accept-Language")
		call.Text("hei")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/greeting")

		assert.Equal(
			t,
			[]string{"Origin, Accept-Language"},
			response.Header.Values("Vary"),
			"Both layers' selecting headers should survive",
		)
	})
}

// TestVaryOnDeclaresARepeatedHeaderOnce covers two layers with the same claim,
// which is what a plugin varying on Origin under a handler that does too looks
// like. Header names are case-insensitive, so the spelling does not decide it.
func TestVaryOnDeclaresARepeatedHeaderOnce(t *testing.T) {
	app := newTestApp()
	app.Before("/greeting", func(call *govalin.Call) bool {
		call.VaryOn("origin")

		return true
	})
	app.Get("/greeting", func(call *govalin.Call) {
		call.VaryOn("Origin")
		call.Text("hei")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/greeting")

		assert.Equal(
			t,
			[]string{"Origin"},
			response.Header.Values("Vary"),
			"One claim named twice should be declared once",
		)
	})
}

// TestVaryOnFoldsRawDeclarationsIntoItsList covers the world VaryOn arrived
// into: a layer declaring Vary through Header sends a field line of its own,
// and RFC 9110 §5.3 reads several of those as the one list.
func TestVaryOnFoldsRawDeclarationsIntoItsList(t *testing.T) {
	app := newTestApp()
	app.Before("/greeting", func(call *govalin.Call) bool {
		call.Header("Vary", "Accept")
		call.Header("Vary", "accept, Accept-Language")

		return true
	})
	app.Get("/greeting", func(call *govalin.Call) {
		call.VaryOn("Accept-Language", "Origin")
		call.Text("hei")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := client.GetResponse("/greeting")

		assert.Equal(
			t,
			[]string{"Accept, Accept-Language, Origin"},
			response.Header.Values("Vary"),
			"Every claim should be declared, and each of them once",
		)
	})
}

// TestVaryOnSurvivesARevalidation covers what the cache does with a 304: it
// updates the stored response from it, and dropping Vary there would leave the
// copy keyed on a URL it was never selected by.
func TestVaryOnSurvivesARevalidation(t *testing.T) {
	app := newTestApp()
	app.Get("/greeting", func(call *govalin.Call) {
		call.VaryOn("Accept-Language")

		if call.NotModified("v3") {
			return
		}

		call.Text("hei")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		response := revalidate(t, client, http.MethodGet, "/greeting", `"v3"`)

		assert.Equal(t, http.StatusNotModified, response.StatusCode, "Should answer 304 on a match")
		assert.Equal(
			t,
			"Accept-Language",
			response.Header.Get("Vary"),
			"A 304 should say which stored response it renews",
		)
	})
}
