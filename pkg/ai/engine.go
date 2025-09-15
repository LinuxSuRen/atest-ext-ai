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

package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/config"
)

// Engine defines the interface for AI SQL generation
type Engine interface {
	GenerateSQL(ctx context.Context, req *GenerateSQLRequest) (*GenerateSQLResponse, error)
	GetCapabilities() *Capabilities
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

// Capabilities represents AI engine capabilities
type Capabilities struct {
	SupportedDatabases []string  `json:"supported_databases"`
	Features           []Feature `json:"features"`
}

// Feature represents a specific AI feature
type Feature struct {
	Name        string            `json:"name"`
	Enabled     bool              `json:"enabled"`
	Description string            `json:"description"`
	Parameters  map[string]string `json:"parameters,omitempty"`
}

// basicEngine is a basic implementation for testing
type basicEngine struct {
	config LegacyAIConfig
}

// NewEngine creates a new AI engine based on configuration
func NewEngine(config config.AIConfig) (Engine, error) {
	// Convert new config format to legacy for basicEngine compatibility
	legacyConfig := convertToLegacyAIConfig(config)

	switch legacyConfig.Provider {
	case "local", "ollama":
		return NewOllamaEngine(legacyConfig)
	case "openai":
		return NewOpenAIEngine(legacyConfig)
	case "claude":
		return NewClaudeEngine(legacyConfig)
	default:
		// Return basic engine for unsupported providers or fallback
		return &basicEngine{config: legacyConfig}, nil
	}
}

// convertToLegacyAIConfig converts new AIConfig to legacy format for backward compatibility
func convertToLegacyAIConfig(newConfig config.AIConfig) LegacyAIConfig {
	legacy := LegacyAIConfig{
		Provider:            newConfig.DefaultService,
		ConfidenceThreshold: 0.7,
		SupportedDatabases:  []string{"mysql", "postgresql", "sqlite"},
		EnableSQLExecution:  true,
		Metadata:            make(map[string]string),
	}

	// Get configuration from the default service
	if service, exists := newConfig.Services[newConfig.DefaultService]; exists {
		legacy.OllamaEndpoint = service.Endpoint
		legacy.APIKey = service.APIKey
		legacy.Model = service.Model

		// Map provider names to legacy format
		if service.Provider == "ollama" {
			legacy.Provider = "local"
		} else {
			legacy.Provider = service.Provider
		}
	}

	return legacy
}

// LegacyAIConfig represents the legacy configuration format for backward compatibility
type LegacyAIConfig struct {
	Provider           string            `json:"provider"`
	OllamaEndpoint     string            `json:"ollama_endpoint"`
	Model              string            `json:"model"`
	APIKey             string            `json:"api_key"`
	ConfidenceThreshold float32           `json:"confidence_threshold"`
	SupportedDatabases []string          `json:"supported_databases"`
	EnableSQLExecution bool              `json:"enable_sql_execution"`
	Metadata           map[string]string `json:"metadata"`
}

// NewOllamaEngine creates an Ollama-based AI engine
func NewOllamaEngine(config LegacyAIConfig) (Engine, error) {
	return &basicEngine{config: config}, nil
}

// NewOpenAIEngine creates an OpenAI-based AI engine
func NewOpenAIEngine(config LegacyAIConfig) (Engine, error) {
	return &basicEngine{config: config}, nil
}

// NewClaudeEngine creates a Claude-based AI engine
func NewClaudeEngine(config LegacyAIConfig) (Engine, error) {
	return &basicEngine{config: config}, nil
}

// GenerateSQL implements Engine.GenerateSQL
func (e *basicEngine) GenerateSQL(ctx context.Context, req *GenerateSQLRequest) (*GenerateSQLResponse, error) {
	start := time.Now()

	// Basic implementation that returns a simple response
	return &GenerateSQLResponse{
		SQL:             "SELECT * FROM table_name;", // Basic SQL as placeholder
		Explanation:     fmt.Sprintf("Generated basic SQL for: %s", req.NaturalLanguage),
		ConfidenceScore: 0.5,
		ProcessingTime:  time.Since(start),
		RequestID:       fmt.Sprintf("req_%d", time.Now().UnixNano()),
		ModelUsed:       e.config.Provider,
		DebugInfo:       []string{"Using basic implementation"},
	}, nil
}

// GetCapabilities implements Engine.GetCapabilities
func (e *basicEngine) GetCapabilities() *Capabilities {
	return &Capabilities{
		SupportedDatabases: []string{"mysql", "postgresql", "sqlite"},
		Features: []Feature{
			{
				Name:        "SQL Generation",
				Enabled:     true,
				Description: "Basic SQL generation from natural language",
			},
		},
	}
}

// IsHealthy implements Engine.IsHealthy
func (e *basicEngine) IsHealthy() bool {
	return true
}

// Close implements Engine.Close
func (e *basicEngine) Close() {
	// No cleanup needed for basic implementation
}