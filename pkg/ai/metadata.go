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
	"runtime"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/config"
)

// PluginMetadata contains comprehensive information about the AI plugin
type PluginMetadata struct {
	Name          string                 `json:"name"`
	Version       string                 `json:"version"`
	Description   string                 `json:"description"`
	Author        string                 `json:"author"`
	License       string                 `json:"license"`
	Homepage      string                 `json:"homepage"`
	Repository    string                 `json:"repository"`
	BuildInfo     BuildInfo              `json:"build_info"`
	Runtime       RuntimeInfo            `json:"runtime"`
	Configuration ConfigurationInfo      `json:"configuration"`
	Dependencies  []DependencyInfo       `json:"dependencies"`
	Compatibility CompatibilityInfo      `json:"compatibility"`
	Features      []string               `json:"features"`
	Tags          []string               `json:"tags"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	Custom        map[string]interface{} `json:"custom,omitempty"`
}

// BuildInfo contains information about the plugin build
type BuildInfo struct {
	Version   string    `json:"version"`
	Commit    string    `json:"commit"`
	Branch    string    `json:"branch"`
	BuildDate time.Time `json:"build_date"`
	BuildUser string    `json:"build_user"`
	GoVersion string    `json:"go_version"`
	Platform  string    `json:"platform"`
	Arch      string    `json:"arch"`
	Compiler  string    `json:"compiler"`
	BuildTags []string  `json:"build_tags,omitempty"`
}

// RuntimeInfo contains information about the current runtime environment
type RuntimeInfo struct {
	GoVersion    string        `json:"go_version"`
	GOOS         string        `json:"goos"`
	GOARCH       string        `json:"goarch"`
	NumCPU       int           `json:"num_cpu"`
	NumGoroutine int           `json:"num_goroutine"`
	MemStats     MemoryStats   `json:"memory_stats"`
	Uptime       time.Duration `json:"uptime"`
	StartTime    time.Time     `json:"start_time"`
	ProcessID    int           `json:"process_id"`
}

// MemoryStats contains memory usage statistics
type MemoryStats struct {
	Alloc        uint64  `json:"alloc"`          // Bytes allocated and not yet freed
	TotalAlloc   uint64  `json:"total_alloc"`    // Bytes allocated (even if freed)
	Sys          uint64  `json:"sys"`            // Bytes obtained from system
	Lookups      uint64  `json:"lookups"`        // Number of pointer lookups
	Mallocs      uint64  `json:"mallocs"`        // Number of mallocs
	Frees        uint64  `json:"frees"`          // Number of frees
	HeapAlloc    uint64  `json:"heap_alloc"`     // Bytes allocated and not yet freed (same as Alloc above)
	HeapSys      uint64  `json:"heap_sys"`       // Bytes obtained from system
	HeapIdle     uint64  `json:"heap_idle"`      // Bytes in idle spans
	HeapInuse    uint64  `json:"heap_inuse"`     // Bytes in non-idle span
	HeapReleased uint64  `json:"heap_released"`  // Bytes released to the OS
	HeapObjects  uint64  `json:"heap_objects"`   // Total number of allocated objects
	GCCPUPercent float64 `json:"gc_cpu_percent"` // Percentage of CPU time spent in GC
}

// ConfigurationInfo contains information about the plugin configuration
type ConfigurationInfo struct {
	Source           string                 `json:"source"`    // Configuration source (file, env, etc.)
	LoadedAt         time.Time              `json:"loaded_at"` // When configuration was loaded
	Version          string                 `json:"version"`   // Configuration version
	Valid            bool                   `json:"valid"`     // Whether configuration is valid
	Providers        []string               `json:"providers"` // Configured AI providers
	Features         []string               `json:"features"`  // Enabled features
	Settings         map[string]interface{} `json:"settings"`  // Key configuration settings
	ValidationErrors []string               `json:"validation_errors,omitempty"`
}

// DependencyInfo contains information about plugin dependencies
type DependencyInfo struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Type        string            `json:"type"` // internal, external, runtime
	Required    bool              `json:"required"`
	Description string            `json:"description,omitempty"`
	Homepage    string            `json:"homepage,omitempty"`
	License     string            `json:"license,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// CompatibilityInfo contains compatibility information
type CompatibilityInfo struct {
	MinAPITestingVersion string            `json:"min_api_testing_version"`
	MaxAPITestingVersion string            `json:"max_api_testing_version,omitempty"`
	SupportedPlatforms   []string          `json:"supported_platforms"`
	RequiredFeatures     []string          `json:"required_features,omitempty"`
	ConflictingPlugins   []string          `json:"conflicting_plugins,omitempty"`
	Deprecations         []DeprecationInfo `json:"deprecations,omitempty"`
}

// DeprecationInfo contains information about deprecated features
type DeprecationInfo struct {
	Feature     string    `json:"feature"`
	Since       string    `json:"since"`
	RemovalDate time.Time `json:"removal_date,omitempty"`
	Replacement string    `json:"replacement,omitempty"`
	Message     string    `json:"message"`
}

// MetadataProvider manages plugin metadata
type MetadataProvider struct {
	metadata  *PluginMetadata
	startTime time.Time
	config    config.Config
}

// NewMetadataProvider creates a new metadata provider
func NewMetadataProvider(cfg config.Config) *MetadataProvider {
	startTime := time.Now()

	provider := &MetadataProvider{
		startTime: startTime,
		config:    cfg,
		metadata: &PluginMetadata{
			Name:        "atest-ext-ai",
			Version:     "1.0.0",
			Description: "AI Extension Plugin for API Testing Framework - Provides AI-powered SQL generation and analysis capabilities",
			Author:      "API Testing Authors",
			License:     "Apache License 2.0",
			Homepage:    "https://github.com/linuxsuren/atest-ext-ai",
			Repository:  "https://github.com/linuxsuren/atest-ext-ai.git",
			Features: []string{
				"sql-generation",
				"natural-language-queries",
				"multi-database-support",
				"query-optimization",
				"sql-validation",
				"health-monitoring",
				"capability-reporting",
			},
			Tags: []string{
				"ai",
				"sql",
				"nlp",
				"database",
				"testing",
				"automation",
			},
			CreatedAt: startTime,
			UpdatedAt: startTime,
			Custom:    make(map[string]interface{}),
		},
	}

	provider.updateRuntimeInfo()
	provider.updateBuildInfo()
	provider.updateConfigurationInfo()
	provider.updateDependencies()
	provider.updateCompatibilityInfo()

	return provider
}

// GetMetadata returns the complete plugin metadata
func (p *MetadataProvider) GetMetadata() *PluginMetadata {
	// Update runtime information on each call
	p.updateRuntimeInfo()
	p.metadata.UpdatedAt = time.Now()

	// Return a copy to prevent external modification
	metadata := *p.metadata
	return &metadata
}

// GetBuildInfo returns build information
func (p *MetadataProvider) GetBuildInfo() BuildInfo {
	return p.metadata.BuildInfo
}

// GetRuntimeInfo returns current runtime information
func (p *MetadataProvider) GetRuntimeInfo() RuntimeInfo {
	p.updateRuntimeInfo()
	return p.metadata.Runtime
}

// GetConfigurationInfo returns configuration information
func (p *MetadataProvider) GetConfigurationInfo() ConfigurationInfo {
	p.updateConfigurationInfo()
	return p.metadata.Configuration
}

// updateRuntimeInfo updates the runtime information
func (p *MetadataProvider) updateRuntimeInfo() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	p.metadata.Runtime = RuntimeInfo{
		GoVersion:    runtime.Version(),
		GOOS:         runtime.GOOS,
		GOARCH:       runtime.GOARCH,
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
		Uptime:       time.Since(p.startTime),
		StartTime:    p.startTime,
		ProcessID:    0, // Could be set from os.Getpid() if needed
		MemStats: MemoryStats{
			Alloc:        memStats.Alloc,
			TotalAlloc:   memStats.TotalAlloc,
			Sys:          memStats.Sys,
			Lookups:      memStats.Lookups,
			Mallocs:      memStats.Mallocs,
			Frees:        memStats.Frees,
			HeapAlloc:    memStats.HeapAlloc,
			HeapSys:      memStats.HeapSys,
			HeapIdle:     memStats.HeapIdle,
			HeapInuse:    memStats.HeapInuse,
			HeapReleased: memStats.HeapReleased,
			HeapObjects:  memStats.HeapObjects,
			GCCPUPercent: memStats.GCCPUFraction * 100,
		},
	}
}

// updateBuildInfo updates build information
func (p *MetadataProvider) updateBuildInfo() {
	// These values would typically be set during build using ldflags
	// For now, using default values
	p.metadata.BuildInfo = BuildInfo{
		Version:   "1.0.0",
		Commit:    "unknown",   // Could be set via ldflags: -X pkg/ai.GitCommit=$(git rev-parse HEAD)
		Branch:    "unknown",   // Could be set via ldflags: -X pkg/ai.GitBranch=$(git rev-parse --abbrev-ref HEAD)
		BuildDate: p.startTime, // Could be set via ldflags: -X pkg/ai.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)
		BuildUser: "unknown",   // Could be set via ldflags: -X pkg/ai.BuildUser=$(whoami)
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS,
		Arch:      runtime.GOARCH,
		Compiler:  runtime.Compiler,
		BuildTags: []string{}, // Could be populated based on build tags
	}
}

// updateConfigurationInfo updates configuration information
func (p *MetadataProvider) updateConfigurationInfo() {
	var providers []string
	var features []string
	var validationErrors []string

	// Extract provider information
	if len(p.config.AI.Services) > 0 {
		for name, service := range p.config.AI.Services {
			if service.Enabled {
				providers = append(providers, name)
			}
		}
	}

	// Extract enabled features
	features = append(features, "sql-generation", "health-monitoring")

	// Configuration settings (non-sensitive)
	settings := map[string]interface{}{
		"default_service":         p.config.AI.DefaultService,
		"provider_count":          len(providers),
		"load_balancer_strategy":  "round_robin", // Default value
		"health_check_enabled":    true,
		"caching_enabled":         true,
		"max_concurrent_requests": 10,
	}

	// Check configuration validity
	valid := p.config.AI.DefaultService != ""

	if !valid {
		validationErrors = append(validationErrors, "default_service not configured")
	}

	p.metadata.Configuration = ConfigurationInfo{
		Source:           "config file", // Could be more specific
		LoadedAt:         p.startTime,
		Version:          "1.0",
		Valid:            valid,
		Providers:        providers,
		Features:         features,
		Settings:         settings,
		ValidationErrors: validationErrors,
	}
}

// updateDependencies updates dependency information
func (p *MetadataProvider) updateDependencies() {
	p.metadata.Dependencies = []DependencyInfo{
		{
			Name:        "github.com/linuxsuren/api-testing",
			Version:     "v0.0.19", // Should match the actual version
			Type:        "external",
			Required:    true,
			Description: "Core API testing framework",
			Homepage:    "https://github.com/linuxsuren/api-testing",
			License:     "MIT",
		},
		{
			Name:        "google.golang.org/grpc",
			Version:     "latest", // Should be actual version
			Type:        "external",
			Required:    true,
			Description: "gRPC framework for communication",
			Homepage:    "https://grpc.io",
			License:     "Apache 2.0",
		},
		{
			Name:        "go runtime",
			Version:     runtime.Version(),
			Type:        "runtime",
			Required:    true,
			Description: "Go runtime environment",
		},
	}
}

// updateCompatibilityInfo updates compatibility information
func (p *MetadataProvider) updateCompatibilityInfo() {
	p.metadata.Compatibility = CompatibilityInfo{
		MinAPITestingVersion: "v0.0.19",
		SupportedPlatforms: []string{
			"linux/amd64",
			"linux/arm64",
			"darwin/amd64",
			"darwin/arm64",
			"windows/amd64",
		},
		RequiredFeatures: []string{
			"grpc-support",
			"plugin-loader",
		},
		ConflictingPlugins: []string{
			// No known conflicts yet
		},
		Deprecations: []DeprecationInfo{
			// No deprecations yet
		},
	}
}

// SetCustomMetadata allows setting custom metadata fields
func (p *MetadataProvider) SetCustomMetadata(key string, value interface{}) {
	if p.metadata.Custom == nil {
		p.metadata.Custom = make(map[string]interface{})
	}
	p.metadata.Custom[key] = value
	p.metadata.UpdatedAt = time.Now()
}

// GetCustomMetadata retrieves custom metadata
func (p *MetadataProvider) GetCustomMetadata(key string) (interface{}, bool) {
	if p.metadata.Custom == nil {
		return nil, false
	}
	value, exists := p.metadata.Custom[key]
	return value, exists
}

// GetSummary returns a brief summary of the plugin
func (p *MetadataProvider) GetSummary() map[string]interface{} {
	return map[string]interface{}{
		"name":        p.metadata.Name,
		"version":     p.metadata.Version,
		"description": p.metadata.Description,
		"uptime":      time.Since(p.startTime).String(),
		"go_version":  runtime.Version(),
		"platform":    runtime.GOOS + "/" + runtime.GOARCH,
		"providers":   len(p.metadata.Configuration.Providers),
		"features":    len(p.metadata.Features),
		"healthy":     p.metadata.Configuration.Valid,
	}
}
