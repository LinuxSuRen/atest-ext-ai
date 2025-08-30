package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"atest-ext-ai-core/internal/config"
	"atest-ext-ai-core/internal/errors"
)

// LogLevel represents the logging level
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger represents the application logger
type Logger struct {
	level  LogLevel
	format string
	logger *log.Logger
}

// NewLogger creates a new logger instance
func NewLogger(cfg *config.LoggingConfig) *Logger {
	level := parseLogLevel(cfg.Level)
	format := cfg.Format
	if format == "" {
		format = "text"
	}

	// Create logger with appropriate output
	var output io.Writer
	if strings.ToLower(cfg.Output) == "stderr" {
		output = os.Stderr
	} else {
		output = os.Stdout
	}

	logger := log.New(output, "", 0)

	return &Logger{
		level:  level,
		format: format,
		logger: logger,
	}
}

// parseLogLevel parses string log level to LogLevel
func parseLogLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN", "WARNING":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return INFO
	}
}

// shouldLog checks if the message should be logged based on level
func (l *Logger) shouldLog(level LogLevel) bool {
	return level >= l.level
}

// formatMessage formats the log message
func (l *Logger) formatMessage(level LogLevel, message string, fields map[string]interface{}) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	if l.format == "json" {
		return l.formatJSON(timestamp, level, message, fields)
	}
	return l.formatText(timestamp, level, message, fields)
}

// formatText formats message as plain text
func (l *Logger) formatText(timestamp string, level LogLevel, message string, fields map[string]interface{}) string {
	result := fmt.Sprintf("[%s] %s: %s", timestamp, level.String(), message)

	if len(fields) > 0 {
		result += " |"
		for key, value := range fields {
			result += fmt.Sprintf(" %s=%v", key, value)
		}
	}

	return result
}

// formatJSON formats message as JSON (simplified)
func (l *Logger) formatJSON(timestamp string, level LogLevel, message string, fields map[string]interface{}) string {
	json := fmt.Sprintf(`{"timestamp":"%s","level":"%s","message":"%s"`, timestamp, level.String(), message)

	for key, value := range fields {
		json += fmt.Sprintf(`,"%s":"%v"`, key, value)
	}

	json += "}"
	return json
}

// log is the internal logging method
func (l *Logger) log(level LogLevel, message string, fields map[string]interface{}) {
	if !l.shouldLog(level) {
		return
	}

	formatted := l.formatMessage(level, message, fields)
	l.logger.Println(formatted)

	// Exit on fatal errors
	if level == FATAL {
		os.Exit(1)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(DEBUG, message, f)
}

// Info logs an info message
func (l *Logger) Info(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(INFO, message, f)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(WARN, message, f)
}

// Error logs an error message
func (l *Logger) Error(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(ERROR, message, f)
}

// ErrorWithErr logs an error with underlying error details
func (l *Logger) ErrorWithErr(msg string, err error) {
	fields := make(map[string]interface{})
	if appErr := errors.GetAppError(err); appErr != nil {
		fields["error_code"] = appErr.Code
		fields["error_details"] = appErr.Details
		fields["error_context"] = appErr.Context
		if appErr.Cause != nil {
			fields["underlying_error"] = appErr.Cause.Error()
		}
	} else {
		fields["error"] = err.Error()
	}
	l.log(ERROR, msg, fields)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(FATAL, message, f)
}

// FatalWithErr logs a fatal message with error details and exits
func (l *Logger) FatalWithErr(msg string, err error) {
	l.ErrorWithErr(msg, err)
	os.Exit(1)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Debug(fmt.Sprintf(format, args...))
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Info(fmt.Sprintf(format, args...))
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Warn(fmt.Sprintf(format, args...))
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Error(fmt.Sprintf(format, args...))
}

// ErrorfWithErr logs a formatted error message with underlying error
func (l *Logger) ErrorfWithErr(err error, format string, args ...interface{}) {
	l.ErrorWithErr(fmt.Sprintf(format, args...), err)
}

// Fatalf logs a formatted fatal message and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Fatal(fmt.Sprintf(format, args...))
}

// FatalfWithErr logs a formatted fatal message with error and exits
func (l *Logger) FatalfWithErr(err error, format string, args ...interface{}) {
	l.FatalWithErr(fmt.Sprintf(format, args...), err)
}

// WithFields returns a new logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *FieldLogger {
	return &FieldLogger{
		logger: l,
		fields: fields,
	}
}

// FieldLogger is a logger with predefined fields
type FieldLogger struct {
	logger *Logger
	fields map[string]interface{}
}

// Debug logs a debug message with fields
func (fl *FieldLogger) Debug(message string) {
	fl.logger.log(DEBUG, message, fl.fields)
}

// Info logs an info message with fields
func (fl *FieldLogger) Info(message string) {
	fl.logger.log(INFO, message, fl.fields)
}

// Warn logs a warning message with fields
func (fl *FieldLogger) Warn(message string) {
	fl.logger.log(WARN, message, fl.fields)
}

// Error logs an error message with fields
func (fl *FieldLogger) Error(message string) {
	fl.logger.log(ERROR, message, fl.fields)
}

// Fatal logs a fatal message with fields and exits
func (fl *FieldLogger) Fatal(message string) {
	fl.logger.log(FATAL, message, fl.fields)
}

// Global logger instance
var globalLogger *Logger

// InitGlobalLogger initializes the global logger
func InitGlobalLogger(cfg *config.LoggingConfig) {
	globalLogger = NewLogger(cfg)
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	if globalLogger == nil {
		// Create default logger if not initialized
		globalLogger = NewLogger(&config.LoggingConfig{
			Level:  "info",
			Format: "text",
		})
	}
	return globalLogger
}

// Global logging functions
func Debug(message string, fields ...map[string]interface{}) {
	GetGlobalLogger().Debug(message, fields...)
}

func Info(message string, fields ...map[string]interface{}) {
	GetGlobalLogger().Info(message, fields...)
}

func Warn(message string, fields ...map[string]interface{}) {
	GetGlobalLogger().Warn(message, fields...)
}

func Error(message string, fields ...map[string]interface{}) {
	GetGlobalLogger().Error(message, fields...)
}

// ErrorWithErr logs an error with underlying error using the global logger
func ErrorWithErr(msg string, err error) {
	GetGlobalLogger().ErrorWithErr(msg, err)
}

func Fatal(message string, fields ...map[string]interface{}) {
	GetGlobalLogger().Fatal(message, fields...)
}

// FatalWithErr logs a fatal message with error using the global logger and exits
func FatalWithErr(msg string, err error) {
	GetGlobalLogger().FatalWithErr(msg, err)
}

func Debugf(format string, args ...interface{}) {
	GetGlobalLogger().Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	GetGlobalLogger().Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	GetGlobalLogger().Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	GetGlobalLogger().Errorf(format, args...)
}

// ErrorfWithErr logs a formatted error message with underlying error using the global logger
func ErrorfWithErr(err error, format string, args ...interface{}) {
	GetGlobalLogger().ErrorfWithErr(err, format, args...)
}

func Fatalf(format string, args ...interface{}) {
	GetGlobalLogger().Fatalf(format, args...)
}

// FatalfWithErr logs a formatted fatal message with error using the global logger and exits
func FatalfWithErr(err error, format string, args ...interface{}) {
	GetGlobalLogger().FatalfWithErr(err, format, args...)
}
