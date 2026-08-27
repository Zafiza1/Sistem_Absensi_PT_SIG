// Package logger provides a single structured logger for the whole
// application, built on the standard library's log/slog. Using slog keeps
// the dependency footprint minimal while still giving us structured,
// leveled, JSON-capable logging suitable for production log aggregation.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New builds a slog.Logger configured for the given environment and level.
//
// In "production"/"staging" it emits JSON (one object per line, easy to
// ship to a log aggregator). In "development" it emits human-readable text
// to make local debugging pleasant.
func New(env, level string) *slog.Logger {
	handlerOpts := &slog.HandlerOptions{
		Level:     parseLevel(level),
		AddSource: false,
	}

	var handler slog.Handler
	if strings.EqualFold(env, "production") || strings.EqualFold(env, "staging") {
		handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	}

	logger := slog.New(handler).With(
		slog.String("service", "absensi-backend"),
		slog.String("env", env),
	)
	slog.SetDefault(logger)
	return logger
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
