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

package ai

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/ai/discovery"
	"github.com/linuxsuren/atest-ext-ai/pkg/ai/providers/universal"
	"github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
)

// ProviderInfo represents information about an AI provider
type ProviderInfo struct {
	Name        string                    `json:"name"`
	Type        string                    `json:"type"`
	Available   bool                      `json:"available"`
	Endpoint    string                    `json:"endpoint"`
	Models      []interfaces.ModelInfo    `json:"models"`
	LastChecked time.Time                 `json:"last_checked"`
	Config      map[string]interface{}    `json:"config,omitempty"`
	Health      *interfaces.HealthStatus  `json:"health,omitempty"`
}

// ConnectionTestResult represents the result of a connection test
type ConnectionTestResult struct {
	Success      bool          `json:"success"`
	Message      string        `json:"message"`
	ResponseTime time.Duration `json:"response_time"`
	Provider     string        `json:"provider"`
	Model        string        `json:"model,omitempty"`
	Error        string        `json:"error,omitempty"`
}

// ProviderManager manages AI providers and their discovery
type ProviderManager struct {
	providers map[string]*ProviderInfo
	clients   map[string]interfaces.AIClient
	discovery *discovery.OllamaDiscovery
	mu        sync.RWMutex
	config    *universal.Config
}

// NewProviderManager creates a new provider manager
func NewProviderManager() *ProviderManager {
	return &ProviderManager{
		providers: make(map[string]*ProviderInfo),
		clients:   make(map[string]interfaces.AIClient),
		discovery: discovery.NewOllamaDiscovery(""),
	}
}

// DiscoverProviders discovers available AI providers
func (pm *ProviderManager) DiscoverProviders(ctx context.Context) ([]*ProviderInfo, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var providers []*ProviderInfo

	// Check for Ollama
	if pm.discovery.IsAvailable(ctx) {
		models, err := pm.discovery.GetModels(ctx)
		if err == nil {
			provider := &ProviderInfo{
				Name:        "ollama",
				Type:        "local",
				Available:   true,
				Endpoint:    "http://localhost:11434",
				Models:      models,
				LastChecked: time.Now(),
			}
			pm.providers["ollama"] = provider
			providers = append(providers, provider)

			// Create Ollama client
			config := &universal.Config{
				Provider:       "ollama",
				Endpoint:       "http://localhost:11434",
				Model:          "llama2",
				Temperature:    0.7,
				MaxTokens:      4096,
				StreamSupported: true,
			}
			client, _ := universal.NewUniversalClient(config)
			pm.clients["ollama"] = client
		}
	}

	// Check for other configured providers
	// This can be extended to check for OpenAI, Anthropic, etc.

	return providers, nil
}

// GetProviders returns all discovered providers
func (pm *ProviderManager) GetProviders() []*ProviderInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	providers := make([]*ProviderInfo, 0, len(pm.providers))
	for _, p := range pm.providers {
		providers = append(providers, p)
	}
	return providers
}

// GetProvider returns a specific provider by name
func (pm *ProviderManager) GetProvider(name string) (*ProviderInfo, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	provider, exists := pm.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	return provider, nil
}

// GetModels returns models for a specific provider
func (pm *ProviderManager) GetModels(ctx context.Context, providerName string) ([]interfaces.ModelInfo, error) {
	pm.mu.RLock()
	provider, exists := pm.providers[providerName]
	pm.mu.RUnlock()

	if !exists {
		// Try to discover the provider first
		pm.DiscoverProviders(ctx)

		pm.mu.RLock()
		provider, exists = pm.providers[providerName]
		pm.mu.RUnlock()

		if !exists {
			return nil, fmt.Errorf("provider %s not found", providerName)
		}
	}

	// If models are cached and recent, return them
	if len(provider.Models) > 0 && time.Since(provider.LastChecked) < 5*time.Minute {
		return provider.Models, nil
	}

	// Otherwise, refresh models
	if providerName == "ollama" {
		models, err := pm.discovery.GetModels(ctx)
		if err != nil {
			return nil, err
		}

		pm.mu.Lock()
		provider.Models = models
		provider.LastChecked = time.Now()
		pm.mu.Unlock()

		return models, nil
	}

	// For other providers, use the client's GetCapabilities
	client, exists := pm.clients[providerName]
	if exists {
		caps, err := client.GetCapabilities(ctx)
		if err != nil {
			return nil, err
		}
		return caps.Models, nil
	}

	return provider.Models, nil
}

// TestConnection tests the connection to a provider
func (pm *ProviderManager) TestConnection(ctx context.Context, config *universal.Config) (*ConnectionTestResult, error) {
	start := time.Now()

	// Create a client with the provided config
	client, err := universal.NewUniversalClient(config)
	if err != nil {
		return &ConnectionTestResult{
			Success:      false,
			Message:      "Failed to create client",
			ResponseTime: time.Since(start),
			Provider:     config.Provider,
			Error:        err.Error(),
		}, nil
	}

	// Test the connection with a health check
	health, err := client.HealthCheck(ctx)
	if err != nil {
		return &ConnectionTestResult{
			Success:      false,
			Message:      "Health check failed",
			ResponseTime: time.Since(start),
			Provider:     config.Provider,
			Model:        config.Model,
			Error:        err.Error(),
		}, nil
	}

	message := "Connection successful"
	if !health.Healthy {
		message = health.Status
	}

	return &ConnectionTestResult{
		Success:      health.Healthy,
		Message:      message,
		ResponseTime: health.ResponseTime,
		Provider:     config.Provider,
		Model:        config.Model,
	}, nil
}

// UpdateConfig updates the configuration for a provider
func (pm *ProviderManager) UpdateConfig(ctx context.Context, providerName string, config *universal.Config) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Create new client with the updated config
	client, err := universal.NewUniversalClient(config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Test the client
	health, err := client.HealthCheck(ctx)
	if err != nil || !health.Healthy {
		return fmt.Errorf("health check failed for new configuration")
	}

	// Update the client
	if oldClient, exists := pm.clients[providerName]; exists {
		oldClient.Close()
	}
	pm.clients[providerName] = client

	// Update provider info
	provider := &ProviderInfo{
		Name:        providerName,
		Type:        config.Provider,
		Available:   true,
		Endpoint:    config.Endpoint,
		LastChecked: time.Now(),
		Config: map[string]interface{}{
			"model":       config.Model,
			"temperature": config.Temperature,
			"max_tokens":  config.MaxTokens,
		},
		Health: health,
	}

	// Try to get models
	if caps, err := client.GetCapabilities(ctx); err == nil {
		provider.Models = caps.Models
	}

	pm.providers[providerName] = provider
	return nil
}

// GetClient returns a client for the specified provider
func (pm *ProviderManager) GetClient(providerName string) (interfaces.AIClient, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	client, exists := pm.clients[providerName]
	if !exists {
		return nil, fmt.Errorf("client for provider %s not found", providerName)
	}
	return client, nil
}

// CreateClient creates a new client with the given configuration
func (pm *ProviderManager) CreateClient(config *universal.Config) (interfaces.AIClient, error) {
	return universal.NewUniversalClient(config)
}

// AddProvider adds a new provider with configuration
func (pm *ProviderManager) AddProvider(ctx context.Context, name string, config *universal.Config) error {
	// Create and test the client
	client, err := universal.NewUniversalClient(config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Test health
	health, err := client.HealthCheck(ctx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Close old client if exists
	if oldClient, exists := pm.clients[name]; exists {
		oldClient.Close()
	}

	// Add the new client
	pm.clients[name] = client

	// Create provider info
	provider := &ProviderInfo{
		Name:        name,
		Type:        config.Provider,
		Available:   health.Healthy,
		Endpoint:    config.Endpoint,
		LastChecked: time.Now(),
		Health:      health,
		Config: map[string]interface{}{
			"model":       config.Model,
			"temperature": config.Temperature,
			"max_tokens":  config.MaxTokens,
		},
	}

	// Try to get models
	if caps, err := client.GetCapabilities(ctx); err == nil {
		provider.Models = caps.Models
	}

	pm.providers[name] = provider
	return nil
}

// RemoveProvider removes a provider
func (pm *ProviderManager) RemoveProvider(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Close and remove client
	if client, exists := pm.clients[name]; exists {
		client.Close()
		delete(pm.clients, name)
	}

	// Remove provider info
	delete(pm.providers, name)
	return nil
}