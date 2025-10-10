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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

// Loader handles loading configuration from various sources and formats
type Loader struct {
	viper  *viper.Viper
	config *Config
}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	v := viper.New()

	// Set default values
	setDefaults(v)

	return &Loader{
		viper:  v,
		config: &Config{},
	}
}

// Load loads configuration from the specified paths with auto-format detection
func (l *Loader) Load(paths ...string) error {
	if len(paths) == 0 {
		paths = []string{"./config", "./", "/etc/atest-ext-ai"}
	}

	// Setup viper configuration
	l.viper.SetConfigName("config")
	l.viper.SetConfigType("yaml") // Default type, will be auto-detected

	// Add search paths
	for _, path := range paths {
		l.viper.AddConfigPath(path)
	}

	// Enable environment variable support
	l.setupEnvironmentVariables()

	// Try to read configuration file
	if err := l.viper.ReadInConfig(); err != nil {
		// If no config file found, that's ok - we'll use defaults and env vars
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Load additional configuration files if they exist
	for _, path := range paths {
		if err := l.loadFromDirectory(path); err != nil {
			return fmt.Errorf("error loading configs from %s: %w", path, err)
		}
	}

	// Unmarshal into our config struct with custom decoder
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			stringToDurationHookFunc(),
			mapstructure.StringToTimeDurationHookFunc(),
		),
		Result:           l.config,
		WeaklyTypedInput: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	}

	if err := decoder.Decode(l.viper.AllSettings()); err != nil {
		return fmt.Errorf("unable to decode config into struct: %w", err)
	}

	// Check for deprecated fields and display warnings
	for serviceName, service := range l.config.AI.Services {
		warnings := service.ValidateAndWarnDeprecated()
		for _, warning := range warnings {
			fmt.Printf("⚠️  [WARNING] Service '%s': %s\n", serviceName, warning)
		}
	}

	return nil
}

// LoadFromFile loads configuration from a specific file
func (l *Loader) LoadFromFile(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("config file does not exist: %s", filePath)
	}

	// Detect format based on file extension
	format := l.detectFormat(filePath)
	if format == "" {
		return fmt.Errorf("unsupported file format: %s", filePath)
	}

	// Read file content
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading config file %s: %w", filePath, err)
	}

	// Parse based on format
	if err := l.parseContent(data, format); err != nil {
		return LoadError{
			Path:   filePath,
			Format: format,
			Err:    err,
		}
	}

	return nil
}

// LoadFromBytes loads configuration from byte data with specified format
func (l *Loader) LoadFromBytes(data []byte, format string) error {
	if len(data) == 0 {
		return fmt.Errorf("data cannot be empty")
	}

	format = strings.ToLower(format)
	if !l.isSupportedFormat(format) {
		return fmt.Errorf("unsupported format: %s", format)
	}

	return l.parseContent(data, format)
}

// GetConfig returns the loaded configuration
func (l *Loader) GetConfig() *Config {
	return l.config
}

// GetViper returns the underlying viper instance
func (l *Loader) GetViper() *viper.Viper {
	return l.viper
}

// Merge merges another configuration into the current one
func (l *Loader) Merge(other *Config) error {
	if other == nil {
		return fmt.Errorf("cannot merge nil config")
	}

	// Directly merge Config structs using reflection
	mergedConfig := l.mergeConfigs(l.config, other)

	// Update the loader's config
	l.config = mergedConfig

	// Convert merged config back to a map and update viper
	configMap := make(map[string]interface{})
	configJSON, err := json.Marshal(mergedConfig)
	if err != nil {
		return fmt.Errorf("error marshaling merged config: %w", err)
	}

	if err := json.Unmarshal(configJSON, &configMap); err != nil {
		return fmt.Errorf("error unmarshaling merged config: %w", err)
	}

	// Create new viper instance and set the merged values
	newViper := viper.New()
	setDefaults(newViper) // Restore defaults
	for key, value := range configMap {
		newViper.Set(key, value)
	}

	l.viper = newViper

	return nil
}

// Export exports configuration in the specified format
func (l *Loader) Export(format string) ([]byte, error) {
	format = strings.ToLower(format)

	switch format {
	case "yaml", "yml":
		return yaml.Marshal(l.config)
	case "json":
		return json.MarshalIndent(l.config, "", "  ")
	case "toml":
		return toml.Marshal(l.config)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// setupEnvironmentVariables configures environment variable handling
func (l *Loader) setupEnvironmentVariables() {
	// Set environment variable prefix
	l.viper.SetEnvPrefix("ATEST_EXT_AI")

	// Enable automatic environment variable binding
	l.viper.AutomaticEnv()

	// Replace dots and hyphens with underscores for environment variables
	l.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	// Bind specific environment variables with consistent ATEST_EXT_AI_ prefix
	envBindings := map[string]string{
		"server.host":                 "SERVER_HOST",
		"server.port":                 "SERVER_PORT",
		"server.socket_path":          "SERVER_SOCKET_PATH",
		"plugin.debug":                "DEBUG",
		"plugin.log_level":            "LOG_LEVEL",
		"plugin.environment":          "ENVIRONMENT",
		"ai.default_service":          "AI_PROVIDER",
		"ai.timeout":                  "AI_TIMEOUT",
		"ai.services.ollama.endpoint": "OLLAMA_ENDPOINT",
		"ai.services.ollama.model":    "AI_MODEL",
		"ai.services.openai.api_key":  "OPENAI_API_KEY",
		"ai.services.openai.model":    "OPENAI_MODEL",
		"ai.services.claude.api_key":  "CLAUDE_API_KEY",
		"ai.services.claude.model":    "CLAUDE_MODEL",
		"database.enabled":            "DATABASE_ENABLED",
		"database.driver":             "DATABASE_DRIVER",
		"database.dsn":                "DATABASE_DSN",
		"logging.format":              "LOG_FORMAT",
		"logging.output":              "LOG_OUTPUT",
	}

	for viperKey, envKey := range envBindings {
		if err := l.viper.BindEnv(viperKey, "ATEST_EXT_AI_"+envKey); err != nil {
			// Log but don't fail - environment binding is optional
			continue
		}
	}
}

// stringToDurationHookFunc creates a decode hook for converting strings to Duration
func stringToDurationHookFunc() mapstructure.DecodeHookFunc {
	return mapstructure.DecodeHookFuncType(func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(Duration{}) {
			return data, nil
		}

		// Parse string to Duration
		s := data.(string)
		duration, err := time.ParseDuration(s)
		if err != nil {
			return nil, fmt.Errorf("invalid duration format: %s", s)
		}

		return Duration{Duration: duration}, nil
	})
}

// loadFromDirectory loads all supported config files from a directory
func (l *Loader) loadFromDirectory(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return nil // Directory doesn't exist, skip
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("error reading directory %s: %w", dirPath, err)
	}

	supportedExtensions := []string{".yaml", ".yml", ".json", ".toml"}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := filepath.Ext(name)

		// Skip if not a supported config file
		if !contains(supportedExtensions, ext) {
			continue
		}

		// Skip if it's the main config file (already loaded)
		if strings.HasPrefix(name, "config.") {
			continue
		}

		filePath := filepath.Join(dirPath, name)
		if err := l.LoadFromFile(filePath); err != nil {
			// Log error but don't fail completely
			fmt.Printf("Warning: failed to load config file %s: %v\n", filePath, err)
		}
	}

	return nil
}

// parseContent parses configuration data based on format
func (l *Loader) parseContent(data []byte, format string) error {
	var configMap map[string]interface{}

	switch strings.ToLower(format) {
	case "yaml", "yml":
		if err := yaml.Unmarshal(data, &configMap); err != nil {
			return fmt.Errorf("error parsing YAML: %w", err)
		}
	case "json":
		if err := json.Unmarshal(data, &configMap); err != nil {
			return fmt.Errorf("error parsing JSON: %w", err)
		}
	case "toml":
		if err := toml.Unmarshal(data, &configMap); err != nil {
			return fmt.Errorf("error parsing TOML: %w", err)
		}
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	// Convert map to Config struct using mapstructure with custom hooks
	var config Config
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			stringToDurationHookFunc(),
			mapstructure.StringToTimeDurationHookFunc(),
		),
		Result:           &config,
		WeaklyTypedInput: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	}

	if err := decoder.Decode(configMap); err != nil {
		return fmt.Errorf("error decoding config map: %w", err)
	}

	// Update the loader's config directly for file parsing (not merging)
	l.config = &config

	// Update viper with the new config values
	viperConfigMap := make(map[string]interface{})
	configJSON, err := json.Marshal(&config)
	if err != nil {
		return fmt.Errorf("error marshaling config: %w", err)
	}

	if err := json.Unmarshal(configJSON, &viperConfigMap); err != nil {
		return fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Clear viper and reset with new values plus defaults
	l.viper = viper.New()
	setDefaults(l.viper) // Set defaults first
	for key, value := range viperConfigMap {
		l.viper.Set(key, value) // Override with file values
	}

	return nil
}

// detectFormat detects configuration format based on file extension
func (l *Loader) detectFormat(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".toml":
		return "toml"
	default:
		return ""
	}
}

// isSupportedFormat checks if the format is supported
func (l *Loader) isSupportedFormat(format string) bool {
	supportedFormats := []string{"yaml", "yml", "json", "toml"}
	return contains(supportedFormats, strings.ToLower(format))
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.timeout", "30s")
	v.SetDefault("server.max_connections", 100)
	v.SetDefault("server.socket_path", "/tmp/atest-ext-ai.sock")
	v.SetDefault("server.read_timeout", "15s")
	v.SetDefault("server.write_timeout", "15s")

	// Plugin defaults
	v.SetDefault("plugin.name", "atest-ext-ai")
	v.SetDefault("plugin.version", "1.0.0")
	v.SetDefault("plugin.debug", false)
	v.SetDefault("plugin.log_level", "info")
	v.SetDefault("plugin.environment", "production")

	// AI defaults
	v.SetDefault("ai.default_service", "ollama")
	v.SetDefault("ai.timeout", "60s")
	v.SetDefault("ai.fallback_order", []string{"ollama"})

	// AI Rate Limiting defaults
	v.SetDefault("ai.rate_limit.enabled", true)
	v.SetDefault("ai.rate_limit.requests_per_minute", 60)
	v.SetDefault("ai.rate_limit.burst_size", 10)
	v.SetDefault("ai.rate_limit.window_size", "1m")

	// AI Circuit Breaker defaults
	v.SetDefault("ai.circuit_breaker.enabled", true)
	v.SetDefault("ai.circuit_breaker.failure_threshold", 5)
	v.SetDefault("ai.circuit_breaker.success_threshold", 3)
	v.SetDefault("ai.circuit_breaker.timeout", "30s")
	v.SetDefault("ai.circuit_breaker.reset_timeout", "60s")

	// AI Retry defaults
	v.SetDefault("ai.retry.enabled", true)
	v.SetDefault("ai.retry.max_attempts", 3)
	v.SetDefault("ai.retry.initial_delay", "1s")
	v.SetDefault("ai.retry.max_delay", "30s")
	v.SetDefault("ai.retry.multiplier", 2.0)
	v.SetDefault("ai.retry.jitter", true)

	// AI Cache defaults
	v.SetDefault("ai.cache.enabled", true)
	v.SetDefault("ai.cache.ttl", "1h")
	v.SetDefault("ai.cache.max_size", 1000)
	v.SetDefault("ai.cache.provider", "memory")

	// AI Service defaults - Ollama
	v.SetDefault("ai.services.ollama.enabled", true)
	v.SetDefault("ai.services.ollama.provider", "ollama")
	// ai.services.ollama.endpoint must be set via OLLAMA_ENDPOINT environment variable
	// AI_MODEL will be auto-detected from available models at runtime
	v.SetDefault("ai.services.ollama.max_tokens", 4096)
	v.SetDefault("ai.services.ollama.priority", 1)
	v.SetDefault("ai.services.ollama.timeout", "60s")

	// Don't set defaults for disabled services - let validation only check enabled ones

	// Database defaults
	v.SetDefault("database.enabled", false)
	v.SetDefault("database.driver", "sqlite")
	v.SetDefault("database.dsn", "file:atest-ext-ai.db?cache=shared&mode=rwc")
	v.SetDefault("database.default_type", "mysql")
	v.SetDefault("database.max_connections", 10)
	v.SetDefault("database.max_idle", 5)
	v.SetDefault("database.max_lifetime", "1h")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.output", "stdout")
	v.SetDefault("logging.file.path", "/var/log/atest-ext-ai.log")
	v.SetDefault("logging.file.max_size", "100MB")
	v.SetDefault("logging.file.max_backups", 3)
	v.SetDefault("logging.file.max_age", 28)
	v.SetDefault("logging.file.compress", true)
	v.SetDefault("logging.rotation.enabled", true)
	v.SetDefault("logging.rotation.size", "100MB")
	v.SetDefault("logging.rotation.count", 5)
	v.SetDefault("logging.rotation.age", "30d")
	v.SetDefault("logging.rotation.compress", true)
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// mergeConfigs merges two Config structs, preserving non-zero values from dst and overriding with non-zero values from src
func (l *Loader) mergeConfigs(dst, src *Config) *Config {
	if dst == nil && src == nil {
		return &Config{}
	}
	if dst == nil {
		return src
	}
	if src == nil {
		return dst
	}

	// Create a copy of dst to avoid modifying the original
	result := *dst

	// Merge Server config
	if src.Server.Host != "" {
		result.Server.Host = src.Server.Host
	}
	if src.Server.Port != 0 {
		result.Server.Port = src.Server.Port
	}
	if src.Server.Timeout.Duration != 0 {
		result.Server.Timeout = src.Server.Timeout
	}
	if src.Server.MaxConns != 0 {
		result.Server.MaxConns = src.Server.MaxConns
	}
	if src.Server.SocketPath != "" {
		result.Server.SocketPath = src.Server.SocketPath
	}
	if src.Server.ReadTimeout.Duration != 0 {
		result.Server.ReadTimeout = src.Server.ReadTimeout
	}
	if src.Server.WriteTimeout.Duration != 0 {
		result.Server.WriteTimeout = src.Server.WriteTimeout
	}

	// Merge Plugin config
	if src.Plugin.Name != "" {
		result.Plugin.Name = src.Plugin.Name
	}
	if src.Plugin.Version != "" {
		result.Plugin.Version = src.Plugin.Version
	}
	if src.Plugin.LogLevel != "" {
		result.Plugin.LogLevel = src.Plugin.LogLevel
	}
	if src.Plugin.Environment != "" {
		result.Plugin.Environment = src.Plugin.Environment
	}
	// Debug is a bool, so we need special handling
	result.Plugin.Debug = src.Plugin.Debug || dst.Plugin.Debug

	// Merge AI config
	if src.AI.DefaultService != "" {
		result.AI.DefaultService = src.AI.DefaultService
	}
	if src.AI.Timeout.Duration != 0 {
		result.AI.Timeout = src.AI.Timeout
	}
	if len(src.AI.Fallback) > 0 {
		result.AI.Fallback = src.AI.Fallback
	}

	// Merge AI services
	if result.AI.Services == nil {
		result.AI.Services = make(map[string]AIService)
	}
	for name, service := range src.AI.Services {
		result.AI.Services[name] = service
	}

	// Merge other AI configs (rate limit, circuit breaker, etc.) with non-zero checks
	if src.AI.RateLimit.RequestsPerMinute != 0 {
		result.AI.RateLimit.RequestsPerMinute = src.AI.RateLimit.RequestsPerMinute
	}
	if src.AI.RateLimit.BurstSize != 0 {
		result.AI.RateLimit.BurstSize = src.AI.RateLimit.BurstSize
	}
	if src.AI.RateLimit.WindowSize.Duration != 0 {
		result.AI.RateLimit.WindowSize = src.AI.RateLimit.WindowSize
	}
	result.AI.RateLimit.Enabled = src.AI.RateLimit.Enabled || dst.AI.RateLimit.Enabled

	// Merge other configs as needed - Database, Logging, etc.
	// For now, just copy them over if they have any meaningful values
	if src.Database.Driver != "" {
		result.Database = src.Database
	}
	if src.Logging.Level != "" {
		result.Logging = src.Logging
	}

	return &result
}
