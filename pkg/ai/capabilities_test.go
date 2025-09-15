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
	"testing"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCapabilityDetector(t *testing.T) {
	cfg := config.AIConfig{
		DefaultService: "test",
		Services: map[string]config.AIService{
			"test": {
				Enabled:  true,
				Provider: "local",
			},
		},
	}

	detector := NewCapabilityDetector(cfg, nil)

	assert.NotNil(t, detector)
	assert.NotNil(t, detector.cache)
	assert.NotNil(t, detector.healthChecker)
	assert.Equal(t, 5*time.Minute, detector.updateInterval)
	assert.Equal(t, cfg, detector.config)
}

func TestCapabilityDetector_GetCapabilities_BasicRequest(t *testing.T) {
	cfg := config.AIConfig{
		DefaultService: "test",
		Services: map[string]config.AIService{
			"test": {
				Enabled:  true,
				Provider: "local",
			},
		},
	}

	detector := NewCapabilityDetector(cfg, nil)
	ctx := context.Background()

	req := &CapabilitiesRequest{
		IncludeModels:    true,
		IncludeDatabases: true,
		IncludeFeatures:  true,
		CheckHealth:      false,
	}

	capabilities, err := detector.GetCapabilities(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, capabilities)
	assert.Equal(t, "1.0.0", capabilities.Version)
	assert.NotEmpty(t, capabilities.Models)
	assert.NotEmpty(t, capabilities.Databases)
	assert.NotEmpty(t, capabilities.Features)
	assert.NotEmpty(t, capabilities.Limits)
}

func TestCapabilityDetector_GetCapabilities_DatabasesOnly(t *testing.T) {
	cfg := config.AIConfig{
		DefaultService: "test",
	}

	detector := NewCapabilityDetector(cfg, nil)
	ctx := context.Background()

	req := &CapabilitiesRequest{
		IncludeModels:    false,
		IncludeDatabases: true,
		IncludeFeatures:  false,
		CheckHealth:      false,
	}

	capabilities, err := detector.GetCapabilities(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, capabilities)
	assert.Empty(t, capabilities.Models)
	assert.NotEmpty(t, capabilities.Databases)
	assert.Empty(t, capabilities.Features)

	// Check that we have expected databases
	dbTypes := make(map[string]bool)
	for _, db := range capabilities.Databases {
		dbTypes[db.Type] = true
	}

	assert.True(t, dbTypes["mysql"])
	assert.True(t, dbTypes["postgresql"])
	assert.True(t, dbTypes["sqlite"])
}

func TestCapabilityDetector_GetCapabilities_FeaturesOnly(t *testing.T) {
	cfg := config.AIConfig{
		DefaultService: "test",
	}

	detector := NewCapabilityDetector(cfg, nil)
	ctx := context.Background()

	req := &CapabilitiesRequest{
		IncludeModels:    false,
		IncludeDatabases: false,
		IncludeFeatures:  true,
		CheckHealth:      false,
	}

	capabilities, err := detector.GetCapabilities(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, capabilities)
	assert.Empty(t, capabilities.Models)
	assert.Empty(t, capabilities.Databases)
	assert.NotEmpty(t, capabilities.Features)

	// Check that we have expected features
	featureNames := make(map[string]bool)
	for _, feature := range capabilities.Features {
		featureNames[feature.Name] = true
	}

	assert.True(t, featureNames["sql-generation"])
	assert.True(t, featureNames["sql-validation"])
	assert.True(t, featureNames["query-explanation"])
}

func TestCapabilityDetector_DetectDatabaseCapabilities(t *testing.T) {
	detector := NewCapabilityDetector(config.AIConfig{}, nil)

	databases := detector.detectDatabaseCapabilities()

	assert.NotEmpty(t, databases)

	// Verify that supported databases are present and configured correctly
	dbMap := make(map[string]DatabaseCapability)
	for _, db := range databases {
		dbMap[db.Type] = db
	}

	// Test MySQL
	mysql := dbMap["mysql"]
	assert.True(t, mysql.Supported)
	assert.Contains(t, mysql.Versions, "8.0")
	assert.Contains(t, mysql.Features, "joins")
	assert.Contains(t, mysql.Features, "subqueries")

	// Test PostgreSQL
	postgresql := dbMap["postgresql"]
	assert.True(t, postgresql.Supported)
	assert.Contains(t, postgresql.Versions, "15")
	assert.Contains(t, postgresql.Features, "cte")
	assert.Contains(t, postgresql.Features, "json-functions")

	// Test SQLite
	sqlite := dbMap["sqlite"]
	assert.True(t, sqlite.Supported)
	assert.Contains(t, sqlite.Versions, "3.x")
	assert.Contains(t, sqlite.Features, "window-functions")
}

func TestCapabilityDetector_DetectFeatureCapabilities(t *testing.T) {
	detector := NewCapabilityDetector(config.AIConfig{}, nil)

	features := detector.detectFeatureCapabilities()

	assert.NotEmpty(t, features)

	// Verify expected features
	featureMap := make(map[string]FeatureCapability)
	for _, feature := range features {
		featureMap[feature.Name] = feature
	}

	// Test SQL generation feature
	sqlGen := featureMap["sql-generation"]
	assert.True(t, sqlGen.Enabled)
	assert.Equal(t, "1.0.0", sqlGen.Version)
	assert.NotEmpty(t, sqlGen.Description)
	assert.NotEmpty(t, sqlGen.Parameters)

	// Test SQL validation feature
	sqlVal := featureMap["sql-validation"]
	assert.True(t, sqlVal.Enabled)
	assert.Contains(t, sqlVal.Parameters, "strict_mode")

	// Test multi-language support
	multiLang := featureMap["multi-language-support"]
	assert.True(t, multiLang.Enabled)
	assert.Contains(t, multiLang.Parameters, "supported_languages")
}

func TestCapabilityDetector_GetResourceLimits(t *testing.T) {
	detector := NewCapabilityDetector(config.AIConfig{}, nil)

	limits := detector.getResourceLimits()

	assert.Greater(t, limits.MaxConcurrentRequests, 0)
	assert.Greater(t, limits.RateLimit.RequestsPerMinute, 0)
	assert.Greater(t, limits.RateLimit.RequestsPerHour, 0)
	assert.Greater(t, limits.Memory.MaxMemoryMB, 0)
	assert.Greater(t, limits.Processing.MaxProcessingTimeSeconds, 0)
	assert.Greater(t, limits.Processing.MaxRetryAttempts, 0)
}

func TestCapabilityDetector_CacheValidity(t *testing.T) {
	detector := NewCapabilityDetector(config.AIConfig{}, nil)

	// Initially cache should be invalid
	assert.False(t, detector.cache.isValid())

	// After getting capabilities, cache should be valid
	ctx := context.Background()
	req := &CapabilitiesRequest{
		IncludeFeatures: true,
	}

	_, err := detector.GetCapabilities(ctx, req)
	require.NoError(t, err)

	assert.True(t, detector.cache.isValid())

	// Test cache invalidation
	detector.InvalidateCache()
	assert.False(t, detector.cache.isValid())
}

func TestCapabilityDetector_SetCacheTTL(t *testing.T) {
	detector := NewCapabilityDetector(config.AIConfig{}, nil)

	newTTL := 10 * time.Minute
	detector.SetCacheTTL(newTTL)

	assert.Equal(t, newTTL, detector.cache.ttl)
}

func TestCapabilityDetector_HealthChecks(t *testing.T) {
	cfg := config.AIConfig{
		DefaultService: "test",
	}

	detector := NewCapabilityDetector(cfg, nil)
	ctx := context.Background()

	req := &CapabilitiesRequest{
		CheckHealth: true,
	}

	capabilities, err := detector.GetCapabilities(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, capabilities)
	assert.NotNil(t, capabilities.Health)
	assert.NotEmpty(t, capabilities.Health.Components)

	// Should have at least engine, cache, and config components
	assert.Contains(t, capabilities.Health.Components, "engine")
	assert.Contains(t, capabilities.Health.Components, "cache")
	assert.Contains(t, capabilities.Health.Components, "config")
}

func TestCapabilityDetector_CheckEngineHealth(t *testing.T) {
	cfg := config.AIConfig{
		DefaultService: "test",
	}

	detector := NewCapabilityDetector(cfg, nil)
	ctx := context.Background()

	health := detector.checkEngineHealth(ctx)

	assert.NotEmpty(t, health.Status)
	assert.False(t, health.Healthy) // Should be false since no client
	assert.NotEmpty(t, health.Message)
	assert.NotZero(t, health.ResponseTime)
}

func TestCapabilityDetector_CheckCacheHealth(t *testing.T) {
	detector := NewCapabilityDetector(config.AIConfig{}, nil)

	health := detector.checkCacheHealth()

	assert.Equal(t, "healthy", health.Status)
	assert.True(t, health.Healthy)
	assert.Equal(t, "Capability cache operational", health.Message)
	assert.NotZero(t, health.ResponseTime)
}

func TestCapabilityDetector_CheckConfigHealth(t *testing.T) {
	tests := []struct {
		name         string
		config       config.AIConfig
		expectHealthy bool
		expectErrors bool
	}{
		{
			name: "valid config",
			config: config.AIConfig{
				DefaultService: "test",
			},
			expectHealthy: true,
			expectErrors:  false,
		},
		{
			name: "missing default service",
			config: config.AIConfig{
				DefaultService: "",
			},
			expectHealthy: false,
			expectErrors:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewCapabilityDetector(tt.config, nil)
			health := detector.checkConfigHealth()

			assert.Equal(t, tt.expectHealthy, health.Healthy)
			if tt.expectErrors {
				assert.NotEmpty(t, health.Errors)
			} else {
				assert.Empty(t, health.Errors)
			}
			assert.NotEmpty(t, health.Status)
			assert.NotEmpty(t, health.Message)
		})
	}
}