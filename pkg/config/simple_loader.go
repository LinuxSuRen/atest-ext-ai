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

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// LoadConfig loads configuration from file and environment variables
func LoadConfig() (*Config, error) {
	// 1. Try to load from config file
	cfg, err := loadConfigFile()
	if err != nil {
		// Config file not found or invalid - use defaults
		cfg = defaultConfig()
	}

	// 2. Apply environment variable overrides
	applyEnvOverrides(cfg)

	// 3. Apply default values for any missing fields
	applyDefaults(cfg)

	// 4. Validate configuration
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// loadConfigFile tries to find and load a config file from standard locations
func loadConfigFile() (*Config, error) {
	// Search paths in priority order
	searchPaths := []string{
		"config.yaml",
		"config.yml",
		"./config.yaml",
		"./config.yml",
		filepath.Join(os.Getenv("HOME"), ".config", "atest", "config.yaml"),
		"/etc/atest/config.yaml",
	}

	var lastErr error
	for _, path := range searchPaths {
		cfg, err := loadYAMLFile(path)
		if err == nil {
			return cfg, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("no config file found: %w", lastErr)
}

// loadYAMLFile loads configuration from a YAML file
func loadYAMLFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &cfg, nil
}

// applyEnvOverrides applies environment variable overrides to the configuration
func applyEnvOverrides(cfg *Config) {
	// Server configuration
	if host := os.Getenv("ATEST_EXT_AI_SERVER_HOST"); host != "" {
		cfg.Server.Host = host
	}
	if port := os.Getenv("ATEST_EXT_AI_SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = p
		}
	}
	if socketPath := os.Getenv("ATEST_EXT_AI_SERVER_SOCKET_PATH"); socketPath != "" {
		cfg.Server.SocketPath = socketPath
	}
	if timeout := os.Getenv("ATEST_EXT_AI_SERVER_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			cfg.Server.Timeout = Duration{Duration: d}
		}
	}

	// Plugin configuration
	if debug := os.Getenv("ATEST_EXT_AI_DEBUG"); debug != "" {
		cfg.Plugin.Debug = strings.ToLower(debug) == "true"
	}
	if logLevel := os.Getenv("ATEST_EXT_AI_LOG_LEVEL"); logLevel != "" {
		cfg.Plugin.LogLevel = logLevel
	}
	if env := os.Getenv("ATEST_EXT_AI_ENVIRONMENT"); env != "" {
		cfg.Plugin.Environment = env
	}

	// AI configuration
	if defaultService := os.Getenv("ATEST_EXT_AI_DEFAULT_SERVICE"); defaultService != "" {
		cfg.AI.DefaultService = defaultService
	}
	if defaultService := os.Getenv("ATEST_EXT_AI_AI_PROVIDER"); defaultService != "" {
		cfg.AI.DefaultService = defaultService
	}
	if timeout := os.Getenv("ATEST_EXT_AI_AI_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			cfg.AI.Timeout = Duration{Duration: d}
		}
	}

	// Initialize services map if nil
	if cfg.AI.Services == nil {
		cfg.AI.Services = make(map[string]AIService)
	}

	// Ollama service configuration
	if endpoint := os.Getenv("ATEST_EXT_AI_OLLAMA_ENDPOINT"); endpoint != "" {
		svc := cfg.AI.Services["ollama"]
		svc.Endpoint = endpoint
		cfg.AI.Services["ollama"] = svc
	}
	if model := os.Getenv("ATEST_EXT_AI_OLLAMA_MODEL"); model != "" {
		svc := cfg.AI.Services["ollama"]
		svc.Model = model
		cfg.AI.Services["ollama"] = svc
	}
	if model := os.Getenv("ATEST_EXT_AI_AI_MODEL"); model != "" {
		// Also check generic AI_MODEL env var
		svc := cfg.AI.Services["ollama"]
		if svc.Model == "" {
			svc.Model = model
		}
		cfg.AI.Services["ollama"] = svc
	}

	// OpenAI service configuration
	if apiKey := os.Getenv("ATEST_EXT_AI_OPENAI_API_KEY"); apiKey != "" {
		svc, ok := cfg.AI.Services["openai"]
		if !ok {
			svc = AIService{
				Enabled:  true,
				Provider: "openai",
			}
		}
		svc.APIKey = apiKey
		cfg.AI.Services["openai"] = svc
	}
	if model := os.Getenv("ATEST_EXT_AI_OPENAI_MODEL"); model != "" {
		svc := cfg.AI.Services["openai"]
		svc.Model = model
		cfg.AI.Services["openai"] = svc
	}

	// Claude service configuration
	if apiKey := os.Getenv("ATEST_EXT_AI_CLAUDE_API_KEY"); apiKey != "" {
		svc, ok := cfg.AI.Services["claude"]
		if !ok {
			svc = AIService{
				Enabled:  true,
				Provider: "claude",
			}
		}
		svc.APIKey = apiKey
		cfg.AI.Services["claude"] = svc
	}
	if model := os.Getenv("ATEST_EXT_AI_CLAUDE_MODEL"); model != "" {
		svc := cfg.AI.Services["claude"]
		svc.Model = model
		cfg.AI.Services["claude"] = svc
	}

	// Database configuration
	if enabled := os.Getenv("ATEST_EXT_AI_DATABASE_ENABLED"); enabled != "" {
		cfg.Database.Enabled = strings.ToLower(enabled) == "true"
	}
	if driver := os.Getenv("ATEST_EXT_AI_DATABASE_DRIVER"); driver != "" {
		cfg.Database.Driver = driver
	}
	if dsn := os.Getenv("ATEST_EXT_AI_DATABASE_DSN"); dsn != "" {
		cfg.Database.DSN = dsn
	}

	// Logging configuration
	if level := os.Getenv("ATEST_EXT_AI_LOG_LEVEL"); level != "" {
		cfg.Logging.Level = level
	}
	if format := os.Getenv("ATEST_EXT_AI_LOG_FORMAT"); format != "" {
		cfg.Logging.Format = format
	}
	if output := os.Getenv("ATEST_EXT_AI_LOG_OUTPUT"); output != "" {
		cfg.Logging.Output = output
	}
}

// applyDefaults applies default values for any missing configuration
func applyDefaults(cfg *Config) {
	// Server defaults
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.SocketPath == "" {
		cfg.Server.SocketPath = "/tmp/atest-ext-ai.sock"
	}
	if cfg.Server.Timeout.Duration == 0 {
		cfg.Server.Timeout = Duration{Duration: 30 * time.Second}
	}
	if cfg.Server.ReadTimeout.Duration == 0 {
		cfg.Server.ReadTimeout = Duration{Duration: 15 * time.Second}
	}
	if cfg.Server.WriteTimeout.Duration == 0 {
		cfg.Server.WriteTimeout = Duration{Duration: 15 * time.Second}
	}
	if cfg.Server.MaxConns == 0 {
		cfg.Server.MaxConns = 100
	}

	// Plugin defaults
	if cfg.Plugin.Name == "" {
		cfg.Plugin.Name = "atest-ext-ai"
	}
	if cfg.Plugin.Version == "" {
		cfg.Plugin.Version = "1.0.0"
	}
	if cfg.Plugin.LogLevel == "" {
		cfg.Plugin.LogLevel = "info"
	}
	if cfg.Plugin.Environment == "" {
		cfg.Plugin.Environment = "production"
	}

	// AI defaults
	if cfg.AI.DefaultService == "" {
		cfg.AI.DefaultService = "ollama"
	}
	if cfg.AI.Timeout.Duration == 0 {
		cfg.AI.Timeout = Duration{Duration: 60 * time.Second}
	}

	// Initialize services map if nil
	if cfg.AI.Services == nil {
		cfg.AI.Services = make(map[string]AIService)
	}

	// Ollama service defaults
	if _, exists := cfg.AI.Services["ollama"]; !exists {
		cfg.AI.Services["ollama"] = AIService{
			Enabled:   true,
			Provider:  "ollama",
			Endpoint:  "http://localhost:11434",
			Model:     "qwen2.5-coder:latest",
			MaxTokens: 4096,
			Priority:  1,
			Timeout:   Duration{Duration: 60 * time.Second},
		}
	} else {
		// Fill in missing fields for existing Ollama service
		svc := cfg.AI.Services["ollama"]
		if svc.Endpoint == "" {
			svc.Endpoint = "http://localhost:11434"
		}
		if svc.Model == "" {
			svc.Model = "qwen2.5-coder:latest"
		}
		if svc.MaxTokens == 0 {
			svc.MaxTokens = 4096
		}
		if svc.Timeout.Duration == 0 {
			svc.Timeout = Duration{Duration: 60 * time.Second}
		}
		if svc.Priority == 0 {
			svc.Priority = 1
		}
		cfg.AI.Services["ollama"] = svc
	}

	// Retry defaults
	if cfg.AI.Retry.MaxAttempts == 0 {
		cfg.AI.Retry.Enabled = true
		cfg.AI.Retry.MaxAttempts = 3
		cfg.AI.Retry.InitialDelay = Duration{Duration: 1 * time.Second}
		cfg.AI.Retry.MaxDelay = Duration{Duration: 30 * time.Second}
		cfg.AI.Retry.Multiplier = 2.0
		cfg.AI.Retry.Jitter = true
	}

	// Rate limit defaults
	if cfg.AI.RateLimit.RequestsPerMinute == 0 {
		cfg.AI.RateLimit.Enabled = true
		cfg.AI.RateLimit.RequestsPerMinute = 60
		cfg.AI.RateLimit.BurstSize = 10
		cfg.AI.RateLimit.WindowSize = Duration{Duration: 1 * time.Minute}
	}

	// Database defaults
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "sqlite"
	}
	if cfg.Database.DSN == "" {
		cfg.Database.DSN = "file:atest-ext-ai.db?cache=shared&mode=rwc"
	}
	if cfg.Database.DefaultType == "" {
		cfg.Database.DefaultType = "mysql"
	}
	if cfg.Database.MaxConns == 0 {
		cfg.Database.MaxConns = 10
	}
	if cfg.Database.MaxIdle == 0 {
		cfg.Database.MaxIdle = 5
	}
	if cfg.Database.MaxLifetime.Duration == 0 {
		cfg.Database.MaxLifetime = Duration{Duration: 1 * time.Hour}
	}

	// Logging defaults
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "json"
	}
	if cfg.Logging.Output == "" {
		cfg.Logging.Output = "stdout"
	}
	if cfg.Logging.File.Path == "" {
		cfg.Logging.File.Path = "/var/log/atest-ext-ai.log"
	}
	if cfg.Logging.File.MaxSize == "" {
		cfg.Logging.File.MaxSize = "100MB"
	}
	if cfg.Logging.File.MaxBackups == 0 {
		cfg.Logging.File.MaxBackups = 3
	}
	if cfg.Logging.File.MaxAge == 0 {
		cfg.Logging.File.MaxAge = 28
	}
}

// validateConfig validates the configuration
func validateConfig(cfg *Config) error {
	var errors []string

	// Validate server configuration
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		errors = append(errors, fmt.Sprintf("invalid server port: %d (must be 1-65535)", cfg.Server.Port))
	}
	if cfg.Server.Host == "" {
		errors = append(errors, "server host cannot be empty")
	}
	if cfg.Server.SocketPath == "" {
		errors = append(errors, "server socket_path cannot be empty")
	}

	// Validate plugin configuration
	if cfg.Plugin.Name == "" {
		errors = append(errors, "plugin name cannot be empty")
	}
	if cfg.Plugin.Version == "" {
		errors = append(errors, "plugin version cannot be empty")
	}
	validLogLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLogLevels, cfg.Plugin.LogLevel) {
		errors = append(errors, fmt.Sprintf("invalid log level: %s (must be one of: %s)", cfg.Plugin.LogLevel, strings.Join(validLogLevels, ", ")))
	}
	validEnvironments := []string{"development", "staging", "production"}
	if !contains(validEnvironments, cfg.Plugin.Environment) {
		errors = append(errors, fmt.Sprintf("invalid environment: %s (must be one of: %s)", cfg.Plugin.Environment, strings.Join(validEnvironments, ", ")))
	}

	// Validate AI configuration
	if cfg.AI.DefaultService == "" {
		errors = append(errors, "AI default_service cannot be empty")
	}

	if len(cfg.AI.Services) == 0 {
		errors = append(errors, "at least one AI service must be configured")
	}

	// Check if default service exists
	if _, exists := cfg.AI.Services[cfg.AI.DefaultService]; !exists {
		errors = append(errors, fmt.Sprintf("default service '%s' not found in services", cfg.AI.DefaultService))
	}

	// Validate each enabled AI service
	validProviders := []string{"ollama", "openai", "claude", "deepseek", "local", "custom"}
	for name, svc := range cfg.AI.Services {
		if !svc.Enabled {
			continue
		}

		if svc.Provider == "" {
			errors = append(errors, fmt.Sprintf("service '%s': provider cannot be empty", name))
		}
		if !contains(validProviders, svc.Provider) {
			errors = append(errors, fmt.Sprintf("service '%s': invalid provider '%s' (must be one of: %s)", name, svc.Provider, strings.Join(validProviders, ", ")))
		}
		if svc.Model == "" {
			errors = append(errors, fmt.Sprintf("service '%s': model cannot be empty", name))
		}
		if svc.MaxTokens < 1 || svc.MaxTokens > 100000 {
			errors = append(errors, fmt.Sprintf("service '%s': max_tokens %d out of range (1-100000)", name, svc.MaxTokens))
		}

		// Validate provider-specific requirements
		if svc.Provider == "openai" || svc.Provider == "claude" || svc.Provider == "deepseek" {
			if svc.APIKey == "" {
				errors = append(errors, fmt.Sprintf("service '%s': API key required for provider '%s'", name, svc.Provider))
			}
		}
	}

	// Validate retry configuration
	if cfg.AI.Retry.MaxAttempts < 1 || cfg.AI.Retry.MaxAttempts > 10 {
		errors = append(errors, fmt.Sprintf("AI retry max_attempts %d out of range (1-10)", cfg.AI.Retry.MaxAttempts))
	}
	if cfg.AI.Retry.Multiplier < 1 {
		errors = append(errors, fmt.Sprintf("AI retry multiplier %.2f must be >= 1", cfg.AI.Retry.Multiplier))
	}

	// Validate rate limit configuration
	if cfg.AI.RateLimit.Enabled {
		if cfg.AI.RateLimit.RequestsPerMinute < 1 {
			errors = append(errors, fmt.Sprintf("AI rate_limit requests_per_minute %d must be >= 1", cfg.AI.RateLimit.RequestsPerMinute))
		}
		if cfg.AI.RateLimit.BurstSize < 1 {
			errors = append(errors, fmt.Sprintf("AI rate_limit burst_size %d must be >= 1", cfg.AI.RateLimit.BurstSize))
		}
	}

	// Validate database configuration
	if cfg.Database.Enabled {
		validDrivers := []string{"sqlite", "mysql", "postgresql"}
		if !contains(validDrivers, cfg.Database.Driver) {
			errors = append(errors, fmt.Sprintf("invalid database driver: %s (must be one of: %s)", cfg.Database.Driver, strings.Join(validDrivers, ", ")))
		}
		if cfg.Database.DSN == "" {
			errors = append(errors, "database DSN cannot be empty when database is enabled")
		}
	}

	// Validate logging configuration
	validFormats := []string{"json", "text"}
	if !contains(validFormats, cfg.Logging.Format) {
		errors = append(errors, fmt.Sprintf("invalid logging format: %s (must be one of: %s)", cfg.Logging.Format, strings.Join(validFormats, ", ")))
	}
	validOutputs := []string{"stdout", "stderr", "file"}
	if !contains(validOutputs, cfg.Logging.Output) {
		errors = append(errors, fmt.Sprintf("invalid logging output: %s (must be one of: %s)", cfg.Logging.Output, strings.Join(validOutputs, ", ")))
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation errors:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// defaultConfig returns a configuration with all default values
func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			SocketPath:   "/tmp/atest-ext-ai.sock",
			Timeout:      Duration{Duration: 30 * time.Second},
			ReadTimeout:  Duration{Duration: 15 * time.Second},
			WriteTimeout: Duration{Duration: 15 * time.Second},
			MaxConns:     100,
		},
		Plugin: PluginConfig{
			Name:        "atest-ext-ai",
			Version:     "1.0.0",
			Debug:       false,
			LogLevel:    "info",
			Environment: "production",
		},
		AI: AIConfig{
			DefaultService: "ollama",
			Timeout:        Duration{Duration: 60 * time.Second},
			Services: map[string]AIService{
				"ollama": {
					Enabled:   true,
					Provider:  "ollama",
					Endpoint:  "http://localhost:11434",
					Model:     "qwen2.5-coder:latest",
					MaxTokens: 4096,
					Priority:  1,
					Timeout:   Duration{Duration: 60 * time.Second},
				},
			},
			Retry: RetryConfig{
				Enabled:      true,
				MaxAttempts:  3,
				InitialDelay: Duration{Duration: 1 * time.Second},
				MaxDelay:     Duration{Duration: 30 * time.Second},
				Multiplier:   2.0,
				Jitter:       true,
			},
			RateLimit: RateLimitConfig{
				Enabled:           true,
				RequestsPerMinute: 60,
				BurstSize:         10,
				WindowSize:        Duration{Duration: 1 * time.Minute},
			},
		},
		Database: DatabaseConfig{
			Enabled:     false,
			Driver:      "sqlite",
			DSN:         "file:atest-ext-ai.db?cache=shared&mode=rwc",
			DefaultType: "mysql",
			MaxConns:    10,
			MaxIdle:     5,
			MaxLifetime: Duration{Duration: 1 * time.Hour},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
			File: LogFileConfig{
				Path:       "/var/log/atest-ext-ai.log",
				MaxSize:    "100MB",
				MaxBackups: 3,
				MaxAge:     28,
				Compress:   true,
			},
		},
	}
}

// contains checks if a string slice contains a specific string (case-insensitive)
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}
