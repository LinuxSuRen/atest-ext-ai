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

package local

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
)

// Client implements the AIClient interface for local models (Ollama)
type Client struct {
	config     *Config
	httpClient *http.Client
}

// Config holds local model configuration
type Config struct {
	BaseURL     string        `json:"base_url"`
	Timeout     time.Duration `json:"timeout"`
	MaxTokens   int           `json:"max_tokens"`
	Model       string        `json:"model"`
	UserAgent   string        `json:"user_agent,omitempty"`
	Temperature float64       `json:"temperature"`
}

// NewClient creates a new local model client
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Set defaults
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:11434"
	}
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 4096
	}
	if config.Model == "" {
		config.Model = "llama2"
	}
	if config.UserAgent == "" {
		config.UserAgent = "atest-ext-ai/1.0"
	}
	if config.Temperature == 0 {
		config.Temperature = 0.7
	}

	client := &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}

	return client, nil
}

// Generate executes a generation request
func (c *Client) Generate(ctx context.Context, req *interfaces.GenerateRequest) (*interfaces.GenerateResponse, error) {
	start := time.Now()

	// Build the prompt with context
	prompt := c.buildPrompt(req)

	// Build the Ollama request
	ollamaReq := &GenerateRequest{
		Model:  c.getModel(req),
		Prompt: prompt,
		Stream: req.Stream,
		Options: map[string]any{
			"temperature": c.getTemperature(req),
			"num_predict": c.getMaxTokens(req),
		},
	}

	// Make the HTTP request
	response, err := c.makeRequest(ctx, "/api/generate", ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	// Convert response
	aiResponse := &interfaces.GenerateResponse{
		Text:            response.Response,
		Model:           response.Model,
		ProcessingTime:  time.Since(start),
		RequestID:       fmt.Sprintf("ollama_%d", time.Now().UnixNano()),
		ConfidenceScore: 1.0, // Ollama doesn't provide confidence scores
		Usage: interfaces.TokenUsage{
			PromptTokens:     response.PromptEvalCount,
			CompletionTokens: response.EvalCount,
			TotalTokens:      response.PromptEvalCount + response.EvalCount,
		},
		Metadata: map[string]any{
			"done":             response.Done,
			"total_duration":   response.TotalDuration,
			"load_duration":    response.LoadDuration,
			"prompt_eval_duration": response.PromptEvalDuration,
			"eval_duration":    response.EvalDuration,
		},
	}

	return aiResponse, nil
}

// GetCapabilities returns the capabilities of the local client
func (c *Client) GetCapabilities(ctx context.Context) (*interfaces.Capabilities, error) {
	// Get available models from Ollama
	models, err := c.getAvailableModels(ctx)
	if err != nil {
		// Return default capabilities if we can't fetch models
		models = []interfaces.ModelInfo{
			{
				ID:           c.config.Model,
				Name:         c.config.Model,
				Description:  "Local model via Ollama",
				MaxTokens:    c.config.MaxTokens,
				Capabilities: []string{"text_generation", "code_generation"},
			},
		}
	}

	return &interfaces.Capabilities{
		Provider:  "local",
		MaxTokens: c.config.MaxTokens,
		Models:    models,
		Features: []interfaces.Feature{
			{
				Name:        "generation",
				Enabled:     true,
				Description: "Text generation via Ollama",
				Version:     "v1",
			},
			{
				Name:        "streaming",
				Enabled:     true,
				Description: "Streaming response support",
				Version:     "v1",
			},
			{
				Name:        "local_execution",
				Enabled:     true,
				Description: "Local model execution without external API calls",
				Version:     "v1",
			},
		},
		SupportedLanguages: []string{"en"}, // Local models typically support English primarily
		RateLimits: &interfaces.RateLimits{
			RequestsPerMinute: -1, // No rate limits for local execution
			TokensPerMinute:   -1,
			RequestsPerDay:    -1,
			TokensPerDay:      -1,
		},
	}, nil
}

// HealthCheck performs a health check
func (c *Client) HealthCheck(ctx context.Context) (*interfaces.HealthStatus, error) {
	start := time.Now()

	// Check if Ollama is running by listing models
	_, err := c.getAvailableModels(ctx)
	duration := time.Since(start)

	status := &interfaces.HealthStatus{
		ResponseTime: duration,
		LastChecked:  time.Now(),
		Metadata: map[string]any{
			"provider": "local",
			"endpoint": c.config.BaseURL,
			"model":    c.config.Model,
		},
	}

	if err != nil {
		status.Healthy = false
		status.Status = fmt.Sprintf("Health check failed: %v", err)
		status.Errors = []string{err.Error()}
	} else {
		status.Healthy = true
		status.Status = "OK"
	}

	return status, nil
}

// Close releases any resources held by the client
func (c *Client) Close() error {
	// No resources to clean up for HTTP client
	return nil
}

// buildPrompt constructs a prompt from the request
func (c *Client) buildPrompt(req *interfaces.GenerateRequest) string {
	var parts []string

	// Add system prompt if provided
	if req.SystemPrompt != "" {
		parts = append(parts, "System: "+req.SystemPrompt)
	}

	// Add context
	for i, context := range req.Context {
		parts = append(parts, fmt.Sprintf("Context %d: %s", i+1, context))
	}

	// Add the main prompt
	parts = append(parts, "User: "+req.Prompt)

	return strings.Join(parts, "\n\n")
}

// getModel returns the model to use for the request
func (c *Client) getModel(req *interfaces.GenerateRequest) string {
	if req.Model != "" {
		return req.Model
	}
	return c.config.Model
}

// getMaxTokens returns the max tokens for the request
func (c *Client) getMaxTokens(req *interfaces.GenerateRequest) int {
	if req.MaxTokens > 0 {
		return req.MaxTokens
	}
	return c.config.MaxTokens
}

// getTemperature returns the temperature for the request
func (c *Client) getTemperature(req *interfaces.GenerateRequest) float64 {
	if req.Temperature > 0 {
		return req.Temperature
	}
	return c.config.Temperature
}

// getAvailableModels retrieves the list of available models from Ollama
func (c *Client) getAvailableModels(ctx context.Context) ([]interfaces.ModelInfo, error) {
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", c.config.BaseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.config.UserAgent)

	// Make the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var tagsResp TagsResponse
	if err := json.Unmarshal(respBody, &tagsResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to AI model info
	var models []interfaces.ModelInfo
	for _, model := range tagsResp.Models {
		models = append(models, interfaces.ModelInfo{
			ID:           model.Name,
			Name:         model.Name,
			Description:  fmt.Sprintf("Local model: %s", model.Name),
			MaxTokens:    c.config.MaxTokens,
			Capabilities: []string{"text_generation"},
		})
	}

	return models, nil
}

// makeRequest makes an HTTP request to the Ollama API
func (c *Client) makeRequest(ctx context.Context, endpoint string, body interface{}) (*GenerateResponse, error) {
	// Marshal the request body
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.config.UserAgent)

	// Make the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var generateResp GenerateResponse
	if err := json.Unmarshal(respBody, &generateResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &generateResp, nil
}

// Ollama API structures

// GenerateRequest represents a generation request to Ollama
type GenerateRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	Stream  bool           `json:"stream,omitempty"`
	Options map[string]any `json:"options,omitempty"`
}

// GenerateResponse represents a generation response from Ollama
type GenerateResponse struct {
	Model               string `json:"model"`
	CreatedAt           string `json:"created_at"`
	Response            string `json:"response"`
	Done                bool   `json:"done"`
	TotalDuration       int64  `json:"total_duration"`
	LoadDuration        int64  `json:"load_duration"`
	PromptEvalCount     int    `json:"prompt_eval_count"`
	PromptEvalDuration  int64  `json:"prompt_eval_duration"`
	EvalCount           int    `json:"eval_count"`
	EvalDuration        int64  `json:"eval_duration"`
}

// TagsResponse represents the response from the tags endpoint
type TagsResponse struct {
	Models []ModelInfo `json:"models"`
}

// ModelInfo represents model information from Ollama
type ModelInfo struct {
	Name       string `json:"name"`
	ModifiedAt string `json:"modified_at"`
	Size       int64  `json:"size"`
}