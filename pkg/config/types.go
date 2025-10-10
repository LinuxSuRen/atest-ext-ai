/*
Copyright 2025 API Testing Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

// No imports needed for this file

// Config represents the complete application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server" yaml:"server" json:"server" toml:"server" validate:"required"`
	Plugin   PluginConfig   `mapstructure:"plugin" yaml:"plugin" json:"plugin" toml:"plugin" validate:"required"`
	AI       AIConfig       `mapstructure:"ai" yaml:"ai" json:"ai" toml:"ai" validate:"required"`
	Database DatabaseConfig `mapstructure:"database" yaml:"database" json:"database" toml:"database"`
	Logging  LoggingConfig  `mapstructure:"logging" yaml:"logging" json:"logging" toml:"logging"`
}

// ServerConfig contains server-specific configuration
type ServerConfig struct {
	Host         string   `mapstructure:"host" yaml:"host" json:"host" toml:"host" validate:"required,hostname_rfc1123"`
	Port         int      `mapstructure:"port" yaml:"port" json:"port" toml:"port" validate:"required,min=1,max=65535"`
	Timeout      Duration `mapstructure:"timeout" yaml:"timeout" json:"timeout" toml:"timeout" validate:"required,duration_gt_zero"`
	MaxConns     int      `mapstructure:"max_connections" yaml:"max_connections" json:"max_connections" toml:"max_connections" validate:"min=1,max=10000"`
	SocketPath   string   `mapstructure:"socket_path" yaml:"socket_path" json:"socket_path" toml:"socket_path" validate:"required"`
	ReadTimeout  Duration `mapstructure:"read_timeout" yaml:"read_timeout" json:"read_timeout" toml:"read_timeout"`
	WriteTimeout Duration `mapstructure:"write_timeout" yaml:"write_timeout" json:"write_timeout" toml:"write_timeout"`
}

// PluginConfig contains plugin-specific configuration
type PluginConfig struct {
	Name        string `mapstructure:"name" yaml:"name" json:"name" toml:"name" validate:"required,min=1"`
	Version     string `mapstructure:"version" yaml:"version" json:"version" toml:"version" validate:"required,semver"`
	Debug       bool   `mapstructure:"debug" yaml:"debug" json:"debug" toml:"debug"`
	LogLevel    string `mapstructure:"log_level" yaml:"log_level" json:"log_level" toml:"log_level" validate:"oneof=debug info warn error"`
	Environment string `mapstructure:"environment" yaml:"environment" json:"environment" toml:"environment" validate:"oneof=development staging production"`
}

// AIConfig contains AI service configuration
type AIConfig struct {
	DefaultService string               `mapstructure:"default_service" yaml:"default_service" json:"default_service" toml:"default_service" validate:"required,oneof=ollama openai claude deepseek local custom"`
	Services       map[string]AIService `mapstructure:"services" yaml:"services" json:"services" toml:"services" validate:"required,dive"`
	Fallback       []string             `mapstructure:"fallback_order" yaml:"fallback_order" json:"fallback_order" toml:"fallback_order"`
	Timeout        Duration             `mapstructure:"timeout" yaml:"timeout" json:"timeout" toml:"timeout" validate:"required"`
	RateLimit      RateLimitConfig      `mapstructure:"rate_limit" yaml:"rate_limit" json:"rate_limit" toml:"rate_limit"`
	Retry          RetryConfig          `mapstructure:"retry" yaml:"retry" json:"retry" toml:"retry"`
}

// AIService represents configuration for a specific AI service
type AIService struct {
	Enabled   bool              `mapstructure:"enabled" yaml:"enabled" json:"enabled" toml:"enabled"`
	Provider  string            `mapstructure:"provider" yaml:"provider" json:"provider" toml:"provider" validate:"required,oneof=ollama openai claude deepseek local custom"`
	Endpoint  string            `mapstructure:"endpoint" yaml:"endpoint" json:"endpoint" toml:"endpoint"`
	APIKey    string            `mapstructure:"api_key" yaml:"api_key" json:"api_key" toml:"api_key"`
	Model     string            `mapstructure:"model" yaml:"model" json:"model" toml:"model" validate:"required,min=1"`
	MaxTokens int               `mapstructure:"max_tokens" yaml:"max_tokens" json:"max_tokens" toml:"max_tokens" validate:"min=1,max=100000"`
	TopP      float32           `mapstructure:"top_p" yaml:"top_p" json:"top_p" toml:"top_p" validate:"min=0,max=1"`
	Headers   map[string]string `mapstructure:"headers" yaml:"headers" json:"headers" toml:"headers"`
	Models    []string          `mapstructure:"models" yaml:"models" json:"models" toml:"models"`
	Priority  int               `mapstructure:"priority" yaml:"priority" json:"priority" toml:"priority" validate:"min=1,max=10"`
	Timeout   Duration          `mapstructure:"timeout" yaml:"timeout" json:"timeout" toml:"timeout"`

	// Deprecated fields (kept for backward compatibility warning)
	Temperature float32 `mapstructure:"temperature" yaml:"temperature" json:"temperature,omitempty" toml:"temperature"`
}

// ValidateAndWarnDeprecated checks for deprecated fields and returns warnings
func (s *AIService) ValidateAndWarnDeprecated() []string {
	var warnings []string
	if s.Temperature != 0 {
		warnings = append(warnings, "Temperature field is deprecated and will be ignored. Configure temperature when creating the LLM client if needed.")
	}
	return warnings
}

// RateLimitConfig contains rate limiting configuration
type RateLimitConfig struct {
	Enabled           bool     `mapstructure:"enabled" yaml:"enabled" json:"enabled" toml:"enabled"`
	RequestsPerMinute int      `mapstructure:"requests_per_minute" yaml:"requests_per_minute" json:"requests_per_minute" toml:"requests_per_minute" validate:"min=1"`
	BurstSize         int      `mapstructure:"burst_size" yaml:"burst_size" json:"burst_size" toml:"burst_size" validate:"min=1"`
	WindowSize        Duration `mapstructure:"window_size" yaml:"window_size" json:"window_size" toml:"window_size"`
}

// RetryConfig contains retry configuration
type RetryConfig struct {
	Enabled      bool     `mapstructure:"enabled" yaml:"enabled" json:"enabled" toml:"enabled"`
	MaxAttempts  int      `mapstructure:"max_attempts" yaml:"max_attempts" json:"max_attempts" toml:"max_attempts" validate:"min=1,max=10"`
	InitialDelay Duration `mapstructure:"initial_delay" yaml:"initial_delay" json:"initial_delay" toml:"initial_delay"`
	MaxDelay     Duration `mapstructure:"max_delay" yaml:"max_delay" json:"max_delay" toml:"max_delay"`
	Multiplier   float32  `mapstructure:"multiplier" yaml:"multiplier" json:"multiplier" toml:"multiplier" validate:"min=1"`
	Jitter       bool     `mapstructure:"jitter" yaml:"jitter" json:"jitter" toml:"jitter"`
}

// DatabaseConfig contains database configuration (optional)
type DatabaseConfig struct {
	Enabled     bool     `mapstructure:"enabled" yaml:"enabled" json:"enabled" toml:"enabled"`
	Driver      string   `mapstructure:"driver" yaml:"driver" json:"driver" toml:"driver"`
	DSN         string   `mapstructure:"dsn" yaml:"dsn" json:"dsn" toml:"dsn"`
	DefaultType string   `mapstructure:"default_type" yaml:"default_type" json:"default_type" toml:"default_type" validate:"oneof=mysql postgresql sqlite oracle sqlserver"`
	MaxConns    int      `mapstructure:"max_connections" yaml:"max_connections" json:"max_connections" toml:"max_connections"`
	MaxIdle     int      `mapstructure:"max_idle" yaml:"max_idle" json:"max_idle" toml:"max_idle"`
	MaxLifetime Duration `mapstructure:"max_lifetime" yaml:"max_lifetime" json:"max_lifetime" toml:"max_lifetime"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string        `mapstructure:"level" yaml:"level" json:"level" toml:"level" validate:"oneof=debug info warn error"`
	Format string        `mapstructure:"format" yaml:"format" json:"format" toml:"format" validate:"oneof=json text"`
	Output string        `mapstructure:"output" yaml:"output" json:"output" toml:"output" validate:"oneof=stdout stderr file"`
	File   LogFileConfig `mapstructure:"file" yaml:"file" json:"file" toml:"file"`
}

// LogFileConfig contains log file configuration
type LogFileConfig struct {
	Path       string `mapstructure:"path" yaml:"path" json:"path" toml:"path"`
	MaxSize    string `mapstructure:"max_size" yaml:"max_size" json:"max_size"`
	MaxBackups int    `mapstructure:"max_backups" yaml:"max_backups" json:"max_backups" validate:"min=0"`
	MaxAge     int    `mapstructure:"max_age" yaml:"max_age" json:"max_age" validate:"min=0"`
	Compress   bool   `mapstructure:"compress" yaml:"compress" json:"compress" toml:"compress"`
}

// ConfigChangeCallback defines the callback function type for configuration changes
type ConfigChangeCallback func(key string, oldValue, newValue interface{})

// ConfigManager interface defines the contract for configuration management
type ConfigManager interface {
	// Load loads configuration from the specified paths
	Load(paths ...string) error

	// Get retrieves a configuration value by key
	Get(key string) interface{}

	// Set sets a configuration value by key
	Set(key string, value interface{}) error

	// Reload reloads the configuration from the source
	Reload() error

	// Watch registers a callback for configuration changes
	Watch(callback ConfigChangeCallback) error

	// Validate validates the current configuration
	Validate() error

	// Export exports configuration in the specified format
	Export(format string) ([]byte, error)

	// GetConfig returns the complete configuration
	GetConfig() *Config

	// Stop stops the configuration manager and cleans up resources
	Stop() error
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string      `json:"field"`
	Value   interface{} `json:"value"`
	Tag     string      `json:"tag"`
	Message string      `json:"message"`
}

func (e ValidationError) Error() string {
	return e.Message
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (errs ValidationErrors) Error() string {
	if len(errs) == 0 {
		return ""
	}
	if len(errs) == 1 {
		return errs[0].Error()
	}
	msg := "validation errors:\n"
	for _, err := range errs {
		msg += "  - " + err.Error() + "\n"
	}
	return msg
}

// LoadError represents a configuration loading error
type LoadError struct {
	Path   string `json:"path"`
	Format string `json:"format"`
	Err    error  `json:"error"`
}

func (e LoadError) Error() string {
	return "failed to load config from " + e.Path + " (" + e.Format + "): " + e.Err.Error()
}

// WatchError represents a configuration watching error
type WatchError struct {
	Path string `json:"path"`
	Err  error  `json:"error"`
}

func (e WatchError) Error() string {
	return "failed to watch config at " + e.Path + ": " + e.Err.Error()
}
