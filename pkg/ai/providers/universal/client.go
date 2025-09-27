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

package universal

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

// UniversalClient implements a universal OpenAI-compatible API client
type UniversalClient struct {
	config     *Config
	httpClient *http.Client
}

// Config holds configuration for the universal client
type Config struct {
	Provider        string                 `json:"provider"`          // Provider name (e.g., "ollama", "openai", "custom")
	Endpoint        string                 `json:"endpoint"`          // API endpoint URL
	APIKey          string                 `json:"api_key,omitempty"` // API key (optional for local services)
	Model           string                 `json:"model"`             // Default model to use
	Temperature     float64                `json:"temperature"`       // Temperature for generation
	MaxTokens       int                    `json:"max_tokens"`        // Maximum tokens for generation
	Timeout         time.Duration          `json:"timeout"`           // Request timeout
	Headers         map[string]string      `json:"headers,omitempty"` // Additional headers
	Parameters      map[string]interface{} `json:"parameters,omitempty"` // Provider-specific parameters
	CompletionPath  string                 `json:"completion_path"`   // API path for completions (default: /v1/chat/completions)
	ModelsPath      string                 `json:"models_path"`       // API path for models (default: /v1/models)
	HealthPath      string                 `json:"health_path"`       // API path for health check
	StreamSupported bool                   `json:"stream_supported"`  // Whether streaming is supported
}

// NewUniversalClient creates a new universal OpenAI-compatible client
func NewUniversalClient(config *Config) (*UniversalClient, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Validate required fields
	if config.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}

	// Set defaults based on provider type
	if config.Provider == "" {
		config.Provider = "custom"
	}

	// Apply provider-specific defaults
	switch config.Provider {
	case "ollama":
		if config.Endpoint == "" {
			return nil, fmt.Errorf("endpoint is required for ollama provider - set OLLAMA_ENDPOINT environment variable")
		}
		if config.CompletionPath == "" {
			config.CompletionPath = "/api/chat"
		}
		if config.ModelsPath == "" {
			config.ModelsPath = "/api/tags"
		}
		config.StreamSupported = true

	case "openai":
		if config.Endpoint == "" {
			config.Endpoint = "https://api.openai.com"
		}
		if config.CompletionPath == "" {
			config.CompletionPath = "/v1/chat/completions"
		}
		if config.ModelsPath == "" {
			config.ModelsPath = "/v1/models"
		}
		config.StreamSupported = true

	default: // custom or unknown provider
		// Use OpenAI-compatible defaults
		if config.CompletionPath == "" {
			config.CompletionPath = "/v1/chat/completions"
		}
		if config.ModelsPath == "" {
			config.ModelsPath = "/v1/models"
		}
		if !config.StreamSupported {
			config.StreamSupported = true // Assume streaming is supported by default
		}
	}

	// Set other defaults
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 4096
	}
	if config.Temperature == 0 {
		config.Temperature = 0.7
	}
	if config.Headers == nil {
		config.Headers = make(map[string]string)
	}

	// Create HTTP client
	client := &UniversalClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}

	return client, nil
}

// Generate executes a generation request
func (c *UniversalClient) Generate(ctx context.Context, req *interfaces.GenerateRequest) (*interfaces.GenerateResponse, error) {
	start := time.Now()

	// Build the request based on provider type
	var requestBody interface{}

	if c.config.Provider == "ollama" {
		// Ollama-specific format
		requestBody = c.buildOllamaRequest(req)
	} else {
		// OpenAI-compatible format
		requestBody = c.buildOpenAIRequest(req)
	}

	// Marshal request
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.Endpoint+c.config.CompletionPath, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}
	for k, v := range c.config.Headers {
		httpReq.Header.Set(k, v)
	}

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse response based on provider
	var response *interfaces.GenerateResponse
	if c.config.Provider == "ollama" {
		response, err = c.parseOllamaResponse(resp.Body, req.Model)
	} else {
		response, err = c.parseOpenAIResponse(resp.Body)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	response.ProcessingTime = time.Since(start)
	return response, nil
}

// GetCapabilities returns the capabilities of this AI client
func (c *UniversalClient) GetCapabilities(ctx context.Context) (*interfaces.Capabilities, error) {
	caps := &interfaces.Capabilities{
		Provider:  c.config.Provider,
		MaxTokens: c.config.MaxTokens,
		Features: []interfaces.Feature{
			{
				Name:        "text-generation",
				Enabled:     true,
				Description: "Text generation capability",
			},
		},
		SupportedLanguages: []string{"en", "zh", "es", "fr", "de", "ja", "ko"},
	}

	// Try to get models list
	models, err := c.getModels(ctx)
	if err == nil {
		caps.Models = models
	} else {
		// If we can't get models, at least add the configured model
		caps.Models = []interfaces.ModelInfo{
			{
				ID:          c.config.Model,
				Name:        c.config.Model,
				Description: "Default configured model",
				MaxTokens:   c.config.MaxTokens,
			},
		}
	}

	// Add streaming feature if supported
	if c.config.StreamSupported {
		caps.Features = append(caps.Features, interfaces.Feature{
			Name:        "streaming",
			Enabled:     true,
			Description: "Streaming response support",
		})
	}

	return caps, nil
}

// HealthCheck performs a health check on the AI service
func (c *UniversalClient) HealthCheck(ctx context.Context) (*interfaces.HealthStatus, error) {
	start := time.Now()

	// Try to get models as a health check
	healthPath := c.config.HealthPath
	if healthPath == "" {
		healthPath = c.config.ModelsPath
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.config.Endpoint+healthPath, nil)
	if err != nil {
		return &interfaces.HealthStatus{
			Healthy:      false,
			Status:       "Failed to create health check request",
			ResponseTime: time.Since(start),
			LastChecked:  time.Now(),
			Errors:       []string{err.Error()},
		}, nil
	}

	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &interfaces.HealthStatus{
			Healthy:      false,
			Status:       "Service unreachable",
			ResponseTime: time.Since(start),
			LastChecked:  time.Now(),
			Errors:       []string{err.Error()},
		}, nil
	}
	defer resp.Body.Close()

	healthy := resp.StatusCode == http.StatusOK
	status := "Healthy"
	if !healthy {
		status = fmt.Sprintf("Unhealthy (status: %d)", resp.StatusCode)
	}

	return &interfaces.HealthStatus{
		Healthy:      healthy,
		Status:       status,
		ResponseTime: time.Since(start),
		LastChecked:  time.Now(),
		Metadata: map[string]any{
			"provider": c.config.Provider,
			"endpoint": c.config.Endpoint,
			"model":    c.config.Model,
		},
	}, nil
}

// Close releases any resources held by the client
func (c *UniversalClient) Close() error {
	// No persistent connections to close
	return nil
}

// buildOllamaRequest builds an Ollama-specific request
func (c *UniversalClient) buildOllamaRequest(req *interfaces.GenerateRequest) map[string]interface{} {
	model := req.Model
	if model == "" {
		model = c.config.Model
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = c.config.Temperature
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = c.config.MaxTokens
	}

	// Build messages for chat format
	messages := []map[string]string{}

	if req.SystemPrompt != "" {
		messages = append(messages, map[string]string{
			"role":    "system",
			"content": req.SystemPrompt,
		})
	}

	// Add context as previous messages
	for _, ctx := range req.Context {
		messages = append(messages, map[string]string{
			"role":    "assistant",
			"content": ctx,
		})
	}

	// Add the main prompt
	messages = append(messages, map[string]string{
		"role":    "user",
		"content": req.Prompt,
	})

	return map[string]interface{}{
		"model":    model,
		"messages": messages,
		"stream":   req.Stream,
		"options": map[string]interface{}{
			"temperature": temperature,
			"num_predict": maxTokens,
		},
	}
}

// buildOpenAIRequest builds an OpenAI-compatible request
func (c *UniversalClient) buildOpenAIRequest(req *interfaces.GenerateRequest) map[string]interface{} {
	model := req.Model
	if model == "" {
		model = c.config.Model
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = c.config.Temperature
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = c.config.MaxTokens
	}

	// Build messages
	messages := []map[string]string{}

	if req.SystemPrompt != "" {
		messages = append(messages, map[string]string{
			"role":    "system",
			"content": req.SystemPrompt,
		})
	}

	// Add context
	for _, ctx := range req.Context {
		messages = append(messages, map[string]string{
			"role":    "assistant",
			"content": ctx,
		})
	}

	// Add the main prompt
	messages = append(messages, map[string]string{
		"role":    "user",
		"content": req.Prompt,
	})

	request := map[string]interface{}{
		"model":       model,
		"messages":    messages,
		"temperature": temperature,
		"max_tokens":  maxTokens,
		"stream":      req.Stream,
	}

	// Add any additional parameters
	for k, v := range c.config.Parameters {
		if _, exists := request[k]; !exists {
			request[k] = v
		}
	}

	return request
}

// parseOllamaResponse parses an Ollama API response
func (c *UniversalClient) parseOllamaResponse(body io.Reader, requestedModel string) (*interfaces.GenerateResponse, error) {
	var resp struct {
		Model              string `json:"model"`
		Message            struct {
			Content string `json:"content"`
		} `json:"message"`
		Done               bool   `json:"done"`
		TotalDuration      int64  `json:"total_duration"`
		LoadDuration       int64  `json:"load_duration"`
		PromptEvalCount    int    `json:"prompt_eval_count"`
		PromptEvalDuration int64  `json:"prompt_eval_duration"`
		EvalCount          int    `json:"eval_count"`
		EvalDuration       int64  `json:"eval_duration"`
	}

	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, err
	}

	return &interfaces.GenerateResponse{
		Text:  resp.Message.Content,
		Model: resp.Model,
		Usage: interfaces.TokenUsage{
			PromptTokens:     resp.PromptEvalCount,
			CompletionTokens: resp.EvalCount,
			TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
		},
		RequestID: fmt.Sprintf("ollama-%d", time.Now().Unix()),
		Metadata: map[string]any{
			"total_duration":   resp.TotalDuration,
			"load_duration":    resp.LoadDuration,
			"prompt_eval_time": resp.PromptEvalDuration,
			"eval_time":        resp.EvalDuration,
		},
	}, nil
}

// parseOpenAIResponse parses an OpenAI-compatible API response
func (c *UniversalClient) parseOpenAIResponse(body io.Reader) (*interfaces.GenerateResponse, error) {
	var resp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &interfaces.GenerateResponse{
		Text:      resp.Choices[0].Message.Content,
		Model:     resp.Model,
		RequestID: resp.ID,
		Usage: interfaces.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		Metadata: map[string]any{
			"finish_reason": resp.Choices[0].FinishReason,
		},
	}, nil
}

// getModels retrieves available models from the API
func (c *UniversalClient) getModels(ctx context.Context) ([]interfaces.ModelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.config.Endpoint+c.config.ModelsPath, nil)
	if err != nil {
		return nil, err
	}

	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get models: status %d", resp.StatusCode)
	}

	// Parse response based on provider
	if c.config.Provider == "ollama" {
		return c.parseOllamaModels(resp.Body)
	}
	return c.parseOpenAIModels(resp.Body)
}

// parseOllamaModels parses Ollama's model list response
func (c *UniversalClient) parseOllamaModels(body io.Reader) ([]interfaces.ModelInfo, error) {
	var resp struct {
		Models []struct {
			Name       string `json:"name"`
			ModifiedAt string `json:"modified_at"`
			Size       int64  `json:"size"`
			Details    struct {
				ParameterSize string `json:"parameter_size"`
			} `json:"details"`
		} `json:"models"`
	}

	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, err
	}

	models := make([]interfaces.ModelInfo, 0, len(resp.Models))
	for _, m := range resp.Models {
		models = append(models, interfaces.ModelInfo{
			ID:          m.Name,
			Name:        m.Name,
			Description: fmt.Sprintf("Ollama model (size: %.2f GB)", float64(m.Size)/(1024*1024*1024)),
			MaxTokens:   c.config.MaxTokens,
		})
	}

	return models, nil
}

// parseOpenAIModels parses OpenAI's model list response
func (c *UniversalClient) parseOpenAIModels(body io.Reader) ([]interfaces.ModelInfo, error) {
	var resp struct {
		Data []struct {
			ID      string `json:"id"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}

	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, err
	}

	models := make([]interfaces.ModelInfo, 0, len(resp.Data))
	for _, m := range resp.Data {
		// Only include chat models
		if strings.Contains(m.ID, "gpt") || strings.Contains(m.ID, "chat") {
			models = append(models, interfaces.ModelInfo{
				ID:          m.ID,
				Name:        m.ID,
				Description: fmt.Sprintf("OpenAI model (owner: %s)", m.OwnedBy),
				MaxTokens:   c.config.MaxTokens,
			})
		}
	}

	return models, nil
}