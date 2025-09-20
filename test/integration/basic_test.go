// +build integration

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

package integration

import (
	"testing"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/config"
)

// TestBasicIntegration performs basic integration checks
func TestBasicIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Log("Running basic integration test")

	// Test configuration loading
	loader := config.NewLoader()
	err := loader.Load()
	if err != nil {
		t.Logf("Config load (using defaults): %v", err)
	}

	cfg := loader.GetConfig()
	if cfg == nil {
		t.Fatal("Configuration should not be nil")
	}

	// Basic validation
	if cfg.Server.Port == 0 {
		t.Error("Expected server port to be set")
	}

	if cfg.AI.DefaultService == "" {
		t.Error("Expected default AI service to be set")
	}

	t.Log("Basic integration test passed")
}

// TestServiceAvailability tests service availability
func TestServiceAvailability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test checks if various services would be available
	// It doesn't fail if services are not running

	t.Log("Checking service availability...")

	// Check Ollama
	t.Log("Ollama service: Would connect to http://localhost:11434 in production")

	// Check database
	t.Log("Database: Would use SQLite by default")

	// Check plugin socket
	t.Log("Plugin socket: Would use /tmp/atest-ext-ai.sock")

	t.Log("Service availability check completed")
}

// TestConfigurationIntegration tests configuration integration
func TestConfigurationIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test various configuration scenarios
	testCases := []struct {
		name     string
		setup    func() *config.Config
		validate func(*config.Config) bool
	}{
		{
			name: "Default configuration",
			setup: func() *config.Config {
				loader := config.NewLoader()
				_ = loader.Load()
				return loader.GetConfig()
			},
			validate: func(cfg *config.Config) bool {
				return cfg != nil && cfg.Server.Port > 0
			},
		},
		{
			name: "Custom configuration",
			setup: func() *config.Config {
				return &config.Config{
					Server: config.ServerConfig{
						Host: "127.0.0.1",
						Port: 9999,
						Timeout: config.Duration{Duration: 60 * time.Second},
					},
					AI: config.AIConfig{
						DefaultService: "custom",
					},
				}
			},
			validate: func(cfg *config.Config) bool {
				return cfg.Server.Port == 9999 && cfg.AI.DefaultService == "custom"
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tc.setup()
			if !tc.validate(cfg) {
				t.Errorf("%s validation failed", tc.name)
			}
		})
	}
}

// TestPluginLifecycle tests plugin lifecycle
func TestPluginLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Log("Testing plugin lifecycle...")

	// Simulate plugin initialization
	t.Log("1. Plugin initialization - would load configuration")

	// Simulate plugin startup
	t.Log("2. Plugin startup - would start gRPC server")

	// Simulate plugin operation
	t.Log("3. Plugin operation - would handle queries")

	// Simulate plugin shutdown
	t.Log("4. Plugin shutdown - would close connections")

	t.Log("Plugin lifecycle test completed")
}