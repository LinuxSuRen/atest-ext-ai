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
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// Client implements the AIClient interface for OpenAI
type Client struct {
	config *Config
	llm    *openai.LLM
}

// Config holds OpenAI-specific configuration
type Config struct {
	APIKey    string        `json:"api_key"`
	BaseURL   string        `json:"base_url"`
	Timeout   time.Duration `json:"timeout"`
	MaxTokens int           `json:"max_tokens"`
	Model     string        `json:"model"`
	OrgID     string        `json:"org_id,omitempty"`

	// Legacy fields for backward compatibility
	UserAgent       string        `json:"user_agent,omitempty"`
	MaxIdleConns    int           `json:"max_idle_conns,omitempty"`
	MaxConnsPerHost int           `json:"max_conns_per_host,omitempty"`
	IdleConnTimeout time.Duration `json:"idle_conn_timeout,omitempty"`
}

// NewClient creates a new OpenAI client using langchaingo
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Set API key from environment if not provided
	if config.APIKey == "" {
		// Try standardized environment variable first, with fallback for compatibility
		if envKey := os.Getenv("ATEST_EXT_AI_OPENAI_API_KEY"); envKey != "" {
			config.APIKey = envKey
		} else if envKey := os.Getenv("OPENAI_API_KEY"); envKey != "" {
			// Legacy compatibility
			config.APIKey = envKey
		} else {
			return nil, fmt.Errorf("API key is required (set ATEST_EXT_AI_OPENAI_API_KEY or OPENAI_API_KEY environment variable or provide in config)")
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

	// Get organization ID from environment if not provided
	if config.OrgID == "" {
		config.OrgID = os.Getenv("OPENAI_ORG_ID")
	}

	// Build langchaingo options
	opts := []openai.Option{
		openai.WithToken(config.APIKey),
		openai.WithModel(config.Model),
	}

	// Add optional configurations
	if config.BaseURL != "" && config.BaseURL != "https://api.openai.com/v1" {
		opts = append(opts, openai.WithBaseURL(config.BaseURL))
	}
	if config.OrgID != "" {
		opts = append(opts, openai.WithOrganization(config.OrgID))
	}

	// Create langchaingo OpenAI LLM
	llm, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI LLM: %w", err)
	}

	client := &Client{
		config: config,
		llm:    llm,
	}

	return client, nil
}

// Generate executes a generation request using langchaingo
func (c *Client) Generate(ctx context.Context, req *interfaces.GenerateRequest) (*interfaces.GenerateResponse, error) {
	start := time.Now()

	// Apply timeout if configured
	if c.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.config.Timeout)
		defer cancel()
	}

	// Build messages in proper MessageContent format
	messages := c.buildMessages(req)

	// Build generation options
	opts := c.buildGenerationOptions(req)

	var responseText string
	var requestID string
	var err error

	if req.Stream {
		// Handle streaming
		responseText, requestID, err = c.generateStream(ctx, messages, opts)
	} else {
		// Non-streaming generation using GenerateContent
		responseText, requestID, err = c.generateContent(ctx, messages, opts)
	}

	if err != nil {
		return nil, fmt.Errorf("generation failed: %w", err)
	}

	// Build response
	aiResponse := &interfaces.GenerateResponse{
		Text:            responseText,
		Model:           c.getModel(req),
		ProcessingTime:  time.Since(start),
		RequestID:       requestID,
		ConfidenceScore: 1.0,
		Metadata: map[string]any{
			"streaming": req.Stream,
			"provider":  "openai",
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
	// No resources to clean up with langchaingo
	return nil
}

// buildMessages constructs chat messages from the request in proper MessageContent format
func (c *Client) buildMessages(req *interfaces.GenerateRequest) []llms.MessageContent {
	var messages []llms.MessageContent

	// Add system prompt if provided
	if req.SystemPrompt != "" {
		messages = append(messages, llms.TextParts(llms.ChatMessageTypeSystem, req.SystemPrompt))
	}

	// Add context messages as alternating user/assistant messages
	for i, contextMsg := range req.Context {
		role := llms.ChatMessageTypeHuman
		if i%2 == 1 {
			// Alternate between user and assistant for conversation context
			role = llms.ChatMessageTypeAI
		}
		messages = append(messages, llms.TextParts(role, contextMsg))
	}

	// Add the main prompt as user message
	messages = append(messages, llms.TextParts(llms.ChatMessageTypeHuman, req.Prompt))

	return messages
}

// buildGenerationOptions constructs generation options from the request
func (c *Client) buildGenerationOptions(req *interfaces.GenerateRequest) []llms.CallOption {
	opts := []llms.CallOption{}

	// Set max tokens
	maxTokens := c.getMaxTokens(req)
	if maxTokens > 0 {
		opts = append(opts, llms.WithMaxTokens(maxTokens))
	}

	// Set model if specified in request
	if req.Model != "" {
		opts = append(opts, llms.WithModel(req.Model))
	}

	return opts
}

// generateContent handles non-streaming generation using GenerateContent
func (c *Client) generateContent(ctx context.Context, messages []llms.MessageContent, opts []llms.CallOption) (string, string, error) {
	resp, err := c.llm.GenerateContent(ctx, messages, opts...)
	if err != nil {
		return "", "", fmt.Errorf("GenerateContent failed: %w", err)
	}

	// Extract response text and metadata
	var responseText string
	var requestID string

	if len(resp.Choices) > 0 {
		responseText = resp.Choices[0].Content

		// Try to extract request ID from generation info
		if genInfo := resp.Choices[0].GenerationInfo; genInfo != nil {
			if id, ok := genInfo["request_id"].(string); ok {
				requestID = id
			} else if id, ok := genInfo["RequestID"].(string); ok {
				requestID = id
			}
		}
	}

	return responseText, requestID, nil
}

// generateStream handles streaming generation using langchaingo
func (c *Client) generateStream(ctx context.Context, messages []llms.MessageContent, opts []llms.CallOption) (string, string, error) {
	var responseText strings.Builder

	// Add streaming callback
	streamingFunc := func(ctx context.Context, chunk []byte) error {
		responseText.Write(chunk)
		return nil
	}

	opts = append(opts, llms.WithStreamingFunc(streamingFunc))

	// Call GenerateContent with streaming enabled
	resp, err := c.llm.GenerateContent(ctx, messages, opts...)
	if err != nil {
		return "", "", fmt.Errorf("streaming generation failed: %w", err)
	}

	// Extract request ID if available
	var requestID string
	if len(resp.Choices) > 0 {
		if genInfo := resp.Choices[0].GenerationInfo; genInfo != nil {
			if id, ok := genInfo["request_id"].(string); ok {
				requestID = id
			} else if id, ok := genInfo["RequestID"].(string); ok {
				requestID = id
			}
		}
	}

	return responseText.String(), requestID, nil
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
