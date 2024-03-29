package logging

import (
	"os"

	"log/slog"

	"github.com/pkkummermo/govalin"
)

type StructuredLoggingConfig struct {
	logLevel slog.Level
}

// NewStructuredLogging configures slog to use structured logging as default.
func NewStructuredLogging() *StructuredLoggingConfig {
	return &StructuredLoggingConfig{
		logLevel: slog.LevelInfo,
	}
}

func (config *StructuredLoggingConfig) Name() string {
	return "Structured Logging plugin"
}

func (config *StructuredLoggingConfig) OnInit(_ *govalin.Config) {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource:   false,
		Level:       config.logLevel,
		ReplaceAttr: nil,
	})
	slog.SetDefault(slog.New(handler))
}

func (config *StructuredLoggingConfig) Apply(_ *govalin.App) {
}

func (config *StructuredLoggingConfig) LogLevel(logLevel slog.Level) *StructuredLoggingConfig {
	config.logLevel = logLevel

	return config
}
