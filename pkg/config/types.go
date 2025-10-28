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
	Server   ServerConfig   `yaml:"server" json:"server" validate:"required"`
	Plugin   PluginConfig   `yaml:"plugin" json:"plugin" validate:"required"`
	AI       AIConfig       `yaml:"ai" json:"ai" validate:"required"`
	Database DatabaseConfig `yaml:"database" json:"database"`
	Logging  LoggingConfig  `yaml:"logging" json:"logging"`
}

// ServerConfig contains server-specific configuration
type ServerConfig struct {
	Host          string   `yaml:"host" json:"host"`
	Port          int      `yaml:"port" json:"port" validate:"min=1,max=65535"`
	Timeout       Duration `yaml:"timeout" json:"timeout"`
	MaxConns      int      `yaml:"max_connections" json:"max_connections"`
	SocketPath    string   `yaml:"socket_path" json:"socket_path"`
	ListenAddress string   `yaml:"listen_address" json:"listen_address"`
	ReadTimeout   Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout  Duration `yaml:"write_timeout" json:"write_timeout"`
}

// PluginConfig contains plugin-specific configuration
type PluginConfig struct {
	Name        string `yaml:"name" json:"name"`
	Version     string `yaml:"version" json:"version"`
	Debug       bool   `yaml:"debug" json:"debug"`
	LogLevel    string `yaml:"log_level" json:"log_level"`
	Environment string `yaml:"environment" json:"environment"`
}

// AIConfig contains AI service configuration
type AIConfig struct {
	DefaultService string               `yaml:"default_service" json:"default_service"`
	Services       map[string]AIService `yaml:"services" json:"services"`
	Fallback       []string             `yaml:"fallback_order" json:"fallback_order"`
	Timeout        Duration             `yaml:"timeout" json:"timeout"`
	RateLimit      RateLimitConfig      `yaml:"rate_limit" json:"rate_limit"`
	Retry          RetryConfig          `yaml:"retry" json:"retry"`
}

// AIService represents configuration for a specific AI service
type AIService struct {
	Enabled   bool              `yaml:"enabled" json:"enabled"`
	Provider  string            `yaml:"provider" json:"provider"`
	Endpoint  string            `yaml:"endpoint" json:"endpoint"`
	APIKey    string            `yaml:"api_key" json:"api_key"`
	Model     string            `yaml:"model" json:"model"`
	MaxTokens int               `yaml:"max_tokens" json:"max_tokens"`
	TopP      float32           `yaml:"top_p" json:"top_p"`
	Headers   map[string]string `yaml:"headers" json:"headers"`
	Models    []string          `yaml:"models" json:"models"`
	Priority  int               `yaml:"priority" json:"priority"`
	Timeout   Duration          `yaml:"timeout" json:"timeout"`

	// Deprecated fields (kept for backward compatibility warning)
	Temperature float32 `yaml:"temperature" json:"temperature,omitempty"`
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
	Enabled           bool     `yaml:"enabled" json:"enabled"`
	RequestsPerMinute int      `yaml:"requests_per_minute" json:"requests_per_minute"`
	BurstSize         int      `yaml:"burst_size" json:"burst_size"`
	WindowSize        Duration `yaml:"window_size" json:"window_size"`
}

// RetryConfig contains retry configuration
type RetryConfig struct {
	Enabled      bool     `yaml:"enabled" json:"enabled"`
	MaxAttempts  int      `yaml:"max_attempts" json:"max_attempts"`
	InitialDelay Duration `yaml:"initial_delay" json:"initial_delay"`
	MaxDelay     Duration `yaml:"max_delay" json:"max_delay"`
	Multiplier   float32  `yaml:"multiplier" json:"multiplier"`
	Jitter       bool     `yaml:"jitter" json:"jitter"`
}

// DatabaseConfig contains database configuration (optional)
type DatabaseConfig struct {
	Enabled     bool     `yaml:"enabled" json:"enabled"`
	Driver      string   `yaml:"driver" json:"driver"`
	DSN         string   `yaml:"dsn" json:"dsn"`
	DefaultType string   `yaml:"default_type" json:"default_type"`
	MaxConns    int      `yaml:"max_connections" json:"max_connections"`
	MaxIdle     int      `yaml:"max_idle" json:"max_idle"`
	MaxLifetime Duration `yaml:"max_lifetime" json:"max_lifetime"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string        `yaml:"level" json:"level"`
	Format string        `yaml:"format" json:"format"`
	Output string        `yaml:"output" json:"output"`
	File   LogFileConfig `yaml:"file" json:"file"`
}

// LogFileConfig contains log file configuration
type LogFileConfig struct {
	Path       string `yaml:"path" json:"path"`
	MaxSize    string `yaml:"max_size" json:"max_size"`
	MaxBackups int    `yaml:"max_backups" json:"max_backups"`
	MaxAge     int    `yaml:"max_age" json:"max_age"`
	Compress   bool   `yaml:"compress" json:"compress"`
}
