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

package errors

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Standard errors that can be used across the application
// These follow the Uber Go Style Guide pattern of declaring sentinel errors
var (
	// ErrProviderNotConfigured indicates that no AI provider is configured or enabled
	ErrProviderNotConfigured = errors.New("AI provider not configured")

	// ErrModelNotFound indicates that the requested model does not exist
	ErrModelNotFound = errors.New("model not found")

	// ErrInvalidConfig indicates that configuration validation failed
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrConnectionFailed indicates that connection to AI service failed
	ErrConnectionFailed = errors.New("connection failed")

	// ErrProviderNotAvailable indicates that the provider exists but is not currently available
	ErrProviderNotAvailable = errors.New("provider not available")

	// ErrInvalidRequest indicates that the request parameters are invalid
	ErrInvalidRequest = errors.New("invalid request")

	// ErrTimeout indicates that an operation timed out
	ErrTimeout = errors.New("operation timed out")

	// ErrResourceExhausted indicates that rate limits or quotas have been exceeded
	ErrResourceExhausted = errors.New("resource exhausted")
)

// ToGRPCError converts internal application errors to gRPC status errors
// with appropriate error codes.
//
// This function implements the error mapping strategy defined in docs/ERROR_HANDLING.md
//
// Usage:
//
//	if err := doSomething(); err != nil {
//	    return ToGRPCError(err)
//	}
func ToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	// Map specific errors to appropriate gRPC codes
	switch {
	case errors.Is(err, ErrProviderNotConfigured):
		return status.Error(codes.FailedPrecondition, err.Error())

	case errors.Is(err, ErrModelNotFound):
		return status.Error(codes.NotFound, err.Error())

	case errors.Is(err, ErrInvalidConfig), errors.Is(err, ErrInvalidRequest):
		return status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, ErrConnectionFailed), errors.Is(err, ErrProviderNotAvailable):
		return status.Error(codes.Unavailable, err.Error())

	case errors.Is(err, ErrTimeout):
		return status.Error(codes.DeadlineExceeded, err.Error())

	case errors.Is(err, ErrResourceExhausted):
		return status.Error(codes.ResourceExhausted, err.Error())

	default:
		// For unknown errors, return as Internal error
		return status.Error(codes.Internal, err.Error())
	}
}

// ToGRPCErrorf is a convenience function that wraps fmt.Errorf and ToGRPCError
//
// Usage:
//
//	return ToGRPCErrorf(ErrModelNotFound, "model %s not found in provider %s", modelID, provider)
func ToGRPCErrorf(err error, format string, args ...interface{}) error {
	wrapped := fmt.Errorf(format+": %w", append(args, err)...)
	return ToGRPCError(wrapped)
}

// IsRetryable checks if an error indicates a retryable condition
//
// Usage:
//
//	if err != nil && IsRetryable(err) {
//	    // Retry the operation
//	}
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific retryable errors
	switch {
	case errors.Is(err, ErrConnectionFailed):
		return true
	case errors.Is(err, ErrProviderNotAvailable):
		return true
	case errors.Is(err, ErrTimeout):
		return true
	case errors.Is(err, ErrResourceExhausted):
		return true
	default:
		// Check gRPC status codes for retryable conditions
		if st, ok := status.FromError(err); ok {
			code := st.Code()
			return code == codes.Unavailable ||
				code == codes.DeadlineExceeded ||
				code == codes.ResourceExhausted ||
				code == codes.Aborted
		}
		return false
	}
}

// ValidationError represents a configuration or request validation error
// with details about which field failed validation
type ValidationError struct {
	Field   string // The field that failed validation
	Value   string // The invalid value (if safe to include)
	Message string // Human-readable error message
}

func (e *ValidationError) Error() string {
	if e.Value != "" {
		return fmt.Sprintf("validation failed for field %q with value %q: %s", e.Field, e.Value, e.Message)
	}
	return fmt.Sprintf("validation failed for field %q: %s", e.Field, e.Message)
}

// NewValidationError creates a new ValidationError
func NewValidationError(field, value, message string) error {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// ConnectionError wraps connection-related errors with additional context
type ConnectionError struct {
	Provider string // AI provider name
	Endpoint string // Connection endpoint
	Err      error  // Underlying error
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("connection to %s (%s) failed: %v", e.Provider, e.Endpoint, e.Err)
}

func (e *ConnectionError) Unwrap() error {
	return e.Err
}

// NewConnectionError creates a new ConnectionError
func NewConnectionError(provider, endpoint string, err error) error {
	return &ConnectionError{
		Provider: provider,
		Endpoint: endpoint,
		Err:      err,
	}
}
