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
	"net"
	"syscall"
	"testing"
	"time"
)

func TestRetryManager_Execute_Success(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:       3,
		BaseDelay:         10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		Jitter:            false,
	}

	rm := NewDefaultRetryManager(config)

	callCount := 0
	err := rm.Execute(context.Background(), func() error {
		callCount++
		return nil // Success on first try
	})

	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
}

func TestRetryManager_Execute_RetrySuccess(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:       3,
		BaseDelay:         1 * time.Millisecond, // Short delay for testing
		MaxDelay:          10 * time.Millisecond,
		BackoffMultiplier: 2.0,
		Jitter:            false,
	}

	rm := NewDefaultRetryManager(config)

	callCount := 0
	err := rm.Execute(context.Background(), func() error {
		callCount++
		if callCount < 3 {
			// Create a retryable network error
			return &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}
		}
		return nil // Success on third try
	})

	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}

	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}
}

func TestRetryManager_Execute_MaxAttemptsReached(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:       2,
		BaseDelay:         1 * time.Millisecond,
		MaxDelay:          10 * time.Millisecond,
		BackoffMultiplier: 2.0,
		Jitter:            false,
	}

	rm := NewDefaultRetryManager(config)

	callCount := 0
	retryableErr := &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}
	err := rm.Execute(context.Background(), func() error {
		callCount++
		return retryableErr
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}

	if callCount != 2 {
		t.Errorf("Expected 2 calls, got %d", callCount)
	}
}

func TestRetryManager_Execute_NonRetryableError(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:       3,
		BaseDelay:         1 * time.Millisecond,
		MaxDelay:          10 * time.Millisecond,
		BackoffMultiplier: 2.0,
		Jitter:            false,
	}

	rm := NewDefaultRetryManager(config)

	callCount := 0
	nonRetryableErr := errors.New("bad request")
	err := rm.Execute(context.Background(), func() error {
		callCount++
		return nonRetryableErr
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call (no retry), got %d", callCount)
	}
}

func TestRetryManager_Execute_ContextCancellation(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:       3,
		BaseDelay:         100 * time.Millisecond, // Long delay to allow cancellation
		MaxDelay:          1 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            false,
	}

	rm := NewDefaultRetryManager(config)

	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := rm.Execute(ctx, func() error {
		callCount++
		return &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}

	// Should have made at least one call before cancellation
	if callCount < 1 {
		t.Errorf("Expected at least 1 call, got %d", callCount)
	}
}

func TestRetryManager_ShouldRetry(t *testing.T) {
	rm := NewDefaultRetryManager(RetryConfig{})

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "network timeout",
			err:      &net.OpError{Op: "dial", Err: syscall.ETIMEDOUT},
			expected: true,
		},
		{
			name:     "connection refused",
			err:      &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED},
			expected: true,
		},
		{
			name:     "DNS error",
			err:      &net.DNSError{IsTemporary: true},
			expected: true,
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: false,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: false,
		},
		{
			name:     "circuit breaker open",
			err:      ErrCircuitBreakerOpen,
			expected: false,
		},
		{
			name:     "rate limit error",
			err:      errors.New("rate limit exceeded"),
			expected: true,
		},
		{
			name:     "server error",
			err:      errors.New("internal server error"),
			expected: true,
		},
		{
			name:     "bad request",
			err:      errors.New("bad request"),
			expected: false,
		},
		{
			name:     "unauthorized",
			err:      errors.New("unauthorized"),
			expected: false,
		},
		{
			name:     "retryable error",
			err:      NewRetryableError(errors.New("custom error"), true),
			expected: true,
		},
		{
			name:     "non-retryable error",
			err:      NewRetryableError(errors.New("custom error"), false),
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := rm.ShouldRetry(test.err)
			if result != test.expected {
				t.Errorf("ShouldRetry() = %v, expected %v", result, test.expected)
			}
		})
	}
}

func TestRetryManager_GetRetryDelay(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:       3,
		BaseDelay:         100 * time.Millisecond,
		MaxDelay:          1 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            false,
	}

	rm := NewDefaultRetryManager(config)

	// Test exponential backoff
	delay0 := rm.GetRetryDelay(0)
	delay1 := rm.GetRetryDelay(1)
	delay2 := rm.GetRetryDelay(2)

	expectedDelay0 := 100 * time.Millisecond
	expectedDelay1 := 200 * time.Millisecond
	expectedDelay2 := 400 * time.Millisecond

	if delay0 != expectedDelay0 {
		t.Errorf("Delay for attempt 0: expected %v, got %v", expectedDelay0, delay0)
	}

	if delay1 != expectedDelay1 {
		t.Errorf("Delay for attempt 1: expected %v, got %v", expectedDelay1, delay1)
	}

	if delay2 != expectedDelay2 {
		t.Errorf("Delay for attempt 2: expected %v, got %v", expectedDelay2, delay2)
	}
}

func TestRetryManager_GetRetryDelay_MaxDelay(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:       10,
		BaseDelay:         100 * time.Millisecond,
		MaxDelay:          300 * time.Millisecond, // Lower than what exponential backoff would produce
		BackoffMultiplier: 2.0,
		Jitter:            false,
	}

	rm := NewDefaultRetryManager(config)

	// After a few attempts, delay should be capped at MaxDelay
	delay := rm.GetRetryDelay(5)
	if delay > config.MaxDelay {
		t.Errorf("Delay %v exceeds MaxDelay %v", delay, config.MaxDelay)
	}
	if delay != config.MaxDelay {
		t.Errorf("Expected delay to be capped at MaxDelay %v, got %v", config.MaxDelay, delay)
	}
}

func TestRetryManager_GetRetryDelay_WithJitter(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:       3,
		BaseDelay:         100 * time.Millisecond,
		MaxDelay:          1 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            true,
	}

	rm := NewDefaultRetryManager(config)

	// With jitter, delays should vary
	delay1 := rm.GetRetryDelay(1)
	delay2 := rm.GetRetryDelay(1)

	// Base delay for attempt 1 should be 200ms
	baseDelay := 200 * time.Millisecond

	// With jitter, both delays should be >= baseDelay and <= baseDelay * 1.1
	minDelay := baseDelay
	maxDelay := time.Duration(float64(baseDelay) * 1.1)

	if delay1 < minDelay || delay1 > maxDelay {
		t.Errorf("Delay1 %v out of expected range [%v, %v]", delay1, minDelay, maxDelay)
	}

	if delay2 < minDelay || delay2 > maxDelay {
		t.Errorf("Delay2 %v out of expected range [%v, %v]", delay2, minDelay, maxDelay)
	}
}

func TestNewRetryableError(t *testing.T) {
	originalErr := errors.New("original error")

	retryableErr := NewRetryableError(originalErr, true)
	if !IsRetryableError(retryableErr) {
		t.Error("Expected error to be retryable")
	}

	nonRetryableErr := NewRetryableError(originalErr, false)
	if IsRetryableError(nonRetryableErr) {
		t.Error("Expected error to be non-retryable")
	}

	// Test error unwrapping
	if !errors.Is(retryableErr, originalErr) {
		t.Error("Retryable error should unwrap to original error")
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		s          string
		substrings []string
		expected   bool
	}{
		{"hello world", []string{"hello"}, true},
		{"hello world", []string{"goodbye"}, false},
		{"hello world", []string{"hello", "goodbye"}, true},
		{"hello world", []string{"goodbye", "farewell"}, false},
		{"", []string{"hello"}, false},
		{"hello", []string{""}, true}, // empty string is contained in any string
		{"hello", []string{}, false},
	}

	for _, test := range tests {
		result := containsAny(test.s, test.substrings)
		if result != test.expected {
			t.Errorf("containsAny(%q, %v) = %v, expected %v",
				test.s, test.substrings, result, test.expected)
		}
	}
}