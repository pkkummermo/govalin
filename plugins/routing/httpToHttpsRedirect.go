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

		// Don't redirect localhost unless configured
		if !config.redirectLocalhost && isLocalhost {
			return true
		}

		// Redirect if scheme is http or forwarded scheme is http
		xForwardedProto := call.Header(headers.XForwardedProto)
		if xForwardedProto == "http" || (xForwardedProto == "" && call.Raw.Req.TLS == nil) {
			// More often than not we do not want to redirect to HTTPS on localhost, at least not permanently
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
