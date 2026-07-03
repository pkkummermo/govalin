package govalin_test

import (
	"net/http"
	"testing"

	"github.com/pkkummermo/govalin/govalintest"
	"github.com/stretchr/testify/assert"
)

func TestServeHTTP(t *testing.T) {
	app := newTestApp()
	app.HTTPServe("/httpserve", func(w http.ResponseWriter, _ *http.Request) {
		// Header must be written before the body; writing it afterwards is a
		// superfluous WriteHeader call that net/http warns about.
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("httpservegovalin"))
		assert.Nil(t, err, "Should write to response writer")
	})

	govalintest.Test(t, app, func(client *govalintest.Client) {
		assert.Equal(
			t,
			"httpservegovalin",
			client.Get("/httpserve"),
			"Should create httpserve GET endpoint",
		)
		assert.Equal(
			t,
			"httpservegovalin",
			client.Post("/httpserve", nil),
			"Should create httpserve POST endpoint",
		)
		assert.Equal(
			t,
			"httpservegovalin",
			client.Patch("/httpserve", nil),
			"Should create httpserve PATCH endpoint",
		)
		assert.Equal(
			t,
			"httpservegovalin",
			client.Delete("/httpserve"),
			"Should create httpserve DELETE endpoint",
		)
		assert.Equal(
			t,
			"httpservegovalin",
			client.Options("/httpserve"),
			"Should create httpserve OPTIONS endpoint",
		)
		assert.Equal(
			t,
			"",
			client.Head("/httpserve"),
			"Should create httpserve HEAD endpoint",
		)
	})
}
