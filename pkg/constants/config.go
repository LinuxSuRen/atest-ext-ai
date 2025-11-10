package constants

import "time"

// TimeoutDefaults groups commonly used timeout values to prevent magic numbers.
type TimeoutDefaults struct {
	Server    time.Duration
	Read      time.Duration
	Write     time.Duration
	AI        time.Duration
	Ollama    time.Duration
	Shutdown  time.Duration
	Discovery time.Duration
}

// Timeouts contains the canonical timeout values for the plugin.
var Timeouts = TimeoutDefaults{
	Server:    30 * time.Second,
	Read:      15 * time.Second,
	Write:     15 * time.Second,
	AI:        60 * time.Second,
	Ollama:    60 * time.Second,
	Shutdown:  30 * time.Second,
	Discovery: 5 * time.Second,
}

// ServerConfigDefaults lists server-specific numeric defaults.
type ServerConfigDefaults struct {
	MaxConnections int
}

// ServerDefaults centralizes limits applied to the embedded gRPC server.
var ServerDefaults = ServerConfigDefaults{
	MaxConnections: 100,
}

// RetryPolicyDefaults captures retry strategy values for AI providers.
type RetryPolicyDefaults struct {
	Enabled      bool
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float32
	Jitter       bool
}

// Retry contains the default retry policy for AI calls.
var Retry = RetryPolicyDefaults{
	Enabled:      true,
	MaxAttempts:  3,
	InitialDelay: 1 * time.Second,
	MaxDelay:     30 * time.Second,
	Multiplier:   2.0,
	Jitter:       true,
}

// RateLimitDefaults describes how API rate limiting should behave by default.
type RateLimitDefaults struct {
	Enabled           bool
	RequestsPerMinute int
	BurstSize         int
	WindowSize        time.Duration
}

// RateLimit provides the builtin rate limit settings for AI requests.
var RateLimit = RateLimitDefaults{
	Enabled:           true,
	RequestsPerMinute: 60,
	BurstSize:         10,
	WindowSize:        1 * time.Minute,
}

// DatabasePoolDefaults outlines default values for database connection pools.
type DatabasePoolDefaults struct {
	MaxConns    int
	MaxIdle     int
	MaxLifetime time.Duration
}

// DatabasePool configures how the default SQL connection pool behaves.
var DatabasePool = DatabasePoolDefaults{
	MaxConns:    10,
	MaxIdle:     5,
	MaxLifetime: 1 * time.Hour,
}

// LogFileDefaults sets the default rolling file configuration.
type LogFileDefaults struct {
	Path       string
	MaxSize    string
	MaxBackups int
	MaxAge     int
	Compress   bool
}

// LogFile contains sane defaults for structured log files.
var LogFile = LogFileDefaults{
	Path:       DefaultLogFilePath,
	MaxSize:    DefaultLogFileSize,
	MaxBackups: 3,
	MaxAge:     28,
	Compress:   true,
}

// RuntimeDefaults configures basic runtime tuning knobs.
type RuntimeDefaults struct {
	GCPercent int
	MaxProcs  int
}

// Runtime centralizes values used to constrain runtime resource usage.
var Runtime = RuntimeDefaults{
	GCPercent: 50,
	MaxProcs:  2,
}
