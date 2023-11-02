package cors

import (
	"net/http"
	"os"
	"strings"

	"log/slog"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/internal/http/headers"
	"github.com/pkkummermo/govalin/internal/util"
)

type Config struct {
	allowedOrigins   []string
	allowedHeaders   []string
	allowedMethods   []string
	allowCredentials bool
}

const (
	wildcard   = "*"
	nullOrigin = "null"
)

func New() *Config {
	return &Config{
		allowedOrigins: []string{},
		allowedMethods: []string{
			"GET", "POST", "PUT", "DELETE", "OPTIONS",
		},
		allowedHeaders:   []string{"*"},
		allowCredentials: false,
	}
}

func (config *Config) Name() string {
	return "CORS plugin"
}

func (config *Config) OnInit(_ *govalin.Config) {
	config.checkConfiguration()
}

func (config *Config) Apply(app *govalin.App) {
	app.Before("*", config.handleCors)
	app.After("*", func(call *govalin.Call) {
		if call.Method() == http.MethodOptions {
			call.Status(http.StatusOK)
		}
	})
}

func (config *Config) checkConfiguration() {
	if config.allowCredentials && util.ContainsSome(config.allowedOrigins, wildcard) {
		slog.Error("CORS plugin has been configured to allow credentials while having " +
			"a wildcard in allowed origins. This is not a secure way of exposing " +
			"CORS headers. For more details search for 'CORS attacks'.")
		os.Exit(1)
	}

	if util.ContainsSome(config.allowedOrigins, nullOrigin) {
		slog.Error("You should never allow the null origin in your CORS config. For more details see " +
			"https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Origin#directives")
		os.Exit(1)
	}
}

func (config *Config) handleCors(call *govalin.Call) bool {
	origin := call.Header(headers.Origin)

	if !util.ContainsSome(config.allowedOrigins, wildcard, origin) {
		return true
	}

	call.Header(headers.AccessControlAllowOrigin, origin)
	call.Header(headers.Vary, headers.AccessControlAllowOrigin)

	if util.ContainsSome(config.allowedHeaders, wildcard) {
		call.Header(headers.AccessControlAllowHeaders, wildcard)
	} else {
		call.Header(headers.AccessControlAllowHeaders, strings.Join(config.allowedHeaders, ", "))
	}

	if util.ContainsSome(config.allowedMethods, wildcard) {
		call.Header(headers.AccessControlAllowMethods, wildcard)
	} else {
		call.Header(headers.AccessControlAllowMethods, strings.Join(config.allowedMethods, ", "))
	}

	if config.allowCredentials {
		call.Header(headers.AccessControlAllowCredentials, "true")
	}

	return true
}

// AllowAllOrigins will explicitly set allowed origins to "*", allowing all origins.
//
// For more details see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Origin
func (config *Config) AllowAllOrigins() *Config {
	config.allowedOrigins = []string{"*"}
	return config
}

// AllowOrigins sets the allowed origins for cross origin requests
//
// For more details see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Origin
func (config *Config) AllowOrigins(origins ...string) *Config {
	config.allowedOrigins = origins
	return config
}

// AllowCredentials will allow for the user to send credentials using cross origin requests
//
// For more details see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Credentials
func (config *Config) AllowCredentials(allow bool) *Config {
	config.allowCredentials = allow
	return config
}

// AllowHeaders sets the allowed headers for CORS in addition to the safelisted headers
// found in https://developer.mozilla.org/en-US/docs/Glossary/CORS-safelisted_request_header. Defaults to "*"
//
// For more details see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Headers
func (config *Config) AllowHeaders(headers ...string) *Config {
	config.allowedHeaders = headers
	return config
}

// AllowMethods sets the allowed methods for CORS. Defaults to GET, POST, PUT, DELETE, OPTIONS.
func (config *Config) AllowMethods(methods ...string) *Config {
	config.allowedMethods = methods
	return config
}
