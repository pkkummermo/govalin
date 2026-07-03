package mdns_test

import (
	"testing"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/govalintest"
	"github.com/pkkummermo/govalin/plugins/mdns"
	"github.com/stretchr/testify/assert"
)

// TestServerServesWithMDNSEnabled verifies that enabling the plugin does not
// interfere with the HTTP server: routes keep serving regardless of whether the
// underlying multicast environment can actually advertise (runtime failures are
// non-fatal). It also exercises the full Apply -> shutdown -> withdraw lifecycle
// since the test harness shuts the server down at the end.
func TestServerServesWithMDNSEnabled(t *testing.T) {
	app := govalin.New(func(config *govalin.Config) {
		config.Plugin(mdns.New())
	}).Get("/govalin", func(call *govalin.Call) { call.Text("govalin") })

	govalintest.Test(t, app, func(client *govalintest.Client) {
		assert.Equal(t, "govalin", client.Get("/govalin"))
	})
}
