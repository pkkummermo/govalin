package routing

import (
	"strings"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/internal/http/headers"
)

type HTTPToHTTPSConfig struct {
	redirectLocalhost bool
}

// NewHTTPtoHTTPS configures the server to redirect HTTP calls to HTTPS.
func NewHTTPtoHTTPS() *HTTPToHTTPSConfig {
	return &HTTPToHTTPSConfig{
		redirectLocalhost: false,
	}
}

func (config *HTTPToHTTPSConfig) Name() string {
	return "HTTP to HTTPS plugin"
}

func (config *HTTPToHTTPSConfig) OnInit(_ *govalin.Config) {
}

func (config *HTTPToHTTPSConfig) Apply(app *govalin.App) {
	app.Before("/*", func(call *govalin.Call) bool {
		callHost := call.Host()
		isLocalhost := strings.HasPrefix(callHost, "localhost")

		if !config.redirectLocalhost && isLocalhost {
			return true
		}

		xForwardedProto := call.Header(headers.XForwardedProto)
		if xForwardedProto == "http" || (xForwardedProto == "" && call.Raw.Req.TLS == nil) {
			// Temporary on localhost: a permanent redirect would stick in the browser's cache.
			if isLocalhost {
				call.Redirect("https://"+callHost+call.URL().Path, false)
			} else {
				call.Redirect("https://"+callHost+call.URL().Path, true)
			}

			return false
		}

		return true
	})
}

func (config *HTTPToHTTPSConfig) RedirectLocalHost(shouldRedirect bool) *HTTPToHTTPSConfig {
	config.redirectLocalhost = shouldRedirect

	return config
}
