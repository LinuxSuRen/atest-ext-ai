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
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestToGRPCError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode codes.Code
	}{
		{
			name:         "nil error",
			err:          nil,
			expectedCode: codes.OK,
		},
		{
			name:         "provider not configured",
			err:          ErrProviderNotConfigured,
			expectedCode: codes.FailedPrecondition,
		},
		{
			name:         "model not found",
			err:          ErrModelNotFound,
			expectedCode: codes.NotFound,
		},
		{
			name:         "invalid config",
			err:          ErrInvalidConfig,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "invalid request",
			err:          ErrInvalidRequest,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "connection failed",
			err:          ErrConnectionFailed,
			expectedCode: codes.Unavailable,
		},
		{
			name:         "provider not available",
			err:          ErrProviderNotAvailable,
			expectedCode: codes.Unavailable,
		},
		{
			name:         "timeout",
			err:          ErrTimeout,
			expectedCode: codes.DeadlineExceeded,
		},
		{
			name:         "resource exhausted",
			err:          ErrResourceExhausted,
			expectedCode: codes.ResourceExhausted,
		},
		{
			name:         "unknown error",
			err:          errors.New("some unknown error"),
			expectedCode: codes.Internal,
		},
		{
			name:         "wrapped error",
			err:          fmt.Errorf("outer: %w", ErrModelNotFound),
			expectedCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grpcErr := ToGRPCError(tt.err)

			if tt.err == nil {
				if grpcErr != nil {
					t.Errorf("expected nil error, got %v", grpcErr)
				}
				return
			}

			st, ok := status.FromError(grpcErr)
			if !ok {
				t.Fatalf("expected gRPC status error, got %T", grpcErr)
			}

			if st.Code() != tt.expectedCode {
				t.Errorf("expected code %v, got %v", tt.expectedCode, st.Code())
			}
		})
	}
}

func TestToGRPCErrorf(t *testing.T) {
	err := ToGRPCErrorf(ErrModelNotFound, "model %s not found in provider %s", "gpt-4", "openai")

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T", err)
	}

	if st.Code() != codes.NotFound {
		t.Errorf("expected code NotFound, got %v", st.Code())
	}

	expectedMsg := "model gpt-4 not found in provider openai: model not found"
	if st.Message() != expectedMsg {
		t.Errorf("expected message %q, got %q", expectedMsg, st.Message())
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "nil error",
			err:       nil,
			retryable: false,
		},
		{
			name:      "connection failed",
			err:       ErrConnectionFailed,
			retryable: true,
		},
		{
			name:      "provider not available",
			err:       ErrProviderNotAvailable,
			retryable: true,
		},
		{
			name:      "timeout",
			err:       ErrTimeout,
			retryable: true,
		},
		{
			name:      "resource exhausted",
			err:       ErrResourceExhausted,
			retryable: true,
		},
		{
			name:      "invalid config",
			err:       ErrInvalidConfig,
			retryable: false,
		},
		{
			name:      "model not found",
			err:       ErrModelNotFound,
			retryable: false,
		},
		{
			name:      "gRPC unavailable",
			err:       status.Error(codes.Unavailable, "service unavailable"),
			retryable: true,
		},
		{
			name:      "gRPC deadline exceeded",
			err:       status.Error(codes.DeadlineExceeded, "timeout"),
			retryable: true,
		},
		{
			name:      "gRPC invalid argument",
			err:       status.Error(codes.InvalidArgument, "bad request"),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryable(tt.err)
			if result != tt.retryable {
				t.Errorf("expected retryable=%v, got %v", tt.retryable, result)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := NewValidationError("model", "invalid-model", "model name contains invalid characters")

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}

	if ve.Field != "model" {
		t.Errorf("expected field 'model', got %q", ve.Field)
	}

	expectedMsg := `validation failed for field "model" with value "invalid-model": model name contains invalid characters`
	if ve.Error() != expectedMsg {
		t.Errorf("expected message %q, got %q", expectedMsg, ve.Error())
	}
}

func TestConnectionError(t *testing.T) {
	underlying := errors.New("dial tcp: connection refused")
	err := NewConnectionError("ollama", "http://localhost:11434", underlying)

	var ce *ConnectionError
	if !errors.As(err, &ce) {
		t.Fatalf("expected *ConnectionError, got %T", err)
	}

	if ce.Provider != "ollama" {
		t.Errorf("expected provider 'ollama', got %q", ce.Provider)
	}

	// Test Unwrap
	if !errors.Is(err, underlying) {
		t.Error("expected ConnectionError to wrap underlying error")
	}

	expectedMsg := "connection to ollama (http://localhost:11434) failed: dial tcp: connection refused"
	if ce.Error() != expectedMsg {
		t.Errorf("expected message %q, got %q", expectedMsg, ce.Error())
	}
}
