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
			name: "empty API key",
			config: &Config{
				APIKey: "",
			},
			expectError: true,
		},
		{
			name: "valid config with defaults",
			config: &Config{
				APIKey: "test-key",
			},
			expectError: false,
		},
		{
			name: "valid config with all fields",
			config: &Config{
				APIKey:    "test-key",
				BaseURL:   "https://api.openai.com/v1",
				Timeout:   30 * time.Second,
				MaxTokens: 4096,
				Model:     "gpt-4",
				OrgID:     "org-123",
				UserAgent: "test-agent",
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
	config := &Config{
		APIKey: "test-key",
	}
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

	if caps.Provider != "openai" {
		t.Errorf("Expected provider 'openai', got '%s'", caps.Provider)
	}

	if len(caps.Models) == 0 {
		t.Error("Expected at least one model")
	}

	if len(caps.Features) == 0 {
		t.Error("Expected at least one feature")
	}
}

func TestClient_Close(t *testing.T) {
	config := &Config{
		APIKey: "test-key",
	}
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestClient_getModel(t *testing.T) {
	config := &Config{
		APIKey: "test-key",
		Model:  "gpt-3.5-turbo",
	}
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test with request-specific model
	req := &interfaces.GenerateRequest{
		Model: "gpt-4",
	}
	model := client.getModel(req)
	if model != "gpt-4" {
		t.Errorf("Expected 'gpt-4', got '%s'", model)
	}

	// Test with default model
	req = &interfaces.GenerateRequest{}
	model = client.getModel(req)
	if model != "gpt-3.5-turbo" {
		t.Errorf("Expected 'gpt-3.5-turbo', got '%s'", model)
	}
}

func TestClient_getMaxTokens(t *testing.T) {
	config := &Config{
		APIKey:    "test-key",
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
		APIKey: "test-key",
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
	if temp != 0.7 {
		t.Errorf("Expected 0.7, got %f", temp)
	}
}

func TestNewClient_EnvironmentVariables(t *testing.T) {
	// Test environment variable for API key
	originalKey := os.Getenv("OPENAI_API_KEY")
	originalOrg := os.Getenv("OPENAI_ORG_ID")
	defer func() {
		if originalKey != "" {
			_ = os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			_ = os.Unsetenv("OPENAI_API_KEY")
		}
		if originalOrg != "" {
			_ = os.Setenv("OPENAI_ORG_ID", originalOrg)
		} else {
			_ = os.Unsetenv("OPENAI_ORG_ID")
		}
	}()

	_ = os.Setenv("OPENAI_API_KEY", "env-test-key")
	_ = os.Setenv("OPENAI_ORG_ID", "env-test-org")

	config := &Config{}
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.config.APIKey != "env-test-key" {
		t.Errorf("Expected API key to be 'env-test-key', got '%s'", client.config.APIKey)
	}
	if client.config.OrgID != "env-test-org" {
		t.Errorf("Expected OrgID to be 'env-test-org', got '%s'", client.config.OrgID)
	}
}

func TestNewClient_ConnectionPooling(t *testing.T) {
	config := &Config{
		APIKey:          "test-key",
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
	config := &Config{
		APIKey: "test-key",
	}
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Verify the client has a langchaingo LLM
	if client.llm == nil {
		t.Error("Expected LLM to be configured")
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestClient_RequestBuilding(t *testing.T) {
	config := &Config{
		APIKey: "test-key",
	}
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	req := &interfaces.GenerateRequest{
		Prompt:       "Test prompt",
		SystemPrompt: "You are a helpful assistant",
		Model:        "gpt-3.5-turbo",
		MaxTokens:    100,
		Temperature:  0.5,
		Stream:       true,
		Context:      []string{"Previous context"},
	}

	// Test message building
	messages := client.buildMessages(req)
	if !strings.Contains(messages, "System: You are a helpful assistant") {
		t.Error("Expected messages to contain system prompt")
	}
	if !strings.Contains(messages, "Test prompt") {
		t.Error("Expected messages to contain main prompt")
	}
	if !strings.Contains(messages, "Previous context") {
		t.Error("Expected messages to contain context")
	}

	// Test generation options building
	opts := client.buildGenerationOptions(req)
	if len(opts) == 0 {
		t.Error("Expected generation options to be built")
	}
}

func TestClient_RateLimits(t *testing.T) {
	config := &Config{
		APIKey: "test-key",
	}
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	caps, err := client.GetCapabilities(ctx)
	if err != nil {
		t.Fatalf("GetCapabilities failed: %v", err)
	}

	if caps.RateLimits == nil {
		t.Error("Expected rate limits to be defined")
	} else {
		if caps.RateLimits.RequestsPerMinute <= 0 {
			t.Error("Expected positive requests per minute")
		}
		if caps.RateLimits.TokensPerMinute <= 0 {
			t.Error("Expected positive tokens per minute")
		}
	}
}
