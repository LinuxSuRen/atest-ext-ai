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

	"github.com/linuxsuren/atest-ext-ai/pkg/config"
	"github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
)

// CapabilitiesRequest defines the request structure for capability queries
type CapabilitiesRequest struct {
	IncludeModels    bool `json:"include_models"`
	IncludeDatabases bool `json:"include_databases"`
	IncludeFeatures  bool `json:"include_features"`
	CheckHealth      bool `json:"check_health"`
}

// CapabilitiesResponse defines the complete capability information for the AI plugin
type CapabilitiesResponse struct {
	Version     string                 `json:"version"`
	Models      []ModelCapability      `json:"models"`
	Databases   []DatabaseCapability   `json:"databases"`
	Features    []FeatureCapability    `json:"features"`
	Health      HealthStatusReport     `json:"health"`
	Limits      ResourceLimits         `json:"limits"`
	LastUpdated time.Time              `json:"last_updated"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ModelCapability represents the capabilities of an AI model
type ModelCapability struct {
	Name        string            `json:"name"`
	Provider    string            `json:"provider"`
	Available   bool              `json:"available"`
	Features    []string          `json:"features"`
	Limitations []string          `json:"limitations"`
	MaxTokens   int               `json:"max_tokens"`
	ContextSize int               `json:"context_size"`
	CostPer1K   *CostInfo         `json:"cost_per_1k,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// CostInfo represents pricing information for a model
type CostInfo struct {
	InputCost  float64 `json:"input_cost"`
	OutputCost float64 `json:"output_cost"`
	Currency   string  `json:"currency"`
}

// DatabaseCapability represents supported database types and features
type DatabaseCapability struct {
	Type        string   `json:"type"`
	Versions    []string `json:"versions"`
	Features    []string `json:"features"`
	Limitations []string `json:"limitations"`
	Supported   bool     `json:"supported"`
}

// FeatureCapability represents a specific feature and its status
type FeatureCapability struct {
	Name         string            `json:"name"`
	Enabled      bool              `json:"enabled"`
	Description  string            `json:"description"`
	Version      string            `json:"version"`
	Parameters   map[string]string `json:"parameters,omitempty"`
	Dependencies []string          `json:"dependencies,omitempty"`
}

// HealthStatusReport provides detailed health information
type HealthStatusReport struct {
	Overall    bool                  `json:"overall"`
	Components map[string]HealthInfo `json:"components"`
	Providers  map[string]HealthInfo `json:"providers"`
	Timestamp  time.Time             `json:"timestamp"`
}

// HealthInfo represents health information for a component
type HealthInfo struct {
	Status       string        `json:"status"`
	Healthy      bool          `json:"healthy"`
	ResponseTime time.Duration `json:"response_time"`
	LastCheck    time.Time     `json:"last_check"`
	Errors       []string      `json:"errors,omitempty"`
	Message      string        `json:"message,omitempty"`
}

// ResourceLimits defines the resource constraints and limits
type ResourceLimits struct {
	MaxConcurrentRequests int              `json:"max_concurrent_requests"`
	RateLimit             RateLimitInfo    `json:"rate_limit"`
	Memory                MemoryLimits     `json:"memory"`
	Processing            ProcessingLimits `json:"processing"`
}

// RateLimitInfo describes rate limiting constraints
type RateLimitInfo struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	RequestsPerHour   int `json:"requests_per_hour"`
	TokensPerMinute   int `json:"tokens_per_minute"`
	TokensPerHour     int `json:"tokens_per_hour"`
}

// MemoryLimits describes memory usage constraints
type MemoryLimits struct {
	MaxMemoryMB  int `json:"max_memory_mb"`
	CacheSizeMB  int `json:"cache_size_mb"`
	BufferSizeMB int `json:"buffer_size_mb"`
}

// ProcessingLimits describes processing constraints
type ProcessingLimits struct {
	MaxProcessingTimeSeconds int `json:"max_processing_time_seconds"`
	MaxQueueSize             int `json:"max_queue_size"`
	MaxRetryAttempts         int `json:"max_retry_attempts"`
}

// CapabilityDetector handles dynamic capability detection and reporting
type CapabilityDetector struct {
	config         config.AIConfig
	client         *Client
	cache          *capabilityCache
	healthChecker  *CapabilityHealthChecker
	mu             sync.RWMutex
	lastUpdate     time.Time
	updateInterval time.Duration
}

// capabilityCache provides caching for capability information
type capabilityCache struct {
	data      *CapabilitiesResponse
	mu        sync.RWMutex
	ttl       time.Duration
	timestamp time.Time
}

// CapabilityHealthChecker manages health checking for various components
type CapabilityHealthChecker struct {
	providers map[string]interfaces.AIClient
	timeout   time.Duration
	mu        sync.RWMutex
}

// NewCapabilityDetector creates a new capability detector
func NewCapabilityDetector(cfg config.AIConfig, client *Client) *CapabilityDetector {
	detector := &CapabilityDetector{
		config:         cfg,
		client:         client,
		updateInterval: 5 * time.Minute, // Default update interval
		cache: &capabilityCache{
			ttl: 5 * time.Minute, // Default cache TTL
		},
		healthChecker: &CapabilityHealthChecker{
			providers: make(map[string]interfaces.AIClient),
			timeout:   10 * time.Second,
		},
	}

	// Initialize health checker with available clients
	if client != nil {
		for name, client := range client.GetAllClients() {
			detector.healthChecker.providers[name] = client
		}
	}

	return detector
}

// GetCapabilities returns the comprehensive capability information
func (d *CapabilityDetector) GetCapabilities(ctx context.Context, req *CapabilitiesRequest) (*CapabilitiesResponse, error) {
	d.mu.RLock()
	// Check if we have cached data that's still valid
	if d.cache.isValid() {
		d.mu.RUnlock()
		return d.getCachedCapabilities(req)
	}
	d.mu.RUnlock()

	// Need to refresh capabilities
	d.mu.Lock()
	defer d.mu.Unlock()

	// Double-check after acquiring write lock
	if d.cache.isValid() {
		return d.getCachedCapabilities(req)
	}

	// Build fresh capability response
	response := &CapabilitiesResponse{
		Version:     "1.0.0",
		LastUpdated: time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	// Collect models if requested
	if req.IncludeModels {
		models, err := d.detectModelCapabilities(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to detect model capabilities: %w", err)
		}
		response.Models = models
	}

	// Collect database capabilities if requested
	if req.IncludeDatabases {
		response.Databases = d.detectDatabaseCapabilities()
	}

	// Collect feature capabilities if requested
	if req.IncludeFeatures {
		response.Features = d.detectFeatureCapabilities()
	}

	// Perform health checks if requested
	if req.CheckHealth {
		health, err := d.performHealthChecks(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to perform health checks: %w", err)
		}
		response.Health = *health
	}

	// Always include resource limits
	response.Limits = d.getResourceLimits()

	// Update cache
	d.cache.update(response)
	d.lastUpdate = time.Now()

	return response, nil
}

// detectModelCapabilities discovers available AI models and their capabilities
func (d *CapabilityDetector) detectModelCapabilities(ctx context.Context) ([]ModelCapability, error) {
	var capabilities []ModelCapability

	if d.client == nil {
		// Return default capabilities if no client available
		return []ModelCapability{
			{
				Name:        "basic",
				Provider:    "ollama",
				Available:   true,
				Features:    []string{"sql-generation", "text-generation"},
				Limitations: []string{"limited-model-capabilities"},
				MaxTokens:   4096,
				ContextSize: 4096,
			},
		}, nil
	}

	// Get all available clients
	clients := d.client.GetAllClients()
	for providerName, client := range clients {
		// Get capabilities from each provider
		clientCaps, err := client.GetCapabilities(ctx)
		if err != nil {
			// Log error but continue with other providers
			capabilities = append(capabilities, ModelCapability{
				Name:        providerName,
				Provider:    providerName,
				Available:   false,
				Limitations: []string{fmt.Sprintf("capability_detection_error: %v", err)},
			})
			continue
		}

		// Convert provider capabilities to our format
		for _, model := range clientCaps.Models {
			capability := ModelCapability{
				Name:        model.ID,
				Provider:    clientCaps.Provider,
				Available:   true,
				Features:    model.Capabilities,
				MaxTokens:   model.MaxTokens,
				ContextSize: model.MaxTokens,
				Metadata: map[string]string{
					"description": model.Description,
					"name":        model.Name,
				},
			}

			// Add cost information if available
			if model.InputCostPer1K > 0 || model.OutputCostPer1K > 0 {
				capability.CostPer1K = &CostInfo{
					InputCost:  model.InputCostPer1K,
					OutputCost: model.OutputCostPer1K,
					Currency:   "USD",
				}
			}

			capabilities = append(capabilities, capability)
		}
	}

	return capabilities, nil
}

// detectDatabaseCapabilities returns supported database types and features
func (d *CapabilityDetector) detectDatabaseCapabilities() []DatabaseCapability {
	// Static database capabilities - could be enhanced with dynamic detection
	return []DatabaseCapability{
		{
			Type:      "mysql",
			Versions:  []string{"5.7", "8.0", "8.1"},
			Features:  []string{"joins", "subqueries", "cte", "window-functions", "stored-procedures"},
			Supported: true,
		},
		{
			Type:      "postgresql",
			Versions:  []string{"12", "13", "14", "15", "16"},
			Features:  []string{"joins", "subqueries", "cte", "window-functions", "stored-procedures", "json-functions", "arrays"},
			Supported: true,
		},
		{
			Type:      "sqlite",
			Versions:  []string{"3.x"},
			Features:  []string{"joins", "subqueries", "cte", "window-functions", "json-functions"},
			Supported: true,
		},
		{
			Type:        "oracle",
			Versions:    []string{"11g", "12c", "19c", "21c"},
			Features:    []string{"joins", "subqueries", "cte", "window-functions", "stored-procedures", "pl-sql"},
			Supported:   false,
			Limitations: []string{"not-implemented"},
		},
		{
			Type:        "sqlserver",
			Versions:    []string{"2016", "2017", "2019", "2022"},
			Features:    []string{"joins", "subqueries", "cte", "window-functions", "stored-procedures", "t-sql"},
			Supported:   false,
			Limitations: []string{"not-implemented"},
		},
	}
}

// detectFeatureCapabilities returns available plugin features
func (d *CapabilityDetector) detectFeatureCapabilities() []FeatureCapability {
	features := []FeatureCapability{
		{
			Name:        "sql-generation",
			Enabled:     true,
			Description: "Generate SQL queries from natural language descriptions",
			Version:     "1.0.0",
			Parameters: map[string]string{
				"max_tokens":    "2000",
				"temperature":   "0.3",
				"database_type": "mysql,postgresql,sqlite",
			},
		},
		{
			Name:         "sql-optimization",
			Enabled:      false,
			Description:  "Optimize existing SQL queries for better performance",
			Version:      "0.9.0",
			Dependencies: []string{"sql-generation"},
		},
		{
			Name:        "sql-validation",
			Enabled:     true,
			Description: "Validate generated SQL queries for syntax and logical correctness",
			Version:     "1.0.0",
			Parameters: map[string]string{
				"strict_mode": "true",
			},
		},
		{
			Name:         "schema-analysis",
			Enabled:      false,
			Description:  "Analyze database schema and suggest improvements",
			Version:      "0.8.0",
			Dependencies: []string{"sql-generation", "sql-validation"},
		},
		{
			Name:        "query-explanation",
			Enabled:     true,
			Description: "Provide detailed explanations for generated SQL queries",
			Version:     "1.0.0",
		},
		{
			Name:        "multi-language-support",
			Enabled:     true,
			Description: "Support for multiple natural languages in queries",
			Version:     "1.0.0",
			Parameters: map[string]string{
				"supported_languages": "en,zh,es,fr,de,ja",
			},
		},
	}

	// Dynamically adjust feature status based on configuration
	if d.client != nil && d.client.GetPrimaryClient() != nil {
		// If we have a working AI client, enable more features
		for i := range features {
			if features[i].Name == "sql-optimization" && len(features[i].Dependencies) == 1 {
				features[i].Enabled = true
			}
		}
	}

	return features
}

// performHealthChecks executes health checks on all components
func (d *CapabilityDetector) performHealthChecks(ctx context.Context) (*HealthStatusReport, error) {
	report := &HealthStatusReport{
		Overall:    true,
		Components: make(map[string]HealthInfo),
		Providers:  make(map[string]HealthInfo),
		Timestamp:  time.Now(),
	}

	// Check component health
	report.Components["engine"] = d.checkEngineHealth(ctx)
	report.Components["cache"] = d.checkCacheHealth()
	report.Components["config"] = d.checkConfigHealth()

	// Check provider health
	d.healthChecker.mu.RLock()
	for name, client := range d.healthChecker.providers {
		report.Providers[name] = d.checkProviderHealth(ctx, client)
	}
	d.healthChecker.mu.RUnlock()

	// Determine overall health
	for _, health := range report.Components {
		if !health.Healthy {
			report.Overall = false
			break
		}
	}

	if report.Overall {
		for _, health := range report.Providers {
			if !health.Healthy {
				report.Overall = false
				break
			}
		}
	}

	return report, nil
}

// checkEngineHealth checks the health of the AI engine
func (d *CapabilityDetector) checkEngineHealth(ctx context.Context) HealthInfo {
	start := time.Now()

	if d.client == nil {
		return HealthInfo{
			Status:       "unavailable",
			Healthy:      false,
			ResponseTime: time.Since(start),
			LastCheck:    time.Now(),
			Errors:       []string{"no AI client available"},
			Message:      "AI client not initialized",
		}
	}

	primaryClient := d.client.GetPrimaryClient()
	if primaryClient == nil {
		return HealthInfo{
			Status:       "degraded",
			Healthy:      false,
			ResponseTime: time.Since(start),
			LastCheck:    time.Now(),
			Errors:       []string{"no primary client available"},
			Message:      "Primary AI client not available",
		}
	}

	return HealthInfo{
		Status:       "healthy",
		Healthy:      true,
		ResponseTime: time.Since(start),
		LastCheck:    time.Now(),
		Message:      "AI engine operational",
	}
}

// checkCacheHealth checks the health of the capability cache
func (d *CapabilityDetector) checkCacheHealth() HealthInfo {
	start := time.Now()

	d.cache.mu.RLock()
	defer d.cache.mu.RUnlock()

	return HealthInfo{
		Status:       "healthy",
		Healthy:      true,
		ResponseTime: time.Since(start),
		LastCheck:    time.Now(),
		Message:      "Capability cache operational",
	}
}

// checkConfigHealth checks the health of the configuration
func (d *CapabilityDetector) checkConfigHealth() HealthInfo {
	start := time.Now()

	var errors []string

	if d.config.DefaultService == "" {
		errors = append(errors, "no default service configured")
	}

	healthy := len(errors) == 0
	status := "healthy"
	message := "Configuration valid"

	if !healthy {
		status = "unhealthy"
		message = "Configuration issues detected"
	}

	return HealthInfo{
		Status:       status,
		Healthy:      healthy,
		ResponseTime: time.Since(start),
		LastCheck:    time.Now(),
		Errors:       errors,
		Message:      message,
	}
}

// checkProviderHealth checks the health of an AI provider
func (d *CapabilityDetector) checkProviderHealth(ctx context.Context, client interfaces.AIClient) HealthInfo {
	start := time.Now()

	// Create context with timeout
	healthCtx, cancel := context.WithTimeout(ctx, d.healthChecker.timeout)
	defer cancel()

	healthStatus, err := client.HealthCheck(healthCtx)
	responseTime := time.Since(start)

	if err != nil {
		return HealthInfo{
			Status:       "unhealthy",
			Healthy:      false,
			ResponseTime: responseTime,
			LastCheck:    time.Now(),
			Errors:       []string{err.Error()},
			Message:      "Health check failed",
		}
	}

	if healthStatus == nil {
		return HealthInfo{
			Status:       "unknown",
			Healthy:      false,
			ResponseTime: responseTime,
			LastCheck:    time.Now(),
			Errors:       []string{"no health status returned"},
			Message:      "Health status unavailable",
		}
	}

	status := "healthy"
	if !healthStatus.Healthy {
		status = "unhealthy"
	}

	return HealthInfo{
		Status:       status,
		Healthy:      healthStatus.Healthy,
		ResponseTime: responseTime,
		LastCheck:    time.Now(),
		Message:      healthStatus.Status,
	}
}

// getResourceLimits returns current resource limits
func (d *CapabilityDetector) getResourceLimits() ResourceLimits {
	return ResourceLimits{
		MaxConcurrentRequests: 10, // Could be configurable
		RateLimit: RateLimitInfo{
			RequestsPerMinute: 60,
			RequestsPerHour:   1000,
			TokensPerMinute:   100000,
			TokensPerHour:     500000,
		},
		Memory: MemoryLimits{
			MaxMemoryMB:  512,
			CacheSizeMB:  64,
			BufferSizeMB: 32,
		},
		Processing: ProcessingLimits{
			MaxProcessingTimeSeconds: 30,
			MaxQueueSize:             100,
			MaxRetryAttempts:         3,
		},
	}
}

// getCachedCapabilities returns filtered cached capabilities
func (d *CapabilityDetector) getCachedCapabilities(req *CapabilitiesRequest) (*CapabilitiesResponse, error) {
	d.cache.mu.RLock()
	defer d.cache.mu.RUnlock()

	if d.cache.data == nil {
		return nil, fmt.Errorf("no cached data available")
	}

	// Create a filtered response based on request
	response := &CapabilitiesResponse{
		Version:     d.cache.data.Version,
		LastUpdated: d.cache.data.LastUpdated,
		Metadata:    d.cache.data.Metadata,
	}

	if req.IncludeModels {
		response.Models = d.cache.data.Models
	}
	if req.IncludeDatabases {
		response.Databases = d.cache.data.Databases
	}
	if req.IncludeFeatures {
		response.Features = d.cache.data.Features
	}
	if req.CheckHealth {
		response.Health = d.cache.data.Health
	}

	// Always include limits
	response.Limits = d.cache.data.Limits

	return response, nil
}

// isValid checks if cached data is still valid
func (c *capabilityCache) isValid() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.data != nil && time.Since(c.timestamp) < c.ttl
}

// update updates the cached data
func (c *capabilityCache) update(data *CapabilitiesResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = data
	c.timestamp = time.Now()
}

// InvalidateCache forces a cache invalidation
func (d *CapabilityDetector) InvalidateCache() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.cache.mu.Lock()
	defer d.cache.mu.Unlock()

	d.cache.data = nil
	d.cache.timestamp = time.Time{}
}

// SetCacheTTL updates the cache TTL
func (d *CapabilityDetector) SetCacheTTL(ttl time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.cache.mu.Lock()
	defer d.cache.mu.Unlock()

	d.cache.ttl = ttl
}

// GetLastUpdate returns the timestamp of the last capability update
func (d *CapabilityDetector) GetLastUpdate() time.Time {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.lastUpdate
}
