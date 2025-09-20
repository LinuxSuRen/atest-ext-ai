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
	"sync"
	"time"
)

// defaultCircuitBreaker implements the CircuitBreaker interface
type defaultCircuitBreaker struct {
	config              CircuitBreakerConfig
	state               CircuitState
	consecutiveFailures int64
	lastFailureTime     *time.Time
	lastSuccessTime     *time.Time
	halfOpenCalls       int
	halfOpenSuccesses   int
	totalRequests       int64
	successfulRequests  int64
	failedRequests      int64
	mu                  sync.RWMutex
}

// NewDefaultCircuitBreaker creates a new default circuit breaker
func NewDefaultCircuitBreaker(config CircuitBreakerConfig) CircuitBreaker {
	// Set default values if not provided
	if config.FailureThreshold == 0 {
		config.FailureThreshold = 5
	}
	if config.ResetTimeout == 0 {
		config.ResetTimeout = 60 * time.Second
	}
	if config.HalfOpenMaxCalls == 0 {
		config.HalfOpenMaxCalls = 3
	}
	if config.SuccessThreshold == 0 {
		config.SuccessThreshold = 2
	}

	return &defaultCircuitBreaker{
		config: config,
		state:  CircuitClosed,
	}
}

// Call executes a function with circuit breaker protection
func (cb *defaultCircuitBreaker) Call(ctx context.Context, fn func() error) error {
	// Check if we can proceed
	if !cb.allowRequest() {
		return ErrCircuitBreakerOpen
	}

	// Increment half-open calls counter if in half-open state
	cb.mu.Lock()
	if cb.state == CircuitHalfOpen {
		cb.halfOpenCalls++
	}
	cb.mu.Unlock()

	start := time.Now()
	err := fn()
	duration := time.Since(start)

	// Record the result
	cb.recordResult(err == nil, duration)

	return err
}

// State returns the current circuit breaker state
func (cb *defaultCircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset manually resets the circuit breaker
func (cb *defaultCircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitClosed
	cb.consecutiveFailures = 0
	cb.halfOpenCalls = 0
	cb.lastFailureTime = nil
}

// GetMetrics returns circuit breaker metrics
func (cb *defaultCircuitBreaker) GetMetrics() *CircuitBreakerMetrics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	metrics := &CircuitBreakerMetrics{
		State:               cb.state,
		TotalRequests:       cb.totalRequests,
		SuccessfulRequests:  cb.successfulRequests,
		FailedRequests:      cb.failedRequests,
		ConsecutiveFailures: cb.consecutiveFailures,
	}

	// Copy time pointers
	if cb.lastFailureTime != nil {
		failureTime := *cb.lastFailureTime
		metrics.LastFailureTime = &failureTime
	}
	if cb.lastSuccessTime != nil {
		successTime := *cb.lastSuccessTime
		metrics.LastSuccessTime = &successTime
	}

	return metrics
}

// allowRequest determines if a request should be allowed
func (cb *defaultCircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if we should transition to half-open
		if cb.shouldTransitionToHalfOpen() {
			cb.state = CircuitHalfOpen
			cb.halfOpenCalls = 0
			cb.halfOpenSuccesses = 0
			return true
		}
		return false
	case CircuitHalfOpen:
		// Allow limited requests in half-open state
		if cb.halfOpenCalls < cb.config.HalfOpenMaxCalls {
			// Don't increment here, wait until call is actually made
			return true
		}
		return false
	default:
		return false
	}
}

// shouldTransitionToHalfOpen checks if the circuit should transition from open to half-open
func (cb *defaultCircuitBreaker) shouldTransitionToHalfOpen() bool {
	if cb.lastFailureTime == nil {
		return false
	}
	return time.Since(*cb.lastFailureTime) >= cb.config.ResetTimeout
}

// recordResult records the result of a function call
func (cb *defaultCircuitBreaker) recordResult(success bool, duration time.Duration) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalRequests++

	if success {
		cb.recordSuccess()
	} else {
		cb.recordFailure()
	}

	// Update state based on the result
	cb.updateState(success)
}

// recordSuccess records a successful request
func (cb *defaultCircuitBreaker) recordSuccess() {
	cb.successfulRequests++
	cb.consecutiveFailures = 0
	now := time.Now()
	cb.lastSuccessTime = &now

	// Track successes in half-open state
	if cb.state == CircuitHalfOpen {
		cb.halfOpenSuccesses++
	}
}

// recordFailure records a failed request
func (cb *defaultCircuitBreaker) recordFailure() {
	cb.failedRequests++
	cb.consecutiveFailures++
	now := time.Now()
	cb.lastFailureTime = &now
}

// updateState updates the circuit breaker state based on the latest result
func (cb *defaultCircuitBreaker) updateState(success bool) {
	switch cb.state {
	case CircuitClosed:
		if !success && cb.consecutiveFailures >= int64(cb.config.FailureThreshold) {
			cb.state = CircuitOpen
		}
	case CircuitHalfOpen:
		if success {
			// Check if we have enough successes to close the circuit
			if cb.halfOpenSuccesses >= cb.config.SuccessThreshold {
				cb.state = CircuitClosed
				cb.halfOpenCalls = 0
				cb.halfOpenSuccesses = 0
			}
			// If we haven't reached the threshold, stay in half-open
		} else {
			// Any failure in half-open state transitions back to open
			cb.state = CircuitOpen
			cb.halfOpenCalls = 0
			cb.halfOpenSuccesses = 0
		}
	case CircuitOpen:
		// State transitions are handled in allowRequest()
	}
}

// CircuitBreakerStats provides additional statistics for monitoring
type CircuitBreakerStats struct {
	State               CircuitState  `json:"state"`
	TotalRequests       int64         `json:"total_requests"`
	SuccessfulRequests  int64         `json:"successful_requests"`
	FailedRequests      int64         `json:"failed_requests"`
	ConsecutiveFailures int64         `json:"consecutive_failures"`
	LastFailureTime     *time.Time    `json:"last_failure_time,omitempty"`
	LastSuccessTime     *time.Time    `json:"last_success_time,omitempty"`
	SuccessRate         float64       `json:"success_rate"`
	FailureRate         float64       `json:"failure_rate"`
	Uptime              time.Duration `json:"uptime,omitempty"`
}

// GetStats returns detailed statistics for the circuit breaker
func (cb *defaultCircuitBreaker) GetStats() *CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	stats := &CircuitBreakerStats{
		State:               cb.state,
		TotalRequests:       cb.totalRequests,
		SuccessfulRequests:  cb.successfulRequests,
		FailedRequests:      cb.failedRequests,
		ConsecutiveFailures: cb.consecutiveFailures,
	}

	// Calculate rates
	if cb.totalRequests > 0 {
		stats.SuccessRate = float64(cb.successfulRequests) / float64(cb.totalRequests)
		stats.FailureRate = float64(cb.failedRequests) / float64(cb.totalRequests)
	}

	// Copy time pointers
	if cb.lastFailureTime != nil {
		failureTime := *cb.lastFailureTime
		stats.LastFailureTime = &failureTime
	}
	if cb.lastSuccessTime != nil {
		successTime := *cb.lastSuccessTime
		stats.LastSuccessTime = &successTime
	}

	// Calculate uptime (time since last failure or circuit creation)
	if cb.lastFailureTime != nil {
		stats.Uptime = time.Since(*cb.lastFailureTime)
	}

	return stats
}

// IsHealthy returns whether the circuit breaker considers the service healthy
func (cb *defaultCircuitBreaker) IsHealthy() bool {
	return cb.State() == CircuitClosed
}

// GetFailureRate returns the current failure rate
func (cb *defaultCircuitBreaker) GetFailureRate() float64 {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.totalRequests == 0 {
		return 0.0
	}

	return float64(cb.failedRequests) / float64(cb.totalRequests)
}
