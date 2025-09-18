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

package anthropic

import (
	"context"
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
				BaseURL:   "https://api.anthropic.com",
				Timeout:   45 * time.Second,
				MaxTokens: 4096,
				Model:     "claude-3-sonnet-20240229",
				Version:   "2023-06-01",
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

	if caps.Provider != "anthropic" {
		t.Errorf("Expected provider 'anthropic', got '%s'", caps.Provider)
	}

	if len(caps.Models) == 0 {
		t.Error("Expected at least one model")
	}

	if len(caps.Features) == 0 {
		t.Error("Expected at least one feature")
	}

	// Check for Claude-specific features
	hasLongContext := false
	for _, feature := range caps.Features {
		if feature.Name == "long_context" {
			hasLongContext = true
			break
		}
	}
	if !hasLongContext {
		t.Error("Expected long_context feature for Claude")
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
		Model:  "claude-3-haiku-20240307",
	}
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test with request-specific model
	req := &interfaces.GenerateRequest{
		Model: "claude-3-opus-20240229",
	}
	model := client.getModel(req)
	if model != "claude-3-opus-20240229" {
		t.Errorf("Expected 'claude-3-opus-20240229', got '%s'", model)
	}

	// Test with default model
	req = &interfaces.GenerateRequest{}
	model = client.getModel(req)
	if model != "claude-3-haiku-20240307" {
		t.Errorf("Expected 'claude-3-haiku-20240307', got '%s'", model)
	}
}

func TestClient_getMaxTokens(t *testing.T) {
	config := &Config{
		APIKey:    "test-key",
		MaxTokens: 8192,
	}
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test with request-specific max tokens
	req := &interfaces.GenerateRequest{
		MaxTokens: 4096,
	}
	maxTokens := client.getMaxTokens(req)
	if maxTokens != 4096 {
		t.Errorf("Expected 4096, got %d", maxTokens)
	}

	// Test with default max tokens
	req = &interfaces.GenerateRequest{}
	maxTokens = client.getMaxTokens(req)
	if maxTokens != 8192 {
		t.Errorf("Expected 8192, got %d", maxTokens)
	}
}