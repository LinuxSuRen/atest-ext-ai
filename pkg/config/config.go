/*
Copyright 2023-2025 API Testing Authors.

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

// Config represents the AI plugin configuration
type Config struct {
	AI AIConfig `yaml:"ai" json:"ai"`
}

// AIConfig contains AI-specific configuration
type AIConfig struct {
	Provider           string            `yaml:"provider" json:"provider"`                       // local, openai, claude
	OllamaEndpoint     string            `yaml:"ollama_endpoint" json:"ollama_endpoint"`         
	Model              string            `yaml:"model" json:"model"`                             
	APIKey             string            `yaml:"api_key" json:"api_key"`                         
	ConfidenceThreshold float32           `yaml:"confidence_threshold" json:"confidence_threshold"`
	SupportedDatabases []string          `yaml:"supported_databases" json:"supported_databases"`
	EnableSQLExecution bool              `yaml:"enable_sql_execution" json:"enable_sql_execution"`
	Metadata           map[string]string `yaml:"metadata" json:"metadata"`
}

// LoadConfig loads configuration from environment variables and stores.yaml format
func LoadConfig() (*Config, error) {
	config := &Config{
		AI: AIConfig{
			Provider:            getEnvWithDefault("AI_PROVIDER", "local"),
			OllamaEndpoint:      getEnvWithDefault("OLLAMA_ENDPOINT", "http://localhost:11434"),
			Model:               getEnvWithDefault("AI_MODEL", "codellama"),
			APIKey:              os.Getenv("AI_API_KEY"),
			ConfidenceThreshold: 0.7,
			SupportedDatabases:  []string{"mysql", "postgresql", "sqlite"},
			EnableSQLExecution:  true,
			Metadata:            make(map[string]string),
		},
	}

	// Try to load from YAML file if specified
	if configFile := os.Getenv("AI_CONFIG_FILE"); configFile != "" {
		if err := loadFromYAML(config, configFile); err != nil {
			return nil, fmt.Errorf("failed to load config from file %s: %w", configFile, err)
		}
	}

	// Load stores.yaml format from environment (main project integration)
	if storeConfig := os.Getenv("STORE_CONFIG"); storeConfig != "" {
		if err := loadFromStoreConfig(config, storeConfig); err != nil {
			return nil, fmt.Errorf("failed to load store config: %w", err)
		}
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// loadFromYAML loads configuration from YAML file
func loadFromYAML(config *Config, filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, config)
}

// loadFromStoreConfig loads configuration from stores.yaml format (main project integration)
func loadFromStoreConfig(config *Config, storeConfigData string) error {
	// Parse stores.yaml format properties
	// Example: "ai_provider=local;ollama_endpoint=http://localhost:11434;model=codellama"
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
			// Parse float32 from string
			if value != "" {
				config.AI.ConfidenceThreshold = 0.7 // Default, could parse from string
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

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	if config.AI.Provider == "" {
		return fmt.Errorf("ai provider is required")
	}

	// Validate provider-specific requirements
	switch config.AI.Provider {
	case "local":
		if config.AI.OllamaEndpoint == "" {
			return fmt.Errorf("ollama_endpoint is required for local provider")
		}
		if config.AI.Model == "" {
			return fmt.Errorf("model is required for local provider")
		}
	case "openai", "claude":
		if config.AI.APIKey == "" {
			return fmt.Errorf("api_key is required for %s provider", config.AI.Provider)
		}
	default:
		return fmt.Errorf("unsupported AI provider: %s", config.AI.Provider)
	}

	if len(config.AI.SupportedDatabases) == 0 {
		return fmt.Errorf("at least one supported database is required")
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