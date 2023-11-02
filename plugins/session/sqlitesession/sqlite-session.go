package sqlitesession

import (
	"log/slog"
	"os"

	"github.com/pkkummermo/govalin"
)

type Config struct {
	connectionString string
	useWAL           bool
}

func New() *Config {
	return &Config{
		connectionString: ":memory:",
		useWAL:           true,
	}
}

func (config *Config) Name() string {
	return "SQLite session plugin"
}

func (config *Config) OnInit(conf *govalin.Config) {
	initiatedStore, err := NewSqliteSessionStore(config.connectionString, config.useWAL)

	if err != nil {
		slog.Error("Failed to initiate the SQLite session store")
		os.Exit(1)
	}

	conf.EnableSessions(func(sessionConfig *govalin.SessionConfiguration) {
		sessionConfig.SessionStore(initiatedStore)
	})
}

func (config *Config) Apply(_ *govalin.App) {}

// ConnectionString sets the connection string for the SQLite database. Default is ":memory:".
func (config *Config) ConnectionString(connectionString string) *Config {
	config.connectionString = connectionString
	return config
}

// UseWAL sets whether to use WAL mode for the SQLite database. Default is true.
func (config *Config) UseWAL(useWAL bool) *Config {
	config.useWAL = useWAL
	return config
}
