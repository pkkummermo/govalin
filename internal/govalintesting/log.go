package govalintesting

import (
	"os"

	"golang.org/x/exp/slog"
)

var (
	log = slog.New(slog.NewJSONHandler(os.Stdout, nil))
)
