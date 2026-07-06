package logs

import (
	"log/slog"
	"os"
)

// Setup configures slog.Default() with a JSON handler wrapped in a
// SanitizingHandler that masks BlockedFields.
// Every log line automatically includes "service" and "env" fields.
func Setup(level, service, env string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	inner := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	sanitized := NewSanitizingHandler(inner, BlockedFields)
	logger := slog.New(sanitized).With(
		slog.String("service", service),
		slog.String("env", env),
	)
	slog.SetDefault(logger)
}
