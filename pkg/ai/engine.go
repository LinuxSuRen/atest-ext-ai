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

package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/config"
	"github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
	"github.com/linuxsuren/atest-ext-ai/pkg/logging"
)

// shouldIncludeDebugInfo checks if debug information should be included in responses
func shouldIncludeDebugInfo() bool {
	return os.Getenv("APP_ENV") == "development" || os.Getenv("LOG_LEVEL") == "debug"
}

// addDebugInfo conditionally adds debug information based on environment
func addDebugInfo(existing []string, info string) []string {
	if shouldIncludeDebugInfo() {
		return append(existing, info)
	}
	return existing
}

// IsProviderNotSupported checks if an error is due to an unsupported provider
func IsProviderNotSupported(err error) bool {
	if err == nil {
		return false
	}
	// Check if the error is from the client factory
	return errors.Is(err, ErrProviderNotSupported)
}

// Engine defines the interface for AI SQL generation
type Engine interface {
	GenerateSQL(ctx context.Context, req *GenerateSQLRequest) (*GenerateSQLResponse, error)
	GetCapabilities() *SQLCapabilities
	IsHealthy() bool
	Close()
}

// GenerateSQLRequest represents an AI SQL generation request
type GenerateSQLRequest struct {
	NaturalLanguage string            `json:"natural_language"`
	DatabaseType    string            `json:"database_type"`
	Context         map[string]string `json:"context,omitempty"`
}

// GenerateSQLResponse represents an AI SQL generation response
type GenerateSQLResponse struct {
	SQL             string        `json:"sql"`
	Explanation     string        `json:"explanation"`
	ConfidenceScore float32       `json:"confidence_score"`
	ProcessingTime  time.Duration `json:"processing_time"`
	RequestID       string        `json:"request_id"`
	ModelUsed       string        `json:"model_used"`
	DebugInfo       []string      `json:"debug_info,omitempty"`
}

// SQLCapabilities represents AI engine capabilities for SQL generation
type SQLCapabilities struct {
	SupportedDatabases []string     `json:"supported_databases"`
	Features           []SQLFeature `json:"features"`
}

// SQLFeature represents a specific AI SQL feature
type SQLFeature struct {
	Name        string            `json:"name"`
	Enabled     bool              `json:"enabled"`
	Description string            `json:"description"`
	Parameters  map[string]string `json:"parameters,omitempty"`
}

// aiEngine is the AI engine implementation using AI clients
type aiEngine struct {
	config    config.AIConfig
	generator *SQLGenerator
	aiClient  interfaces.AIClient
	client    *Client
}

// NewEngine creates a new AI engine based on configuration
func NewEngine(cfg config.AIConfig) (Engine, error) {
	// Try to create a full AI client first
	client, err := NewClient(cfg)
	if err != nil {
		// Check if this is an unsupported provider error
		if IsProviderNotSupported(err) {
			logging.Logger.Error("Provider not supported - please use one of: openai, local, deepseek, custom", "error", err, "provider", cfg.DefaultService)
			return nil, fmt.Errorf("unsupported AI provider '%s': %w. Supported providers: openai, local (ollama), deepseek, custom", cfg.DefaultService, err)
		}
		// For other errors, also fail instead of silent fallback
		logging.Logger.Error("Failed to create AI client", "error", err, "provider", cfg.DefaultService)
		return nil, fmt.Errorf("failed to create AI client for provider '%s': %w", cfg.DefaultService, err)
	}

	// Get the AI client from the Client
	aiClient := client.GetPrimaryClient()
	if aiClient == nil {
		logging.Logger.Error("No primary AI client available - check your configuration", "provider", cfg.DefaultService)
		return nil, fmt.Errorf("no primary AI client available for provider '%s' - please check your configuration", cfg.DefaultService)
	}

	// Create SQL generator with AI client
	generator, err := NewSQLGenerator(aiClient, cfg)
	if err != nil {
		logging.Logger.Error("Failed to create SQL generator", "error", err, "provider", cfg.DefaultService)
		return nil, fmt.Errorf("failed to create SQL generator for provider '%s': %w", cfg.DefaultService, err)
	}

	logging.Logger.Info("AI engine created successfully", "provider", cfg.DefaultService)
	return &aiEngine{
		config:    cfg,
		generator: generator,
		aiClient:  aiClient,
		client:    client,
	}, nil
}

// NewOllamaEngine creates an Ollama-based AI engine
func NewOllamaEngine(cfg config.AIConfig) (Engine, error) {
	return NewEngine(cfg)
}

// NewOpenAIEngine creates an OpenAI-based AI engine
func NewOpenAIEngine(cfg config.AIConfig) (Engine, error) {
	return NewEngine(cfg)
}

// NewClaudeEngine creates a Claude-based AI engine
func NewClaudeEngine(cfg config.AIConfig) (Engine, error) {
	return NewEngine(cfg)
}


// GenerateSQL implements Engine.GenerateSQL with full AI integration
func (e *aiEngine) GenerateSQL(ctx context.Context, req *GenerateSQLRequest) (*GenerateSQLResponse, error) {
	if e.generator == nil {
		return nil, fmt.Errorf("SQL generator not initialized")
	}

	// Get default max tokens from configuration
	defaultMaxTokens := 2000 // fallback if config not available
	if service, ok := e.config.Services[e.config.DefaultService]; ok && service.MaxTokens > 0 {
		defaultMaxTokens = service.MaxTokens
	}

	// Convert request to generator options
	options := &GenerateOptions{
		DatabaseType:       req.DatabaseType,
		ValidateSQL:        true,
		OptimizeQuery:      false,
		IncludeExplanation: true,
		SafetyMode:         true,
		MaxTokens:          defaultMaxTokens,
	}

	// Add context if provided and extract preferred_model and runtime config
	var runtimeConfig map[string]interface{}
	if len(req.Context) > 0 {
		options.Context = make([]string, 0, len(req.Context))
		for key, value := range req.Context {
			if key == "preferred_model" {
				// Set the preferred model directly in options
				options.Model = value
				logging.Logger.Debug("AI engine: setting model from context", "model", value)
			} else if key == "config" {
				// Parse runtime configuration for API keys etc.
				if err := json.Unmarshal([]byte(value), &runtimeConfig); err != nil {
					logging.Logger.Warn("Failed to parse runtime config", "error", err)
				} else {
					logging.Logger.Debug("AI engine: parsed runtime config",
						"provider", runtimeConfig["provider"],
						"has_api_key", runtimeConfig["api_key"] != nil)
					// Extract configuration for dynamic client creation
					if provider, ok := runtimeConfig["provider"].(string); ok {
						// Map "local" to "ollama" for consistency
						if provider == "local" {
							provider = "ollama"
						}
						options.Provider = provider
					}
					if apiKey, ok := runtimeConfig["api_key"].(string); ok && apiKey != "" {
						options.APIKey = apiKey
					}
					if endpoint, ok := runtimeConfig["endpoint"].(string); ok && endpoint != "" {
						options.Endpoint = endpoint
					}
					if maxTokens, ok := runtimeConfig["max_tokens"].(float64); ok {
						options.MaxTokens = int(maxTokens)
					}
				}
			} else {
				// Add other context as strings
				options.Context = append(options.Context, fmt.Sprintf("%s: %s", key, value))
			}
		}
	}

	// Generate SQL using the generator
	result, err := e.generator.Generate(ctx, req.NaturalLanguage, options)
	if err != nil {
		return nil, fmt.Errorf("failed to generate SQL: %w", err)
	}

	// Convert generator result to engine response
	return &GenerateSQLResponse{
		SQL:             result.SQL,
		Explanation:     result.Explanation,
		ConfidenceScore: float32(result.ConfidenceScore),
		ProcessingTime:  result.Metadata.ProcessingTime,
		RequestID:       result.Metadata.RequestID,
		ModelUsed:       result.Metadata.ModelUsed,
		DebugInfo:       addDebugInfo(result.Metadata.DebugInfo, fmt.Sprintf("Query complexity: %s", result.Metadata.Complexity)),
	}, nil
}


// GetCapabilities implements Engine.GetCapabilities for AI engine
func (e *aiEngine) GetCapabilities() *SQLCapabilities {
	if e.generator != nil {
		return e.generator.GetCapabilities()
	}
	// Fallback to basic capabilities
	return &SQLCapabilities{
		SupportedDatabases: []string{"mysql", "postgresql", "sqlite"},
		Features: []SQLFeature{
			{
				Name:        "SQL Generation",
				Enabled:     true,
				Description: "AI-powered SQL generation from natural language",
			},
		},
	}
}


// IsHealthy implements Engine.IsHealthy for AI engine
func (e *aiEngine) IsHealthy() bool {
	if e.client != nil {
		// Check if primary client is healthy
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		healthStatus, err := e.aiClient.HealthCheck(ctx)
		return err == nil && healthStatus != nil && healthStatus.Healthy
	}
	return false
}

// Close implements Engine.Close for AI engine
func (e *aiEngine) Close() {
	if e.client != nil {
		_ = e.client.Close()
	}
	if e.aiClient != nil {
		_ = e.aiClient.Close()
	}
}
