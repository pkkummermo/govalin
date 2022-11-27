package govalin

const (
	defaultPort                      = 6060 // govalin default port.
	defaultMaxReadTimeout            = 10   // maximum read timeout for requests.
	defaultMaxBodyReadSize     int64 = 4096 //  Default max body read size.
	defaultShutdownTimeoutInMS       = 200  // Max time for shutdown.
)

// ConfigFunc gives a config function that will generate a Config
// for the Govalin object.
type ConfigFunc func(config *Config)

type serverConfig struct {
	port                uint16
	maxReadTimeout      int64
	maxBodyReadSize     int64
	shutdownTimeoutInMS int64
	plugins             []Plugin
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

// ServerMaxReadTimeout sets the max read timeout for requests towards the Govalin server.
func (config *Config) ServerMaxReadTimeout(timeout int64) *Config {
	config.server.maxReadTimeout = timeout
	return config
}

// ServerShutdownTimeout sets the max timeout for before forcefully shutting the server down.
func (config *Config) ServerShutdownTimeout(timeout int64) *Config {
	config.server.shutdownTimeoutInMS = timeout
	return config
}

func newConfig() *Config {
	return &Config{
		server: serverConfig{
			port:                defaultPort,
			maxReadTimeout:      defaultMaxReadTimeout,
			maxBodyReadSize:     defaultMaxBodyReadSize,
			shutdownTimeoutInMS: defaultShutdownTimeoutInMS,
		},
	}
}
