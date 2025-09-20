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

package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
)

// Client implements the AIClient interface for OpenAI
type Client struct {
	config     *Config
	httpClient *http.Client
}

// Config holds OpenAI-specific configuration
type Config struct {
	APIKey          string        `json:"api_key"`
	BaseURL         string        `json:"base_url"`
	Timeout         time.Duration `json:"timeout"`
	MaxTokens       int           `json:"max_tokens"`
	Model           string        `json:"model"`
	OrgID           string        `json:"org_id,omitempty"`
	UserAgent       string        `json:"user_agent,omitempty"`
	MaxIdleConns    int           `json:"max_idle_conns,omitempty"`
	MaxConnsPerHost int           `json:"max_conns_per_host,omitempty"`
	IdleConnTimeout time.Duration `json:"idle_conn_timeout,omitempty"`
}

// NewClient creates a new OpenAI client
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Set API key from environment if not provided
	if config.APIKey == "" {
		if envKey := os.Getenv("OPENAI_API_KEY"); envKey != "" {
			config.APIKey = envKey
		} else {
			return nil, fmt.Errorf("API key is required (set OPENAI_API_KEY environment variable or provide in config)")
		}
	}

	// Set defaults
	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com/v1"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 4096
	}
	if config.Model == "" {
		config.Model = "gpt-3.5-turbo"
	}
	if config.UserAgent == "" {
		config.UserAgent = "atest-ext-ai/1.0"
	}
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 100
	}
	if config.MaxConnsPerHost == 0 {
		config.MaxConnsPerHost = 10
	}
	if config.IdleConnTimeout == 0 {
		config.IdleConnTimeout = 90 * time.Second
	}

	// Get organization ID from environment if not provided
	if config.OrgID == "" {
		config.OrgID = os.Getenv("OPENAI_ORG_ID")
	}

	// Create HTTP transport with connection pooling
	transport := &http.Transport{
		MaxIdleConns:        config.MaxIdleConns,
		MaxIdleConnsPerHost: config.MaxConnsPerHost,
		IdleConnTimeout:     config.IdleConnTimeout,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := &Client{
		config: config,
		httpClient: &http.Client{
			Timeout:   config.Timeout,
			Transport: transport,
		},
	}

	return client, nil
}

// Generate executes a generation request
func (c *Client) Generate(ctx context.Context, req *interfaces.GenerateRequest) (*interfaces.GenerateResponse, error) {
	start := time.Now()

	// Build the OpenAI request
	openaiReq := &ChatCompletionRequest{
		Model:       c.getModel(req),
		MaxTokens:   c.getMaxTokens(req),
		Temperature: c.getTemperature(req),
		Stream:      req.Stream,
	}

	// Build messages
	if req.SystemPrompt != "" {
		openaiReq.Messages = append(openaiReq.Messages, Message{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}

	// Add context messages
	for _, contextMsg := range req.Context {
		openaiReq.Messages = append(openaiReq.Messages, Message{
			Role:    "user",
			Content: contextMsg,
		})
	}

	// Add the main prompt
	openaiReq.Messages = append(openaiReq.Messages, Message{
		Role:    "user",
		Content: req.Prompt,
	})

	if req.Stream {
		return c.generateStream(ctx, openaiReq, start)
	}

	// Make the HTTP request for non-streaming
	response, err := c.makeRequest(ctx, "/chat/completions", openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	// Convert response
	aiResponse := &interfaces.GenerateResponse{
		Text:            response.Choices[0].Message.Content,
		Model:           response.Model,
		ProcessingTime:  time.Since(start),
		RequestID:       response.ID,
		ConfidenceScore: 1.0, // OpenAI doesn't provide confidence scores
		Usage: interfaces.TokenUsage{
			PromptTokens:     response.Usage.PromptTokens,
			CompletionTokens: response.Usage.CompletionTokens,
			TotalTokens:      response.Usage.TotalTokens,
		},
		Metadata: map[string]any{
			"finish_reason": response.Choices[0].FinishReason,
			"created":       response.Created,
			"streaming":     false,
		},
	}

	return aiResponse, nil
}

// GetCapabilities returns the capabilities of the OpenAI client
func (c *Client) GetCapabilities(ctx context.Context) (*interfaces.Capabilities, error) {
	return &interfaces.Capabilities{
		Provider:  "openai",
		MaxTokens: c.config.MaxTokens,
		Models: []interfaces.ModelInfo{
			{
				ID:              "gpt-4",
				Name:            "GPT-4",
				Description:     "Most capable GPT-4 model",
				MaxTokens:       8192,
				InputCostPer1K:  0.03,
				OutputCostPer1K: 0.06,
				Capabilities:    []string{"text_generation", "code_generation", "analysis"},
			},
			{
				ID:              "gpt-4-turbo",
				Name:            "GPT-4 Turbo",
				Description:     "Latest GPT-4 model with improved performance",
				MaxTokens:       128000,
				InputCostPer1K:  0.01,
				OutputCostPer1K: 0.03,
				Capabilities:    []string{"text_generation", "code_generation", "analysis", "long_context"},
			},
			{
				ID:              "gpt-3.5-turbo",
				Name:            "GPT-3.5 Turbo",
				Description:     "Fast and efficient GPT-3.5 model",
				MaxTokens:       4096,
				InputCostPer1K:  0.0015,
				OutputCostPer1K: 0.002,
				Capabilities:    []string{"text_generation", "code_generation"},
			},
		},
		Features: []interfaces.Feature{
			{
				Name:        "chat_completions",
				Enabled:     true,
				Description: "Chat-based text generation",
				Version:     "v1",
			},
			{
				Name:        "streaming",
				Enabled:     true,
				Description: "Streaming response support",
				Version:     "v1",
			},
		},
		SupportedLanguages: []string{"en", "es", "fr", "de", "it", "pt", "ru", "ja", "ko", "zh"},
		RateLimits: &interfaces.RateLimits{
			RequestsPerMinute: 3500,
			TokensPerMinute:   90000,
			RequestsPerDay:    -1, // No daily limit
			TokensPerDay:      -1, // No daily limit
		},
	}, nil
}

// HealthCheck performs a health check
func (c *Client) HealthCheck(ctx context.Context) (*interfaces.HealthStatus, error) {
	start := time.Now()

	// Make a simple request to check if the service is available
	req := &interfaces.GenerateRequest{
		Prompt:    "Hello",
		MaxTokens: 1,
	}

	_, err := c.Generate(ctx, req)
	duration := time.Since(start)

	status := &interfaces.HealthStatus{
		ResponseTime: duration,
		LastChecked:  time.Now(),
		Metadata: map[string]any{
			"provider": "openai",
			"endpoint": c.config.BaseURL,
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
	// Close idle connections in the transport
	if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
	return nil
}

// generateStream handles streaming generation requests
func (c *Client) generateStream(ctx context.Context, openaiReq *ChatCompletionRequest, start time.Time) (*interfaces.GenerateResponse, error) {
	// Marshal the request body
	jsonBody, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("User-Agent", c.config.UserAgent)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	if c.config.OrgID != "" {
		req.Header.Set("OpenAI-Organization", c.config.OrgID)
	}

	// Make the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		var errorResp ErrorResponse
		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			return nil, fmt.Errorf("API error: %s", errorResp.Error.Message)
		}
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Read streaming response
	var responseText strings.Builder
	var model string
	var finishReason string
	var requestID string
	var totalTokens, promptTokens, completionTokens int
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var streamResp StreamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			continue // Skip malformed lines
		}

		if len(streamResp.Choices) > 0 {
			choice := streamResp.Choices[0]
			if choice.Delta.Content != "" {
				responseText.WriteString(choice.Delta.Content)
			}
			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
			}
		}

		if streamResp.Model != "" {
			model = streamResp.Model
		}
		if streamResp.ID != "" {
			requestID = streamResp.ID
		}

		// Token usage is typically only provided in the final message
		if streamResp.Usage != nil {
			totalTokens = streamResp.Usage.TotalTokens
			promptTokens = streamResp.Usage.PromptTokens
			completionTokens = streamResp.Usage.CompletionTokens
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading stream: %w", err)
	}

	// Build the final response
	aiResponse := &interfaces.GenerateResponse{
		Text:            responseText.String(),
		Model:           model,
		ProcessingTime:  time.Since(start),
		RequestID:       requestID,
		ConfidenceScore: 1.0,
		Usage: interfaces.TokenUsage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      totalTokens,
		},
		Metadata: map[string]any{
			"finish_reason": finishReason,
			"streaming":     true,
		},
	}

	return aiResponse, nil
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
	return 0.7 // Default temperature
}

// makeRequest makes an HTTP request to the OpenAI API
func (c *Client) makeRequest(ctx context.Context, endpoint string, body interface{}) (*ChatCompletionResponse, error) {
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
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("User-Agent", c.config.UserAgent)

	if c.config.OrgID != "" {
		req.Header.Set("OpenAI-Organization", c.config.OrgID)
	}

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
		var errorResp ErrorResponse
		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			return nil, fmt.Errorf("API error: %s", errorResp.Error.Message)
		}
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &chatResp, nil
}

// OpenAI API structures

// ChatCompletionRequest represents a chat completion request
type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionResponse represents a chat completion response
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// StreamResponse represents a streaming response from OpenAI
type StreamResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
	Usage   *Usage         `json:"usage,omitempty"`
}

// StreamChoice represents a choice in a streaming response
type StreamChoice struct {
	Index        int         `json:"index"`
	Delta        StreamDelta `json:"delta"`
	FinishReason string      `json:"finish_reason"`
}

// StreamDelta represents the delta content in a streaming response
type StreamDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}
