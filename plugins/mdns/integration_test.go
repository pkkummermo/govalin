package mdns_test

import (
	"testing"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/internal/govalintesting"
	"github.com/pkkummermo/govalin/plugins/mdns"
	"github.com/stretchr/testify/assert"
)

// TestServerServesWithMDNSEnabled verifies that enabling the plugin does not
// interfere with the HTTP server: routes keep serving regardless of whether the
// underlying multicast environment can actually advertise (runtime failures are
// non-fatal). It also exercises the full Apply -> shutdown -> withdraw lifecycle
// since HTTPTestUtil shuts the server down at the end.
func TestServerServesWithMDNSEnabled(t *testing.T) {
	govalintesting.HTTPTestUtil(func(_ *govalin.App) *govalin.App {
		return govalin.New(func(config *govalin.Config) {
			config.Plugin(mdns.New())
		}).Get("/govalin", func(call *govalin.Call) { call.Text("govalin") })
	}, func(http govalintesting.GovalinHTTP) {
		assert.Equal(t, "govalin", http.Get("/govalin"))
	})
}
