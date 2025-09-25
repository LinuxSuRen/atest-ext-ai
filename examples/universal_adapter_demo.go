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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/linuxsuren/atest-ext-ai/pkg/ai"
	"github.com/linuxsuren/atest-ext-ai/pkg/ai/discovery"
	"github.com/linuxsuren/atest-ext-ai/pkg/ai/providers/universal"
	"github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
)

func main() {
	ctx := context.Background()

	// Example 1: Ollama Auto-Discovery
	fmt.Println("=== Ollama Auto-Discovery Demo ===")
	demoOllamaDiscovery(ctx)

	// Example 2: Universal OpenAI-Compatible Adapter
	fmt.Println("\n=== Universal Adapter Demo ===")
	demoUniversalAdapter(ctx)

	// Example 3: Provider Manager
	fmt.Println("\n=== Provider Manager Demo ===")
	demoProviderManager(ctx)

	// Example 4: Dynamic Configuration
	fmt.Println("\n=== Dynamic Configuration Demo ===")
	demoDynamicConfiguration(ctx)
}

func demoOllamaDiscovery(ctx context.Context) {
	discovery := discovery.NewOllamaDiscovery("")

	// Check if Ollama is available
	if !discovery.IsAvailable(ctx) {
		fmt.Println("❌ Ollama is not running. Please start Ollama first.")
		return
	}

	fmt.Println("✅ Ollama detected and running!")

	// Get available models
	models, err := discovery.GetModels(ctx)
	if err != nil {
		log.Printf("Failed to get models: %v", err)
		return
	}

	fmt.Printf("Found %d models:\n", len(models))
	for _, model := range models {
		fmt.Printf("  • %s: %s\n", model.ID, model.Description)
		if len(model.Capabilities) > 0 {
			fmt.Printf("    Capabilities: %v\n", model.Capabilities)
		}
	}
}

func demoUniversalAdapter(ctx context.Context) {
	// Configuration for Ollama
	ollamaConfig := &universal.Config{
		Provider:    "ollama",
		Endpoint:    "http://localhost:11434",
		Model:       "gemma3:1b", // Use a model you have installed
		Temperature: 0.7,
		MaxTokens:   100,
	}

	// Create universal client
	client, err := universal.NewUniversalClient(ollamaConfig)
	if err != nil {
		log.Printf("Failed to create client: %v", err)
		return
	}
	defer client.Close()

	// Test health check
	health, err := client.HealthCheck(ctx)
	if err != nil {
		log.Printf("Health check failed: %v", err)
		return
	}

	fmt.Printf("Health Status:\n")
	fmt.Printf("  • Healthy: %v\n", health.Healthy)
	fmt.Printf("  • Status: %s\n", health.Status)
	fmt.Printf("  • Response Time: %v\n", health.ResponseTime)

	// Get capabilities
	caps, err := client.GetCapabilities(ctx)
	if err != nil {
		log.Printf("Failed to get capabilities: %v", err)
		return
	}

	fmt.Printf("Capabilities:\n")
	fmt.Printf("  • Provider: %s\n", caps.Provider)
	fmt.Printf("  • Models: %d available\n", len(caps.Models))
	fmt.Printf("  • Features: %v\n", caps.Features)

	// Test generation (if healthy)
	if health.Healthy {
		fmt.Println("\nTesting generation...")
		response, err := client.Generate(ctx, &interfaces.GenerateRequest{
			Prompt:       "What is SQL?",
			Model:        ollamaConfig.Model,
			MaxTokens:    50,
			Temperature:  0.7,
			SystemPrompt: "You are a database expert. Give a brief answer.",
		})

		if err != nil {
			log.Printf("Generation failed: %v", err)
			return
		}

		fmt.Printf("Generated Response:\n%s\n", response.Text)
		fmt.Printf("Model Used: %s\n", response.Model)
		fmt.Printf("Tokens Used: %d\n", response.Usage.TotalTokens)
	}
}

func demoProviderManager(ctx context.Context) {
	manager := ai.NewProviderManager()

	// Discover available providers
	providers, err := manager.DiscoverProviders(ctx)
	if err != nil {
		log.Printf("Failed to discover providers: %v", err)
		return
	}

	fmt.Printf("Discovered %d providers:\n", len(providers))
	for _, provider := range providers {
		fmt.Printf("\nProvider: %s\n", provider.Name)
		fmt.Printf("  • Type: %s\n", provider.Type)
		fmt.Printf("  • Available: %v\n", provider.Available)
		fmt.Printf("  • Endpoint: %s\n", provider.Endpoint)
		fmt.Printf("  • Models: %d available\n", len(provider.Models))
		fmt.Printf("  • Last Checked: %s\n", provider.LastChecked.Format("15:04:05"))
	}

	// Get models for a specific provider
	if len(providers) > 0 {
		providerName := providers[0].Name
		models, err := manager.GetModels(ctx, providerName)
		if err != nil {
			log.Printf("Failed to get models for %s: %v", providerName, err)
			return
		}

		fmt.Printf("\nModels for %s:\n", providerName)
		for i, model := range models {
			if i >= 3 {
				fmt.Printf("  ... and %d more models\n", len(models)-3)
				break
			}
			fmt.Printf("  • %s: %s\n", model.ID, model.Description)
		}
	}
}

func demoDynamicConfiguration(ctx context.Context) {
	manager := ai.NewProviderManager()

	// Test connection with custom configuration
	testConfig := &universal.Config{
		Provider:    "ollama",
		Endpoint:    "http://localhost:11434",
		Model:       "gemma3:1b",
		Temperature: 0.5,
		MaxTokens:   200,
	}

	fmt.Println("Testing connection with custom configuration...")
	result, err := manager.TestConnection(ctx, testConfig)
	if err != nil {
		log.Printf("Connection test failed: %v", err)
		return
	}

	fmt.Printf("Connection Test Result:\n")
	fmt.Printf("  • Success: %v\n", result.Success)
	fmt.Printf("  • Message: %s\n", result.Message)
	fmt.Printf("  • Response Time: %v\n", result.ResponseTime)
	fmt.Printf("  • Provider: %s\n", result.Provider)
	if result.Model != "" {
		fmt.Printf("  • Model: %s\n", result.Model)
	}

	// Add a custom provider
	if result.Success {
		fmt.Println("\nAdding custom provider...")
		err = manager.AddProvider(ctx, "my-ollama", testConfig)
		if err != nil {
			log.Printf("Failed to add provider: %v", err)
			return
		}
		fmt.Println("✅ Custom provider 'my-ollama' added successfully!")

		// Get the provider info
		provider, err := manager.GetProvider("my-ollama")
		if err != nil {
			log.Printf("Failed to get provider info: %v", err)
			return
		}

		configJSON, _ := json.MarshalIndent(provider.Config, "    ", "  ")
		fmt.Printf("Provider Configuration:\n%s\n", string(configJSON))
	}
}

// Example configurations for different providers
func printExampleConfigurations() {
	fmt.Println("\n=== Example Configurations ===")

	// Ollama configuration
	ollamaConfig := universal.Config{
		Provider:    "ollama",
		Endpoint:    "http://localhost:11434",
		Model:       "llama2",
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	// OpenAI configuration
	openAIConfig := universal.Config{
		Provider:    "openai",
		Endpoint:    "https://api.openai.com",
		APIKey:      "your-api-key-here",
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	// Custom OpenAI-compatible service
	customConfig := universal.Config{
		Provider:       "custom",
		Endpoint:       "https://your-ai-service.com",
		APIKey:         "your-api-key",
		Model:          "custom-model",
		Temperature:    0.8,
		MaxTokens:      2048,
		CompletionPath: "/v1/chat/completions",
		ModelsPath:     "/v1/models",
		Headers: map[string]string{
			"X-Custom-Header": "value",
		},
	}

	configs := []struct {
		Name   string
		Config universal.Config
	}{
		{"Ollama", ollamaConfig},
		{"OpenAI", openAIConfig},
		{"Custom Service", customConfig},
	}

	for _, cfg := range configs {
		fmt.Printf("\n%s Configuration:\n", cfg.Name)
		jsonData, _ := json.MarshalIndent(cfg.Config, "  ", "  ")
		fmt.Println(string(jsonData))
	}
}