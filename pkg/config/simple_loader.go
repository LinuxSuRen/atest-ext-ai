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

// Package config provides simplified configuration loading using YAML and environment variables.
// This replaces the previous Viper-based configuration system with a lightweight, direct approach.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/constants"
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
	var attemptedPaths []string

	for _, path := range searchPaths {
		attemptedPaths = append(attemptedPaths, path)
		cfg, err := loadYAMLFile(path)
		if err == nil {
			fmt.Fprintf(os.Stderr, "Configuration loaded from: %s\n", path)
			return cfg, nil
		}
		lastErr = err
	}

	// Log all attempted paths for troubleshooting
	fmt.Fprintf(os.Stderr, "Warning: No configuration file found. Attempted paths:\n")
	for i, path := range attemptedPaths {
		fmt.Fprintf(os.Stderr, "  %d. %s\n", i+1, path)
	}
	fmt.Fprintf(os.Stderr, "Using default configuration. Last error: %v\n", lastErr)
	fmt.Fprintf(os.Stderr, "To customize: Create config.yaml in current directory or ~/.config/atest/\n")

	return nil, fmt.Errorf("no config file found (tried %d paths): %w", len(attemptedPaths), lastErr)
}

// loadYAMLFile loads configuration from a YAML file
func loadYAMLFile(path string) (*Config, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- configuration paths are restricted to trusted locations
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &cfg, nil
}

// applyEnvOverrides applies environment variable overrides to the configuration.
// GUI-driven configuration is the primary workflow; environment overrides remain
// for legacy automation scenarios and may be removed in future versions.
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
	if listenAddr := os.Getenv("ATEST_EXT_AI_SERVER_LISTEN_ADDR"); listenAddr != "" {
		cfg.Server.ListenAddress = listenAddr
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
		cfg.Server.Host = constants.DefaultServerHost
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = constants.DefaultServerPort
	}
	if runtime.GOOS == "windows" {
		if cfg.Server.ListenAddress == "" {
			cfg.Server.ListenAddress = constants.DefaultWindowsListenAddress
		}
	} else {
		if cfg.Server.SocketPath == "" {
			cfg.Server.SocketPath = constants.DefaultUnixSocketPath
		}
	}
	if cfg.Server.Timeout.Duration == 0 {
		cfg.Server.Timeout = Duration{Duration: constants.Timeouts.Server}
	}
	if cfg.Server.ReadTimeout.Duration == 0 {
		cfg.Server.ReadTimeout = Duration{Duration: constants.Timeouts.Read}
	}
	if cfg.Server.WriteTimeout.Duration == 0 {
		cfg.Server.WriteTimeout = Duration{Duration: constants.Timeouts.Write}
	}
	if cfg.Server.MaxConns == 0 {
		cfg.Server.MaxConns = constants.ServerDefaults.MaxConnections
	}

	// Plugin defaults
	if cfg.Plugin.Name == "" {
		cfg.Plugin.Name = constants.DefaultPluginName
	}
	if cfg.Plugin.Version == "" {
		cfg.Plugin.Version = constants.DefaultPluginVersion
	}
	if cfg.Plugin.LogLevel == "" {
		cfg.Plugin.LogLevel = constants.DefaultPluginLogLevel
	}
	if cfg.Plugin.Environment == "" {
		cfg.Plugin.Environment = constants.DefaultPluginEnvironment
	}

	// AI defaults
	if cfg.AI.DefaultService == "" {
		cfg.AI.DefaultService = constants.DefaultAIService
	}
	if cfg.AI.Timeout.Duration == 0 {
		cfg.AI.Timeout = Duration{Duration: constants.Timeouts.AI}
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
			Endpoint:  constants.DefaultOllamaEndpoint,
			Model:     constants.DefaultOllamaModel,
			MaxTokens: constants.DefaultOllamaMaxTokens,
			Priority:  constants.DefaultOllamaPriority,
			Timeout:   Duration{Duration: constants.Timeouts.Ollama},
		}
	} else {
		// Fill in missing fields for existing Ollama service
		svc := cfg.AI.Services["ollama"]
		if svc.Endpoint == "" {
			svc.Endpoint = constants.DefaultOllamaEndpoint
		}
		if svc.Model == "" {
			svc.Model = constants.DefaultOllamaModel
		}
		if svc.MaxTokens == 0 {
			svc.MaxTokens = constants.DefaultOllamaMaxTokens
		}
		if svc.Timeout.Duration == 0 {
			svc.Timeout = Duration{Duration: constants.Timeouts.Ollama}
		}
		if svc.Priority == 0 {
			svc.Priority = constants.DefaultOllamaPriority
		}
		cfg.AI.Services["ollama"] = svc
	}

	// Retry defaults
	if cfg.AI.Retry.MaxAttempts == 0 {
		cfg.AI.Retry.Enabled = constants.Retry.Enabled
		cfg.AI.Retry.MaxAttempts = constants.Retry.MaxAttempts
		cfg.AI.Retry.InitialDelay = Duration{Duration: constants.Retry.InitialDelay}
		cfg.AI.Retry.MaxDelay = Duration{Duration: constants.Retry.MaxDelay}
		cfg.AI.Retry.Multiplier = constants.Retry.Multiplier
		cfg.AI.Retry.Jitter = constants.Retry.Jitter
	}

	// Rate limit defaults
	if cfg.AI.RateLimit.RequestsPerMinute == 0 {
		cfg.AI.RateLimit.Enabled = constants.RateLimit.Enabled
		cfg.AI.RateLimit.RequestsPerMinute = constants.RateLimit.RequestsPerMinute
		cfg.AI.RateLimit.BurstSize = constants.RateLimit.BurstSize
		cfg.AI.RateLimit.WindowSize = Duration{Duration: constants.RateLimit.WindowSize}
	}

	// Database defaults
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = constants.DefaultDatabaseDriver
	}
	if cfg.Database.DSN == "" {
		cfg.Database.DSN = constants.DefaultDatabaseDSN
	}
	if cfg.Database.DefaultType == "" {
		cfg.Database.DefaultType = constants.DefaultDatabaseType
	}
	if cfg.Database.MaxConns == 0 {
		cfg.Database.MaxConns = constants.DatabasePool.MaxConns
	}
	if cfg.Database.MaxIdle == 0 {
		cfg.Database.MaxIdle = constants.DatabasePool.MaxIdle
	}
	if cfg.Database.MaxLifetime.Duration == 0 {
		cfg.Database.MaxLifetime = Duration{Duration: constants.DatabasePool.MaxLifetime}
	}

	// Logging defaults
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = constants.DefaultLoggingLevel
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = constants.DefaultLoggingFormat
	}
	if cfg.Logging.Output == "" {
		cfg.Logging.Output = constants.DefaultLoggingOutput
	}
	if cfg.Logging.File.Path == "" {
		cfg.Logging.File.Path = constants.LogFile.Path
	}
	if cfg.Logging.File.MaxSize == "" {
		cfg.Logging.File.MaxSize = constants.LogFile.MaxSize
	}
	if cfg.Logging.File.MaxBackups == 0 {
		cfg.Logging.File.MaxBackups = constants.LogFile.MaxBackups
	}
	if cfg.Logging.File.MaxAge == 0 {
		cfg.Logging.File.MaxAge = constants.LogFile.MaxAge
	}
}

// validateConfig validates the configuration with relaxed rules
// Only critical configuration errors cause failure - the plugin can start with minimal config
func validateConfig(cfg *Config) error {
	result := cfg.Validate()

	for _, warning := range result.Warnings {
		fmt.Fprintf(os.Stderr, "Configuration warning: %s\n", warning.Error())
	}

	if err := result.Error(); err != nil {
		return err
	}

	return nil
}

// defaultConfig returns a configuration with all default values
func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:          constants.DefaultServerHost,
			Port:          constants.DefaultServerPort,
			SocketPath:    constants.DefaultUnixSocketPath,
			ListenAddress: constants.DefaultWindowsListenAddress,
			Timeout:       Duration{Duration: constants.Timeouts.Server},
			ReadTimeout:   Duration{Duration: constants.Timeouts.Read},
			WriteTimeout:  Duration{Duration: constants.Timeouts.Write},
			MaxConns:      constants.ServerDefaults.MaxConnections,
		},
		Plugin: PluginConfig{
			Name:        constants.DefaultPluginName,
			Version:     constants.DefaultPluginVersion,
			Debug:       false,
			LogLevel:    constants.DefaultPluginLogLevel,
			Environment: constants.DefaultPluginEnvironment,
		},
		AI: AIConfig{
			DefaultService: constants.DefaultAIService,
			Timeout:        Duration{Duration: constants.Timeouts.AI},
			Services: map[string]AIService{
				"ollama": {
					Enabled:   true,
					Provider:  "ollama",
					Endpoint:  constants.DefaultOllamaEndpoint,
					Model:     constants.DefaultOllamaModel,
					MaxTokens: constants.DefaultOllamaMaxTokens,
					Priority:  constants.DefaultOllamaPriority,
					Timeout:   Duration{Duration: constants.Timeouts.Ollama},
				},
			},
			Retry: RetryConfig{
				Enabled:      constants.Retry.Enabled,
				MaxAttempts:  constants.Retry.MaxAttempts,
				InitialDelay: Duration{Duration: constants.Retry.InitialDelay},
				MaxDelay:     Duration{Duration: constants.Retry.MaxDelay},
				Multiplier:   constants.Retry.Multiplier,
				Jitter:       constants.Retry.Jitter,
			},
			RateLimit: RateLimitConfig{
				Enabled:           constants.RateLimit.Enabled,
				RequestsPerMinute: constants.RateLimit.RequestsPerMinute,
				BurstSize:         constants.RateLimit.BurstSize,
				WindowSize:        Duration{Duration: constants.RateLimit.WindowSize},
			},
		},
		Database: DatabaseConfig{
			Enabled:     false,
			Driver:      constants.DefaultDatabaseDriver,
			DSN:         constants.DefaultDatabaseDSN,
			DefaultType: constants.DefaultDatabaseType,
			MaxConns:    constants.DatabasePool.MaxConns,
			MaxIdle:     constants.DatabasePool.MaxIdle,
			MaxLifetime: Duration{Duration: constants.DatabasePool.MaxLifetime},
		},
		Logging: LoggingConfig{
			Level:  constants.DefaultLoggingLevel,
			Format: constants.DefaultLoggingFormat,
			Output: constants.DefaultLoggingOutput,
			File: LogFileConfig{
				Path:       constants.LogFile.Path,
				MaxSize:    constants.LogFile.MaxSize,
				MaxBackups: constants.LogFile.MaxBackups,
				MaxAge:     constants.LogFile.MaxAge,
				Compress:   constants.LogFile.Compress,
			},
		},
	}
}
