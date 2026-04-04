package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

func Default(logger *slog.Logger) *slog.Logger {
	if logger != nil {
		return logger
	}

	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func New() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: parseLevel(os.Getenv("LOG_LEVEL")),
	}))
}

func parseLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "info", "":
		fallthrough
	default:
		return slog.LevelInfo
	}
}

func Event(ctx context.Context, logger *slog.Logger, level slog.Level, operation string, connector string, outcome string, attrs ...slog.Attr) {
	base := []slog.Attr{
		slog.String("operation", operation),
		slog.String("connector", connector),
		slog.String("outcome", outcome),
	}
	base = append(base, attrs...)

	Default(logger).LogAttrs(ctx, level, "auth operation", base...)
}
