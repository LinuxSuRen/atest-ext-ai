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
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"syscall"
	"time"
)

// retryableError is an error that can be retried
type retryableError struct {
	err       error
	retryable bool
}

func (e *retryableError) Error() string {
	return e.err.Error()
}

func (e *retryableError) Unwrap() error {
	return e.err
}

// IsRetryable returns whether the error is retryable
func (e *retryableError) IsRetryable() bool {
	return e.retryable
}

// defaultRetryManager implements the RetryManager interface
type defaultRetryManager struct {
	config RetryConfig
}

// NewDefaultRetryManager creates a new default retry manager
func NewDefaultRetryManager(config RetryConfig) RetryManager {
	// Set default values if not provided
	if config.MaxAttempts == 0 {
		config.MaxAttempts = 3
	}
	if config.BaseDelay == 0 {
		config.BaseDelay = time.Second
	}
	if config.MaxDelay == 0 {
		config.MaxDelay = 30 * time.Second
	}
	if config.BackoffMultiplier == 0 {
		config.BackoffMultiplier = 2.0
	}

	return &defaultRetryManager{
		config: config,
	}
}

// Execute executes a function with retry logic
func (rm *defaultRetryManager) Execute(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt < rm.config.MaxAttempts; attempt++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute the function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !rm.ShouldRetry(err) {
			return err
		}

		// Don't wait after the last attempt
		if attempt == rm.config.MaxAttempts-1 {
			break
		}

		// Calculate delay
		delay := rm.GetRetryDelay(attempt)

		// Wait before retrying
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("all retry attempts failed, last error: %w", lastErr)
}

// ExecuteWithResult executes a function with retry logic and returns a result
func (rm *defaultRetryManager) ExecuteWithResult(ctx context.Context, fn func() (*GenerateResponse, error)) (*GenerateResponse, error) {
	var lastErr error

	for attempt := 0; attempt < rm.config.MaxAttempts; attempt++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Execute the function
		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if we should retry
		if !rm.ShouldRetry(err) {
			return nil, err
		}

		// Don't wait after the last attempt
		if attempt == rm.config.MaxAttempts-1 {
			break
		}

		// Calculate delay
		delay := rm.GetRetryDelay(attempt)

		// Wait before retrying
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}

	return nil, fmt.Errorf("all retry attempts failed, last error: %w", lastErr)
}

// ShouldRetry determines if an error should trigger a retry
func (rm *defaultRetryManager) ShouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// Check for retryable error interface
	var retryableErr *retryableError
	if errors.As(err, &retryableErr) {
		return retryableErr.IsRetryable()
	}

	// Check for specific error types that should be retried
	return rm.isRetryableError(err)
}

// GetRetryDelay calculates the delay before the next retry attempt
func (rm *defaultRetryManager) GetRetryDelay(attempt int) time.Duration {
	delay := float64(rm.config.BaseDelay) * math.Pow(rm.config.BackoffMultiplier, float64(attempt))

	// Apply jitter if enabled
	if rm.config.Jitter {
		jitter := rand.Float64() * 0.1 // 10% jitter
		delay = delay * (1 + jitter)
	}

	// Ensure we don't exceed max delay
	if time.Duration(delay) > rm.config.MaxDelay {
		delay = float64(rm.config.MaxDelay)
	}

	return time.Duration(delay)
}

// isRetryableError checks if an error is retryable based on error type
func (rm *defaultRetryManager) isRetryableError(err error) bool {
	// Context cancellation and timeout are not retryable - check first
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Circuit breaker errors are not retryable
	if errors.Is(err, ErrCircuitBreakerOpen) {
		return false
	}

	// Network errors are generally retryable
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
	}

	// DNS errors are retryable
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	// Connection refused errors are retryable
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if opErr.Op == "dial" {
			return true
		}
	}

	// System call errors
	var syscallErr *syscall.Errno
	if errors.As(err, &syscallErr) {
		switch *syscallErr {
		case syscall.ECONNREFUSED, syscall.ECONNRESET, syscall.ETIMEDOUT:
			return true
		}
	}

	// Provider-specific errors can be checked here
	return rm.isProviderErrorRetryable(err)
}

// isProviderErrorRetryable checks if provider-specific errors are retryable
func (rm *defaultRetryManager) isProviderErrorRetryable(err error) bool {
	errorMsg := err.Error()

	// Rate limiting errors are retryable
	if containsAny(errorMsg, []string{
		"rate limit",
		"too many requests",
		"quota exceeded",
		"429",
	}) {
		return true
	}

	// Server errors are retryable
	if containsAny(errorMsg, []string{
		"internal server error",
		"service unavailable",
		"bad gateway",
		"gateway timeout",
		"500", "502", "503", "504",
	}) {
		return true
	}

	// Authentication errors are generally not retryable
	if containsAny(errorMsg, []string{
		"unauthorized",
		"forbidden",
		"invalid api key",
		"authentication failed",
		"401", "403",
	}) {
		return false
	}

	// Bad request errors are not retryable
	if containsAny(errorMsg, []string{
		"bad request",
		"invalid request",
		"malformed",
		"400",
	}) {
		return false
	}

	// Default to not retryable for unknown errors
	return false
}

// containsAny checks if a string contains any of the given substrings
func containsAny(s string, substrings []string) bool {
	for _, substring := range substrings {
		if len(s) >= len(substring) {
			for i := 0; i <= len(s)-len(substring); i++ {
				if s[i:i+len(substring)] == substring {
					return true
				}
			}
		}
	}
	return false
}

// NewRetryableError creates a new retryable error
func NewRetryableError(err error, retryable bool) error {
	return &retryableError{
		err:       err,
		retryable: retryable,
	}
}

// IsRetryableError checks if an error is marked as retryable
func IsRetryableError(err error) bool {
	var retryableErr *retryableError
	if errors.As(err, &retryableErr) {
		return retryableErr.IsRetryable()
	}
	return false
}