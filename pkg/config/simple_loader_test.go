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

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
	// Change to temp directory to avoid loading real config
	tempDir := t.TempDir()
	switchToDir(t, tempDir)

	// Load configuration (should use defaults when no file exists)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load default configuration: %v", err)
	}

	// Verify default values
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Expected default host '0.0.0.0', got '%s'", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Plugin.Name != "atest-ext-ai" {
		t.Errorf("Expected default plugin name 'atest-ext-ai', got '%s'", cfg.Plugin.Name)
	}
	if cfg.AI.DefaultService != "ollama" {
		t.Errorf("Expected default service 'ollama', got '%s'", cfg.AI.DefaultService)
	}
}

func TestLoadConfigFromYAML(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	configData := `
server:
  host: "test-host"
  port: 9090
  timeout: "45s"
  max_connections: 200
  socket_path: "/tmp/test.sock"

plugin:
  name: "test-plugin"
  version: "2.0.0"
  debug: true
  log_level: "debug"
  environment: "production"

ai:
  default_service: "openai"
  timeout: "120s"
  services:
    openai:
      enabled: true
      provider: "openai"
      endpoint: "https://api.openai.com/v1"
      api_key: "test-key"
      model: "gpt-4"
      timeout: "30s"
      max_tokens: 8192
      priority: 1
`

	if err := os.WriteFile(configFile, []byte(configData), 0o600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Change directory to where the config file is
	switchToDir(t, tempDir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load configuration from YAML: %v", err)
	}

	// Verify loaded values
	if cfg.Server.Host != "test-host" {
		t.Errorf("Expected host 'test-host', got '%s'", cfg.Server.Host)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Plugin.Name != "test-plugin" {
		t.Errorf("Expected plugin name 'test-plugin', got '%s'", cfg.Plugin.Name)
	}
	if cfg.AI.DefaultService != "openai" {
		t.Errorf("Expected default service 'openai', got '%s'", cfg.AI.DefaultService)
	}
}

func TestLoadConfigWithEnvOverrides(t *testing.T) {
	// Set environment variables
	_ = os.Setenv("ATEST_EXT_AI_SERVER_HOST", "env-host")
	_ = os.Setenv("ATEST_EXT_AI_SERVER_PORT", "5555")
	_ = os.Setenv("ATEST_EXT_AI_LOG_LEVEL", "debug")
	// Note: Not setting AI_PROVIDER to avoid validation issues with default services
	defer func() {
		_ = os.Unsetenv("ATEST_EXT_AI_SERVER_HOST")
		_ = os.Unsetenv("ATEST_EXT_AI_SERVER_PORT")
		_ = os.Unsetenv("ATEST_EXT_AI_LOG_LEVEL")
	}()

	// Change to temp directory to avoid loading real config
	tempDir := t.TempDir()
	switchToDir(t, tempDir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load configuration with env overrides: %v", err)
	}

	// Verify environment variables override defaults
	if cfg.Server.Host != "env-host" {
		t.Errorf("Expected host from env 'env-host', got '%s'", cfg.Server.Host)
	}
	if cfg.Server.Port != 5555 {
		t.Errorf("Expected port from env 5555, got %d", cfg.Server.Port)
	}
	if cfg.Plugin.LogLevel != "debug" {
		t.Errorf("Expected log level from env 'debug', got '%s'", cfg.Plugin.LogLevel)
	}
	// Verify AI default service is still 'ollama' (default, since we didn't override it)
	if cfg.AI.DefaultService != "ollama" {
		t.Errorf("Expected default service 'ollama', got '%s'", cfg.AI.DefaultService)
	}
}

func TestApplyDefaults(t *testing.T) {
	cfg := &Config{}
	applyDefaults(cfg)

	// Verify defaults are applied
	if cfg.Server.Host == "" {
		t.Error("Expected server host to have default value")
	}
	if cfg.Server.Port == 0 {
		t.Error("Expected server port to have default value")
	}
	if cfg.Plugin.Name == "" {
		t.Error("Expected plugin name to have default value")
	}
	if cfg.AI.DefaultService == "" {
		t.Error("Expected AI default service to have default value")
	}
	if len(cfg.AI.Services) == 0 {
		t.Error("Expected AI services to have default values")
	}
	if cfg.AI.Retry.MaxAttempts == 0 {
		t.Error("Expected retry max attempts to have default value")
	}
}

func TestLoadYAMLFile(t *testing.T) {
	tempDir := t.TempDir()
	validFile := filepath.Join(tempDir, "valid.yaml")
	invalidFile := filepath.Join(tempDir, "invalid.yaml")
	nonexistentFile := filepath.Join(tempDir, "nonexistent.yaml")

	// Create valid YAML file
	validData := `
server:
  host: "localhost"
  port: 8080
plugin:
  name: "test"
  version: "1.0.0"
  log_level: "info"
  environment: "production"
ai:
  default_service: "ollama"
  services:
    ollama:
      enabled: true
      provider: "ollama"
      model: "test-model"
      max_tokens: 4096
      priority: 1
`
	if err := os.WriteFile(validFile, []byte(validData), 0o600); err != nil {
		t.Fatalf("Failed to write valid file: %v", err)
	}

	// Create invalid YAML file
	invalidData := `
server:
  host: "localhost"
  - invalid syntax
`
	if err := os.WriteFile(invalidFile, []byte(invalidData), 0o600); err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}

	// Test valid file
	cfg, err := loadYAMLFile(validFile)
	if err != nil {
		t.Errorf("Expected no error for valid file, got: %v", err)
	}
	if cfg == nil {
		t.Error("Expected config to be loaded, got nil")
	}

	// Test invalid file
	_, err = loadYAMLFile(invalidFile)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}

	// Test nonexistent file
	_, err = loadYAMLFile(nonexistentFile)
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

// switchToDir changes the current working directory for the duration of the test.
func switchToDir(t *testing.T, dir string) {
	t.Helper()

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change directory to %s: %v", dir, err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})
}
