package govalintesting

import (
	"os"

	"log/slog"
)

var (
	log = slog.New(slog.NewJSONHandler(os.Stdout, nil))
)
