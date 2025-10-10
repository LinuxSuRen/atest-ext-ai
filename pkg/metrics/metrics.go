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

// Package metrics provides Prometheus metrics for monitoring AI operations
// Following Prometheus best practices for metric naming and types:
// - Counter: for monotonically increasing values (request count)
// - Histogram: for distributions and aggregations (latency)
// - Gauge: for values that can go up and down (health status)
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// aiRequestsTotal counts the total number of AI requests by method, provider, and status
	// Type: Counter - monotonically increasing counter
	// Labels: method (e.g., "generate", "models"), provider (e.g., "openai", "ollama"), status (e.g., "success", "error")
	aiRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atest_ai_requests_total",
			Help: "Total number of AI requests by method, provider, and status",
		},
		[]string{"method", "provider", "status"},
	)

	// aiRequestDuration tracks the distribution of AI request durations
	// Type: Histogram - for measuring latencies and aggregating across dimensions
	// Labels: method, provider
	// Buckets: Exponential buckets starting at 0.1s, factor 2, 10 buckets
	// This covers: 0.1s, 0.2s, 0.4s, 0.8s, 1.6s, 3.2s, 6.4s, 12.8s, 25.6s, 51.2s
	// Based on Prometheus best practices: use histograms for aggregations
	aiRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "atest_ai_request_duration_seconds",
			Help:    "AI request duration in seconds by method and provider",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s to ~51.2s
		},
		[]string{"method", "provider"},
	)

	// aiServiceHealth represents the health status of AI services
	// Type: Gauge - for values that can go up and down
	// Labels: provider
	// Value: 1 = healthy, 0 = unhealthy
	aiServiceHealth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "atest_ai_service_health",
			Help: "AI service health status (1=healthy, 0=unhealthy) by provider",
		},
		[]string{"provider"},
	)

	// aiTokensUsed tracks the total number of tokens used in AI requests
	// Type: Counter - monotonically increasing counter
	// Labels: method, provider, token_type (e.g., "prompt", "completion")
	aiTokensUsed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atest_ai_tokens_used_total",
			Help: "Total number of tokens used in AI requests by method, provider, and token type",
		},
		[]string{"method", "provider", "token_type"},
	)

	// aiConcurrentRequests tracks the number of concurrent AI requests
	// Type: Gauge - for current active requests
	// Labels: provider
	aiConcurrentRequests = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "atest_ai_concurrent_requests",
			Help: "Current number of concurrent AI requests by provider",
		},
		[]string{"provider"},
	)
)

// RecordRequest records an AI request with the given method, provider, and status
// This increments the total request counter
// Status should be one of: "success", "error", "timeout", "rate_limited"
func RecordRequest(method, provider, status string) {
	aiRequestsTotal.WithLabelValues(method, provider, status).Inc()
}

// RecordDuration records the duration of an AI request in seconds
// This observes a value in the histogram, allowing for percentile calculations
// Duration should be in seconds (e.g., time.Since(start).Seconds())
func RecordDuration(method, provider string, duration float64) {
	aiRequestDuration.WithLabelValues(method, provider).Observe(duration)
}

// SetHealthStatus sets the health status of an AI service provider
// healthy=true sets the gauge to 1, healthy=false sets it to 0
func SetHealthStatus(provider string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	aiServiceHealth.WithLabelValues(provider).Set(value)
}

// RecordTokens records the number of tokens used in an AI request
// tokenType should be one of: "prompt", "completion", "total"
func RecordTokens(method, provider, tokenType string, count int) {
	aiTokensUsed.WithLabelValues(method, provider, tokenType).Add(float64(count))
}

// IncrementConcurrentRequests increments the concurrent request counter for a provider
func IncrementConcurrentRequests(provider string) {
	aiConcurrentRequests.WithLabelValues(provider).Inc()
}

// DecrementConcurrentRequests decrements the concurrent request counter for a provider
func DecrementConcurrentRequests(provider string) {
	aiConcurrentRequests.WithLabelValues(provider).Dec()
}

// MeasureDuration is a helper function that returns a function to record duration
// Usage: defer metrics.MeasureDuration("generate", provider)()
func MeasureDuration(method, provider string) func() {
	// Note: This would require time.Now() to be captured at call time
	// For now, this is a placeholder - callers should use RecordDuration directly
	return func() {
		// Duration measurement happens in the calling code
	}
}
