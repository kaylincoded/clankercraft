package log

import (
	"log/slog"
	"os"
	"strings"

	charmlog "github.com/charmbracelet/log"
)

// Setup creates a pretty terminal logger writing to stderr.
func Setup(level string) *slog.Logger {
	var lvl charmlog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = charmlog.DebugLevel
	case "warn":
		lvl = charmlog.WarnLevel
	case "error":
		lvl = charmlog.ErrorLevel
	default:
		lvl = charmlog.InfoLevel
	}

	charm := charmlog.NewWithOptions(os.Stderr, charmlog.Options{
		Level:           lvl,
		ReportTimestamp: true,
	})

	logger := slog.New(charm)
	slog.SetDefault(logger)
	return logger
}
