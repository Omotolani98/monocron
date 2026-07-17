package logging

import (
	"log/slog"
	"os"
)

// New creates a structured logger with the requested level.
func New(level string) *slog.Logger {
	var lv slog.Level
	switch level {
	case "debug":
		lv = slog.LevelDebug
	case "warn":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lv})
	return slog.New(h)
}

// SetDefault configures the default slog logger.
func SetDefault(level string) {
	slog.SetDefault(New(level))
}
