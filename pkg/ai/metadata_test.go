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
	"runtime"
	"testing"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestNewMetadataProvider(t *testing.T) {
	cfg := config.Config{
		AI: config.AIConfig{
			DefaultService: "test",
			Services: map[string]config.AIService{
				"test": {
					Enabled:  true,
					Provider: "local",
				},
			},
		},
	}

	provider := NewMetadataProvider(cfg)

	assert.NotNil(t, provider)
	assert.NotNil(t, provider.metadata)
	assert.Equal(t, cfg, provider.config)
	assert.NotZero(t, provider.startTime)
}

func TestMetadataProvider_GetMetadata(t *testing.T) {
	cfg := config.Config{
		AI: config.AIConfig{
			DefaultService: "test",
		},
	}

	provider := NewMetadataProvider(cfg)
	metadata := provider.GetMetadata()

	assert.NotNil(t, metadata)
	assert.Equal(t, "atest-ext-ai", metadata.Name)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.NotEmpty(t, metadata.Description)
	assert.Equal(t, "API Testing Authors", metadata.Author)
	assert.Equal(t, "Apache License 2.0", metadata.License)
	assert.NotEmpty(t, metadata.Features)
	assert.NotEmpty(t, metadata.Tags)
	assert.NotNil(t, metadata.BuildInfo)
	assert.NotNil(t, metadata.Runtime)
	assert.NotNil(t, metadata.Configuration)
	assert.NotNil(t, metadata.Dependencies)
	assert.NotNil(t, metadata.Compatibility)
}

func TestMetadataProvider_GetBuildInfo(t *testing.T) {
	provider := NewMetadataProvider(config.Config{})
	buildInfo := provider.GetBuildInfo()

	assert.Equal(t, "1.0.0", buildInfo.Version)
	assert.Equal(t, runtime.Version(), buildInfo.GoVersion)
	assert.Equal(t, runtime.GOOS, buildInfo.Platform)
	assert.Equal(t, runtime.GOARCH, buildInfo.Arch)
	assert.Equal(t, runtime.Compiler, buildInfo.Compiler)
	assert.NotZero(t, buildInfo.BuildDate)
}

func TestMetadataProvider_GetRuntimeInfo(t *testing.T) {
	provider := NewMetadataProvider(config.Config{})
	runtimeInfo := provider.GetRuntimeInfo()

	assert.Equal(t, runtime.Version(), runtimeInfo.GoVersion)
	assert.Equal(t, runtime.GOOS, runtimeInfo.GOOS)
	assert.Equal(t, runtime.GOARCH, runtimeInfo.GOARCH)
	assert.Equal(t, runtime.NumCPU(), runtimeInfo.NumCPU)
	assert.Greater(t, runtimeInfo.NumGoroutine, 0)
	assert.Greater(t, runtimeInfo.Uptime, time.Duration(0))
	assert.NotZero(t, runtimeInfo.StartTime)
	assert.NotNil(t, runtimeInfo.MemStats)

	// Check memory stats
	assert.Greater(t, runtimeInfo.MemStats.Alloc, uint64(0))
	assert.Greater(t, runtimeInfo.MemStats.TotalAlloc, uint64(0))
	assert.GreaterOrEqual(t, runtimeInfo.MemStats.GCCPUPercent, float64(0))
}

func TestMetadataProvider_GetConfigurationInfo(t *testing.T) {
	tests := []struct {
		name         string
		config       config.Config
		expectValid  bool
		expectErrors bool
	}{
		{
			name: "valid configuration",
			config: config.Config{
				AI: config.AIConfig{
					DefaultService: "test",
					Services: map[string]config.AIService{
						"test": {
							Enabled:  true,
							Provider: "local",
						},
					},
				},
			},
			expectValid:  true,
			expectErrors: false,
		},
		{
			name: "invalid configuration - no default service",
			config: config.Config{
				AI: config.AIConfig{
					DefaultService: "",
				},
			},
			expectValid:  false,
			expectErrors: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewMetadataProvider(tt.config)
			configInfo := provider.GetConfigurationInfo()

			assert.Equal(t, tt.expectValid, configInfo.Valid)
			if tt.expectErrors {
				assert.NotEmpty(t, configInfo.ValidationErrors)
			} else {
				assert.Empty(t, configInfo.ValidationErrors)
			}

			assert.NotEmpty(t, configInfo.Source)
			assert.NotZero(t, configInfo.LoadedAt)
			assert.NotEmpty(t, configInfo.Version)
			assert.NotEmpty(t, configInfo.Features)
			assert.NotNil(t, configInfo.Settings)
		})
	}
}

func TestMetadataProvider_CustomMetadata(t *testing.T) {
	provider := NewMetadataProvider(config.Config{})

	// Test setting custom metadata
	key := "test_key"
	value := "test_value"
	provider.SetCustomMetadata(key, value)

	// Test getting custom metadata
	retrievedValue, exists := provider.GetCustomMetadata(key)
	assert.True(t, exists)
	assert.Equal(t, value, retrievedValue)

	// Test getting non-existent metadata
	_, exists = provider.GetCustomMetadata("non_existent")
	assert.False(t, exists)

	// Verify custom metadata appears in full metadata
	metadata := provider.GetMetadata()
	assert.Contains(t, metadata.Custom, key)
	assert.Equal(t, value, metadata.Custom[key])
}

func TestMetadataProvider_GetSummary(t *testing.T) {
	cfg := config.Config{
		AI: config.AIConfig{
			DefaultService: "test",
			Services: map[string]config.AIService{
				"test1": {
					Enabled:  true,
					Provider: "local",
				},
				"test2": {
					Enabled:  false,
					Provider: "openai",
				},
			},
		},
	}

	provider := NewMetadataProvider(cfg)

	// Give some time for uptime calculation
	time.Sleep(10 * time.Millisecond)

	summary := provider.GetSummary()

	assert.NotNil(t, summary)

	// Check required fields
	expectedFields := []string{
		"name", "version", "description", "uptime",
		"go_version", "platform", "providers",
		"features", "healthy",
	}

	for _, field := range expectedFields {
		assert.Contains(t, summary, field, "Summary should contain field: %s", field)
	}

	assert.Equal(t, "atest-ext-ai", summary["name"])
	assert.Equal(t, "1.0.0", summary["version"])
	assert.NotEmpty(t, summary["uptime"])
	assert.Equal(t, runtime.Version(), summary["go_version"])
	assert.Equal(t, runtime.GOOS+"/"+runtime.GOARCH, summary["platform"])
	assert.Equal(t, 1, summary["providers"]) // Only enabled providers
	assert.Greater(t, summary["features"], 0)
}

func TestMetadataProvider_Dependencies(t *testing.T) {
	provider := NewMetadataProvider(config.Config{})
	metadata := provider.GetMetadata()

	assert.NotEmpty(t, metadata.Dependencies)

	// Check for expected dependencies
	depMap := make(map[string]DependencyInfo)
	for _, dep := range metadata.Dependencies {
		depMap[dep.Name] = dep
	}

	// Check for API testing framework dependency
	apiTesting, exists := depMap["github.com/linuxsuren/api-testing"]
	assert.True(t, exists)
	assert.Equal(t, "external", apiTesting.Type)
	assert.True(t, apiTesting.Required)

	// Check for gRPC dependency
	grpc, exists := depMap["google.golang.org/grpc"]
	assert.True(t, exists)
	assert.Equal(t, "external", grpc.Type)
	assert.True(t, grpc.Required)

	// Check for Go runtime dependency
	goRuntime, exists := depMap["go runtime"]
	assert.True(t, exists)
	assert.Equal(t, "runtime", goRuntime.Type)
	assert.True(t, goRuntime.Required)
	assert.Equal(t, runtime.Version(), goRuntime.Version)
}

func TestMetadataProvider_Compatibility(t *testing.T) {
	provider := NewMetadataProvider(config.Config{})
	metadata := provider.GetMetadata()

	compatibility := metadata.Compatibility

	assert.Equal(t, "v0.0.19", compatibility.MinAPITestingVersion)
	assert.NotEmpty(t, compatibility.SupportedPlatforms)
	assert.Contains(t, compatibility.SupportedPlatforms, "linux/amd64")
	assert.Contains(t, compatibility.SupportedPlatforms, "darwin/amd64")
	assert.Contains(t, compatibility.SupportedPlatforms, "windows/amd64")
	assert.NotEmpty(t, compatibility.RequiredFeatures)
	assert.Contains(t, compatibility.RequiredFeatures, "grpc-support")
}

func TestMetadataProvider_Features(t *testing.T) {
	provider := NewMetadataProvider(config.Config{})
	metadata := provider.GetMetadata()

	expectedFeatures := []string{
		"sql-generation",
		"natural-language-queries",
		"multi-database-support",
		"query-optimization",
		"sql-validation",
		"health-monitoring",
		"capability-reporting",
	}

	for _, expectedFeature := range expectedFeatures {
		assert.Contains(t, metadata.Features, expectedFeature)
	}
}

func TestMetadataProvider_Tags(t *testing.T) {
	provider := NewMetadataProvider(config.Config{})
	metadata := provider.GetMetadata()

	expectedTags := []string{"ai", "sql", "nlp", "database", "testing", "automation"}

	for _, expectedTag := range expectedTags {
		assert.Contains(t, metadata.Tags, expectedTag)
	}
}

func TestMetadataProvider_TimestampUpdates(t *testing.T) {
	provider := NewMetadataProvider(config.Config{})

	initialMetadata := provider.GetMetadata()
	initialTime := initialMetadata.UpdatedAt

	// Wait a bit to ensure timestamp difference
	time.Sleep(10 * time.Millisecond)

	// Adding custom metadata should update the timestamp
	provider.SetCustomMetadata("test", "value")

	updatedMetadata := provider.GetMetadata()
	updatedTime := updatedMetadata.UpdatedAt

	assert.True(t, updatedTime.After(initialTime), "UpdatedAt should be more recent after modification")
}

func BenchmarkMetadataProvider_GetMetadata(b *testing.B) {
	provider := NewMetadataProvider(config.Config{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = provider.GetMetadata()
	}
}

func BenchmarkMetadataProvider_GetSummary(b *testing.B) {
	provider := NewMetadataProvider(config.Config{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = provider.GetSummary()
	}
}