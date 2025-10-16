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
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
)

// AIClient is retained for backward compatibility; prefer interfaces.AIClient.
//
//revive:disable:exported
type AIClient = interfaces.AIClient

// GenerateRequest is retained for backward compatibility.
type GenerateRequest = interfaces.GenerateRequest

// GenerateResponse is retained for backward compatibility.
type GenerateResponse = interfaces.GenerateResponse

// Capabilities is retained for backward compatibility.
type Capabilities = interfaces.Capabilities

// ModelInfo is retained for backward compatibility.
type ModelInfo = interfaces.ModelInfo

// Feature is retained for backward compatibility.
type Feature = interfaces.Feature

// RateLimits is retained for backward compatibility.
type RateLimits = interfaces.RateLimits

// HealthStatus is retained for backward compatibility.
type HealthStatus = interfaces.HealthStatus

//revive:enable:exported

// ProviderConfig represents configuration for a specific AI provider
type ProviderConfig struct {
	// Name is the provider name (openai, ollama, deepseek, custom, etc.)
	// Note: "local" is accepted as an alias for "ollama" for backward compatibility
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

// ServiceConfig represents the complete AI service configuration.
type ServiceConfig struct {
	// Providers lists all configured AI providers
	Providers []ProviderConfig `json:"providers"`

	// Retry configures the retry behavior
	Retry RetryConfig `json:"retry"`
}

//revive:disable-next-line exported
type AIServiceConfig = ServiceConfig

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
