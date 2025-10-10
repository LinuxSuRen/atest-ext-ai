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
	"strings"

	"gopkg.in/yaml.v2"
)

// LegacyConfig represents the legacy AI plugin configuration for backward compatibility
type LegacyConfig struct {
	AI LegacyAIConfig `yaml:"ai" json:"ai"`
}

// LegacyAIConfig contains legacy AI-specific configuration
type LegacyAIConfig struct {
	Provider            string   `yaml:"provider" json:"provider"` // local, openai, claude
	OllamaEndpoint      string   `yaml:"ollama_endpoint" json:"ollama_endpoint"`
	Model               string   `yaml:"model" json:"model"`
	APIKey              string   `yaml:"api_key" json:"api_key"`
	ConfidenceThreshold float32  `yaml:"confidence_threshold" json:"confidence_threshold"`
	SupportedDatabases  []string `yaml:"supported_databases" json:"supported_databases"`
	EnableSQLExecution  bool     `yaml:"enable_sql_execution" json:"enable_sql_execution"`
}

// LoadConfig loads configuration using simplified loader
func LoadConfig() (*Config, error) {
	// Create a new loader
	loader := NewLoader()

	// Load configuration from default paths
	if err := loader.Load(); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return loader.GetConfig(), nil
}

// LoadLegacyConfig loads configuration using the old format for backward compatibility
func LoadLegacyConfig() (*LegacyConfig, error) {
	config := &LegacyConfig{
		AI: LegacyAIConfig{
			Provider:            getEnvWithDefault("AI_PROVIDER", "local"),
			OllamaEndpoint:      getEnvWithDefault("OLLAMA_ENDPOINT", "http://localhost:11434"),
			Model:               getEnvWithFallback("AI_MODEL"), // Auto-detect from available models if not set
			APIKey:              os.Getenv("AI_API_KEY"),
			ConfidenceThreshold: 0.7,
			SupportedDatabases:  []string{"mysql", "postgresql", "sqlite"},
			EnableSQLExecution:  true,
		},
	}

	// Try to load from YAML file if specified
	if configFile := os.Getenv("AI_CONFIG_FILE"); configFile != "" {
		if err := loadLegacyFromYAML(config, configFile); err != nil {
			return nil, fmt.Errorf("failed to load config from file %s: %w", configFile, err)
		}
	}

	// Load stores.yaml format from environment (main project integration)
	if storeConfig := os.Getenv("STORE_CONFIG"); storeConfig != "" {
		if err := loadLegacyFromStoreConfig(config, storeConfig); err != nil {
			return nil, fmt.Errorf("failed to load store config: %w", err)
		}
	}

	// Validate configuration
	if err := validateLegacyConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// getEnvironment returns the environment setting with production as safe default
func getEnvironment() string {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = os.Getenv("ENVIRONMENT")
	}
	if env == "" {
		return "production"
	}
	return env
}

// ConvertLegacyToNew converts legacy configuration to new format
func ConvertLegacyToNew(legacy *LegacyConfig) *Config {
	if legacy == nil {
		return nil
	}

	// Map legacy provider names to new format
	provider := legacy.AI.Provider
	if provider == "local" {
		provider = "ollama"
	}

	// Create new configuration with converted values
	config := &Config{
		Server: ServerConfig{
			Host:       "0.0.0.0",
			Port:       8080,
			SocketPath: "/tmp/atest-ext-ai.sock",
		},
		Plugin: PluginConfig{
			Name:        "atest-ext-ai",
			Version:     "1.0.0",
			Environment: getEnvironment(),
		},
		AI: AIConfig{
			DefaultService: provider,
			Services: map[string]AIService{
				provider: {
					Enabled:  true,
					Provider: provider,
					Endpoint: legacy.AI.OllamaEndpoint,
					APIKey:   legacy.AI.APIKey,
					Model:    legacy.AI.Model,
					Priority: 1,
				},
			},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}

	return config
}

// ConvertNewToLegacy converts new configuration to legacy format
func ConvertNewToLegacy(config *Config) *LegacyConfig {
	if config == nil {
		return nil
	}

	legacy := &LegacyConfig{
		AI: LegacyAIConfig{
			Provider:            config.AI.DefaultService,
			ConfidenceThreshold: 0.7,
			SupportedDatabases:  []string{"mysql", "postgresql", "sqlite"},
			EnableSQLExecution:  true,
		},
	}

	// Get default service configuration
	if service, exists := config.AI.Services[config.AI.DefaultService]; exists {
		legacy.AI.OllamaEndpoint = service.Endpoint
		legacy.AI.APIKey = service.APIKey
		legacy.AI.Model = service.Model

		// Map provider names back to legacy format
		if service.Provider == "ollama" {
			legacy.AI.Provider = "local"
		}
	}

	return legacy
}

// loadLegacyFromYAML loads legacy configuration from YAML file
func loadLegacyFromYAML(config *LegacyConfig, filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, config)
}

// loadLegacyFromStoreConfig loads legacy configuration from stores.yaml format
func loadLegacyFromStoreConfig(config *LegacyConfig, storeConfigData string) error {
	// Parse stores.yaml format properties
	properties := strings.Split(storeConfigData, ";")
	for _, prop := range properties {
		parts := strings.SplitN(prop, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		switch key {
		case "ai_provider":
			config.AI.Provider = value
		case "ollama_endpoint":
			config.AI.OllamaEndpoint = value
		case "model":
			config.AI.Model = value
		case "api_key":
			config.AI.APIKey = value
		case "confidence_threshold":
			if value != "" {
				config.AI.ConfidenceThreshold = 0.7
			}
		case "enable_sql_execution":
			config.AI.EnableSQLExecution = value == "true"
		case "supported_databases":
			if value != "" {
				config.AI.SupportedDatabases = strings.Split(value, ",")
			}
		}
	}

	return nil
}

// validateLegacyConfig validates legacy configuration
func validateLegacyConfig(config *LegacyConfig) error {
	if config.AI.Provider == "" {
		return fmt.Errorf("ai provider is required")
	}

	switch config.AI.Provider {
	case "local":
		if config.AI.OllamaEndpoint == "" {
			return fmt.Errorf("ollama_endpoint is required for local provider")
		}
		// Model is optional for local provider - will be auto-detected from available models
	case "openai", "claude":
		if config.AI.APIKey == "" {
			return fmt.Errorf("api_key is required for %s provider", config.AI.Provider)
		}
		if config.AI.Model == "" {
			return fmt.Errorf("model is required for %s provider", config.AI.Provider)
		}
	default:
		return fmt.Errorf("unsupported provider: %s", config.AI.Provider)
	}

	return nil
}

// getEnvWithDefault returns environment variable value or default
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvWithFallback returns environment variable value
func getEnvWithFallback(key string) string {
	return os.Getenv(key)
}
