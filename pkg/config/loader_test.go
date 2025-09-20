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

func TestLoaderCreation(t *testing.T) {
	loader := NewLoader()
	if loader == nil {
		t.Fatal("Expected loader to be created, got nil")
	}
}

func TestLoadFromYAML(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	configData := `
server:
  host: "test-host"
  port: 9090
  timeout: "45s"
  max_connections: 200
  socket_path: "/tmp/yaml-test.sock"

plugin:
  name: "yaml-plugin"
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
      api_key: "test-key"
      model: "gpt-4"
      max_tokens: 8192
      temperature: 0.5
      priority: 1
`

	if err := os.WriteFile(configFile, []byte(configData), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	loader := NewLoader()
	err := loader.LoadFromFile(configFile)
	if err != nil {
		t.Fatalf("Failed to load YAML configuration: %v", err)
	}

	config := loader.GetConfig()
	if config.Server.Host != "test-host" {
		t.Errorf("Expected host 'test-host', got '%s'", config.Server.Host)
	}

	if config.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", config.Server.Port)
	}

	if config.Plugin.Name != "yaml-plugin" {
		t.Errorf("Expected plugin name 'yaml-plugin', got '%s'", config.Plugin.Name)
	}

	if config.AI.DefaultService != "openai" {
		t.Errorf("Expected default service 'openai', got '%s'", config.AI.DefaultService)
	}
}

func TestLoadFromJSON(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.json")

	configData := `{
  "server": {
    "host": "json-host",
    "port": 8888,
    "timeout": "60s",
    "max_connections": 150,
    "socket_path": "/tmp/json-test.sock"
  },
  "plugin": {
    "name": "json-plugin",
    "version": "1.5.0",
    "debug": false,
    "log_level": "warn",
    "environment": "staging"
  },
  "ai": {
    "default_service": "claude",
    "timeout": "90s",
    "services": {
      "claude": {
        "enabled": true,
        "provider": "claude",
        "api_key": "claude-key",
        "model": "claude-3-haiku",
        "max_tokens": 2048,
        "temperature": 0.3,
        "priority": 1
      }
    }
  }
}`

	if err := os.WriteFile(configFile, []byte(configData), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	loader := NewLoader()
	err := loader.LoadFromFile(configFile)
	if err != nil {
		t.Fatalf("Failed to load JSON configuration: %v", err)
	}

	config := loader.GetConfig()
	if config.Server.Host != "json-host" {
		t.Errorf("Expected host 'json-host', got '%s'", config.Server.Host)
	}

	if config.Server.Port != 8888 {
		t.Errorf("Expected port 8888, got %d", config.Server.Port)
	}

	if config.Plugin.Name != "json-plugin" {
		t.Errorf("Expected plugin name 'json-plugin', got '%s'", config.Plugin.Name)
	}

	if config.AI.DefaultService != "claude" {
		t.Errorf("Expected default service 'claude', got '%s'", config.AI.DefaultService)
	}
}

func TestLoadFromTOML(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.toml")

	configData := `[server]
host = "toml-host"
port = 7777
timeout = "25s"
max_connections = 75
socket_path = "/tmp/toml-test.sock"

[plugin]
name = "toml-plugin"
version = "3.0.0"
debug = true
log_level = "error"
environment = "development"

[ai]
default_service = "ollama"
timeout = "180s"

[ai.services.ollama]
enabled = true
provider = "ollama"
endpoint = "http://localhost:11434"
model = "llama2"
max_tokens = 1024
temperature = 0.8
priority = 1
`

	if err := os.WriteFile(configFile, []byte(configData), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	loader := NewLoader()
	err := loader.LoadFromFile(configFile)
	if err != nil {
		t.Fatalf("Failed to load TOML configuration: %v", err)
	}

	config := loader.GetConfig()
	if config.Server.Host != "toml-host" {
		t.Errorf("Expected host 'toml-host', got '%s'", config.Server.Host)
	}

	if config.Server.Port != 7777 {
		t.Errorf("Expected port 7777, got %d", config.Server.Port)
	}

	if config.Plugin.Name != "toml-plugin" {
		t.Errorf("Expected plugin name 'toml-plugin', got '%s'", config.Plugin.Name)
	}

	if config.AI.DefaultService != "ollama" {
		t.Errorf("Expected default service 'ollama', got '%s'", config.AI.DefaultService)
	}
}

func TestLoadFromBytes(t *testing.T) {
	loader := NewLoader()

	yamlData := []byte(`
server:
  host: "bytes-host"
  port: 6666
  timeout: "20s"
  max_connections: 50
  socket_path: "/tmp/bytes-test.sock"

plugin:
  name: "bytes-plugin"
  version: "1.1.0"
  debug: false
  log_level: "info"
  environment: "development"

ai:
  default_service: "ollama"
  timeout: "30s"
  services:
    ollama:
      enabled: true
      provider: "ollama"
      endpoint: "http://localhost:11434"
      model: "codellama"
      priority: 1
`)

	err := loader.LoadFromBytes(yamlData, "yaml")
	if err != nil {
		t.Fatalf("Failed to load from bytes: %v", err)
	}

	config := loader.GetConfig()
	if config.Server.Host != "bytes-host" {
		t.Errorf("Expected host 'bytes-host', got '%s'", config.Server.Host)
	}

	if config.Server.Port != 6666 {
		t.Errorf("Expected port 6666, got %d", config.Server.Port)
	}
}

func TestLoadDefaults(t *testing.T) {
	loader := NewLoader()

	// Load without any files (should use defaults)
	err := loader.Load()
	if err != nil {
		t.Fatalf("Failed to load defaults: %v", err)
	}

	config := loader.GetConfig()

	// Check some default values
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
		t.Errorf("Expected default service 'ollama', got '%s'", config.AI.DefaultService)
	}
}

func TestEnvironmentVariableOverrides(t *testing.T) {
	// Set environment variables
	os.Setenv("ATEST_EXT_AI_SERVER_HOST", "env-host")
	os.Setenv("ATEST_EXT_AI_SERVER_PORT", "5555")
	os.Setenv("ATEST_EXT_AI_PLUGIN_NAME", "env-plugin")
	os.Setenv("ATEST_EXT_AI_AI_DEFAULT_SERVICE", "claude")
	defer func() {
		os.Unsetenv("ATEST_EXT_AI_SERVER_HOST")
		os.Unsetenv("ATEST_EXT_AI_SERVER_PORT")
		os.Unsetenv("ATEST_EXT_AI_PLUGIN_NAME")
		os.Unsetenv("ATEST_EXT_AI_AI_DEFAULT_SERVICE")
	}()

	loader := NewLoader()
	err := loader.Load()
	if err != nil {
		t.Fatalf("Failed to load with environment variables: %v", err)
	}

	config := loader.GetConfig()

	// Verify environment variables override defaults
	if config.Server.Host != "env-host" {
		t.Errorf("Expected host from env 'env-host', got '%s'", config.Server.Host)
	}

	if config.Server.Port != 5555 {
		t.Errorf("Expected port from env 5555, got %d", config.Server.Port)
	}

	if config.Plugin.Name != "env-plugin" {
		t.Errorf("Expected plugin name from env 'env-plugin', got '%s'", config.Plugin.Name)
	}

	if config.AI.DefaultService != "claude" {
		t.Errorf("Expected service from env 'claude', got '%s'", config.AI.DefaultService)
	}
}

func TestExport(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	configData := `
server:
  host: "export-host"
  port: 4444
  timeout: "15s"
  max_connections: 25
  socket_path: "/tmp/export-test.sock"

plugin:
  name: "export-plugin"
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
      priority: 1
`

	if err := os.WriteFile(configFile, []byte(configData), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	loader := NewLoader()
	err := loader.LoadFromFile(configFile)
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Test YAML export
	yamlData, err := loader.Export("yaml")
	if err != nil {
		t.Fatalf("Failed to export YAML: %v", err)
	}

	if len(yamlData) == 0 {
		t.Error("Expected YAML data, got empty")
	}

	// Test JSON export
	jsonData, err := loader.Export("json")
	if err != nil {
		t.Fatalf("Failed to export JSON: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("Expected JSON data, got empty")
	}

	// Test TOML export
	tomlData, err := loader.Export("toml")
	if err != nil {
		t.Fatalf("Failed to export TOML: %v", err)
	}

	if len(tomlData) == 0 {
		t.Error("Expected TOML data, got empty")
	}

	// Test unsupported format
	_, err = loader.Export("xml")
	if err == nil {
		t.Error("Expected error for unsupported export format")
	}
}

func TestMergeConfigurations(t *testing.T) {
	loader1 := NewLoader()
	loader2 := NewLoader()

	// Load first configuration
	config1Data := `
server:
  host: "host1"
  port: 1111
plugin:
  name: "plugin1"
ai:
  default_service: "ollama"
  services:
    ollama:
      enabled: true
      provider: "ollama"
      model: "model1"
      priority: 1
`

	err := loader1.LoadFromBytes([]byte(config1Data), "yaml")
	if err != nil {
		t.Fatalf("Failed to load first configuration: %v", err)
	}

	// Load second configuration
	config2Data := `
server:
  port: 2222
plugin:
  version: "2.0.0"
ai:
  timeout: "120s"
  services:
    openai:
      enabled: true
      provider: "openai"
      model: "gpt-4"
      priority: 1
`

	err = loader2.LoadFromBytes([]byte(config2Data), "yaml")
	if err != nil {
		t.Fatalf("Failed to load second configuration: %v", err)
	}

	// Merge configurations
	config2 := loader2.GetConfig()
	err = loader1.Merge(config2)
	if err != nil {
		t.Fatalf("Failed to merge configurations: %v", err)
	}

	// Verify merged configuration
	mergedConfig := loader1.GetConfig()

	// Values from first config should remain
	if mergedConfig.Server.Host != "host1" {
		t.Errorf("Expected merged host 'host1', got '%s'", mergedConfig.Server.Host)
	}

	// Values from second config should override
	if mergedConfig.Server.Port != 2222 {
		t.Errorf("Expected merged port 2222, got %d", mergedConfig.Server.Port)
	}

	if mergedConfig.Plugin.Version != "2.0.0" {
		t.Errorf("Expected merged version '2.0.0', got '%s'", mergedConfig.Plugin.Version)
	}
}

func TestLoadNonexistentFile(t *testing.T) {
	loader := NewLoader()

	err := loader.LoadFromFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Expected error when loading nonexistent file")
	}
}

func TestLoadInvalidFormat(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.txt")

	configData := `invalid config format`

	if err := os.WriteFile(configFile, []byte(configData), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	loader := NewLoader()
	err := loader.LoadFromFile(configFile)
	if err == nil {
		t.Error("Expected error when loading file with unsupported format")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	configData := `
server:
  host: "test-host"
  port: 8080
plugin:
  name: "test-plugin"
  - invalid yaml syntax
`

	if err := os.WriteFile(configFile, []byte(configData), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	loader := NewLoader()
	err := loader.LoadFromFile(configFile)
	if err == nil {
		t.Error("Expected error when loading invalid YAML")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	loader := NewLoader()

	invalidJSON := []byte(`{
  "server": {
    "host": "test-host",
    "port": 8080,
  },
  "plugin": {
    "name": "test-plugin"
    // invalid JSON syntax
  }
}`)

	err := loader.LoadFromBytes(invalidJSON, "json")
	if err == nil {
		t.Error("Expected error when loading invalid JSON")
	}
}

func TestDetectFormat(t *testing.T) {
	loader := NewLoader()

	tests := []struct {
		filename string
		expected string
	}{
		{"config.yaml", "yaml"},
		{"config.yml", "yaml"},
		{"config.json", "json"},
		{"config.toml", "toml"},
		{"config.txt", ""},
		{"config", ""},
	}

	for _, test := range tests {
		format := loader.detectFormat(test.filename)
		if format != test.expected {
			t.Errorf("Expected format '%s' for file '%s', got '%s'", test.expected, test.filename, format)
		}
	}
}
