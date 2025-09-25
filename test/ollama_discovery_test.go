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

package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/ai"
	"github.com/linuxsuren/atest-ext-ai/pkg/ai/discovery"
	"github.com/linuxsuren/atest-ext-ai/pkg/ai/providers/universal"
)

// TestOllamaDiscovery tests Ollama auto-discovery functionality
func TestOllamaDiscovery(t *testing.T) {
	// Skip if Ollama is not available
	discovery := discovery.NewOllamaDiscovery("")
	ctx := context.Background()

	if !discovery.IsAvailable(ctx) {
		t.Skip("Ollama is not available, skipping test")
	}

	// Test getting models
	models, err := discovery.GetModels(ctx)
	if err != nil {
		t.Errorf("Failed to get models: %v", err)
		return
	}

	fmt.Printf("Found %d Ollama models:\n", len(models))
	for _, model := range models {
		fmt.Printf("  - %s: %s\n", model.ID, model.Description)
	}
}

// TestProviderManager tests the provider manager functionality
func TestProviderManager(t *testing.T) {
	ctx := context.Background()
	manager := ai.NewProviderManager()

	// Discover providers
	providers, err := manager.DiscoverProviders(ctx)
	if err != nil {
		t.Errorf("Failed to discover providers: %v", err)
		return
	}

	fmt.Printf("Discovered %d providers:\n", len(providers))
	for _, provider := range providers {
		fmt.Printf("  - %s: %s, %d models available\n",
			provider.Name, provider.Endpoint, len(provider.Models))
	}

	// Test getting models from a provider
	if len(providers) > 0 {
		models, err := manager.GetModels(ctx, providers[0].Name)
		if err != nil {
			t.Errorf("Failed to get models: %v", err)
			return
		}
		fmt.Printf("Provider %s has %d models\n", providers[0].Name, len(models))
	}
}

// TestUniversalClient tests the universal OpenAI-compatible client
func TestUniversalClient(t *testing.T) {
	// Test with Ollama configuration
	config := &universal.Config{
		Provider:    "ollama",
		Endpoint:    "http://localhost:11434",
		Model:       "llama2",
		Temperature: 0.7,
		MaxTokens:   100,
	}

	client, err := universal.NewUniversalClient(config)
	if err != nil {
		t.Errorf("Failed to create client: %v", err)
		return
	}

	ctx := context.Background()

	// Test health check
	health, err := client.HealthCheck(ctx)
	if err != nil {
		t.Logf("Health check failed (this is ok if Ollama is not running): %v", err)
		return
	}

	fmt.Printf("Health check result: %v\n", health.Healthy)
	if health.Healthy {
		fmt.Printf("  Status: %s\n", health.Status)
		fmt.Printf("  Response time: %v\n", health.ResponseTime)
	}

	// Test capabilities
	caps, err := client.GetCapabilities(ctx)
	if err != nil {
		t.Logf("Failed to get capabilities: %v", err)
		return
	}

	fmt.Printf("Provider capabilities:\n")
	fmt.Printf("  Provider: %s\n", caps.Provider)
	fmt.Printf("  Models: %d\n", len(caps.Models))
	fmt.Printf("  Features: %d\n", len(caps.Features))
}

// TestConnectionTest tests the connection testing functionality
func TestConnectionTest(t *testing.T) {
	manager := ai.NewProviderManager()
	ctx := context.Background()

	// Test Ollama connection
	config := &universal.Config{
		Provider: "ollama",
		Endpoint: "http://localhost:11434",
		Model:    "llama2",
	}

	result, err := manager.TestConnection(ctx, config)
	if err != nil {
		t.Errorf("Failed to test connection: %v", err)
		return
	}

	fmt.Printf("Connection test result:\n")
	fmt.Printf("  Success: %v\n", result.Success)
	fmt.Printf("  Message: %s\n", result.Message)
	fmt.Printf("  Response time: %v\n", result.ResponseTime)
	if result.Error != "" {
		fmt.Printf("  Error: %s\n", result.Error)
	}
}

// TestDynamicConfiguration tests dynamic configuration updates
func TestDynamicConfiguration(t *testing.T) {
	manager := ai.NewProviderManager()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Add a new provider with custom configuration
	config := &universal.Config{
		Provider:    "custom",
		Endpoint:    "http://localhost:8080",
		Model:       "custom-model",
		Temperature: 0.5,
		MaxTokens:   2048,
		CompletionPath: "/v1/completions",
		ModelsPath:     "/v1/models",
	}

	err := manager.AddProvider(ctx, "custom-provider", config)
	if err != nil {
		// This is expected if the endpoint doesn't exist
		t.Logf("Failed to add provider (expected if endpoint doesn't exist): %v", err)
	}

	// List all providers
	providers := manager.GetProviders()
	fmt.Printf("Total providers after adding custom: %d\n", len(providers))
}