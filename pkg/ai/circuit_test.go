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
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_NewDefaultCircuitBreaker(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		ResetTimeout:     30 * time.Second,
		HalfOpenMaxCalls: 2,
		SuccessThreshold: 1,
	}

	cb := NewDefaultCircuitBreaker(config)
	if cb == nil {
		t.Fatal("Circuit breaker is nil")
	}

	if cb.State() != CircuitClosed {
		t.Errorf("Initial state should be closed, got %s", cb.State().String())
	}
}

func TestCircuitBreaker_DefaultConfig(t *testing.T) {
	// Test with empty config to verify defaults
	cb := NewDefaultCircuitBreaker(CircuitBreakerConfig{})

	metrics := cb.GetMetrics()
	if metrics.State != CircuitClosed {
		t.Errorf("Initial state should be closed, got %s", metrics.State.String())
	}
}

func TestCircuitBreaker_Call_Success(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		ResetTimeout:     30 * time.Second,
		HalfOpenMaxCalls: 2,
		SuccessThreshold: 1,
	}

	cb := NewDefaultCircuitBreaker(config)

	err := cb.Call(context.Background(), func() error {
		return nil // Success
	})

	if err != nil {
		t.Errorf("Call failed: %v", err)
	}

	if cb.State() != CircuitClosed {
		t.Errorf("State should remain closed after success, got %s", cb.State().String())
	}

	metrics := cb.GetMetrics()
	if metrics.TotalRequests != 1 {
		t.Errorf("Expected 1 total request, got %d", metrics.TotalRequests)
	}
	if metrics.SuccessfulRequests != 1 {
		t.Errorf("Expected 1 successful request, got %d", metrics.SuccessfulRequests)
	}
	if metrics.FailedRequests != 0 {
		t.Errorf("Expected 0 failed requests, got %d", metrics.FailedRequests)
	}
}

func TestCircuitBreaker_Call_Failure(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		ResetTimeout:     30 * time.Second,
		HalfOpenMaxCalls: 2,
		SuccessThreshold: 1,
	}

	cb := NewDefaultCircuitBreaker(config)
	testError := errors.New("test error")

	err := cb.Call(context.Background(), func() error {
		return testError
	})

	if !errors.Is(err, testError) {
		t.Errorf("Expected test error, got: %v", err)
	}

	if cb.State() != CircuitClosed {
		t.Errorf("State should remain closed after single failure, got %s", cb.State().String())
	}

	metrics := cb.GetMetrics()
	if metrics.ConsecutiveFailures != 1 {
		t.Errorf("Expected 1 consecutive failure, got %d", metrics.ConsecutiveFailures)
	}
}

func TestCircuitBreaker_TransitionToOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 2, // Low threshold for testing
		ResetTimeout:     30 * time.Second,
		HalfOpenMaxCalls: 2,
		SuccessThreshold: 1,
	}

	cb := NewDefaultCircuitBreaker(config)
	testError := errors.New("test error")

	// Make failures to reach threshold
	for i := 0; i < config.FailureThreshold; i++ {
		err := cb.Call(context.Background(), func() error {
			return testError
		})
		if !errors.Is(err, testError) {
			t.Errorf("Expected test error, got: %v", err)
		}
	}

	// Circuit should now be open
	if cb.State() != CircuitOpen {
		t.Errorf("State should be open after %d failures, got %s", config.FailureThreshold, cb.State().String())
	}

	// Next call should be blocked
	err := cb.Call(context.Background(), func() error {
		return nil // This should not be called
	})

	if !errors.Is(err, ErrCircuitBreakerOpen) {
		t.Errorf("Expected circuit breaker open error, got: %v", err)
	}
}

func TestCircuitBreaker_TransitionToHalfOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     10 * time.Millisecond, // Short timeout for testing
		HalfOpenMaxCalls: 2,
		SuccessThreshold: 1,
	}

	cb := NewDefaultCircuitBreaker(config)
	testError := errors.New("test error")

	// Force circuit to open
	for i := 0; i < config.FailureThreshold; i++ {
		cb.Call(context.Background(), func() error {
			return testError
		})
	}

	// Wait for reset timeout
	time.Sleep(config.ResetTimeout + time.Millisecond)

	// Next call should transition to half-open
	callMade := false
	err := cb.Call(context.Background(), func() error {
		callMade = true
		return nil // Success
	})

	if err != nil {
		t.Errorf("Call failed: %v", err)
	}

	if !callMade {
		t.Error("Function should have been called in half-open state")
	}

	// Circuit should transition back to closed after successful call
	if cb.State() != CircuitClosed {
		t.Errorf("State should be closed after successful call in half-open, got %s", cb.State().String())
	}
}

func TestCircuitBreaker_HalfOpenFailure(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     10 * time.Millisecond,
		HalfOpenMaxCalls: 2,
		SuccessThreshold: 1,
	}

	cb := NewDefaultCircuitBreaker(config)
	testError := errors.New("test error")

	// Force circuit to open
	for i := 0; i < config.FailureThreshold; i++ {
		cb.Call(context.Background(), func() error {
			return testError
		})
	}

	// Wait for reset timeout
	time.Sleep(config.ResetTimeout + time.Millisecond)

	// First call in half-open fails - should transition back to open
	err := cb.Call(context.Background(), func() error {
		return testError
	})

	if !errors.Is(err, testError) {
		t.Errorf("Expected test error, got: %v", err)
	}

	// Circuit should be open again
	if cb.State() != CircuitOpen {
		t.Errorf("State should be open after failure in half-open, got %s", cb.State().String())
	}
}

func TestCircuitBreaker_HalfOpenMaxCalls(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     10 * time.Millisecond,
		HalfOpenMaxCalls: 1, // Only allow one call
		SuccessThreshold: 2, // Require 2 successes to close (more than max calls)
	}

	cb := NewDefaultCircuitBreaker(config)
	testError := errors.New("test error")

	// Force circuit to open
	for i := 0; i < config.FailureThreshold; i++ {
		cb.Call(context.Background(), func() error {
			return testError
		})
	}

	// Wait for reset timeout
	time.Sleep(config.ResetTimeout + time.Millisecond)

	// First call should succeed
	callCount := 0
	err := cb.Call(context.Background(), func() error {
		callCount++
		return nil
	})
	if err != nil {
		t.Errorf("First call failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}

	// Second call should be blocked (exceeds HalfOpenMaxCalls)
	err = cb.Call(context.Background(), func() error {
		callCount++
		return nil
	})

	if !errors.Is(err, ErrCircuitBreakerOpen) {
		t.Errorf("Expected circuit breaker open error, got: %v", err)
	}

	// Function should not have been called
	if callCount != 1 {
		t.Errorf("Expected 1 call total, got %d", callCount)
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     30 * time.Second,
		HalfOpenMaxCalls: 2,
		SuccessThreshold: 1,
	}

	cb := NewDefaultCircuitBreaker(config)
	testError := errors.New("test error")

	// Force circuit to open
	for i := 0; i < config.FailureThreshold; i++ {
		cb.Call(context.Background(), func() error {
			return testError
		})
	}

	if cb.State() != CircuitOpen {
		t.Errorf("State should be open, got %s", cb.State().String())
	}

	// Reset the circuit breaker
	cb.Reset()

	if cb.State() != CircuitClosed {
		t.Errorf("State should be closed after reset, got %s", cb.State().String())
	}

	// Should be able to make calls again
	err := cb.Call(context.Background(), func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Call failed after reset: %v", err)
	}
}

func TestCircuitBreaker_GetMetrics(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		ResetTimeout:     30 * time.Second,
		HalfOpenMaxCalls: 2,
		SuccessThreshold: 1,
	}

	cb := NewDefaultCircuitBreaker(config)
	testError := errors.New("test error")

	// Make some successful and failed calls
	cb.Call(context.Background(), func() error { return nil })
	cb.Call(context.Background(), func() error { return testError })
	cb.Call(context.Background(), func() error { return nil })
	cb.Call(context.Background(), func() error { return testError })

	metrics := cb.GetMetrics()

	if metrics.TotalRequests != 4 {
		t.Errorf("Expected 4 total requests, got %d", metrics.TotalRequests)
	}
	if metrics.SuccessfulRequests != 2 {
		t.Errorf("Expected 2 successful requests, got %d", metrics.SuccessfulRequests)
	}
	if metrics.FailedRequests != 2 {
		t.Errorf("Expected 2 failed requests, got %d", metrics.FailedRequests)
	}
	if metrics.ConsecutiveFailures != 1 {
		t.Errorf("Expected 1 consecutive failure, got %d", metrics.ConsecutiveFailures)
	}
	if metrics.State != CircuitClosed {
		t.Errorf("Expected closed state, got %s", metrics.State.String())
	}
	if metrics.LastSuccessTime == nil {
		t.Error("LastSuccessTime should not be nil")
	}
	if metrics.LastFailureTime == nil {
		t.Error("LastFailureTime should not be nil")
	}
}

func TestCircuitBreaker_GetStats(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		ResetTimeout:     30 * time.Second,
		HalfOpenMaxCalls: 2,
		SuccessThreshold: 1,
	}

	cb := NewDefaultCircuitBreaker(config)

	// Cast to access GetStats method
	if cbImpl, ok := cb.(*defaultCircuitBreaker); ok {
		stats := cbImpl.GetStats()

		if stats.TotalRequests != 0 {
			t.Errorf("Expected 0 total requests initially, got %d", stats.TotalRequests)
		}

		if stats.SuccessRate != 0.0 {
			t.Errorf("Expected 0.0 success rate initially, got %f", stats.SuccessRate)
		}

		// Make some calls
		cb.Call(context.Background(), func() error { return nil })
		cb.Call(context.Background(), func() error { return errors.New("error") })

		stats = cbImpl.GetStats()

		if stats.TotalRequests != 2 {
			t.Errorf("Expected 2 total requests, got %d", stats.TotalRequests)
		}

		expectedSuccessRate := 0.5
		if stats.SuccessRate != expectedSuccessRate {
			t.Errorf("Expected success rate %f, got %f", expectedSuccessRate, stats.SuccessRate)
		}

		expectedFailureRate := 0.5
		if stats.FailureRate != expectedFailureRate {
			t.Errorf("Expected failure rate %f, got %f", expectedFailureRate, stats.FailureRate)
		}
	} else {
		t.Error("Could not cast circuit breaker to defaultCircuitBreaker")
	}
}

func TestCircuitBreaker_IsHealthy(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     30 * time.Second,
		HalfOpenMaxCalls: 2,
		SuccessThreshold: 1,
	}

	cb := NewDefaultCircuitBreaker(config)

	// Cast to access IsHealthy method
	if cbImpl, ok := cb.(*defaultCircuitBreaker); ok {
		// Initially healthy (closed)
		if !cbImpl.IsHealthy() {
			t.Error("Circuit breaker should be healthy when closed")
		}

		// Force circuit to open
		testError := errors.New("test error")
		for i := 0; i < config.FailureThreshold; i++ {
			cb.Call(context.Background(), func() error {
				return testError
			})
		}

		// Should not be healthy when open
		if cbImpl.IsHealthy() {
			t.Error("Circuit breaker should not be healthy when open")
		}
	} else {
		t.Error("Could not cast circuit breaker to defaultCircuitBreaker")
	}
}

func TestCircuitBreaker_GetFailureRate(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		ResetTimeout:     30 * time.Second,
		HalfOpenMaxCalls: 2,
		SuccessThreshold: 1,
	}

	cb := NewDefaultCircuitBreaker(config)

	// Cast to access GetFailureRate method
	if cbImpl, ok := cb.(*defaultCircuitBreaker); ok {
		// Initially no failures
		if rate := cbImpl.GetFailureRate(); rate != 0.0 {
			t.Errorf("Expected failure rate 0.0 initially, got %f", rate)
		}

		// Make some calls: 2 successes, 1 failure
		cb.Call(context.Background(), func() error { return nil })
		cb.Call(context.Background(), func() error { return nil })
		cb.Call(context.Background(), func() error { return errors.New("error") })

		expectedRate := 1.0 / 3.0 // 1 failure out of 3 total
		if rate := cbImpl.GetFailureRate(); rate != expectedRate {
			t.Errorf("Expected failure rate %f, got %f", expectedRate, rate)
		}
	} else {
		t.Error("Could not cast circuit breaker to defaultCircuitBreaker")
	}
}
