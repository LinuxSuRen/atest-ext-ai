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
	"testing"
	"time"
)

func TestValidatorCreation(t *testing.T) {
	validator := NewValidator()
	if validator == nil {
		t.Fatal("Expected validator to be created, got nil")
	}
}

func TestValidateValidConfiguration(t *testing.T) {
	validator := NewValidator()

	config := &Config{
		Server: ServerConfig{
			Host:         "localhost",
			Port:         8080,
			Timeout:      NewDuration(30 * time.Second),
			MaxConns:     100,
			SocketPath:   "/tmp/test.sock",
			ReadTimeout:  NewDuration(15 * time.Second),
			WriteTimeout: NewDuration(15 * time.Second),
		},
		Plugin: PluginConfig{
			Name:        "test-plugin",
			Version:     "1.0.0",
			Debug:       false,
			LogLevel:    "info",
			Environment: "development",
			Metadata:    map[string]string{"key": "value"},
		},
		AI: AIConfig{
			DefaultService: "ollama",
			Services: map[string]AIService{
				"ollama": {
					Enabled:     true,
					Provider:    "ollama",
					Endpoint:    "http://localhost:11434",
					Model:       "codellama",
					MaxTokens:   4096,
					Temperature: 0.7,
					TopP:        0.9,
					Priority:    1,
					Timeout:     NewDuration(60 * time.Second),
				},
			},
			Fallback: []string{"ollama"},
			Timeout:  NewDuration(60 * time.Second),
			RateLimit: RateLimitConfig{
				Enabled:           true,
				RequestsPerMinute: 60,
				BurstSize:         10,
				WindowSize:        NewDuration(time.Minute),
			},
			CircuitBreaker: CircuitBreakerConfig{
				Enabled:          true,
				FailureThreshold: 5,
				SuccessThreshold: 3,
				Timeout:          NewDuration(30 * time.Second),
				ResetTimeout:     NewDuration(60 * time.Second),
			},
			Retry: RetryConfig{
				Enabled:      true,
				MaxAttempts:  3,
				InitialDelay: NewDuration(time.Second),
				MaxDelay:     NewDuration(30 * time.Second),
				Multiplier:   2.0,
				Jitter:       true,
			},
			Cache: CacheConfig{
				Enabled:  true,
				TTL:      NewDuration(time.Hour),
				MaxSize:  1000,
				Provider: "memory",
			},
			Security: SecurityConfig{
				EncryptCredentials: false,
				AllowedHosts:      []string{"localhost"},
				TLSEnabled:        false,
			},
		},
		Database: DatabaseConfig{
			Enabled:     false,
			Driver:      "sqlite",
			DSN:         "file:test.db?cache=shared&mode=rwc",
			MaxConns:    10,
			MaxIdle:     5,
			MaxLifetime: NewDuration(time.Hour),
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
			File: LogFileConfig{
				Path:       "/var/log/test.log",
				MaxSize:    "100MB",
				MaxBackups: 3,
				MaxAge:     28,
				Compress:   true,
			},
			Rotation: LogRotationConfig{
				Enabled:  true,
				Size:     "100MB",
				Count:    5,
				Age:      "30d",
				Compress: true,
			},
		},
	}

	err := validator.ValidateConfig(config)
	if err != nil {
		t.Fatalf("Expected valid configuration to pass validation, got error: %v", err)
	}
}

func TestValidateInvalidServerConfiguration(t *testing.T) {
	validator := NewValidator()

	config := &Config{
		Server: ServerConfig{
			Host:       "", // Invalid: empty host
			Port:       -1, // Invalid: negative port
			Timeout:    NewDuration(0),  // Invalid: zero timeout
			MaxConns:   -1, // Invalid: negative max connections
			SocketPath: "", // Invalid: empty socket path
		},
		Plugin: PluginConfig{
			Name:        "test-plugin",
			Version:     "1.0.0",
			LogLevel:    "info",
			Environment: "development",
		},
		AI: AIConfig{
			DefaultService: "ollama",
			Services: map[string]AIService{
				"ollama": {
					Enabled:  true,
					Provider: "ollama",
					Endpoint: "http://localhost:11434",
					Model:    "codellama",
					Priority: 1,
				},
			},
			Timeout: NewDuration(60 * time.Second),
		},
	}

	err := validator.ValidateConfig(config)
	if err == nil {
		t.Fatal("Expected validation to fail for invalid server configuration")
	}

	if validationErrs, ok := err.(ValidationErrors); ok {
		if len(validationErrs) == 0 {
			t.Fatal("Expected validation errors, got none")
		}

		// Check for specific validation errors
		foundHostError := false
		foundPortError := false
		foundTimeoutError := false

		for _, validationErr := range validationErrs {
			if validationErr.Field == "Host" {
				foundHostError = true
			}
			if validationErr.Field == "Port" {
				foundPortError = true
			}
			if validationErr.Field == "Timeout" {
				foundTimeoutError = true
			}
		}

		if !foundHostError {
			t.Error("Expected host validation error")
		}
		if !foundPortError {
			t.Error("Expected port validation error")
		}
		if !foundTimeoutError {
			t.Error("Expected timeout validation error")
		}
	} else {
		t.Fatalf("Expected ValidationErrors, got %T", err)
	}
}

func TestValidateInvalidPluginConfiguration(t *testing.T) {
	validator := NewValidator()

	config := &Config{
		Server: ServerConfig{
			Host:       "localhost",
			Port:       8080,
			Timeout:    NewDuration(30 * time.Second),
			MaxConns:   100,
			SocketPath: "/tmp/test.sock",
		},
		Plugin: PluginConfig{
			Name:        "", // Invalid: empty name
			Version:     "invalid-version", // Invalid: not semver
			LogLevel:    "invalid", // Invalid: not a valid log level
			Environment: "invalid", // Invalid: not a valid environment
		},
		AI: AIConfig{
			DefaultService: "ollama",
			Services: map[string]AIService{
				"ollama": {
					Enabled:  true,
					Provider: "ollama",
					Endpoint: "http://localhost:11434",
					Model:    "codellama",
					Priority: 1,
				},
			},
			Timeout: NewDuration(60 * time.Second),
		},
	}

	err := validator.ValidateConfig(config)
	if err == nil {
		t.Fatal("Expected validation to fail for invalid plugin configuration")
	}

	if validationErrs, ok := err.(ValidationErrors); ok {
		if len(validationErrs) == 0 {
			t.Fatal("Expected validation errors, got none")
		}
	} else {
		t.Fatalf("Expected ValidationErrors, got %T", err)
	}
}

func TestValidateInvalidAIConfiguration(t *testing.T) {
	validator := NewValidator()

	config := &Config{
		Server: ServerConfig{
			Host:       "localhost",
			Port:       8080,
			Timeout:    NewDuration(30 * time.Second),
			MaxConns:   100,
			SocketPath: "/tmp/test.sock",
		},
		Plugin: PluginConfig{
			Name:        "test-plugin",
			Version:     "1.0.0",
			LogLevel:    "info",
			Environment: "development",
		},
		AI: AIConfig{
			DefaultService: "nonexistent", // Invalid: service doesn't exist
			Services: map[string]AIService{
				"ollama": {
					Enabled:     false, // Invalid: default service is disabled
					Provider:    "ollama",
					Endpoint:    "invalid-url", // Invalid: not a valid URL
					Model:       "",            // Invalid: empty model
					Temperature: 3.0,           // Invalid: temperature > 2
					TopP:        1.5,           // Invalid: top_p > 1
					Priority:    0,             // Invalid: priority < 1
				},
			},
			Fallback: []string{"nonexistent"}, // Invalid: fallback service doesn't exist
			Timeout:  NewDuration(0),                       // Invalid: zero timeout
		},
	}

	err := validator.ValidateConfig(config)
	if err == nil {
		t.Fatal("Expected validation to fail for invalid AI configuration")
	}

	if validationErrs, ok := err.(ValidationErrors); ok {
		if len(validationErrs) == 0 {
			t.Fatal("Expected validation errors, got none")
		}
	} else {
		t.Fatalf("Expected ValidationErrors, got %T", err)
	}
}

func TestValidateAIServiceConfiguration(t *testing.T) {
	validator := NewValidator()

	// Test Ollama service validation
	ollamaConfig := &Config{
		Server: ServerConfig{
			Host:       "localhost",
			Port:       8080,
			Timeout:    NewDuration(30 * time.Second),
			MaxConns:   100,
			SocketPath: "/tmp/test.sock",
		},
		Plugin: PluginConfig{
			Name:        "test-plugin",
			Version:     "1.0.0",
			LogLevel:    "info",
			Environment: "development",
		},
		AI: AIConfig{
			DefaultService: "ollama",
			Services: map[string]AIService{
				"ollama": {
					Enabled:  true,
					Provider: "ollama",
					Endpoint: "", // Invalid: empty endpoint for Ollama
					Model:    "codellama",
					Priority: 1,
				},
			},
			Timeout: NewDuration(60 * time.Second),
		},
	}

	err := validator.ValidateConfig(ollamaConfig)
	if err == nil {
		t.Fatal("Expected validation to fail for Ollama service without endpoint")
	}

	// Test OpenAI service validation
	openaiConfig := &Config{
		Server: ServerConfig{
			Host:       "localhost",
			Port:       8080,
			Timeout:    NewDuration(30 * time.Second),
			MaxConns:   100,
			SocketPath: "/tmp/test.sock",
		},
		Plugin: PluginConfig{
			Name:        "test-plugin",
			Version:     "1.0.0",
			LogLevel:    "info",
			Environment: "development",
		},
		AI: AIConfig{
			DefaultService: "openai",
			Services: map[string]AIService{
				"openai": {
					Enabled:  true,
					Provider: "openai",
					APIKey:   "", // Invalid: empty API key for OpenAI
					Model:    "gpt-4",
					Priority: 1,
				},
			},
			Timeout: NewDuration(60 * time.Second),
		},
	}

	err = validator.ValidateConfig(openaiConfig)
	if err == nil {
		t.Fatal("Expected validation to fail for OpenAI service without API key")
	}

	// Test Claude service validation
	claudeConfig := &Config{
		Server: ServerConfig{
			Host:       "localhost",
			Port:       8080,
			Timeout:    NewDuration(30 * time.Second),
			MaxConns:   100,
			SocketPath: "/tmp/test.sock",
		},
		Plugin: PluginConfig{
			Name:        "test-plugin",
			Version:     "1.0.0",
			LogLevel:    "info",
			Environment: "development",
		},
		AI: AIConfig{
			DefaultService: "claude",
			Services: map[string]AIService{
				"claude": {
					Enabled:  true,
					Provider: "claude",
					APIKey:   "", // Invalid: empty API key for Claude
					Model:    "claude-3-sonnet",
					Priority: 1,
				},
			},
			Timeout: NewDuration(60 * time.Second),
		},
	}

	err = validator.ValidateConfig(claudeConfig)
	if err == nil {
		t.Fatal("Expected validation to fail for Claude service without API key")
	}
}

func TestValidateRateLimitConfiguration(t *testing.T) {
	validator := NewValidator()

	config := &Config{
		Server: ServerConfig{
			Host:       "localhost",
			Port:       8080,
			Timeout:    NewDuration(30 * time.Second),
			MaxConns:   100,
			SocketPath: "/tmp/test.sock",
		},
		Plugin: PluginConfig{
			Name:        "test-plugin",
			Version:     "1.0.0",
			LogLevel:    "info",
			Environment: "development",
		},
		AI: AIConfig{
			DefaultService: "ollama",
			Services: map[string]AIService{
				"ollama": {
					Enabled:  true,
					Provider: "ollama",
					Endpoint: "http://localhost:11434",
					Model:    "codellama",
					Priority: 1,
				},
			},
			Timeout: NewDuration(60 * time.Second),
			RateLimit: RateLimitConfig{
				Enabled:           true,
				RequestsPerMinute: 0,  // Invalid: must be > 0
				BurstSize:         0,  // Invalid: must be > 0
			},
		},
	}

	err := validator.ValidateConfig(config)
	if err == nil {
		t.Fatal("Expected validation to fail for invalid rate limit configuration")
	}
}

func TestValidateLoggingConfiguration(t *testing.T) {
	validator := NewValidator()

	config := &Config{
		Server: ServerConfig{
			Host:       "localhost",
			Port:       8080,
			Timeout:    NewDuration(30 * time.Second),
			MaxConns:   100,
			SocketPath: "/tmp/test.sock",
		},
		Plugin: PluginConfig{
			Name:        "test-plugin",
			Version:     "1.0.0",
			LogLevel:    "info",
			Environment: "development",
		},
		AI: AIConfig{
			DefaultService: "ollama",
			Services: map[string]AIService{
				"ollama": {
					Enabled:  true,
					Provider: "ollama",
					Endpoint: "http://localhost:11434",
					Model:    "codellama",
					Priority: 1,
				},
			},
			Timeout: NewDuration(60 * time.Second),
		},
		Logging: LoggingConfig{
			Level:  "invalid", // Invalid: not a valid log level
			Format: "invalid", // Invalid: not a valid format
			Output: "file",    // Requires file configuration
			File: LogFileConfig{
				Path:    "", // Invalid: empty path when output is file
				MaxSize: "invalid-size", // Invalid: not a valid size format
			},
		},
	}

	err := validator.ValidateConfig(config)
	if err == nil {
		t.Fatal("Expected validation to fail for invalid logging configuration")
	}
}

func TestCustomValidations(t *testing.T) {
	validator := NewValidator()

	// Test semver validation
	type TestSemver struct {
		Version string `validate:"semver"`
	}

	validSemver := TestSemver{Version: "1.0.0"}
	err := validator.ValidateStruct(validSemver)
	if err != nil {
		t.Errorf("Expected valid semver to pass, got error: %v", err)
	}

	invalidSemver := TestSemver{Version: "invalid-version"}
	err = validator.ValidateStruct(invalidSemver)
	if err == nil {
		t.Error("Expected invalid semver to fail validation")
	}

	// Test log level validation
	type TestLogLevel struct {
		Level string `validate:"log_level"`
	}

	validLevel := TestLogLevel{Level: "info"}
	err = validator.ValidateStruct(validLevel)
	if err != nil {
		t.Errorf("Expected valid log level to pass, got error: %v", err)
	}

	invalidLevel := TestLogLevel{Level: "invalid"}
	err = validator.ValidateStruct(invalidLevel)
	if err == nil {
		t.Error("Expected invalid log level to fail validation")
	}

	// Test duration validation
	type TestDuration struct {
		Duration string `validate:"duration"`
	}

	validDuration := TestDuration{Duration: "30s"}
	err = validator.ValidateStruct(validDuration)
	if err != nil {
		t.Errorf("Expected valid duration to pass, got error: %v", err)
	}

	invalidDuration := TestDuration{Duration: "invalid-duration"}
	err = validator.ValidateStruct(invalidDuration)
	if err == nil {
		t.Error("Expected invalid duration to fail validation")
	}
}

func TestValidationErrorMessages(t *testing.T) {
	validator := NewValidator()

	config := &Config{
		Server: ServerConfig{
			Host:       "",
			Port:       -1,
			Timeout:    NewDuration(0),
			MaxConns:   0,
			SocketPath: "",
		},
		Plugin: PluginConfig{
			Name:        "",
			Version:     "",
			LogLevel:    "",
			Environment: "",
		},
		AI: AIConfig{
			DefaultService: "",
			Services:       nil,
			Timeout:        NewDuration(0),
		},
	}

	err := validator.ValidateConfig(config)
	if err == nil {
		t.Fatal("Expected validation to fail")
	}

	if validationErrs, ok := err.(ValidationErrors); ok {
		if len(validationErrs) == 0 {
			t.Fatal("Expected validation errors, got none")
		}

		// Check that error messages are human-readable
		for _, validationErr := range validationErrs {
			if validationErr.Message == "" {
				t.Errorf("Expected error message for field %s, got empty", validationErr.Field)
			}
		}
	} else {
		t.Fatalf("Expected ValidationErrors, got %T", err)
	}
}