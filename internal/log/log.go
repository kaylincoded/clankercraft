package log

import (
	"log/slog"
	"os"
	"strings"
)

// Setup creates a structured JSON logger writing to stderr.
func Setup(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: lvl})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}
