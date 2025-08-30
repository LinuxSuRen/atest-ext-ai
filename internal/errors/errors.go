package errors

import (
	"fmt"
	"runtime"
	"strings"
)

// ErrorCode represents different types of errors
type ErrorCode string

const (
	// Configuration errors
	ErrConfigInvalid    ErrorCode = "CONFIG_INVALID"
	ErrConfigMissing    ErrorCode = "CONFIG_MISSING"
	ErrConfigLoadFailed ErrorCode = "CONFIG_LOAD_FAILED"

	// AI service errors
	ErrAIServiceInit     ErrorCode = "AI_SERVICE_INIT"
	ErrAIServiceUnavail  ErrorCode = "AI_SERVICE_UNAVAILABLE"
	ErrAIModelNotFound   ErrorCode = "AI_MODEL_NOT_FOUND"
	ErrAIRequestFailed   ErrorCode = "AI_REQUEST_FAILED"
	ErrAIResponseInvalid ErrorCode = "AI_RESPONSE_INVALID"

	// gRPC errors
	ErrGRPCServerStart    ErrorCode = "GRPC_SERVER_START"
	ErrGRPCServerStop     ErrorCode = "GRPC_SERVER_STOP"
	ErrGRPCRequestInvalid ErrorCode = "GRPC_REQUEST_INVALID"
	ErrGRPCHandlerFailed  ErrorCode = "GRPC_HANDLER_FAILED"

	// Database errors
	ErrDatabaseConnect   ErrorCode = "DATABASE_CONNECT"
	ErrDatabaseQuery     ErrorCode = "DATABASE_QUERY"
	ErrDatabaseMigration ErrorCode = "DATABASE_MIGRATION"

	// Cache errors
	ErrCacheInit      ErrorCode = "CACHE_INIT"
	ErrCacheOperation ErrorCode = "CACHE_OPERATION"

	// General errors
	ErrInternal     ErrorCode = "INTERNAL_ERROR"
	ErrInvalidInput ErrorCode = "INVALID_INPUT"
	ErrNotFound     ErrorCode = "NOT_FOUND"
	ErrUnauthorized ErrorCode = "UNAUTHORIZED"
	ErrTimeout      ErrorCode = "TIMEOUT"
)

// AppError represents an application error with context
type AppError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    string                 `json:"details,omitempty"`
	Cause      error                  `json:"-"`
	Context    map[string]interface{} `json:"context,omitempty"`
	StackTrace string                 `json:"stack_trace,omitempty"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s - %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause
func (e *AppError) Unwrap() error {
	return e.Cause
}

// WithContext adds context to the error
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithDetails adds details to the error
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// WithCause adds the underlying cause
func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	return e
}

// NewAppError creates a new application error
func NewAppError(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StackTrace: getStackTrace(),
	}
}

// NewAppErrorWithCause creates a new application error with a cause
func NewAppErrorWithCause(code ErrorCode, message string, cause error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		Cause:      cause,
		StackTrace: getStackTrace(),
	}
}

// getStackTrace captures the current stack trace
func getStackTrace() string {
	var buf strings.Builder
	for i := 2; i < 10; i++ { // Skip getStackTrace and NewAppError
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		// Only include relevant files (skip runtime internals)
		if strings.Contains(file, "github.com/Linuxsuren/atest-ext-ai") {
			buf.WriteString(fmt.Sprintf("%s:%d\n", file, line))
		}
	}
	return buf.String()
}

// Predefined error constructors

// Configuration errors
func ErrInvalidConfig(message string) *AppError {
	return NewAppError(ErrConfigInvalid, message)
}

func ErrMissingConfig(message string) *AppError {
	return NewAppError(ErrConfigMissing, message)
}

func ErrConfigLoadFailure(cause error) *AppError {
	return NewAppErrorWithCause(ErrConfigLoadFailed, "Failed to load configuration", cause)
}

// AI service errors
func ErrAIServiceInitialization(cause error) *AppError {
	return NewAppErrorWithCause(ErrAIServiceInit, "Failed to initialize AI service", cause)
}

func ErrAIServiceUnavailable(message string) *AppError {
	return NewAppError(ErrAIServiceUnavail, message)
}

func ErrModelNotFound(modelName string) *AppError {
	return NewAppError(ErrAIModelNotFound, fmt.Sprintf("AI model not found: %s", modelName))
}

func ErrAIRequestFailure(cause error) *AppError {
	return NewAppErrorWithCause(ErrAIRequestFailed, "AI request failed", cause)
}

func ErrInvalidAIResponse(message string) *AppError {
	return NewAppError(ErrAIResponseInvalid, message)
}

// gRPC errors
func ErrGRPCServerStartFailure(cause error) *AppError {
	return NewAppErrorWithCause(ErrGRPCServerStart, "Failed to start gRPC server", cause)
}

func ErrGRPCServerStopFailure(cause error) *AppError {
	return NewAppErrorWithCause(ErrGRPCServerStop, "Failed to stop gRPC server", cause)
}

func ErrInvalidGRPCRequest(message string) *AppError {
	return NewAppError(ErrGRPCRequestInvalid, message)
}

func ErrGRPCHandlerFailure(handler string, cause error) *AppError {
	return NewAppErrorWithCause(ErrGRPCHandlerFailed, fmt.Sprintf("gRPC handler '%s' failed", handler), cause)
}

// Database errors
func ErrDatabaseConnectionFailure(cause error) *AppError {
	return NewAppErrorWithCause(ErrDatabaseConnect, "Failed to connect to database", cause)
}

func ErrDatabaseQueryFailure(query string, cause error) *AppError {
	return NewAppErrorWithCause(ErrDatabaseQuery, fmt.Sprintf("Database query failed: %s", query), cause)
}

// Cache errors
func ErrCacheInitialization(cause error) *AppError {
	return NewAppErrorWithCause(ErrCacheInit, "Failed to initialize cache", cause)
}

func ErrCacheOperationFailure(operation string, cause error) *AppError {
	return NewAppErrorWithCause(ErrCacheOperation, fmt.Sprintf("Cache operation '%s' failed", operation), cause)
}

// General errors
func ErrInternalError(message string) *AppError {
	return NewAppError(ErrInternal, message)
}

func ErrInvalidInputData(message string) *AppError {
	return NewAppError(ErrInvalidInput, message)
}

func ErrResourceNotFound(resource string) *AppError {
	return NewAppError(ErrNotFound, fmt.Sprintf("Resource not found: %s", resource))
}

func ErrUnauthorizedAccess(message string) *AppError {
	return NewAppError(ErrUnauthorized, message)
}

func ErrOperationTimeout(operation string) *AppError {
	return NewAppError(ErrTimeout, fmt.Sprintf("Operation timeout: %s", operation))
}

// Error handling utilities

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// GetAppError extracts AppError from error, returns nil if not an AppError
func GetAppError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return nil
}

// WrapError wraps a regular error as an internal AppError
func WrapError(err error, message string) *AppError {
	if err == nil {
		return nil
	}
	return NewAppErrorWithCause(ErrInternal, message, err)
}

// ErrorResponse represents an error response for API
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details string                 `json:"details,omitempty"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// ToErrorResponse converts an AppError to ErrorResponse
func (e *AppError) ToErrorResponse() *ErrorResponse {
	return &ErrorResponse{
		Error:   "true",
		Code:    string(e.Code),
		Message: e.Message,
		Details: e.Details,
		Context: e.Context,
	}
}

// ToErrorResponse converts any error to ErrorResponse
func ToErrorResponse(err error) *ErrorResponse {
	if appErr := GetAppError(err); appErr != nil {
		return appErr.ToErrorResponse()
	}

	// Handle regular errors
	return &ErrorResponse{
		Error:   "true",
		Code:    string(ErrInternal),
		Message: "Internal server error",
		Details: err.Error(),
	}
}
