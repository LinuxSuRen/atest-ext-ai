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
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
)

// Type aliases for compatibility
type AIClient = interfaces.AIClient
type GenerateRequest = interfaces.GenerateRequest
type GenerateResponse = interfaces.GenerateResponse
type TokenUsage = interfaces.TokenUsage
type Capabilities = interfaces.Capabilities
type ModelInfo = interfaces.ModelInfo
type Feature = interfaces.Feature
type RateLimits = interfaces.RateLimits
type HealthStatus = interfaces.HealthStatus

// ClientFactory creates AI clients based on provider configuration
type ClientFactory interface {
	// CreateClient creates a new AI client for the specified provider
	CreateClient(provider string, config map[string]any) (interfaces.AIClient, error)

	// GetSupportedProviders returns a list of supported provider names
	GetSupportedProviders() []string

	// ValidateConfig validates the configuration for a specific provider
	ValidateConfig(provider string, config map[string]any) error
}


// RetryManager handles retry logic for failed requests
type RetryManager interface {
	// Execute executes a function with retry logic
	Execute(ctx context.Context, fn func() error) error

	// ShouldRetry determines if an error should trigger a retry
	ShouldRetry(err error) bool

	// GetRetryDelay calculates the delay before the next retry attempt
	GetRetryDelay(attempt int) time.Duration
}


// ProviderConfig represents configuration for a specific AI provider
type ProviderConfig struct {
	// Name is the provider name (openai, anthropic, local, etc.)
	Name string `json:"name"`

	// Enabled indicates if this provider is enabled
	Enabled bool `json:"enabled"`

	// Priority indicates the priority of this provider (higher = more preferred)
	Priority int `json:"priority"`

	// Config contains provider-specific configuration
	Config map[string]any `json:"config"`

	// Models lists the models available for this provider
	Models []string `json:"models,omitempty"`

	// Timeout specifies the request timeout for this provider
	Timeout time.Duration `json:"timeout,omitempty"`

	// MaxRetries specifies the maximum number of retries for this provider
	MaxRetries int `json:"max_retries,omitempty"`
}

// AIServiceConfig represents the complete AI service configuration
type AIServiceConfig struct {
	// Providers lists all configured AI providers
	Providers []ProviderConfig `json:"providers"`


	// Retry configures the retry behavior
	Retry RetryConfig `json:"retry"`


}


// RetryConfig configures retry behavior
type RetryConfig struct {
	// MaxAttempts is the maximum number of retry attempts
	MaxAttempts int `json:"max_attempts"`

	// BaseDelay is the base delay between retries
	BaseDelay time.Duration `json:"base_delay"`

	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration `json:"max_delay"`

	// BackoffMultiplier is the multiplier for exponential backoff
	BackoffMultiplier float64 `json:"backoff_multiplier"`

	// Jitter enables random jitter in retry delays
	Jitter bool `json:"jitter"`
}


