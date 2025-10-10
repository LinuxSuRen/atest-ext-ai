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

package interfaces

import (
	"context"
	"time"
)

// GenerateRequest represents a unified AI generation request
type GenerateRequest struct {
	// Prompt is the input text for generation
	Prompt string `json:"prompt"`

	// Model specifies which model to use for generation
	Model string `json:"model"`

	// MaxTokens limits the maximum number of tokens in the response
	MaxTokens int `json:"max_tokens,omitempty"`

	// Context provides additional context for the generation
	Context []string `json:"context,omitempty"`

	// Options allows provider-specific parameters
	Options map[string]any `json:"options,omitempty"`

	// SystemPrompt provides system-level instructions
	SystemPrompt string `json:"system_prompt,omitempty"`

	// Stream indicates whether to stream the response
	Stream bool `json:"stream,omitempty"`
}

// GenerateResponse represents a unified AI generation response
type GenerateResponse struct {
	// Text is the generated content
	Text string `json:"text"`

	// Model indicates which model was actually used
	Model string `json:"model"`

	// Metadata contains provider-specific metadata
	Metadata map[string]any `json:"metadata,omitempty"`

	// RequestID is a unique identifier for this request
	RequestID string `json:"request_id,omitempty"`

	// ProcessingTime indicates how long the generation took
	ProcessingTime time.Duration `json:"processing_time"`

	// ConfidenceScore indicates the model's confidence in the response
	ConfidenceScore float64 `json:"confidence_score,omitempty"`
}

// HealthStatus represents the health status of an AI service
type HealthStatus struct {
	// Healthy indicates if the service is healthy
	Healthy bool `json:"healthy"`

	// Status provides a human-readable status message
	Status string `json:"status"`

	// ResponseTime indicates the response time for the health check
	ResponseTime time.Duration `json:"response_time"`

	// LastChecked indicates when the health check was last performed
	LastChecked time.Time `json:"last_checked"`

	// Metadata contains additional health-related information
	Metadata map[string]any `json:"metadata,omitempty"`

	// Errors contains any errors encountered during health check
	Errors []string `json:"errors,omitempty"`
}

// Capabilities describes the capabilities of an AI client
type Capabilities struct {
	// Provider identifies the AI service provider
	Provider string `json:"provider"`

	// Models lists the available models
	Models []ModelInfo `json:"models"`

	// Features lists the supported features
	Features []Feature `json:"features"`

	// MaxTokens indicates the maximum token limit
	MaxTokens int `json:"max_tokens"`

	// SupportedLanguages lists supported programming/natural languages
	SupportedLanguages []string `json:"supported_languages,omitempty"`

	// RateLimits describes the rate limiting information
	RateLimits *RateLimits `json:"rate_limits,omitempty"`
}

// ModelInfo provides information about a specific model
type ModelInfo struct {
	// ID is the model identifier
	ID string `json:"id"`

	// Name is the human-readable model name
	Name string `json:"name"`

	// Description describes the model's capabilities
	Description string `json:"description,omitempty"`

	// MaxTokens is the maximum context length for this model
	MaxTokens int `json:"max_tokens"`

	// InputCostPer1K is the cost per 1K input tokens (if applicable)
	InputCostPer1K float64 `json:"input_cost_per_1k,omitempty"`

	// OutputCostPer1K is the cost per 1K output tokens (if applicable)
	OutputCostPer1K float64 `json:"output_cost_per_1k,omitempty"`

	// Capabilities lists model-specific capabilities
	Capabilities []string `json:"capabilities,omitempty"`
}

// Feature represents a specific AI feature
type Feature struct {
	// Name is the feature identifier
	Name string `json:"name"`

	// Enabled indicates if the feature is currently enabled
	Enabled bool `json:"enabled"`

	// Description describes what the feature does
	Description string `json:"description"`

	// Parameters contains feature-specific parameters
	Parameters map[string]string `json:"parameters,omitempty"`

	// Version indicates the feature version
	Version string `json:"version,omitempty"`
}

// RateLimits describes rate limiting information
type RateLimits struct {
	// RequestsPerMinute is the limit on requests per minute
	RequestsPerMinute int `json:"requests_per_minute,omitempty"`

	// TokensPerMinute is the limit on tokens per minute
	TokensPerMinute int `json:"tokens_per_minute,omitempty"`

	// RequestsPerDay is the limit on requests per day
	RequestsPerDay int `json:"requests_per_day,omitempty"`

	// TokensPerDay is the limit on tokens per day
	TokensPerDay int `json:"tokens_per_day,omitempty"`
}

// AIClient defines the unified interface for AI service providers
type AIClient interface {
	// Generate executes an AI generation request
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)

	// GetCapabilities returns the capabilities of this AI client
	GetCapabilities(ctx context.Context) (*Capabilities, error)

	// HealthCheck performs a health check on the AI service
	HealthCheck(ctx context.Context) (*HealthStatus, error)

	// Close releases any resources held by the client
	Close() error
}
