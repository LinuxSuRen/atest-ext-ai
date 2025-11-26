package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// Logger is the shared structured logger used throughout the plugin.
var (
	Logger           *slog.Logger
	requestIDKey     contextKey = "atest-ext-ai-request-id"
	contextLoggerKey contextKey = "atest-ext-ai-context-logger"
)

type contextKey string

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

// WithRequestID embeds the provided request ID into the context for downstream logging.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestIDFromContext extracts the request ID from context, if present.
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if value, ok := ctx.Value(requestIDKey).(string); ok {
		return value
	}
	return ""
}

// FromContext returns a logger enriched with the request ID if present.
func FromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return Logger
	}
	if logger, ok := ctx.Value(contextLoggerKey).(*slog.Logger); ok && logger != nil {
		return logger
	}
	if requestID := RequestIDFromContext(ctx); requestID != "" {
		logger := Logger.With(slog.String("request_id", requestID))
		return logger
	}
	return Logger
}

// ContextWithLogger stores a preconfigured logger in the context.
func ContextWithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, contextLoggerKey, logger)
}
