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

// LoadBalancer manages multiple AI clients and selects the best one for requests
type LoadBalancer interface {
	// SelectClient selects the best available client for the given request
	SelectClient(req *GenerateRequest) (interfaces.AIClient, error)

	// UpdateHealth updates the health status of a client
	UpdateHealth(clientID string, healthy bool)

	// GetHealthyClients returns a list of currently healthy clients
	GetHealthyClients() []string

	// GetStats returns load balancing statistics
	GetStats() *LoadBalancerStats
}

// LoadBalancerStats provides statistics about load balancing
type LoadBalancerStats struct {
	// TotalRequests is the total number of requests processed
	TotalRequests int64 `json:"total_requests"`

	// ClientStats provides per-client statistics
	ClientStats map[string]*ClientStats `json:"client_stats"`

	// LastUpdated indicates when the stats were last updated
	LastUpdated time.Time `json:"last_updated"`
}

// ClientStats provides statistics for a specific client
type ClientStats struct {
	// Requests is the number of requests sent to this client
	Requests int64 `json:"requests"`

	// Successes is the number of successful requests
	Successes int64 `json:"successes"`

	// Failures is the number of failed requests
	Failures int64 `json:"failures"`

	// AverageResponseTime is the average response time
	AverageResponseTime time.Duration `json:"average_response_time"`

	// LastUsed indicates when this client was last used
	LastUsed time.Time `json:"last_used"`

	// HealthStatus indicates the current health status
	HealthStatus bool `json:"health_status"`
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

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker interface {
	// Call executes a function with circuit breaker protection
	Call(ctx context.Context, fn func() error) error

	// State returns the current circuit breaker state
	State() CircuitState

	// Reset manually resets the circuit breaker
	Reset()

	// GetMetrics returns circuit breaker metrics
	GetMetrics() *CircuitBreakerMetrics
}

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	// CircuitClosed indicates the circuit is closed (normal operation)
	CircuitClosed CircuitState = iota

	// CircuitOpen indicates the circuit is open (blocking requests)
	CircuitOpen

	// CircuitHalfOpen indicates the circuit is half-open (testing recovery)
	CircuitHalfOpen
)

// String returns the string representation of the circuit state
func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerMetrics provides metrics for a circuit breaker
type CircuitBreakerMetrics struct {
	// State is the current circuit state
	State CircuitState `json:"state"`

	// TotalRequests is the total number of requests
	TotalRequests int64 `json:"total_requests"`

	// SuccessfulRequests is the number of successful requests
	SuccessfulRequests int64 `json:"successful_requests"`

	// FailedRequests is the number of failed requests
	FailedRequests int64 `json:"failed_requests"`

	// ConsecutiveFailures is the number of consecutive failures
	ConsecutiveFailures int64 `json:"consecutive_failures"`

	// LastFailureTime is the time of the last failure
	LastFailureTime *time.Time `json:"last_failure_time,omitempty"`

	// LastSuccessTime is the time of the last success
	LastSuccessTime *time.Time `json:"last_success_time,omitempty"`
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

	// LoadBalancer configures the load balancing strategy
	LoadBalancer LoadBalancerConfig `json:"load_balancer"`

	// Retry configures the retry behavior
	Retry RetryConfig `json:"retry"`

	// CircuitBreaker configures the circuit breaker behavior
	CircuitBreaker CircuitBreakerConfig `json:"circuit_breaker"`

	// Monitoring configures monitoring and metrics
	Monitoring MonitoringConfig `json:"monitoring,omitempty"`
}

// LoadBalancerConfig configures load balancing behavior
type LoadBalancerConfig struct {
	// Strategy specifies the load balancing strategy
	Strategy string `json:"strategy"` // round_robin, weighted, least_connections, failover

	// HealthCheckInterval specifies how often to check provider health
	HealthCheckInterval time.Duration `json:"health_check_interval"`

	// HealthCheckTimeout specifies the timeout for health checks
	HealthCheckTimeout time.Duration `json:"health_check_timeout"`
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

// CircuitBreakerConfig configures circuit breaker behavior
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of consecutive failures before opening
	FailureThreshold int `json:"failure_threshold"`

	// ResetTimeout is how long to wait before attempting to close the circuit
	ResetTimeout time.Duration `json:"reset_timeout"`

	// HalfOpenMaxCalls is the maximum number of calls in half-open state
	HalfOpenMaxCalls int `json:"half_open_max_calls"`

	// SuccessThreshold is the number of successes needed to close the circuit
	SuccessThreshold int `json:"success_threshold"`
}

// MonitoringConfig configures monitoring and metrics
type MonitoringConfig struct {
	// Enabled indicates if monitoring is enabled
	Enabled bool `json:"enabled"`

	// MetricsPort specifies the port for metrics endpoint
	MetricsPort int `json:"metrics_port,omitempty"`

	// LogLevel specifies the logging level
	LogLevel string `json:"log_level,omitempty"`

	// TracingEnabled indicates if distributed tracing is enabled
	TracingEnabled bool `json:"tracing_enabled,omitempty"`
}
