package models

import (
	"encoding/json"
	"time"
)

// AIRequest represents the request structure for AI operations
type AIRequest struct {
	Action    string                 `json:"action"`     // "convert_to_sql", "ping", etc.
	Data      map[string]interface{} `json:"data"`       // Request-specific data
	RequestID string                 `json:"request_id"` // Unique request identifier
	Timestamp time.Time              `json:"timestamp"`  // Request timestamp
}

// AIResponse represents the response structure for AI operations
type AIResponse struct {
	Success   bool                   `json:"success"`    // Operation success status
	Data      map[string]interface{} `json:"data"`       // Response data
	Message   string                 `json:"message"`    // Response message or error description
	RequestID string                 `json:"request_id"` // Matching request identifier
	Timestamp time.Time              `json:"timestamp"`  // Response timestamp
}

// SQLConversionRequest represents a request to convert natural language to SQL
type SQLConversionRequest struct {
	Query       string            `json:"query"`        // Natural language query
	Context     string            `json:"context"`      // Database context or schema info
	TableSchema map[string]string `json:"table_schema"` // Table schema information
	Dialect     string            `json:"dialect"`      // SQL dialect (mysql, postgresql, etc.)
}

// SQLConversionResponse represents the response for SQL conversion
type SQLConversionResponse struct {
	SQL         string   `json:"sql"`         // Generated SQL query
	Success     bool     `json:"success"`     // Operation success status
	Confidence  float64  `json:"confidence"`  // Confidence score (0-1)
	Explanation string   `json:"explanation"` // Explanation of the generated SQL
	Warnings    []string `json:"warnings"`    // Any warnings about the conversion
	Model       string   `json:"model"`       // AI model used
	Provider    string   `json:"provider"`    // AI provider used
}

// HealthCheckRequest represents a health check request
type HealthCheckRequest struct {
	Service string `json:"service"` // Service to check ("ai", "database", "all")
}

// HealthCheckResponse represents a health check response
type HealthCheckResponse struct {
	Status   string            `json:"status"`   // "healthy", "unhealthy", "degraded"
	Services map[string]string `json:"services"` // Status of individual services
	Uptime   time.Duration     `json:"uptime"`   // Service uptime
	Version  string            `json:"version"`  // Service version
}

// ModelInfo represents information about an AI model
type ModelInfo struct {
	Name         string            `json:"name"`         // Model name
	Provider     string            `json:"provider"`     // Model provider
	Version      string            `json:"version"`      // Model version
	Capabilities []string          `json:"capabilities"` // Model capabilities
	Limits       map[string]int    `json:"limits"`       // Model limits (tokens, etc.)
	Metadata     map[string]string `json:"metadata"`     // Additional metadata
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Code      string    `json:"code"`       // Error code
	Message   string    `json:"message"`    // Error message
	Details   string    `json:"details"`    // Error details
	Timestamp time.Time `json:"timestamp"`  // Error timestamp
	RequestID string    `json:"request_id"` // Associated request ID
}

// ToJSON converts a struct to JSON string
func (r *AIRequest) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	return string(data), err
}

// FromJSON creates AIRequest from JSON string
func (r *AIRequest) FromJSON(jsonStr string) error {
	return json.Unmarshal([]byte(jsonStr), r)
}

// ToJSON converts a struct to JSON string
func (r *AIResponse) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	return string(data), err
}

// FromJSON creates AIResponse from JSON string
func (r *AIResponse) FromJSON(jsonStr string) error {
	return json.Unmarshal([]byte(jsonStr), r)
}

// NewAIRequest creates a new AI request with default values
func NewAIRequest(action string, data map[string]interface{}) *AIRequest {
	return &AIRequest{
		Action:    action,
		Data:      data,
		RequestID: generateRequestID(),
		Timestamp: time.Now(),
	}
}

// NewAIResponse creates a new AI response with default values
func NewAIResponse(success bool, data map[string]interface{}, message, requestID string) *AIResponse {
	return &AIResponse{
		Success:   success,
		Data:      data,
		Message:   message,
		RequestID: requestID,
		Timestamp: time.Now(),
	}
}

// NewErrorResponse creates a new error response
func NewErrorResponse(code, message, details, requestID string) *ErrorResponse {
	return &ErrorResponse{
		Code:      code,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
		RequestID: requestID,
	}
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	// Simple implementation - in production, use UUID or similar
	return time.Now().Format("20060102150405") + "-" + time.Now().Format("000")
}
