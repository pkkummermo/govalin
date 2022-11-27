package cors

import (
	"log"
	"strings"

	"github.com/pkkummermo/govalin"
	"github.com/pkkummermo/govalin/internal/http/headers"
	"github.com/pkkummermo/govalin/internal/util"
)

type Config struct {
	enabled bool
	// defaultScheme string

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
		enabled:        false,
		allowedOrigins: []string{},
		allowedMethods: []string{
			"GET", "POST", "PUT", "DELETE", "OPTIONS",
		},
		allowedHeaders:   []string{"*"},
		allowCredentials: false,
	}
}

func (config *Config) Apply(app *govalin.App) {
	app.Before("*", config.handleCors)
}

// Enable will configure the server to handle OPTIONS preflight requests
// according to your CORS configuration.
func (config *Config) Enable(conf EnableConfig) *Config {
	conf(newConfigFunc(config))
	config.enabled = true
	config.checkConfiguration()

	return config
}

func (config *Config) checkConfiguration() {
	if config.allowCredentials && util.ContainsSome(config.allowedOrigins, wildcard) {
		log.Fatal("CORS plugin has been configured to allow credentials while having " +
			"a wildcard in allowed origins. This is not a secure way of exposing " +
			"CORS headers. For more details search for 'CORS attacks'.")
	}

	if util.ContainsSome(config.allowedOrigins, nullOrigin) {
		log.Fatal("You should never allow the null origin in your CORS config. For more details see " +
			"https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Origin#directives")
	}
}

func (config *Config) handleCors(call *govalin.Call) bool {
	origin := call.Header(headers.Origin)

	if !config.enabled || !util.ContainsSome(config.allowedOrigins, wildcard, origin) {
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

func newConfigFunc(config *Config) *ConfigFunc {
	return &ConfigFunc{
		config: config,
	}
}

type ConfigFunc struct {
	config *Config
}

type EnableConfig func(config *ConfigFunc)

// AllowAllOrigins will explicitly set allowed origins to "*", allowing all origins.
//
// For more details see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Origin
func (corsF *ConfigFunc) AllowAllOrigins() *ConfigFunc {
	corsF.config.allowedOrigins = []string{"*"}
	return corsF
}

// AllowOrigins sets the allowed origins for cross origin requests
//
// For more details see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Origin
func (corsF *ConfigFunc) AllowOrigins(origins ...string) *ConfigFunc {
	corsF.config.allowedOrigins = origins
	return corsF
}

// AllowCredentials will allow for the user to send credentials using cross origin requests
//
// For more details see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Credentials
func (corsF *ConfigFunc) AllowCredentials() *ConfigFunc {
	corsF.config.allowCredentials = true
	return corsF
}

// AllowHeaders sets the allowed headers for CORS in addition to the safelisted headers
// found in https://developer.mozilla.org/en-US/docs/Glossary/CORS-safelisted_request_header. Defaults to "*"
//
// For more details see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Headers
func (corsF *ConfigFunc) AllowHeaders(headers ...string) *ConfigFunc {
	corsF.config.allowedHeaders = headers
	return corsF
}
