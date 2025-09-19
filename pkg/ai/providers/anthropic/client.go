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

package anthropic

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

// Client implements the AIClient interface for Anthropic Claude
type Client struct {
	config     *Config
	httpClient *http.Client
}

// Config holds Anthropic-specific configuration
type Config struct {
	APIKey          string        `json:"api_key"`
	BaseURL         string        `json:"base_url"`
	Timeout         time.Duration `json:"timeout"`
	MaxTokens       int           `json:"max_tokens"`
	Model           string        `json:"model"`
	Version         string        `json:"version"`
	UserAgent       string        `json:"user_agent,omitempty"`
	MaxIdleConns    int           `json:"max_idle_conns,omitempty"`
	MaxConnsPerHost int           `json:"max_conns_per_host,omitempty"`
	IdleConnTimeout time.Duration `json:"idle_conn_timeout,omitempty"`
}

// NewClient creates a new Anthropic client
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Set API key from environment if not provided
	if config.APIKey == "" {
		if envKey := os.Getenv("ANTHROPIC_API_KEY"); envKey != "" {
			config.APIKey = envKey
		} else {
			return nil, fmt.Errorf("API key is required (set ANTHROPIC_API_KEY environment variable or provide in config)")
		}
	}

	// Set defaults
	if config.BaseURL == "" {
		config.BaseURL = "https://api.anthropic.com"
	}
	if config.Timeout == 0 {
		config.Timeout = 45 * time.Second
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 4096
	}
	if config.Model == "" {
		config.Model = "claude-3-sonnet-20240229"
	}
	if config.Version == "" {
		config.Version = "2023-06-01"
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

	// Build the Claude request
	claudeReq := &MessagesRequest{
		Model:     c.getModel(req),
		MaxTokens: c.getMaxTokens(req),
		Stream:    req.Stream,
	}

	// Set temperature if provided
	if req.Temperature > 0 {
		claudeReq.Temperature = &req.Temperature
	}

	// Set system prompt if provided
	if req.SystemPrompt != "" {
		claudeReq.System = req.SystemPrompt
	}

	// Build messages - Claude uses a different format than OpenAI
	// Context messages are treated as conversation history
	for i, contextMsg := range req.Context {
		role := "user"
		if i%2 == 1 { // Alternate between user and assistant
			role = "assistant"
		}
		claudeReq.Messages = append(claudeReq.Messages, Message{
			Role:    role,
			Content: contextMsg,
		})
	}

	// Add the main prompt as user message
	claudeReq.Messages = append(claudeReq.Messages, Message{
		Role:    "user",
		Content: req.Prompt,
	})

	if req.Stream {
		return c.generateStream(ctx, claudeReq, start)
	}

	// Make the HTTP request for non-streaming
	response, err := c.makeRequest(ctx, "/v1/messages", claudeReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	// Extract content from response
	var text string
	if len(response.Content) > 0 {
		text = response.Content[0].Text
	}

	// Convert response
	aiResponse := &interfaces.GenerateResponse{
		Text:            text,
		Model:           response.Model,
		ProcessingTime:  time.Since(start),
		RequestID:       response.ID,
		ConfidenceScore: 1.0, // Claude doesn't provide confidence scores
		Usage: interfaces.TokenUsage{
			PromptTokens:     response.Usage.InputTokens,
			CompletionTokens: response.Usage.OutputTokens,
			TotalTokens:      response.Usage.InputTokens + response.Usage.OutputTokens,
		},
		Metadata: map[string]any{
			"stop_reason": response.StopReason,
			"role":        response.Role,
			"streaming":   false,
		},
	}

	return aiResponse, nil
}

// GetCapabilities returns the capabilities of the Claude client
func (c *Client) GetCapabilities(ctx context.Context) (*interfaces.Capabilities, error) {
	return &interfaces.Capabilities{
		Provider:  "anthropic",
		MaxTokens: c.config.MaxTokens,
		Models: []interfaces.ModelInfo{
			{
				ID:               "claude-3-opus-20240229",
				Name:             "Claude 3 Opus",
				Description:      "Most powerful Claude 3 model for complex tasks",
				MaxTokens:        200000,
				InputCostPer1K:   15.0,
				OutputCostPer1K:  75.0,
				Capabilities:     []string{"text_generation", "code_generation", "analysis", "long_context", "multimodal"},
			},
			{
				ID:               "claude-3-sonnet-20240229",
				Name:             "Claude 3 Sonnet",
				Description:      "Balanced Claude 3 model for general use",
				MaxTokens:        200000,
				InputCostPer1K:   3.0,
				OutputCostPer1K:  15.0,
				Capabilities:     []string{"text_generation", "code_generation", "analysis", "long_context"},
			},
			{
				ID:               "claude-3-haiku-20240307",
				Name:             "Claude 3 Haiku",
				Description:      "Fast and efficient Claude 3 model",
				MaxTokens:        200000,
				InputCostPer1K:   0.25,
				OutputCostPer1K:  1.25,
				Capabilities:     []string{"text_generation", "code_generation", "long_context"},
			},
		},
		Features: []interfaces.Feature{
			{
				Name:        "messages",
				Enabled:     true,
				Description: "Claude Messages API",
				Version:     "v1",
			},
			{
				Name:        "streaming",
				Enabled:     true,
				Description: "Streaming response support",
				Version:     "v1",
			},
			{
				Name:        "long_context",
				Enabled:     true,
				Description: "Support for very long context windows",
				Version:     "v1",
			},
		},
		SupportedLanguages: []string{"en", "es", "fr", "de", "it", "pt", "ru", "ja", "ko", "zh"},
		RateLimits: &interfaces.RateLimits{
			RequestsPerMinute: 1000,
			TokensPerMinute:   40000,
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
			"provider": "anthropic",
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
func (c *Client) generateStream(ctx context.Context, claudeReq *MessagesRequest, start time.Time) (*interfaces.GenerateResponse, error) {
	// Marshal the request body
	jsonBody, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/v1/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.config.APIKey)
	req.Header.Set("anthropic-version", c.config.Version)
	req.Header.Set("User-Agent", c.config.UserAgent)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

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
	var stopReason string
	var requestID string
	var inputTokens, outputTokens int
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if strings.HasPrefix(data, "[DONE]") {
			break
		}

		var streamResp StreamEvent
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			continue // Skip malformed lines
		}

		switch streamResp.Type {
		case "message_start":
			if streamResp.Message != nil {
				model = streamResp.Message.Model
				requestID = streamResp.Message.ID
				if streamResp.Message.Usage.InputTokens > 0 {
					inputTokens = streamResp.Message.Usage.InputTokens
				}
			}
		case "content_block_delta":
			if streamResp.Delta != nil && streamResp.Delta.Text != "" {
				responseText.WriteString(streamResp.Delta.Text)
			}
		case "message_delta":
			if streamResp.Delta != nil && streamResp.Delta.StopReason != "" {
				stopReason = streamResp.Delta.StopReason
			}
			if streamResp.Usage != nil && streamResp.Usage.OutputTokens > 0 {
				outputTokens = streamResp.Usage.OutputTokens
			}
		case "message_stop":
			break
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
			PromptTokens:     inputTokens,
			CompletionTokens: outputTokens,
			TotalTokens:      inputTokens + outputTokens,
		},
		Metadata: map[string]any{
			"stop_reason": stopReason,
			"streaming":   true,
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

// makeRequest makes an HTTP request to the Claude API
func (c *Client) makeRequest(ctx context.Context, endpoint string, body interface{}) (*MessagesResponse, error) {
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
	req.Header.Set("x-api-key", c.config.APIKey)
	req.Header.Set("anthropic-version", c.config.Version)
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
		var errorResp ErrorResponse
		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			return nil, fmt.Errorf("API error: %s", errorResp.Error.Message)
		}
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var messagesResp MessagesResponse
	if err := json.Unmarshal(respBody, &messagesResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &messagesResp, nil
}

// Claude API structures

// MessagesRequest represents a messages request
type MessagesRequest struct {
	Model       string     `json:"model"`
	MaxTokens   int        `json:"max_tokens"`
	Messages    []Message  `json:"messages"`
	System      string     `json:"system,omitempty"`
	Temperature *float64   `json:"temperature,omitempty"`
	Stream      bool       `json:"stream,omitempty"`
}

// Message represents a conversation message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// MessagesResponse represents a messages response
type MessagesResponse struct {
	ID         string           `json:"id"`
	Type       string           `json:"type"`
	Role       string           `json:"role"`
	Content    []ContentBlock   `json:"content"`
	Model      string           `json:"model"`
	StopReason string           `json:"stop_reason"`
	Usage      UsageInfo        `json:"usage"`
}

// ContentBlock represents a piece of content in the response
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// UsageInfo represents token usage information
type UsageInfo struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// StreamEvent represents a streaming event from Anthropic
type StreamEvent struct {
	Type    string                 `json:"type"`
	Message *MessagesResponse      `json:"message,omitempty"`
	Delta   *StreamDelta          `json:"delta,omitempty"`
	Usage   *UsageInfo            `json:"usage,omitempty"`
}

// StreamDelta represents the delta content in a streaming response
type StreamDelta struct {
	Type       string `json:"type,omitempty"`
	Text       string `json:"text,omitempty"`
	StopReason string `json:"stop_reason,omitempty"`
}