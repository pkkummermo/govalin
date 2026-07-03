package govalin_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/pkkummermo/govalin"
)

// newTestApp creates an app with logs disabled to keep test output clean,
// optionally applying further config on top.
func newTestApp(configFuncs ...govalin.ConfigFunc) *govalin.App {
	return govalin.New(func(config *govalin.Config) {
		config.EnableAccessLog(false)
		config.EnableStartupLog(false)

		for _, configFunc := range configFuncs {
			configFunc(config)
		}
	})
}

// readBody reads and closes a response body, failing the test on error.
func readBody(t *testing.T, response *http.Response) string {
	t.Helper()

	defer func() {
		_ = response.Body.Close()
	}()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	return string(data)
}
