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
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
	"github.com/linuxsuren/atest-ext-ai/pkg/logging"
)

// Global HTTP client pool for connection reuse across providers
// Using sync.Map for concurrent-safe access without explicit locking on read
var (
	httpClientPool = &sync.Map{} // key: provider name (string), value: *http.Client
	httpClientMu   sync.Mutex    // Mutex for client creation to prevent duplicate creation
)

// getOrCreateHTTPClient retrieves an existing HTTP client from the pool or creates a new one
// This implements connection pooling to improve performance and resource utilization
// Based on Go net/http best practices for Transport configuration
func getOrCreateHTTPClient(provider string, timeout time.Duration) *http.Client {
	// Try to get existing client from pool (fast path, no locking)
	if client, ok := httpClientPool.Load(provider); ok {
		logging.Logger.Debug("Reusing HTTP client from pool",
			"provider", provider)
		return client.(*http.Client)
	}

	// Client not found, need to create (slow path with locking)
	httpClientMu.Lock()
	defer httpClientMu.Unlock()

	// Double-check: another goroutine might have created the client while we waited for the lock
	if client, ok := httpClientPool.Load(provider); ok {
		logging.Logger.Debug("HTTP client created by another goroutine",
			"provider", provider)
		return client.(*http.Client)
	}

	// Create new HTTP client with optimized transport settings
	// Configuration follows Go net/http best practices:
	// - MaxIdleConns: Total maximum idle connections across all hosts
	// - MaxIdleConnsPerHost: Maximum idle connections per host (important for AI APIs)
	// - IdleConnTimeout: How long idle connections remain in the pool
	// - DisableCompression: Disabled for better compatibility with AI APIs
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,              // Total pool size across all hosts
			MaxIdleConnsPerHost: 10,               // Per-host idle connection limit (AI APIs typically use 1 host)
			IdleConnTimeout:     90 * time.Second, // Keep idle connections for 90s
			DisableCompression:  false,            // Enable compression for better bandwidth utilization
			// Additional recommended settings for production use:
			MaxConnsPerHost:        0,                // No limit on active connections (0 = unlimited)
			ResponseHeaderTimeout:  30 * time.Second, // Timeout for reading response headers
			ExpectContinueTimeout:  1 * time.Second,  // Timeout for 100-Continue handshake
			ForceAttemptHTTP2:      true,             // Enable HTTP/2 when available
			DisableKeepAlives:      false,            // Enable keep-alives for connection reuse
			TLSHandshakeTimeout:    10 * time.Second, // Timeout for TLS handshake
		},
	}

	// Store in pool for reuse
	httpClientPool.Store(provider, client)

	logging.Logger.Info("Created new HTTP client with connection pooling",
		"provider", provider,
		"timeout", timeout,
		"max_idle_conns", 100,
		"max_idle_conns_per_host", 10,
		"idle_conn_timeout", "90s")

	return client
}

// UniversalClient implements a universal OpenAI-compatible API client
type UniversalClient struct {
	config     *Config
	httpClient *http.Client
	strategy   ProviderStrategy // Strategy pattern to handle provider-specific logic
}

// Config holds configuration for the universal client
type Config struct {
	Provider        string            `json:"provider"`             // Provider name (e.g., "ollama", "openai", "custom")
	Endpoint        string            `json:"endpoint"`             // API endpoint URL
	APIKey          string            `json:"api_key,omitempty"`    // API key (optional for local services)
	Model           string            `json:"model"`                // Default model to use
	MaxTokens       int               `json:"max_tokens"`           // Maximum tokens for generation
	Timeout         time.Duration     `json:"timeout"`              // Request timeout
	Headers         map[string]string `json:"headers,omitempty"`    // Additional headers
	Parameters      map[string]any    `json:"parameters,omitempty"` // Provider-specific parameters
	CompletionPath  string            `json:"completion_path"`      // API path for completions (default: /v1/chat/completions)
	ModelsPath      string            `json:"models_path"`          // API path for models (default: /v1/models)
	HealthPath      string            `json:"health_path"`          // API path for health check
	StreamSupported bool              `json:"stream_supported"`     // Whether streaming is supported
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

	// Get strategy for this provider
	strategy := GetStrategy(config.Provider)

	// Apply provider-specific defaults using strategy
	paths := strategy.GetDefaultPaths()
	if config.CompletionPath == "" {
		config.CompletionPath = paths.CompletionPath
	}
	if config.ModelsPath == "" {
		config.ModelsPath = paths.ModelsPath
	}
	if config.HealthPath == "" {
		config.HealthPath = paths.HealthPath
	}
	config.StreamSupported = strategy.SupportsStreaming()

	// Apply endpoint defaults for specific providers
	if config.Provider == "openai" && config.Endpoint == "" {
		config.Endpoint = "https://api.openai.com"
	} else if config.Provider == "deepseek" && config.Endpoint == "" {
		config.Endpoint = "https://api.deepseek.com"
	}

	// Set other defaults
	if config.Timeout == 0 {
		// Increase timeout for reasoning/thinking models
		if strings.Contains(strings.ToLower(config.Model), "think") || strings.Contains(strings.ToLower(config.Model), "reason") {
			config.Timeout = 300 * time.Second // 5 minutes for thinking models
		} else {
			config.Timeout = 120 * time.Second // 2 minutes for regular models
		}
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 4096
	}
	if config.Headers == nil {
		config.Headers = make(map[string]string)
	}

	// Create HTTP client using connection pool for better performance
	// This reuses connections across requests to the same provider
	httpClient := getOrCreateHTTPClient(config.Provider, config.Timeout)

	client := &UniversalClient{
		config:     config,
		strategy:   strategy,
		httpClient: httpClient,
	}

	logging.Logger.Debug("Universal client created",
		"provider", config.Provider,
		"endpoint", config.Endpoint,
		"model", config.Model,
		"timeout", config.Timeout)

	return client, nil
}

// Generate executes a generation request
func (c *UniversalClient) Generate(ctx context.Context, req *interfaces.GenerateRequest) (*interfaces.GenerateResponse, error) {
	start := time.Now()

	// Build request using strategy pattern
	requestBody, err := c.strategy.BuildRequest(req, c.config)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
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
	defer func() { _ = resp.Body.Close() }()

	// Check status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse response using strategy pattern
	response, err := c.strategy.ParseResponse(resp.Body, req.Model)
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
		// If we can't get models, provide default models for the provider
		caps.Models = c.getDefaultModelsForProvider()
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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get models: status %d", resp.StatusCode)
	}

	// Parse response using strategy pattern
	return c.strategy.ParseModels(resp.Body, c.config.MaxTokens)
}

// getDefaultModelsForProvider returns default models using strategy pattern
func (c *UniversalClient) getDefaultModelsForProvider() []interfaces.ModelInfo {
	models := c.strategy.GetDefaultModels(c.config.MaxTokens)

	// If strategy returns empty list and we have a configured model, use it as fallback
	if len(models) == 0 && c.config.Model != "" {
		return []interfaces.ModelInfo{
			{
				ID:          c.config.Model,
				Name:        c.config.Model,
				Description: "Default configured model",
				MaxTokens:   c.config.MaxTokens,
			},
		}
	}

	return models
}
