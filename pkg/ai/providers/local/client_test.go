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

package local

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
		},
		{
			name:        "valid config with defaults",
			config:      &Config{},
			expectError: false,
		},
		{
			name: "valid config with all fields",
			config: &Config{
				BaseURL:     "http://localhost:11434",
				Timeout:     60 * time.Second,
				MaxTokens:   4096,
				Model:       "llama2",
				UserAgent:   "test-agent",
				Temperature: 0.8,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectError && client == nil {
				t.Errorf("Expected client but got nil")
			}
		})
	}
}

func TestClient_GetCapabilities(t *testing.T) {
	config := &Config{}
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	caps, err := client.GetCapabilities(ctx)
	if err != nil {
		t.Fatalf("GetCapabilities failed: %v", err)
	}

	if caps == nil {
		t.Fatal("Expected capabilities but got nil")
	}

	if caps.Provider != "local" {
		t.Errorf("Expected provider 'local', got '%s'", caps.Provider)
	}

	if len(caps.Models) == 0 {
		t.Error("Expected at least one model")
	}

	if len(caps.Features) == 0 {
		t.Error("Expected at least one feature")
	}

	// Check for local-specific features
	hasLocalExecution := false
	for _, feature := range caps.Features {
		if feature.Name == "local_execution" {
			hasLocalExecution = true
			break
		}
	}
	if !hasLocalExecution {
		t.Error("Expected local_execution feature for local provider")
	}

	// Check that rate limits are unlimited
	if caps.RateLimits != nil {
		if caps.RateLimits.RequestsPerMinute != -1 {
			t.Error("Expected unlimited requests per minute for local provider")
		}
	}
}

func TestClient_Close(t *testing.T) {
	config := &Config{}
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestClient_buildPrompt(t *testing.T) {
	config := &Config{}
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tests := []struct {
		name     string
		req      *interfaces.GenerateRequest
		expected []string // Parts that should be in the prompt
	}{
		{
			name: "simple prompt",
			req: &interfaces.GenerateRequest{
				Prompt: "Hello world",
			},
			expected: []string{"User: Hello world"},
		},
		{
			name: "prompt with system",
			req: &interfaces.GenerateRequest{
				SystemPrompt: "You are a helpful assistant",
				Prompt:       "Hello world",
			},
			expected: []string{"System: You are a helpful assistant", "User: Hello world"},
		},
		{
			name: "prompt with context",
			req: &interfaces.GenerateRequest{
				Context: []string{"Previous message 1", "Previous message 2"},
				Prompt:  "Hello world",
			},
			expected: []string{"Context 1: Previous message 1", "Context 2: Previous message 2", "User: Hello world"},
		},
		{
			name: "prompt with all parts",
			req: &interfaces.GenerateRequest{
				SystemPrompt: "You are a helpful assistant",
				Context:      []string{"Previous message"},
				Prompt:       "Hello world",
			},
			expected: []string{"System: You are a helpful assistant", "Context 1: Previous message", "User: Hello world"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := client.buildPrompt(tt.req)

			for _, expectedPart := range tt.expected {
				if !strings.Contains(prompt, expectedPart) {
					t.Errorf("Expected prompt to contain '%s', but got: %s", expectedPart, prompt)
				}
			}
		})
	}
}

func TestClient_getModel(t *testing.T) {
	config := &Config{
		Model: "llama2",
	}
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test with request-specific model
	req := &interfaces.GenerateRequest{
		Model: "codellama",
	}
	model := client.getModel(req)
	if model != "codellama" {
		t.Errorf("Expected 'codellama', got '%s'", model)
	}

	// Test with default model
	req = &interfaces.GenerateRequest{}
	model = client.getModel(req)
	if model != "llama2" {
		t.Errorf("Expected 'llama2', got '%s'", model)
	}
}

func TestClient_getMaxTokens(t *testing.T) {
	config := &Config{
		MaxTokens: 2048,
	}
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test with request-specific max tokens
	req := &interfaces.GenerateRequest{
		MaxTokens: 1024,
	}
	maxTokens := client.getMaxTokens(req)
	if maxTokens != 1024 {
		t.Errorf("Expected 1024, got %d", maxTokens)
	}

	// Test with default max tokens
	req = &interfaces.GenerateRequest{}
	maxTokens = client.getMaxTokens(req)
	if maxTokens != 2048 {
		t.Errorf("Expected 2048, got %d", maxTokens)
	}
}

func TestClient_getTemperature(t *testing.T) {
	config := &Config{
		Temperature: 0.8,
	}
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test with request-specific temperature
	req := &interfaces.GenerateRequest{
		Temperature: 0.5,
	}
	temp := client.getTemperature(req)
	if temp != 0.5 {
		t.Errorf("Expected 0.5, got %f", temp)
	}

	// Test with default temperature
	req = &interfaces.GenerateRequest{}
	temp = client.getTemperature(req)
	if temp != 0.8 {
		t.Errorf("Expected 0.8, got %f", temp)
	}
}

func TestNewClient_EnvironmentVariables(t *testing.T) {
	// Test environment variable override for base URL
	originalURL := os.Getenv("OLLAMA_BASE_URL")
	defer func() {
		if originalURL != "" {
			_ = os.Setenv("OLLAMA_BASE_URL", originalURL)
		} else {
			_ = os.Unsetenv("OLLAMA_BASE_URL")
		}
	}()

	_ = os.Setenv("OLLAMA_BASE_URL", "http://custom:11434")

	config := &Config{}
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.config.BaseURL != "http://custom:11434" {
		t.Errorf("Expected BaseURL to be 'http://custom:11434', got '%s'", client.config.BaseURL)
	}
}

func TestNewClient_ConnectionPooling(t *testing.T) {
	config := &Config{
		MaxIdleConns:    50,
		MaxConnsPerHost: 5,
		IdleConnTimeout: 60 * time.Second,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.config.MaxIdleConns != 50 {
		t.Errorf("Expected MaxIdleConns to be 50, got %d", client.config.MaxIdleConns)
	}
	if client.config.MaxConnsPerHost != 5 {
		t.Errorf("Expected MaxConnsPerHost to be 5, got %d", client.config.MaxConnsPerHost)
	}
	if client.config.IdleConnTimeout != 60*time.Second {
		t.Errorf("Expected IdleConnTimeout to be 60s, got %v", client.config.IdleConnTimeout)
	}
}

func TestClient_CloseWithConnectionPool(t *testing.T) {
	config := &Config{}
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Verify the client has an HTTP transport
	if client.httpClient.Transport == nil {
		t.Error("Expected HTTP transport to be configured")
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestClient_GenerateOptions(t *testing.T) {
	config := &Config{}
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	req := &interfaces.GenerateRequest{
		Prompt:      "Test prompt",
		Model:       "test-model",
		MaxTokens:   100,
		Temperature: 0.5,
		Stream:      false,
	}

	// Test that streaming flag is properly handled in the request building
	ollamaReq := &GenerateRequest{
		Model:  client.getModel(req),
		Prompt: client.buildPrompt(req),
		Stream: req.Stream,
		Options: map[string]any{
			"temperature": client.getTemperature(req),
			"num_predict": client.getMaxTokens(req),
		},
	}

	if ollamaReq.Stream != false {
		t.Errorf("Expected Stream to be false, got %v", ollamaReq.Stream)
	}
	if ollamaReq.Model != "test-model" {
		t.Errorf("Expected Model to be 'test-model', got '%s'", ollamaReq.Model)
	}
	if ollamaReq.Options["temperature"] != 0.5 {
		t.Errorf("Expected temperature to be 0.5, got %v", ollamaReq.Options["temperature"])
	}
	if ollamaReq.Options["num_predict"] != 100 {
		t.Errorf("Expected num_predict to be 100, got %v", ollamaReq.Options["num_predict"])
	}
}
