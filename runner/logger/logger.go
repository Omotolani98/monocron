package logger

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/google/uuid"
)

const (
	INFO  = "INFO"
	DEBUG = "DEBUG"
	WARN  = "WARN"
)

var Child *slog.Logger
var level slog.Level

func InitLogger() {
	switch os.Getenv("LEVEL") {
	case DEBUG:
		level = slog.LevelDebug
	case WARN:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	logger := slog.New(slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level: level,
		}),
	)
	slog.SetDefault(logger)
	Child = logger.WithGroup("info").With("request-id", uuid.New().String())
}

func LogReq(msg string, err error, method string, body any) {
	switch method {
	case http.MethodGet:
		if err != nil {
			Child.LogAttrs(
				context.Background(),
				level,
				msg,
				slog.String("method", method),
				slog.Any("error", err),
			)
		}
		Child.LogAttrs(
			context.Background(),
			level,
			msg,
			slog.String("method", method),
		)
	case http.MethodPost:
		Child.LogAttrs(
			context.Background(),
			level,
			msg,
			slog.String("method", method),
			slog.Any("body", body),
		)
	}
}
