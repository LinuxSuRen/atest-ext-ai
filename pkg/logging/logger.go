package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Logger is the shared structured logger used throughout the plugin.
var Logger *slog.Logger

func init() {
	level := strings.ToLower(os.Getenv("LOG_LEVEL"))

	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	// Use JSON handler for production, text handler for development
	var handler slog.Handler
	if os.Getenv("APP_ENV") == "development" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	Logger = slog.New(handler)
}
