package govalin

import (
	"time"

	"github.com/pkkummermo/govalin/internal/session"
)

const (
	defaultPort                      = 6060               // govalin default port.
	defaultMaxReadTimeout            = 10                 // maximum read timeout for requests.
	defaultMaxBodyReadSize     int64 = 4096               //  Default max body read size.
	defaultMaxMultipartMemory  int64 = 32 << 20           // Default in-memory budget for multipart parts.
	defaultMaxMultipartSize    int64 = 128 << 20          // Default max size of a multipart body.
	defaultShutdownTimeoutInMS       = 200                // Max time for shutdown.
	defaultSessionExpireTime         = 3600 * time.Second // Default session expire time.
)

// ConfigFunc gives a config function that will generate a Config
// for the Govalin object.
type ConfigFunc func(config *Config)

type serverConfig struct {
	port                uint16
	maxReadTimeout      int64
	maxBodyReadSize     int64
	maxMultipartMemory  int64
	maxMultipartSize    int64
	shutdownTimeoutInMS int64
	accessLogEnabled    bool
	startupLogEnabled   bool
	plugins             []Plugin
	sessionsEnabled     bool
	sessionStore        session.Store
	sessionExpireTime   time.Duration
	events              ServerEvents
}

type ServerEvents struct {
	onServerStartup  []OnServerStartup
	onServerShutdown []OnServerShutdown
	onRouteAdded     []OnRouteAdded
}

type OnServerStartup func()
type OnServerShutdown func()
type OnRouteAdded func(method string, path string, handler HandlerFunc)

func (events *ServerEvents) AddOnServerStartup(event OnServerStartup) {
	events.onServerStartup = append(events.onServerStartup, event)
}
func (events *ServerEvents) AddOnServerShutdown(event OnServerShutdown) {
	events.onServerShutdown = append(events.onServerShutdown, event)
}
func (events *ServerEvents) AddOnRouteAdded(event OnRouteAdded) {
	events.onRouteAdded = append(events.onRouteAdded, event)
}

// Config contains configuration for a Govalin instance.
type Config struct {
	server serverConfig
}

// Plugin lets you to provide a Plugin that can interact on the Govalin
// instance.
func (config *Config) Plugin(plugin Plugin) *Config {
	config.server.plugins = append(config.server.plugins, plugin)
	return config
}

// Events allows you to subscribe to server events such as startup and shutdown on the Govalin instance.
func (config *Config) Events(eventFunc func(serverEvents *ServerEvents)) *Config {
	eventFunc(&config.server.events)
	return config
}

// Port sets the default port of the Govalin instance.
func (config *Config) Port(port uint16) *Config {
	config.server.port = port
	return config
}

// ServerMaxBodyReadSize sets the max read size to accept from POST requests.
//
// The server will error if the body size is too big and refuse to handle the
// request further. This is to control DDoS attacks using big body sizes.
func (config *Config) ServerMaxBodyReadSize(maxReadSize int64) *Config {
	config.server.maxBodyReadSize = maxReadSize
	return config
}

// ServerMaxMultipartMemory sets how much of a multipart form is held in memory.
//
// Parts beyond it spill to temporary files. It does not cap the request itself;
// use ServerMaxMultipartSize for that.
func (config *Config) ServerMaxMultipartMemory(maxMemory int64) *Config {
	config.server.maxMultipartMemory = maxMemory
	return config
}

// ServerMaxMultipartSize sets the max size to accept for a multipart request body.
//
// Uploads are bounded by this rather than by ServerMaxBodyReadSize, whose
// default is sized for JSON payloads.
func (config *Config) ServerMaxMultipartSize(maxSize int64) *Config {
	config.server.maxMultipartSize = maxSize
	return config
}

// ServerMaxReadTimeout sets the max read timeout for requests towards the Govalin server.
func (config *Config) ServerMaxReadTimeout(timeout int64) *Config {
	config.server.maxReadTimeout = timeout
	return config
}

// EnableSessions configures govalin to use sessions for all requests.
func (config *Config) EnableSessions(confFunc ...SessionConfigFunc) *Config {
	configuredSession := SessionConfiguration{
		sessionExpireTime: defaultSessionExpireTime,
		sessionStore:      session.NewInMemoryStore(),
	}

	if (len(confFunc)) > 0 {
		confFunc[0](&configuredSession)
	}

	config.server.sessionsEnabled = true
	config.server.sessionExpireTime = configuredSession.sessionExpireTime
	config.server.sessionStore = configuredSession.sessionStore

	return config
}

// ServerShutdownTimeout sets the max timeout for before forcefully shutting the server down.
func (config *Config) ServerShutdownTimeout(timeout int64) *Config {
	config.server.shutdownTimeoutInMS = timeout
	return config
}

// EnableAccessLog enables access logging for the server. Default is enabled.
func (config *Config) EnableAccessLog(enabled bool) *Config {
	config.server.accessLogEnabled = enabled
	return config
}

func (config *Config) EnableStartupLog(enabled bool) *Config {
	config.server.startupLogEnabled = enabled
	return config
}

func newConfig() *Config {
	return &Config{
		server: serverConfig{
			port:                defaultPort,
			maxReadTimeout:      defaultMaxReadTimeout,
			maxBodyReadSize:     defaultMaxBodyReadSize,
			maxMultipartMemory:  defaultMaxMultipartMemory,
			maxMultipartSize:    defaultMaxMultipartSize,
			shutdownTimeoutInMS: defaultShutdownTimeoutInMS,
			sessionsEnabled:     false,
			accessLogEnabled:    true,
			startupLogEnabled:   true,
			events: ServerEvents{
				onServerStartup:  []OnServerStartup{},
				onServerShutdown: []OnServerShutdown{},
				onRouteAdded:     []OnRouteAdded{},
			},
		},
	}
}

type SessionConfiguration struct {
	sessionExpireTime time.Duration
	sessionStore      session.Store
}

// SessionExpireTime sets the expire time for sessions.
func (config *SessionConfiguration) SessionExpireTime(expireTime time.Duration) *SessionConfiguration {
	config.sessionExpireTime = expireTime
	return config
}

// SessionStore sets the session store to use.
func (config *SessionConfiguration) SessionStore(store session.Store) *SessionConfiguration {
	config.sessionStore = store
	return config
}

type SessionConfigFunc func(sessionConfig *SessionConfiguration)
