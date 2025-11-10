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
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/ai/discovery"
	"github.com/linuxsuren/atest-ext-ai/pkg/ai/models"
	"github.com/linuxsuren/atest-ext-ai/pkg/ai/providers/universal"
	"github.com/linuxsuren/atest-ext-ai/pkg/config"
	"github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
	"github.com/linuxsuren/atest-ext-ai/pkg/logging"
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

// ProviderConfigInfo captures metadata about a provider's requirements.
type ProviderConfigInfo struct {
	RequiresAPIKey bool   `json:"requires_api_key"`
	ProviderType   string `json:"provider_type"`
}

// ProviderInfo represents information about an AI provider
type ProviderInfo struct {
	Name        string                   `json:"name"`
	Type        string                   `json:"type"`
	Available   bool                     `json:"available"`
	Endpoint    string                   `json:"endpoint"`
	Models      []interfaces.ModelInfo   `json:"models"`
	LastChecked time.Time                `json:"last_checked"`
	Config      ProviderConfigInfo       `json:"config"`
	Health      *interfaces.HealthStatus `json:"health,omitempty"`
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

// AddClientOptions configures how a client is added to the manager
type AddClientOptions struct {
	SkipHealthCheck    bool          // If true, skip health check during client addition
	HealthCheckTimeout time.Duration // Timeout for health check (default: 5 seconds)
}

// Manager is the unified manager for all AI clients.
// It merges the functionality of ClientManager and ProviderManager.
type Manager struct {
	clients   map[string]interfaces.AIClient
	config    config.AIConfig
	discovery *discovery.OllamaDiscovery
	mu        sync.RWMutex
}

// NewAIManager creates a new unified AI manager.
func NewAIManager(cfg config.AIConfig) (*Manager, error) {
	// The GUI drives provider configuration, so we only consume data from cfg.
	endpoint := discovery.DefaultOllamaEndpoint
	if ollamaSvc, ok := cfg.Services["ollama"]; ok {
		if ep := strings.TrimSpace(ollamaSvc.Endpoint); ep != "" {
			endpoint = ep
		}
	}

	manager := &Manager{
		clients:   make(map[string]interfaces.AIClient),
		config:    cfg,
		discovery: discovery.NewOllamaDiscovery(endpoint),
	}

	// Initialize configured clients
	if err := manager.initializeClients(); err != nil {
		return nil, fmt.Errorf("failed to initialize clients: %w", err)
	}

	return manager, nil
}

// ===== Client Management (from ClientManager) =====

// initializeClients creates clients for all enabled services
func (m *Manager) initializeClients() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, svc := range m.config.Services {
		if !svc.Enabled {
			continue
		}

		client, err := createClient(name, svc)
		if err != nil {
			return fmt.Errorf("failed to create client %s: %w", name, err)
		}

		m.clients[name] = client
	}

	return nil
}

// Generate executes an AI generation request with inline retry logic
func (m *Manager) Generate(ctx context.Context, req *interfaces.GenerateRequest) (*interfaces.GenerateResponse, error) {
	var lastErr error
	maxAttempts := 3

	// Apply retry configuration if available
	if m.config.Retry.MaxAttempts > 0 {
		maxAttempts = m.config.Retry.MaxAttempts
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Calculate backoff delay for retry attempts
		if attempt > 0 {
			delay := calculateBackoff(attempt, m.config.Retry)

			select {
			case <-time.After(delay):
				// Continue with retry
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		// Select a healthy client
		client := m.selectHealthyClient()
		if client == nil {
			lastErr = ErrNoHealthyClients
			continue
		}

		// Execute the generation request
		resp, err := client.Generate(ctx, req)
		if err != nil {
			// Check if error is retryable
			if !isRetryableError(err) {
				return nil, err
			}
			lastErr = err
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("all retry attempts failed: %w", lastErr)
}

// selectHealthyClient selects the best available client
func (m *Manager) selectHealthyClient() interfaces.AIClient {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try default service first
	if m.config.DefaultService != "" {
		if client, ok := m.clients[m.config.DefaultService]; ok {
			return client
		}
	}

	// Return any available client
	for _, client := range m.clients {
		return client
	}

	return nil
}

// GetClient returns a specific client by name
func (m *Manager) GetClient(name string) (interfaces.AIClient, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrClientNotFound, name)
	}

	return client, nil
}

// GetAllClients returns all available clients
func (m *Manager) GetAllClients() map[string]interfaces.AIClient {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a copy to avoid concurrent access issues
	clients := make(map[string]interfaces.AIClient)
	for name, client := range m.clients {
		clients[name] = client
	}

	return clients
}

// GetPrimaryClient returns the primary (default) client
func (m *Manager) GetPrimaryClient() interfaces.AIClient {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try to get default service client
	if m.config.DefaultService != "" {
		if client, ok := m.clients[m.config.DefaultService]; ok {
			return client
		}
	}

	// Return any available client as fallback
	for _, client := range m.clients {
		return client
	}

	return nil
}

// AddClient adds a new client with the given configuration
func (m *Manager) AddClient(ctx context.Context, name string, svc config.AIService, opts *AddClientOptions) error {
	// Set default options if not provided
	if opts == nil {
		opts = &AddClientOptions{
			SkipHealthCheck:    false,
			HealthCheckTimeout: 5 * time.Second,
		}
	}

	// Set default timeout if not specified
	if opts.HealthCheckTimeout == 0 {
		opts.HealthCheckTimeout = 5 * time.Second
	}

	client, err := createClient(name, svc)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Optional health check
	if !opts.SkipHealthCheck {
		healthCtx, cancel := context.WithTimeout(ctx, opts.HealthCheckTimeout)
		defer cancel()

		health, err := client.HealthCheck(healthCtx)
		if err != nil {
			logging.Logger.Warn("Health check failed during client addition",
				"client", name,
				"error", err,
				"action", "client will be added but may be unhealthy")
			// Don't return error, just log warning
		} else if health != nil && !health.Healthy {
			logging.Logger.Warn("Client added but reports unhealthy status",
				"client", name,
				"status", health.Status)
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Close old client if exists
	if oldClient, exists := m.clients[name]; exists {
		if err := oldClient.Close(); err != nil {
			logging.Logger.Warn("Failed to close existing AI client",
				"client", name,
				"error", err)
		}
	}

	m.clients[name] = client
	logging.Logger.Info("AI client added successfully",
		"client", name,
		"skip_health_check", opts.SkipHealthCheck)

	return nil
}

// RemoveClient removes a client
func (m *Manager) RemoveClient(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[name]
	if !exists {
		return fmt.Errorf("%w: %s", ErrClientNotFound, name)
	}

	if err := client.Close(); err != nil {
		logging.Logger.Warn("Failed to close AI client",
			"client", name,
			"error", err)
	}
	delete(m.clients, name)
	return nil
}

// ===== Provider Discovery (from ProviderManager) =====

// DiscoverProviders discovers available AI providers
func (m *Manager) DiscoverProviders(ctx context.Context) ([]*ProviderInfo, error) {
	var providers []*ProviderInfo

	// Check for Ollama
	if m.discovery.IsAvailable(ctx) {
		endpoint := m.discovery.GetBaseURL()

		// Create temporary Ollama client for discovery
		config := &universal.Config{
			Provider:  "ollama",
			Endpoint:  endpoint,
			Model:     "llama2",
			MaxTokens: 4096,
		}

		client, err := universal.NewUniversalClient(config)
		if err == nil {
			// Get models
			var models []interfaces.ModelInfo
			if caps, err := client.GetCapabilities(ctx); err == nil {
				models = caps.Models
			}

			provider := &ProviderInfo{
				Name:        "ollama",
				Type:        "local",
				Available:   true,
				Endpoint:    endpoint,
				Models:      models,
				LastChecked: time.Now(),
				Config: ProviderConfigInfo{
					ProviderType:   "local",
					RequiresAPIKey: false,
				},
			}

			providers = append(providers, provider)
			if err := client.Close(); err != nil {
				logging.Logger.Warn("Failed to close discovery client",
					"provider", provider.Name,
					"error", err)
			}
		}
	}

	// Add online providers
	providers = append(providers, m.getOnlineProviders()...)

	return providers, nil
}

// GetModels returns models for a specific provider
func (m *Manager) GetModels(ctx context.Context, providerName string) ([]interfaces.ModelInfo, error) {
	// Normalize provider name (local -> ollama)
	providerName = normalizeProviderName(providerName)

	m.mu.RLock()
	client, exists := m.clients[providerName]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("provider %s not found", providerName)
	}

	caps, err := client.GetCapabilities(ctx)
	if err != nil {
		return nil, err
	}

	return caps.Models, nil
}

// TestConnection tests the connection to a provider
func (m *Manager) TestConnection(ctx context.Context, cfg *universal.Config) (*ConnectionTestResult, error) {
	start := time.Now()

	if cfg == nil {
		return &ConnectionTestResult{
			Success:      false,
			Message:      "Invalid configuration",
			ResponseTime: time.Since(start),
			Error:        "configuration cannot be nil",
		}, nil
	}

	client, err := universal.NewUniversalClient(cfg)
	if err != nil {
		return &ConnectionTestResult{
			Success:      false,
			Message:      "Failed to create client",
			ResponseTime: time.Since(start),
			Provider:     cfg.Provider,
			Error:        err.Error(),
		}, nil
	}
	defer func() {
		if err := client.Close(); err != nil {
			logging.Logger.Warn("Failed to close test connection client",
				"provider", cfg.Provider,
				"error", err)
		}
	}()

	health, err := client.HealthCheck(ctx)
	if err != nil {
		return &ConnectionTestResult{
			Success:      false,
			Message:      "Health check failed",
			ResponseTime: time.Since(start),
			Provider:     cfg.Provider,
			Model:        cfg.Model,
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
		Provider:     cfg.Provider,
		Model:        cfg.Model,
	}, nil
}

// ===== On-Demand Health Checking =====

// HealthCheck checks health of a specific provider
func (m *Manager) HealthCheck(ctx context.Context, provider string) (*interfaces.HealthStatus, error) {
	provider = normalizeProviderName(provider)

	m.mu.RLock()
	client, exists := m.clients[provider]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("provider not found: %s", provider)
	}

	return client.HealthCheck(ctx)
}

// HealthCheckAll checks health of all providers
func (m *Manager) HealthCheckAll(ctx context.Context) map[string]*interfaces.HealthStatus {
	m.mu.RLock()
	clients := make(map[string]interfaces.AIClient)
	for name, client := range m.clients {
		clients[name] = client
	}
	m.mu.RUnlock()

	results := make(map[string]*interfaces.HealthStatus)

	// Check each client concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex

	for name, client := range clients {
		wg.Add(1)

		go func(name string, client interfaces.AIClient) {
			defer wg.Done()

			status, err := client.HealthCheck(ctx)
			if err != nil {
				status = &interfaces.HealthStatus{
					Healthy: false,
					Status:  err.Error(),
				}
			}

			mu.Lock()
			results[name] = status
			mu.Unlock()
		}(name, client)
	}

	wg.Wait()
	return results
}

// Close closes all clients
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errors []error
	for name, client := range m.clients {
		if err := client.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close client %s: %w", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors occurred while closing clients: %v", errors)
	}

	return nil
}

// ===== Helper Functions =====

// createClient creates a client based on provider name and configuration
func createClient(provider string, cfg config.AIService) (interfaces.AIClient, error) {
	// Normalize provider name
	provider = normalizeProviderName(provider)

	switch provider {
	case "openai", "deepseek", "custom":
		return createOpenAICompatibleClient(provider, cfg)

	case "ollama":
		return createOllamaClient(cfg)

	default:
		return nil, fmt.Errorf("%w: %s", ErrProviderNotSupported, provider)
	}
}

// createOpenAICompatibleClient creates an OpenAI-compatible client
func createOpenAICompatibleClient(provider string, cfg config.AIService) (interfaces.AIClient, error) {
	normalized := strings.ToLower(provider)

	uniCfg := &universal.Config{
		Provider:  normalized,
		Endpoint:  normalizeProviderEndpoint(normalized, cfg.Endpoint),
		APIKey:    cfg.APIKey,
		Model:     cfg.Model,
		MaxTokens: cfg.MaxTokens,
		Timeout:   cfg.Timeout.Value(),
	}

	if uniCfg.Endpoint == "" {
		if endpoint := models.EndpointForProvider(normalized); endpoint != "" {
			uniCfg.Endpoint = endpoint
		} else if normalized == "custom" {
			return nil, fmt.Errorf("endpoint is required for custom provider")
		}
	}

	return universal.NewUniversalClient(uniCfg)
}

// createOllamaClient creates an Ollama client
func createOllamaClient(cfg config.AIService) (interfaces.AIClient, error) {
	config := &universal.Config{
		Provider:  "ollama",
		Endpoint:  cfg.Endpoint,
		Model:     cfg.Model,
		MaxTokens: cfg.MaxTokens,
		Timeout:   cfg.Timeout.Value(),
	}

	// Default endpoint
	if config.Endpoint == "" {
		config.Endpoint = "http://localhost:11434"
	}

	return universal.NewUniversalClient(config)
}

// normalizeProviderName normalizes provider name (local -> ollama)
func normalizeProviderName(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "local" {
		return "ollama"
	}
	return provider
}

// getOnlineProviders returns predefined online providers
func (m *Manager) getOnlineProviders() []*ProviderInfo {
	catalog, err := models.GetCatalog()
	if err != nil {
		logging.Logger.Warn("Failed to load model catalog", "error", err)
		return nil
	}

	var providers []*ProviderInfo
	for _, name := range catalog.ProviderNames() {
		entry, ok := catalog.Provider(name)
		if !ok {
			continue
		}

		providerType := entry.Category
		if providerType == "" {
			providerType = "cloud"
		}
		if providerType != "cloud" && providerType != "online" {
			continue
		}

		providers = append(providers, &ProviderInfo{
			Name:        entry.Name,
			Type:        providerType,
			Available:   true,
			Endpoint:    entry.Endpoint,
			Models:      entry.Models,
			LastChecked: time.Now(),
			Config: ProviderConfigInfo{
				RequiresAPIKey: entry.RequiresAPIKey,
				ProviderType:   providerType,
			},
		})
	}

	return providers
}

// ===== Retry Logic =====

// calculateBackoff calculates exponential backoff delay
func calculateBackoff(attempt int, retryCfg config.RetryConfig) time.Duration {
	if attempt == 0 {
		return 0
	}

	// Use configured values or defaults
	baseDelay := 1 * time.Second
	maxDelay := 10 * time.Second
	multiplier := 2.0
	if retryCfg.InitialDelay.Duration > 0 {
		baseDelay = retryCfg.InitialDelay.Duration
	}
	if retryCfg.MaxDelay.Duration > 0 {
		maxDelay = retryCfg.MaxDelay.Duration
	}
	if retryCfg.Multiplier > 0 {
		multiplier = float64(retryCfg.Multiplier)
	}
	jitter := retryCfg.Jitter

	// Calculate exponential backoff
	delay := baseDelay
	for i := 1; i < attempt; i++ {
		delay = time.Duration(float64(delay) * multiplier)
		if delay > maxDelay {
			delay = maxDelay
			break
		}
	}

	// Add jitter
	if jitter {
		jitterRange := delay / 4
		if jitterRange > 0 {
			rangeLimit := big.NewInt(int64(jitterRange))
			n, err := cryptorand.Int(cryptorand.Reader, rangeLimit)
			if err != nil {
				logging.Logger.Debug("failed to generate crypto jitter, using deterministic midpoint", "error", err)
				delay += jitterRange / 2
			} else {
				delay += time.Duration(n.Int64())
			}
		}
	}

	return delay
}

// isRetryableError determines if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Context errors are not retryable
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Network errors are retryable
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// DNS errors are retryable
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	// Connection errors are retryable
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Op == "dial" {
		return true
	}

	// System call errors
	var syscallErr *syscall.Errno
	if errors.As(err, &syscallErr) {
		switch *syscallErr {
		case syscall.ECONNREFUSED, syscall.ECONNRESET, syscall.ETIMEDOUT:
			return true
		}
	}

	// Check error message for retryable patterns
	errMsg := strings.ToLower(err.Error())

	// Retryable errors
	retryablePatterns := []string{
		"rate limit", "too many requests", "quota exceeded",
		"service unavailable", "bad gateway", "gateway timeout",
		"connection refused", "connection reset",
		"500", "502", "503", "504", "429",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	// Non-retryable errors
	nonRetryablePatterns := []string{
		"unauthorized", "forbidden", "invalid api key",
		"authentication failed", "bad request", "malformed",
		"400", "401", "403", "404",
	}

	for _, pattern := range nonRetryablePatterns {
		if strings.Contains(errMsg, pattern) {
			return false
		}
	}

	// Default: not retryable
	return false
}
