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
	"time"
)

func TestManagerCreation(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() {
		if err := manager.Stop(); err != nil {
			t.Logf("Error stopping manager: %v", err)
		}
	}()

	if manager == nil {
		t.Fatal("Expected manager to be created, got nil")
	}
}

func TestManagerLoadConfiguration(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	configData := `
server:
  host: "localhost"
  port: 8080
  timeout: "30s"
  max_connections: 100
  socket_path: "/tmp/test.sock"

plugin:
  name: "test-plugin"
  version: "1.0.0"
  debug: false
  log_level: "info"
  environment: "development"

ai:
  default_service: "ollama"
  timeout: "60s"
  rate_limit:
    enabled: true
    requests_per_minute: 60
    burst_size: 10
    window_size: "1m"
  circuit_breaker:
    enabled: true
    failure_threshold: 5
    success_threshold: 3
    timeout: "30s"
    reset_timeout: "60s"
  retry:
    enabled: true
    max_attempts: 3
    initial_delay: "1s"
    max_delay: "30s"
    multiplier: 2.0
    jitter: true
  cache:
    enabled: true
    ttl: "1h"
    max_size: 1000
    provider: "memory"
  services:
    ollama:
      enabled: true
      provider: "ollama"
      endpoint: "http://localhost:11434"
      model: "codellama"
      max_tokens: 4096
      temperature: 0.7
      priority: 1

database:
  enabled: false

logging:
  level: "info"
  format: "json"
  output: "stdout"
  rotation:
    enabled: true
    size: "100MB"
    count: 5
    age: "30d"
    compress: true
`

	if err := os.WriteFile(configFile, []byte(configData), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Test loading configuration
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() {
		if err := manager.Stop(); err != nil {
			t.Logf("Error stopping manager: %v", err)
		}
	}()

	err = manager.LoadFromFile(configFile)
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	config := manager.GetConfig()
	if config == nil {
		t.Fatal("Expected configuration to be loaded, got nil")
	}

	// Verify configuration values
	if config.Server.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", config.Server.Host)
	}

	if config.Server.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", config.Server.Port)
	}

	if config.Plugin.Name != "test-plugin" {
		t.Errorf("Expected plugin name 'test-plugin', got '%s'", config.Plugin.Name)
	}

	if config.AI.DefaultService != "ollama" {
		t.Errorf("Expected default AI service 'ollama', got '%s'", config.AI.DefaultService)
	}
}

func TestManagerValidation(t *testing.T) {
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() {
		if err := manager.Stop(); err != nil {
			t.Logf("Error stopping manager: %v", err)
		}
	}()

	// Test validation with invalid configuration
	invalidConfigData := `
server:
  host: ""
  port: -1
  timeout: "30s"
  socket_path: "/tmp/test.sock"

plugin:
  name: ""
  version: "invalid-version"

ai:
  default_service: "nonexistent"
  services: {}
`

	err = manager.LoadFromBytes([]byte(invalidConfigData), "yaml")
	if err == nil {
		t.Fatal("Expected validation error for invalid configuration, got nil")
	}

	if validationErrs, ok := err.(ValidationErrors); ok {
		if len(validationErrs) == 0 {
			t.Fatal("Expected validation errors, got none")
		}
	} else {
		t.Fatalf("Expected ValidationErrors, got %T", err)
	}
}

func TestManagerGetSet(t *testing.T) {
	// Create a temporary config file with valid data
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	configData := `
server:
  host: "localhost"
  port: 8080
  timeout: "30s"
  max_connections: 100
  socket_path: "/tmp/test.sock"

plugin:
  name: "test-plugin"
  version: "1.0.0"
  debug: false
  log_level: "info"
  environment: "development"

ai:
  default_service: "ollama"
  timeout: "60s"
  rate_limit:
    enabled: true
    requests_per_minute: 60
    burst_size: 10
    window_size: "1m"
  circuit_breaker:
    enabled: true
    failure_threshold: 5
    success_threshold: 3
    timeout: "30s"
    reset_timeout: "60s"
  retry:
    enabled: true
    max_attempts: 3
    initial_delay: "1s"
    max_delay: "30s"
    multiplier: 2.0
    jitter: true
  cache:
    enabled: true
    ttl: "1h"
    max_size: 1000
    provider: "memory"
  services:
    ollama:
      enabled: true
      provider: "ollama"
      endpoint: "http://localhost:11434"
      model: "codellama"
      max_tokens: 4096
      temperature: 0.7
      priority: 1

database:
  enabled: false

logging:
  level: "info"
  format: "json"
  output: "stdout"
  rotation:
    enabled: true
    size: "100MB"
    count: 5
    age: "30d"
    compress: true
`

	if err := os.WriteFile(configFile, []byte(configData), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() {
		if err := manager.Stop(); err != nil {
			t.Logf("Error stopping manager: %v", err)
		}
	}()

	if err := manager.LoadFromFile(configFile); err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Test Get
	host := manager.Get("server.host")
	if host != "localhost" {
		t.Errorf("Expected host 'localhost', got %v", host)
	}

	port := manager.Get("server.port")
	if port != 8080 {
		t.Errorf("Expected port 8080, got %v", port)
	}

	// Test Set
	err = manager.Set("server.port", 9090)
	if err != nil {
		t.Fatalf("Failed to set configuration value: %v", err)
	}

	newPort := manager.Get("server.port")
	if newPort != 9090 {
		t.Errorf("Expected port 9090 after set, got %v", newPort)
	}

	// Verify the configuration struct was also updated
	config := manager.GetConfig()
	if config.Server.Port != 9090 {
		t.Errorf("Expected config struct port 9090, got %d", config.Server.Port)
	}
}

func TestManagerCallbacks(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	configData := `
server:
  host: "localhost"
  port: 8080
  timeout: "30s"
  max_connections: 100
  socket_path: "/tmp/test.sock"

plugin:
  name: "test-plugin"
  version: "1.0.0"
  debug: false
  log_level: "info"
  environment: "development"

ai:
  default_service: "ollama"
  timeout: "60s"
  rate_limit:
    enabled: true
    requests_per_minute: 60
    burst_size: 10
    window_size: "1m"
  circuit_breaker:
    enabled: true
    failure_threshold: 5
    success_threshold: 3
    timeout: "30s"
    reset_timeout: "60s"
  retry:
    enabled: true
    max_attempts: 3
    initial_delay: "1s"
    max_delay: "30s"
    multiplier: 2.0
    jitter: true
  cache:
    enabled: true
    ttl: "1h"
    max_size: 1000
    provider: "memory"
  services:
    ollama:
      enabled: true
      provider: "ollama"
      endpoint: "http://localhost:11434"
      model: "codellama"
      max_tokens: 4096
      temperature: 0.7
      priority: 1

database:
  enabled: false

logging:
  level: "info"
  format: "json"
  output: "stdout"
  rotation:
    enabled: true
    size: "100MB"
    count: 5
    age: "30d"
    compress: true
`

	if err := os.WriteFile(configFile, []byte(configData), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() {
		if err := manager.Stop(); err != nil {
			t.Logf("Error stopping manager: %v", err)
		}
	}()

	if err := manager.LoadFromFile(configFile); err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Test callback registration
	callbackExecuted := false
	var callbackKey string
	var callbackOldValue, callbackNewValue interface{}

	callback := func(key string, oldValue, newValue interface{}) {
		callbackExecuted = true
		callbackKey = key
		callbackOldValue = oldValue
		callbackNewValue = newValue
	}

	err = manager.Watch(callback)
	if err != nil {
		t.Fatalf("Failed to register callback: %v", err)
	}

	// Make a change to trigger callback
	err = manager.Set("server.port", 9090)
	if err != nil {
		t.Fatalf("Failed to set configuration value: %v", err)
	}

	// Wait a bit for async callback execution
	time.Sleep(100 * time.Millisecond)

	if !callbackExecuted {
		t.Error("Expected callback to be executed")
	}

	if callbackKey != "server.port" {
		t.Errorf("Expected callback key 'server.port', got '%s'", callbackKey)
	}

	if callbackOldValue != 8080 {
		t.Errorf("Expected callback old value 8080, got %v", callbackOldValue)
	}

	if callbackNewValue != 9090 {
		t.Errorf("Expected callback new value 9090, got %v", callbackNewValue)
	}
}

func TestManagerExport(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	configData := `
server:
  host: "localhost"
  port: 8080
  timeout: "30s"
  max_connections: 100
  socket_path: "/tmp/test.sock"

plugin:
  name: "test-plugin"
  version: "1.0.0"
  debug: false
  log_level: "info"
  environment: "development"

ai:
  default_service: "ollama"
  timeout: "60s"
  rate_limit:
    enabled: true
    requests_per_minute: 60
    burst_size: 10
    window_size: "1m"
  circuit_breaker:
    enabled: true
    failure_threshold: 5
    success_threshold: 3
    timeout: "30s"
    reset_timeout: "60s"
  retry:
    enabled: true
    max_attempts: 3
    initial_delay: "1s"
    max_delay: "30s"
    multiplier: 2.0
    jitter: true
  cache:
    enabled: true
    ttl: "1h"
    max_size: 1000
    provider: "memory"
  services:
    ollama:
      enabled: true
      provider: "ollama"
      endpoint: "http://localhost:11434"
      model: "codellama"
      max_tokens: 4096
      temperature: 0.7
      priority: 1

database:
  enabled: false

logging:
  level: "info"
  format: "json"
  output: "stdout"
  rotation:
    enabled: true
    size: "100MB"
    count: 5
    age: "30d"
    compress: true
`

	if err := os.WriteFile(configFile, []byte(configData), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() {
		if err := manager.Stop(); err != nil {
			t.Logf("Error stopping manager: %v", err)
		}
	}()

	if err := manager.LoadFromFile(configFile); err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Test YAML export
	yamlData, err := manager.Export("yaml")
	if err != nil {
		t.Fatalf("Failed to export YAML: %v", err)
	}

	if len(yamlData) == 0 {
		t.Error("Expected YAML export data, got empty")
	}

	// Test JSON export
	jsonData, err := manager.Export("json")
	if err != nil {
		t.Fatalf("Failed to export JSON: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("Expected JSON export data, got empty")
	}

	// Test TOML export
	tomlData, err := manager.Export("toml")
	if err != nil {
		t.Fatalf("Failed to export TOML: %v", err)
	}

	if len(tomlData) == 0 {
		t.Error("Expected TOML export data, got empty")
	}
}

func TestManagerDefaults(t *testing.T) {
	// Create manager with hot reload disabled for this test
	opts := DefaultManagerOptions()
	opts.EnableHotReload = false
	manager, err := NewManager(opts)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() {
		if err := manager.Stop(); err != nil {
			t.Logf("Error stopping manager: %v", err)
		}
	}()

	// Load with default values (empty paths)
	err = manager.Load()
	if err != nil {
		t.Fatalf("Failed to load default configuration: %v", err)
	}

	config := manager.GetConfig()
	if config == nil {
		t.Fatal("Expected default configuration to be loaded, got nil")
	}

	// Verify some default values
	if config.Server.Host != "0.0.0.0" {
		t.Errorf("Expected default host '0.0.0.0', got '%s'", config.Server.Host)
	}

	if config.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", config.Server.Port)
	}

	if config.Plugin.Name != "atest-ext-ai" {
		t.Errorf("Expected default plugin name 'atest-ext-ai', got '%s'", config.Plugin.Name)
	}

	if config.AI.DefaultService != "ollama" {
		t.Errorf("Expected default AI service 'ollama', got '%s'", config.AI.DefaultService)
	}
}

func TestManagerEnvironmentVariables(t *testing.T) {
	// Set environment variables
	_ = os.Setenv("ATEST_EXT_AI_SERVER_HOST", "custom-host")
	_ = os.Setenv("ATEST_EXT_AI_SERVER_PORT", "9999")
	_ = os.Setenv("ATEST_EXT_AI_AI_DEFAULT_SERVICE", "ollama")
	defer func() {
		_ = os.Unsetenv("ATEST_EXT_AI_SERVER_HOST")
		_ = os.Unsetenv("ATEST_EXT_AI_SERVER_PORT")
		_ = os.Unsetenv("ATEST_EXT_AI_AI_DEFAULT_SERVICE")
	}()

	// Create manager with hot reload disabled for this test
	opts := DefaultManagerOptions()
	opts.EnableHotReload = false
	manager, err := NewManager(opts)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() {
		if err := manager.Stop(); err != nil {
			t.Logf("Error stopping manager: %v", err)
		}
	}()

	err = manager.Load()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	config := manager.GetConfig()

	// Verify environment variables override defaults
	if config.Server.Host != "custom-host" {
		t.Errorf("Expected host from environment 'custom-host', got '%s'", config.Server.Host)
	}

	if config.Server.Port != 9999 {
		t.Errorf("Expected port from environment 9999, got %d", config.Server.Port)
	}

	if config.AI.DefaultService != "ollama" {
		t.Errorf("Expected AI service from environment 'ollama', got '%s'", config.AI.DefaultService)
	}
}

func TestManagerMultipleFormats(t *testing.T) {
	tempDir := t.TempDir()

	// Test YAML format
	yamlFile := filepath.Join(tempDir, "config.yaml")
	yamlData := `
server:
  host: "yaml-host"
  port: 8081
  timeout: "30s"
  max_connections: 100
  socket_path: "/tmp/yaml.sock"

plugin:
  name: "yaml-plugin"
  version: "1.0.0"
  debug: false
  log_level: "info"
  environment: "development"

ai:
  default_service: "ollama"
  timeout: "60s"
  services:
    ollama:
      enabled: true
      provider: "ollama"
      endpoint: "http://localhost:11434"
      model: "codellama"
      max_tokens: 4096
      temperature: 0.7
      priority: 1
`

	if err := os.WriteFile(yamlFile, []byte(yamlData), 0644); err != nil {
		t.Fatalf("Failed to write YAML config file: %v", err)
	}

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() {
		if err := manager.Stop(); err != nil {
			t.Logf("Error stopping manager: %v", err)
		}
	}()

	err = manager.LoadFromFile(yamlFile)
	if err != nil {
		t.Fatalf("Failed to load YAML configuration: %v", err)
	}

	config := manager.GetConfig()
	if config.Server.Host != "yaml-host" {
		t.Errorf("Expected YAML host 'yaml-host', got '%s'", config.Server.Host)
	}

	// Test JSON format
	jsonFile := filepath.Join(tempDir, "config.json")
	jsonData := `{
  "server": {
    "host": "json-host",
    "port": 8082,
    "timeout": "30s",
    "max_connections": 100,
    "socket_path": "/tmp/json.sock"
  },
  "plugin": {
    "name": "json-plugin",
    "version": "1.0.0",
    "debug": false,
    "log_level": "info",
    "environment": "development"
  },
  "ai": {
    "default_service": "ollama",
    "timeout": "60s",
    "services": {
      "ollama": {
        "enabled": true,
        "provider": "ollama",
        "endpoint": "http://localhost:11434",
        "model": "codellama",
        "max_tokens": 4096,
        "temperature": 0.7,
        "priority": 1
      }
    }
  }
}`

	if err := os.WriteFile(jsonFile, []byte(jsonData), 0644); err != nil {
		t.Fatalf("Failed to write JSON config file: %v", err)
	}

	err = manager.LoadFromFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to load JSON configuration: %v", err)
	}

	config = manager.GetConfig()
	if config.Server.Host != "json-host" {
		t.Errorf("Expected JSON host 'json-host', got '%s'", config.Server.Host)
	}

	// Test TOML format
	tomlFile := filepath.Join(tempDir, "config.toml")
	tomlData := `[server]
host = "toml-host"
port = 8083
timeout = "30s"
max_connections = 100
socket_path = "/tmp/toml.sock"

[plugin]
name = "toml-plugin"
version = "1.0.0"
debug = false
log_level = "info"
environment = "development"

[ai]
default_service = "ollama"
timeout = "60s"

[ai.services.ollama]
enabled = true
provider = "ollama"
endpoint = "http://localhost:11434"
model = "codellama"
max_tokens = 4096
temperature = 0.7
priority = 1
`

	if err := os.WriteFile(tomlFile, []byte(tomlData), 0644); err != nil {
		t.Fatalf("Failed to write TOML config file: %v", err)
	}

	err = manager.LoadFromFile(tomlFile)
	if err != nil {
		t.Fatalf("Failed to load TOML configuration: %v", err)
	}

	config = manager.GetConfig()
	if config.Server.Host != "toml-host" {
		t.Errorf("Expected TOML host 'toml-host', got '%s'", config.Server.Host)
	}
}

func TestManagerStats(t *testing.T) {
	// Create manager with hot reload disabled for this test
	opts := DefaultManagerOptions()
	opts.EnableHotReload = false
	manager, err := NewManager(opts)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() {
		if err := manager.Stop(); err != nil {
			t.Logf("Error stopping manager: %v", err)
		}
	}()

	stats := manager.GetStats()
	if stats == nil {
		t.Fatal("Expected stats to be returned, got nil")
	}

	// Check that stats contain expected keys
	expectedKeys := []string{"config_loaded", "is_watching", "watch_paths", "callback_count", "options"}
	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Expected stats key '%s' not found", key)
		}
	}

	// Load a configuration and check stats again
	err = manager.Load()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	stats = manager.GetStats()
	if configLoaded, ok := stats["config_loaded"].(bool); !ok || !configLoaded {
		t.Error("Expected config_loaded to be true after loading configuration")
	}
}
