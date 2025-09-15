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

package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
)

// mockAIClient is a mock implementation of AIClient for testing
type mockAIClient struct {
	name         string
	healthy      bool
	generateErr  error
	capabilities *interfaces.Capabilities
}

func (m *mockAIClient) Generate(ctx context.Context, req *interfaces.GenerateRequest) (*interfaces.GenerateResponse, error) {
	if m.generateErr != nil {
		return nil, m.generateErr
	}

	return &interfaces.GenerateResponse{
		Text:            "Mock response for: " + req.Prompt,
		Model:           "mock-model",
		ProcessingTime:  10 * time.Millisecond,
		RequestID:       "mock-request-id",
		ConfidenceScore: 0.9,
		Usage: interfaces.TokenUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}, nil
}

func (m *mockAIClient) GetCapabilities(ctx context.Context) (*interfaces.Capabilities, error) {
	if m.capabilities != nil {
		return m.capabilities, nil
	}

	return &interfaces.Capabilities{
		Provider:  m.name,
		MaxTokens: 4096,
		Models: []interfaces.ModelInfo{
			{
				ID:          "mock-model",
				Name:        "Mock Model",
				Description: "Mock model for testing",
				MaxTokens:   4096,
			},
		},
		Features: []interfaces.Feature{
			{
				Name:    "generation",
				Enabled: true,
			},
		},
	}, nil
}

func (m *mockAIClient) HealthCheck(ctx context.Context) (*interfaces.HealthStatus, error) {
	return &interfaces.HealthStatus{
		Healthy:      m.healthy,
		Status:       "Mock status",
		ResponseTime: 5 * time.Millisecond,
		LastChecked:  time.Now(),
	}, nil
}

func (m *mockAIClient) Close() error {
	return nil
}

// mockClientFactory is a mock implementation of ClientFactory for testing
type mockClientFactory struct {
	clients map[string]*mockAIClient
}

func newMockClientFactory() *mockClientFactory {
	return &mockClientFactory{
		clients: make(map[string]*mockAIClient),
	}
}

func (f *mockClientFactory) CreateClient(provider string, config map[string]any) (interfaces.AIClient, error) {
	if client, exists := f.clients[provider]; exists {
		return client, nil
	}
	return nil, ErrProviderNotSupported
}

func (f *mockClientFactory) GetSupportedProviders() []string {
	var providers []string
	for provider := range f.clients {
		providers = append(providers, provider)
	}
	return providers
}

func (f *mockClientFactory) ValidateConfig(provider string, config map[string]any) error {
	if _, exists := f.clients[provider]; !exists {
		return ErrProviderNotSupported
	}
	return nil
}

func (f *mockClientFactory) AddMockClient(name string, client *mockAIClient) {
	f.clients[name] = client
}

func TestClientManager_NewClientManager(t *testing.T) {
	config := &AIServiceConfig{
		Providers: []ProviderConfig{
			{
				Name:     "mock1",
				Enabled:  true,
				Priority: 1,
				Config:   map[string]any{},
			},
		},
		LoadBalancer: LoadBalancerConfig{
			Strategy:            "round_robin",
			HealthCheckInterval: 30 * time.Second,
		},
		Retry: RetryConfig{
			MaxAttempts:       3,
			BaseDelay:         time.Second,
			MaxDelay:          30 * time.Second,
			BackoffMultiplier: 2.0,
		},
		CircuitBreaker: CircuitBreakerConfig{
			FailureThreshold: 5,
			ResetTimeout:     60 * time.Second,
		},
	}

	_, err := NewClientManager(config)
	if err == nil {
		t.Errorf("Expected error for unsupported provider, got nil")
	}
}

func TestClientManager_Generate(t *testing.T) {
	// Create mock factory with clients
	factory := newMockClientFactory()
	factory.AddMockClient("mock1", &mockAIClient{
		name:    "mock1",
		healthy: true,
	})

	// Create load balancer
	loadBalancer := NewDefaultLoadBalancer(LoadBalancerConfig{
		Strategy:            "round_robin",
		HealthCheckInterval: 30 * time.Second,
	})

	// Register mock client with load balancer
	mockClient := factory.clients["mock1"]
	if lb, ok := loadBalancer.(*defaultLoadBalancer); ok {
		lb.RegisterClient("mock1", mockClient)
	}

	// Create client manager
	manager := &ClientManager{
		clients:       map[string]interfaces.AIClient{"mock1": mockClient},
		factory:       factory,
		loadBalancer:  loadBalancer,
		retryManager:  NewDefaultRetryManager(RetryConfig{MaxAttempts: 3}),
		circuitBreaker: NewDefaultCircuitBreaker(CircuitBreakerConfig{FailureThreshold: 5}),
		healthChecker: NewHealthChecker(30 * time.Second),
	}

	req := &GenerateRequest{
		Prompt:    "Hello, world!",
		MaxTokens: 100,
	}

	ctx := context.Background()
	response, err := manager.Generate(ctx, req)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	if response.Text != "Mock response for: Hello, world!" {
		t.Errorf("Unexpected response text: %s", response.Text)
	}

	if response.Usage.TotalTokens != 30 {
		t.Errorf("Unexpected token usage: %d", response.Usage.TotalTokens)
	}
}

func TestClientManager_Generate_WithError(t *testing.T) {
	// Create mock factory with failing client
	factory := newMockClientFactory()
	mockErr := errors.New("mock error")
	factory.AddMockClient("mock1", &mockAIClient{
		name:        "mock1",
		healthy:     true,
		generateErr: mockErr,
	})

	// Create load balancer
	loadBalancer := NewDefaultLoadBalancer(LoadBalancerConfig{
		Strategy:            "round_robin",
		HealthCheckInterval: 30 * time.Second,
	})

	// Register mock client with load balancer
	mockClient := factory.clients["mock1"]
	if lb, ok := loadBalancer.(*defaultLoadBalancer); ok {
		lb.RegisterClient("mock1", mockClient)
	}

	// Create client manager
	manager := &ClientManager{
		clients:       map[string]interfaces.AIClient{"mock1": mockClient},
		factory:       factory,
		loadBalancer:  loadBalancer,
		retryManager:  NewDefaultRetryManager(RetryConfig{MaxAttempts: 1}), // Don't retry
		circuitBreaker: NewDefaultCircuitBreaker(CircuitBreakerConfig{FailureThreshold: 5}),
		healthChecker: NewHealthChecker(30 * time.Second),
	}

	req := &GenerateRequest{
		Prompt: "Hello, world!",
	}

	ctx := context.Background()
	_, err := manager.Generate(ctx, req)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !errors.Is(err, mockErr) {
		t.Errorf("Expected mock error, got: %v", err)
	}
}

func TestClientManager_GetCapabilities(t *testing.T) {
	// Create mock factory with clients
	factory := newMockClientFactory()
	factory.AddMockClient("mock1", &mockAIClient{
		name:    "mock1",
		healthy: true,
	})
	factory.AddMockClient("mock2", &mockAIClient{
		name:    "mock2",
		healthy: true,
	})

	// Create client manager
	manager := &ClientManager{
		clients: map[string]interfaces.AIClient{
			"mock1": factory.clients["mock1"],
			"mock2": factory.clients["mock2"],
		},
		factory:       factory,
		healthChecker: NewHealthChecker(30 * time.Second),
	}

	ctx := context.Background()
	capabilities, err := manager.GetCapabilities(ctx)
	if err != nil {
		t.Fatalf("GetCapabilities failed: %v", err)
	}

	if len(capabilities) != 2 {
		t.Errorf("Expected 2 capabilities, got %d", len(capabilities))
	}

	if _, exists := capabilities["mock1"]; !exists {
		t.Error("Expected mock1 capabilities")
	}

	if _, exists := capabilities["mock2"]; !exists {
		t.Error("Expected mock2 capabilities")
	}
}

func TestClientManager_AddRemoveClient(t *testing.T) {
	// Create mock factory
	factory := newMockClientFactory()
	factory.AddMockClient("mock1", &mockAIClient{
		name:    "mock1",
		healthy: true,
	})

	// Create client manager
	config := &AIServiceConfig{
		Providers: []ProviderConfig{},
		LoadBalancer: LoadBalancerConfig{
			Strategy:            "round_robin",
			HealthCheckInterval: 30 * time.Second,
		},
		Retry:          RetryConfig{MaxAttempts: 3},
		CircuitBreaker: CircuitBreakerConfig{FailureThreshold: 5},
	}

	manager := &ClientManager{
		clients:       make(map[string]AIClient),
		factory:       factory,
		loadBalancer:  NewDefaultLoadBalancer(config.LoadBalancer),
		retryManager:  NewDefaultRetryManager(config.Retry),
		circuitBreaker: NewDefaultCircuitBreaker(config.CircuitBreaker),
		config:        config,
		healthChecker: NewHealthChecker(config.LoadBalancer.HealthCheckInterval),
	}

	// Add client
	err := manager.AddClient("mock1", map[string]any{})
	if err != nil {
		t.Fatalf("AddClient failed: %v", err)
	}

	// Verify client was added
	client, err := manager.GetClient("mock1")
	if err != nil {
		t.Fatalf("GetClient failed: %v", err)
	}
	if client == nil {
		t.Fatal("Client is nil")
	}

	// Remove client
	err = manager.RemoveClient("mock1")
	if err != nil {
		t.Fatalf("RemoveClient failed: %v", err)
	}

	// Verify client was removed
	_, err = manager.GetClient("mock1")
	if !errors.Is(err, ErrClientNotFound) {
		t.Errorf("Expected ErrClientNotFound, got: %v", err)
	}
}

func TestDefaultClientFactory(t *testing.T) {
	factory, err := NewDefaultClientFactory()
	if err != nil {
		t.Fatalf("NewDefaultClientFactory failed: %v", err)
	}

	// Test supported providers
	providers := factory.GetSupportedProviders()
	expectedProviders := map[string]bool{
		"openai":    true,
		"anthropic": true,
		"local":     true,
	}

	if len(providers) != len(expectedProviders) {
		t.Errorf("Expected %d providers, got %d", len(expectedProviders), len(providers))
	}

	for _, provider := range providers {
		if !expectedProviders[provider] {
			t.Errorf("Unexpected provider: %s", provider)
		}
	}

	// Test validation
	err = factory.ValidateConfig("openai", map[string]any{"api_key": "test"})
	if err != nil {
		t.Errorf("ValidateConfig failed for valid config: %v", err)
	}

	err = factory.ValidateConfig("unsupported", map[string]any{})
	if !errors.Is(err, ErrProviderNotSupported) {
		t.Errorf("Expected ErrProviderNotSupported, got: %v", err)
	}

	err = factory.ValidateConfig("openai", nil)
	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("Expected ErrInvalidConfig, got: %v", err)
	}
}

func TestCircuitState_String(t *testing.T) {
	tests := []struct {
		state    CircuitState
		expected string
	}{
		{CircuitClosed, "closed"},
		{CircuitOpen, "open"},
		{CircuitHalfOpen, "half-open"},
		{CircuitState(999), "unknown"},
	}

	for _, test := range tests {
		if result := test.state.String(); result != test.expected {
			t.Errorf("CircuitState(%d).String() = %s, expected %s", test.state, result, test.expected)
		}
	}
}