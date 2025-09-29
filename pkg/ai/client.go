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
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/ai/providers/anthropic"
	"github.com/linuxsuren/atest-ext-ai/pkg/ai/providers/local"
	"github.com/linuxsuren/atest-ext-ai/pkg/ai/providers/openai"
	"github.com/linuxsuren/atest-ext-ai/pkg/config"
	"github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
)

var (
	// ErrProviderNotSupported is returned when an unsupported provider is requested
	ErrProviderNotSupported = errors.New("provider not supported")

	// ErrNoHealthyClients is returned when no healthy clients are available
	ErrNoHealthyClients = errors.New("no healthy clients available")

	// ErrClientNotFound is returned when a specific client is not found
	ErrClientNotFound = errors.New("client not found")

	// ErrInvalidConfig is returned when the configuration is invalid
	ErrInvalidConfig = errors.New("invalid configuration")
)

// Client provides a simplified interface for AI interactions
type Client struct {
	manager        *ClientManager
	primaryClient  interfaces.AIClient
	defaultService string
}

// NewClient creates a new AI client from configuration
func NewClient(cfg interface{}) (*Client, error) {
	// Handle different config types for backward compatibility
	var serviceConfig *AIServiceConfig
	var defaultService string

	switch c := cfg.(type) {
	case config.AIConfig:
		// Convert new config to service config
		serviceConfig = convertAIConfigToServiceConfig(c)
		defaultService = c.DefaultService
	default:
		return nil, fmt.Errorf("unsupported configuration type")
	}

	if len(serviceConfig.Providers) == 0 {
		return nil, fmt.Errorf("no providers configured")
	}

	manager, err := NewClientManager(serviceConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create client manager: %w", err)
	}

	// Get primary client
	var primaryClient interfaces.AIClient
	if defaultService != "" {
		if client, err := manager.GetClient(defaultService); err == nil {
			primaryClient = client
		}
	}

	// If no primary client, use the first available one
	if primaryClient == nil {
		for _, provider := range serviceConfig.Providers {
			if provider.Enabled {
				if client, err := manager.GetClient(provider.Name); err == nil {
					primaryClient = client
					defaultService = provider.Name
					break
				}
			}
		}
	}

	return &Client{
		manager:        manager,
		primaryClient:  primaryClient,
		defaultService: defaultService,
	}, nil
}

// GetPrimaryClient returns the primary AI client
func (c *Client) GetPrimaryClient() interfaces.AIClient {
	return c.primaryClient
}

// Generate executes an AI generation request
func (c *Client) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	return c.manager.Generate(ctx, req)
}

// Close closes the client and all its resources
func (c *Client) Close() error {
	return c.manager.Close()
}

// GetAllClients returns all available clients from the client manager
func (c *Client) GetAllClients() map[string]interfaces.AIClient {
	if c.manager == nil {
		return make(map[string]interfaces.AIClient)
	}

	c.manager.mu.RLock()
	defer c.manager.mu.RUnlock()

	// Create a copy to avoid concurrent access issues
	clients := make(map[string]interfaces.AIClient)
	for name, client := range c.manager.clients {
		clients[name] = client
	}

	return clients
}

// convertAIConfigToServiceConfig converts the new config format to the service config format
func convertAIConfigToServiceConfig(cfg config.AIConfig) *AIServiceConfig {
	serviceConfig := &AIServiceConfig{
		Providers: make([]ProviderConfig, 0, len(cfg.Services)),
		Retry: RetryConfig{
			MaxAttempts:       3,
			BaseDelay:         100 * time.Millisecond,
			MaxDelay:          5 * time.Second,
			BackoffMultiplier: 2.0,
			Jitter:            true,
		},
	}

	// Convert services to providers
	for name, service := range cfg.Services {
		if !service.Enabled {
			continue
		}

		providerConfig := ProviderConfig{
			Name:     name,
			Enabled:  service.Enabled,
			Priority: service.Priority,
			Config: map[string]any{
				"api_key":     service.APIKey,
				"base_url":    service.Endpoint,
				"model":       service.Model,
				"max_tokens":  service.MaxTokens,
				"temperature": service.Temperature,
				"timeout":     service.Timeout,
			},
			Models:     service.Models,
			Timeout:    service.Timeout.Value(),
			MaxRetries: 3,
		}

		serviceConfig.Providers = append(serviceConfig.Providers, providerConfig)
	}

	return serviceConfig
}

// ClientManager manages multiple AI clients and provides unified access
type ClientManager struct {
	clients       map[string]interfaces.AIClient
	factory       ClientFactory
	retryManager  RetryManager
	config        *AIServiceConfig
	mu            sync.RWMutex
	healthChecker *HealthChecker
}

// NewClientManager creates a new client manager with the given configuration
func NewClientManager(config *AIServiceConfig) (*ClientManager, error) {
	if config == nil {
		return nil, fmt.Errorf("%w: config is nil", ErrInvalidConfig)
	}

	manager := &ClientManager{
		clients: make(map[string]interfaces.AIClient),
		config:  config,
	}

	// Initialize factory
	factory, err := NewDefaultClientFactory()
	if err != nil {
		return nil, fmt.Errorf("failed to create client factory: %w", err)
	}
	manager.factory = factory

	// Initialize retry manager
	manager.retryManager = NewDefaultRetryManager(config.Retry)

	// Initialize health checker
	manager.healthChecker = NewHealthChecker(30 * time.Second)

	// Create clients for enabled providers
	if err := manager.initializeClients(); err != nil {
		return nil, fmt.Errorf("failed to initialize clients: %w", err)
	}

	// Start health checking
	manager.healthChecker.Start(manager.clients)

	return manager, nil
}

// initializeClients creates clients for all enabled providers
func (cm *ClientManager) initializeClients() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for _, providerConfig := range cm.config.Providers {
		if !providerConfig.Enabled {
			continue
		}

		client, err := cm.factory.CreateClient(providerConfig.Name, providerConfig.Config)
		if err != nil {
			return fmt.Errorf("failed to create client for provider %s: %w", providerConfig.Name, err)
		}

		cm.clients[providerConfig.Name] = client
	}

	return nil
}

// Generate executes an AI generation request with retry logic
func (cm *ClientManager) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	var result *GenerateResponse

	err := cm.retryManager.Execute(ctx, func() error {
		client, err := cm.selectFirstHealthyClient()
		if err != nil {
			return err
		}

		response, err := client.Generate(ctx, req)
		if err != nil {
			return err
		}

		result = response
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// selectFirstHealthyClient selects the first available healthy client
func (cm *ClientManager) selectFirstHealthyClient() (interfaces.AIClient, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Check health status and return first healthy client
	healthStatus := cm.healthChecker.GetHealthStatus()

	for name, client := range cm.clients {
		if health, exists := healthStatus[name]; exists && health.Healthy {
			return client, nil
		}
	}

	// If no healthy clients, return first available client as fallback
	for _, client := range cm.clients {
		return client, nil
	}

	return nil, ErrNoHealthyClients
}

// GetCapabilities returns the capabilities of all available clients
func (cm *ClientManager) GetCapabilities(ctx context.Context) (map[string]*Capabilities, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	capabilities := make(map[string]*Capabilities)

	for name, client := range cm.clients {
		caps, err := client.GetCapabilities(ctx)
		if err != nil {
			// Log error but continue with other clients
			continue
		}
		// Convert from interfaces.Capabilities to Capabilities (alias)
		capabilities[name] = (*Capabilities)(caps)
	}

	return capabilities, nil
}

// GetClient returns a specific client by name
func (cm *ClientManager) GetClient(name string) (interfaces.AIClient, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	client, exists := cm.clients[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrClientNotFound, name)
	}

	return client, nil
}

// AddClient adds a new client with the given name and configuration
func (cm *ClientManager) AddClient(name string, config map[string]any) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Validate configuration
	if err := cm.factory.ValidateConfig(name, config); err != nil {
		return fmt.Errorf("invalid config for provider %s: %w", name, err)
	}

	client, err := cm.factory.CreateClient(name, config)
	if err != nil {
		return fmt.Errorf("failed to create client for provider %s: %w", name, err)
	}

	// Close existing client if it exists
	if existingClient, exists := cm.clients[name]; exists {
		_ = existingClient.Close()
	}

	cm.clients[name] = client
	return nil
}

// RemoveClient removes a client by name
func (cm *ClientManager) RemoveClient(name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	client, exists := cm.clients[name]
	if !exists {
		return fmt.Errorf("%w: %s", ErrClientNotFound, name)
	}

	_ = client.Close()
	delete(cm.clients, name)
	return nil
}

// GetHealthStatus returns the health status of all clients
func (cm *ClientManager) GetHealthStatus() map[string]*HealthStatus {
	return cm.healthChecker.GetHealthStatus()
}

// Close closes all clients and stops the health checker
func (cm *ClientManager) Close() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Stop health checker
	cm.healthChecker.Stop()

	// Close all clients
	var errors []error
	for name, client := range cm.clients {
		if err := client.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close client %s: %w", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors occurred while closing clients: %v", errors)
	}

	return nil
}

// defaultClientFactory is the default implementation of ClientFactory
type defaultClientFactory struct {
	providers map[string]func(config map[string]any) (AIClient, error)
}

// NewDefaultClientFactory creates a new default client factory
func NewDefaultClientFactory() (ClientFactory, error) {
	factory := &defaultClientFactory{
		providers: make(map[string]func(config map[string]any) (AIClient, error)),
	}

	// Register supported providers
	factory.registerProviders()

	return factory, nil
}

// registerProviders registers all supported AI providers
func (f *defaultClientFactory) registerProviders() {
	f.providers["openai"] = f.createOpenAIClient
	f.providers["anthropic"] = f.createAnthropicClient
	f.providers["local"] = f.createLocalClient
	// Register ollama as an alias for local provider for backward compatibility
	f.providers["ollama"] = f.createLocalClient
	// Register OpenAI-compatible providers
	f.providers["deepseek"] = f.createDeepSeekClient
	f.providers["moonshot"] = f.createMoonshotClient
	f.providers["zhipu"] = f.createZhipuClient
	f.providers["baichuan"] = f.createBaichuanClient
	f.providers["custom"] = f.createCustomOpenAIClient
}

// CreateClient creates a new AI client for the specified provider
func (f *defaultClientFactory) CreateClient(provider string, config map[string]any) (AIClient, error) {
	creator, exists := f.providers[provider]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotSupported, provider)
	}

	return creator(config)
}

// GetSupportedProviders returns a list of supported provider names
func (f *defaultClientFactory) GetSupportedProviders() []string {
	providers := make([]string, 0, len(f.providers))
	for provider := range f.providers {
		providers = append(providers, provider)
	}
	return providers
}

// ValidateConfig validates the configuration for a specific provider
func (f *defaultClientFactory) ValidateConfig(provider string, config map[string]any) error {
	_, exists := f.providers[provider]
	if !exists {
		return fmt.Errorf("%w: %s", ErrProviderNotSupported, provider)
	}

	// Basic validation - specific providers will implement more detailed validation
	if config == nil {
		return fmt.Errorf("%w: config is nil", ErrInvalidConfig)
	}

	return nil
}

// createOpenAIClient creates an OpenAI client from config
func (f *defaultClientFactory) createOpenAIClient(config map[string]any) (interfaces.AIClient, error) {
	openaiConfig := &openai.Config{}

	if apiKey, ok := config["api_key"].(string); ok {
		openaiConfig.APIKey = apiKey
	}
	if baseURL, ok := config["base_url"].(string); ok {
		openaiConfig.BaseURL = baseURL
	}
	if model, ok := config["model"].(string); ok {
		openaiConfig.Model = model
	}
	if orgID, ok := config["org_id"].(string); ok {
		openaiConfig.OrgID = orgID
	}
	if timeout, ok := config["timeout"].(time.Duration); ok {
		openaiConfig.Timeout = timeout
	} else if timeoutStr, ok := config["timeout"].(string); ok {
		if duration, err := time.ParseDuration(timeoutStr); err == nil {
			openaiConfig.Timeout = duration
		}
	}
	if maxTokens, ok := config["max_tokens"].(int); ok {
		openaiConfig.MaxTokens = maxTokens
	}

	return openai.NewClient(openaiConfig)
}

// createAnthropicClient creates an Anthropic client from config
func (f *defaultClientFactory) createAnthropicClient(config map[string]any) (interfaces.AIClient, error) {
	anthropicConfig := &anthropic.Config{}

	if apiKey, ok := config["api_key"].(string); ok {
		anthropicConfig.APIKey = apiKey
	}
	if baseURL, ok := config["base_url"].(string); ok {
		anthropicConfig.BaseURL = baseURL
	}
	if model, ok := config["model"].(string); ok {
		anthropicConfig.Model = model
	}
	if version, ok := config["version"].(string); ok {
		anthropicConfig.Version = version
	}
	if timeout, ok := config["timeout"].(time.Duration); ok {
		anthropicConfig.Timeout = timeout
	} else if timeoutStr, ok := config["timeout"].(string); ok {
		if duration, err := time.ParseDuration(timeoutStr); err == nil {
			anthropicConfig.Timeout = duration
		}
	}
	if maxTokens, ok := config["max_tokens"].(int); ok {
		anthropicConfig.MaxTokens = maxTokens
	}

	return anthropic.NewClient(anthropicConfig)
}

// createLocalClient creates a local client from config
func (f *defaultClientFactory) createLocalClient(config map[string]any) (interfaces.AIClient, error) {
	localConfig := &local.Config{}

	if baseURL, ok := config["base_url"].(string); ok {
		localConfig.BaseURL = baseURL
	}
	if model, ok := config["model"].(string); ok {
		localConfig.Model = model
	}
	if timeout, ok := config["timeout"].(time.Duration); ok {
		localConfig.Timeout = timeout
	} else if timeoutStr, ok := config["timeout"].(string); ok {
		if duration, err := time.ParseDuration(timeoutStr); err == nil {
			localConfig.Timeout = duration
		}
	}
	if maxTokens, ok := config["max_tokens"].(int); ok {
		localConfig.MaxTokens = maxTokens
	}
	if temperature, ok := config["temperature"].(float64); ok {
		localConfig.Temperature = temperature
	}

	return local.NewClient(localConfig)
}

// createDeepSeekClient creates a DeepSeek client using OpenAI-compatible interface
func (f *defaultClientFactory) createDeepSeekClient(config map[string]any) (interfaces.AIClient, error) {
	openaiConfig := &openai.Config{}

	if apiKey, ok := config["api_key"].(string); ok {
		openaiConfig.APIKey = apiKey
	}
	// DeepSeek API endpoint
	if baseURL, ok := config["base_url"].(string); ok {
		openaiConfig.BaseURL = baseURL
	} else {
		openaiConfig.BaseURL = "https://api.deepseek.com/v1"
	}
	if model, ok := config["model"].(string); ok {
		openaiConfig.Model = model
	} else {
		openaiConfig.Model = "deepseek-chat"
	}
	if timeout, ok := config["timeout"].(time.Duration); ok {
		openaiConfig.Timeout = timeout
	} else if timeoutStr, ok := config["timeout"].(string); ok {
		if duration, err := time.ParseDuration(timeoutStr); err == nil {
			openaiConfig.Timeout = duration
		}
	}
	if maxTokens, ok := config["max_tokens"].(int); ok {
		openaiConfig.MaxTokens = maxTokens
	}

	return openai.NewClient(openaiConfig)
}

// createMoonshotClient creates a Moonshot client using OpenAI-compatible interface
func (f *defaultClientFactory) createMoonshotClient(config map[string]any) (interfaces.AIClient, error) {
	openaiConfig := &openai.Config{}

	if apiKey, ok := config["api_key"].(string); ok {
		openaiConfig.APIKey = apiKey
	}
	if baseURL, ok := config["base_url"].(string); ok {
		openaiConfig.BaseURL = baseURL
	} else {
		openaiConfig.BaseURL = "https://api.moonshot.cn/v1"
	}
	if model, ok := config["model"].(string); ok {
		openaiConfig.Model = model
	} else {
		openaiConfig.Model = "moonshot-v1-8k"
	}
	if timeout, ok := config["timeout"].(time.Duration); ok {
		openaiConfig.Timeout = timeout
	} else if timeoutStr, ok := config["timeout"].(string); ok {
		if duration, err := time.ParseDuration(timeoutStr); err == nil {
			openaiConfig.Timeout = duration
		}
	}
	if maxTokens, ok := config["max_tokens"].(int); ok {
		openaiConfig.MaxTokens = maxTokens
	}

	return openai.NewClient(openaiConfig)
}

// createZhipuClient creates a Zhipu AI client using OpenAI-compatible interface
func (f *defaultClientFactory) createZhipuClient(config map[string]any) (interfaces.AIClient, error) {
	openaiConfig := &openai.Config{}

	if apiKey, ok := config["api_key"].(string); ok {
		openaiConfig.APIKey = apiKey
	}
	if baseURL, ok := config["base_url"].(string); ok {
		openaiConfig.BaseURL = baseURL
	} else {
		openaiConfig.BaseURL = "https://open.bigmodel.cn/api/paas/v4"
	}
	if model, ok := config["model"].(string); ok {
		openaiConfig.Model = model
	} else {
		openaiConfig.Model = "glm-4"
	}
	if timeout, ok := config["timeout"].(time.Duration); ok {
		openaiConfig.Timeout = timeout
	} else if timeoutStr, ok := config["timeout"].(string); ok {
		if duration, err := time.ParseDuration(timeoutStr); err == nil {
			openaiConfig.Timeout = duration
		}
	}
	if maxTokens, ok := config["max_tokens"].(int); ok {
		openaiConfig.MaxTokens = maxTokens
	}

	return openai.NewClient(openaiConfig)
}

// createBaichuanClient creates a Baichuan client using OpenAI-compatible interface
func (f *defaultClientFactory) createBaichuanClient(config map[string]any) (interfaces.AIClient, error) {
	openaiConfig := &openai.Config{}

	if apiKey, ok := config["api_key"].(string); ok {
		openaiConfig.APIKey = apiKey
	}
	if baseURL, ok := config["base_url"].(string); ok {
		openaiConfig.BaseURL = baseURL
	} else {
		openaiConfig.BaseURL = "https://api.baichuan-ai.com/v1"
	}
	if model, ok := config["model"].(string); ok {
		openaiConfig.Model = model
	} else {
		openaiConfig.Model = "Baichuan2-Turbo"
	}
	if timeout, ok := config["timeout"].(time.Duration); ok {
		openaiConfig.Timeout = timeout
	} else if timeoutStr, ok := config["timeout"].(string); ok {
		if duration, err := time.ParseDuration(timeoutStr); err == nil {
			openaiConfig.Timeout = duration
		}
	}
	if maxTokens, ok := config["max_tokens"].(int); ok {
		openaiConfig.MaxTokens = maxTokens
	}

	return openai.NewClient(openaiConfig)
}

// createCustomOpenAIClient creates a custom OpenAI-compatible client
func (f *defaultClientFactory) createCustomOpenAIClient(config map[string]any) (interfaces.AIClient, error) {
	openaiConfig := &openai.Config{}

	if apiKey, ok := config["api_key"].(string); ok {
		openaiConfig.APIKey = apiKey
	}
	if baseURL, ok := config["base_url"].(string); ok {
		openaiConfig.BaseURL = baseURL
	} else {
		return nil, fmt.Errorf("base_url is required for custom provider")
	}
	if model, ok := config["model"].(string); ok {
		openaiConfig.Model = model
	} else {
		return nil, fmt.Errorf("model is required for custom provider")
	}
	if timeout, ok := config["timeout"].(time.Duration); ok {
		openaiConfig.Timeout = timeout
	} else if timeoutStr, ok := config["timeout"].(string); ok {
		if duration, err := time.ParseDuration(timeoutStr); err == nil {
			openaiConfig.Timeout = duration
		}
	}
	if maxTokens, ok := config["max_tokens"].(int); ok {
		openaiConfig.MaxTokens = maxTokens
	}

	return openai.NewClient(openaiConfig)
}

// HealthChecker monitors the health of AI clients
type HealthChecker struct {
	interval     time.Duration
	clients      map[string]interfaces.AIClient
	healthStatus map[string]*HealthStatus
	mu           sync.RWMutex
	stopCh       chan struct{}
	stopped      bool
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(interval time.Duration) *HealthChecker {
	return &HealthChecker{
		interval:     interval,
		healthStatus: make(map[string]*HealthStatus),
		stopCh:       make(chan struct{}),
	}
}

// Start begins health checking for the given clients
func (hc *HealthChecker) Start(clients map[string]interfaces.AIClient) {
	hc.mu.Lock()
	hc.clients = clients
	hc.stopped = false
	hc.mu.Unlock()

	go hc.healthCheckLoop()
}

// Stop stops the health checker
func (hc *HealthChecker) Stop() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if !hc.stopped {
		close(hc.stopCh)
		hc.stopped = true
	}
}

// GetHealthStatus returns the current health status of all clients
func (hc *HealthChecker) GetHealthStatus() map[string]*HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	// Create a copy of the health status map
	status := make(map[string]*HealthStatus)
	for name, health := range hc.healthStatus {
		statusCopy := *health
		status[name] = &statusCopy
	}

	return status
}

// healthCheckLoop performs periodic health checks
func (hc *HealthChecker) healthCheckLoop() {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hc.performHealthChecks()
		case <-hc.stopCh:
			return
		}
	}
}

// performHealthChecks checks the health of all clients
func (hc *HealthChecker) performHealthChecks() {
	hc.mu.RLock()
	clients := make(map[string]interfaces.AIClient)
	for name, client := range hc.clients {
		clients[name] = client
	}
	hc.mu.RUnlock()

	for name, client := range clients {
		go hc.checkClientHealth(name, client)
	}
}

// checkClientHealth checks the health of a specific client
func (hc *HealthChecker) checkClientHealth(name string, client AIClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	health, err := client.HealthCheck(ctx)
	duration := time.Since(start)

	hc.mu.Lock()
	defer hc.mu.Unlock()

	if err != nil {
		hc.healthStatus[name] = &HealthStatus{
			Healthy:      false,
			Status:       fmt.Sprintf("Health check failed: %v", err),
			ResponseTime: duration,
			LastChecked:  time.Now(),
			Errors:       []string{err.Error()},
		}
	} else {
		hc.healthStatus[name] = health
	}
}
