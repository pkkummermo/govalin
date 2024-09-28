package govalin_test

import (
	"net/http"
	"testing"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/internal/govalintesting"
	"github.com/stretchr/testify/assert"
)

func TestServeHTTP(t *testing.T) {
	govalintesting.HTTPTestUtil(func(app *govalin.App) *govalin.App {
		app.HTTPServe("/httpserve", func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write([]byte("httpservegovalin"))
			w.WriteHeader(http.StatusOK)
			assert.Nil(t, err, "Should write to response writer")
		})

		return app
	}, func(http govalintesting.GovalinHTTP) {
		assert.Equal(
			t,
			"httpservegovalin",
			http.Get("/httpserve", nil),
			"Should create httpserve GET endpoint",
		)
		assert.Equal(
			t,
			"httpservegovalin",
			http.Post("/httpserve", nil),
			"Should create httpserve POST endpoint",
		)
		assert.Equal(
			t,
			"httpservegovalin",
			http.Patch("/httpserve", nil),
			"Should create httpserve PATCH endpoint",
		)
		assert.Equal(
			t,
			"httpservegovalin",
			http.Delete("/httpserve", nil),
			"Should create httpserve DELETE endpoint",
		)
		assert.Equal(
			t,
			"httpservegovalin",
			http.Options("/httpserve", nil),
			"Should create httpserve OPTIONS endpoint",
		)
		assert.Equal(
			t,
			"",
			http.Head("/httpserve"),
			"Should create httpserve HEAD endpoint",
		)
	})
}
