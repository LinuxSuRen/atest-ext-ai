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
	"log"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/config"
	"github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
)

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

// basicEngine is a basic implementation for testing
type basicEngine struct {
	config      config.AIConfig
	generator   *SQLGenerator
	aiClient    interfaces.AIClient
}

// aiEngine is a full-featured implementation using AI clients
type aiEngine struct {
	config      config.AIConfig
	generator   *SQLGenerator
	aiClient    interfaces.AIClient
	client      *Client
}

// NewEngine creates a new AI engine based on configuration
func NewEngine(cfg config.AIConfig) (Engine, error) {
	// Try to create a full AI client first
	client, err := NewClient(cfg)
	if err != nil {
		log.Printf("Failed to create AI client, falling back to basic engine: %v", err)
		return &basicEngine{config: cfg}, nil
	}

	// Get the AI client from the Client
	aiClient := client.GetPrimaryClient()
	if aiClient == nil {
		log.Printf("No primary AI client available, using basic engine")
		return &basicEngine{config: cfg}, nil
	}

	// Create SQL generator with AI client
	generator, err := NewSQLGenerator(aiClient, cfg)
	if err != nil {
		log.Printf("Failed to create SQL generator: %v", err)
		return &basicEngine{config: cfg}, nil
	}

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
		ModelUsed:       e.config.DefaultService,
		DebugInfo:       []string{"Using basic implementation"},
	}, nil
}

// GenerateSQL implements Engine.GenerateSQL with full AI integration
func (e *aiEngine) GenerateSQL(ctx context.Context, req *GenerateSQLRequest) (*GenerateSQLResponse, error) {
	if e.generator == nil {
		return nil, fmt.Errorf("SQL generator not initialized")
	}

	// Convert request to generator options
	options := &GenerateOptions{
		DatabaseType:       req.DatabaseType,
		ValidateSQL:        true,
		OptimizeQuery:      false,
		IncludeExplanation: true,
		SafetyMode:         true,
		Temperature:        0.3,
		MaxTokens:          2000,
	}

	// Add context if provided
	if len(req.Context) > 0 {
		options.Context = make([]string, 0, len(req.Context))
		for key, value := range req.Context {
			options.Context = append(options.Context, fmt.Sprintf("%s: %s", key, value))
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
		DebugInfo:       append(result.Metadata.DebugInfo, fmt.Sprintf("Query complexity: %s", result.Metadata.Complexity)),
	}, nil
}
// GetCapabilities implements Engine.GetCapabilities
func (e *basicEngine) GetCapabilities() *SQLCapabilities {
	return &SQLCapabilities{
		SupportedDatabases: []string{"mysql", "postgresql", "sqlite"},
		Features: []SQLFeature{
			{
				Name:        "SQL Generation",
				Enabled:     true,
				Description: "Basic SQL generation from natural language",
			},
		},
	}
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
// IsHealthy implements Engine.IsHealthy
func (e *basicEngine) IsHealthy() bool {
	return true
}

// Close implements Engine.Close
func (e *basicEngine) Close() {
	// No cleanup needed for basic implementation
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
		e.client.Close()
	}
	if e.aiClient != nil {
		e.aiClient.Close()
	}
}