package config

import (
	"testing"
	"time"
)

func TestValidate_DefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	result := cfg.Validate()

	if result.HasErrors() {
		t.Fatalf("expected default config to be valid, got errors: %v", result.Errors)
	}
}

func TestValidate_DefaultServiceMustExist(t *testing.T) {
	cfg := defaultConfig()
	cfg.AI.DefaultService = "missing"

	result := cfg.Validate()
	if !result.HasErrors() {
		t.Fatalf("expected validation error for missing default service")
	}
	if !hasErrorFor(result, "ai.default_service") {
		t.Errorf("expected error for ai.default_service, got %v", result.Errors)
	}
}

func TestValidate_OpenAIRequiresAPIKey(t *testing.T) {
	cfg := defaultConfig()
	cfg.AI.Services["openai"] = AIService{
		Enabled:   true,
		Provider:  "openai",
		Endpoint:  "https://api.openai.com",
		Model:     "gpt-4",
		APIKey:    "",
		MaxTokens: 1000,
		Timeout:   NewDuration(30 * time.Second),
	}

	result := cfg.Validate()
	if !hasErrorFor(result, "ai.services.openai.api_key") {
		t.Fatalf("expected API key error, got %v", result.Errors)
	}
}

func TestValidate_OllamaRequiresEndpoint(t *testing.T) {
	cfg := defaultConfig()
	ollama := cfg.AI.Services["ollama"]
	ollama.Endpoint = ""
	cfg.AI.Services["ollama"] = ollama

	result := cfg.Validate()
	if !hasErrorFor(result, "ai.services.ollama.endpoint") {
		t.Fatalf("expected endpoint error for ollama provider")
	}
}

func TestValidate_FallbackMustExist(t *testing.T) {
	cfg := defaultConfig()
	cfg.AI.Fallback = []string{"missing-service"}

	result := cfg.Validate()
	if !hasErrorFor(result, "ai.fallback_order[0]") {
		t.Fatalf("expected fallback error when referencing missing service")
	}
}

func TestValidate_DatabaseDriverRequiredWhenEnabled(t *testing.T) {
	cfg := defaultConfig()
	cfg.Database.Enabled = true
	cfg.Database.Driver = "oracle"
	cfg.Database.DSN = ""

	result := cfg.Validate()
	if !hasErrorFor(result, "database.driver") {
		t.Fatalf("expected error for invalid database driver")
	}
	if !hasErrorFor(result, "database.dsn") {
		t.Fatalf("expected error for empty database dsn")
	}
}

func hasErrorFor(result *ValidationResult, field string) bool {
	for _, issue := range result.Errors {
		if issue.Field == field {
			return true
		}
	}
	return false
}
