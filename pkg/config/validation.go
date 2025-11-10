package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// ValidationSeverity indicates whether an issue is an error or warning.
type ValidationSeverity string

const (
	// SeverityError represents a configuration error and should block startup.
	SeverityError ValidationSeverity = "error"
	// SeverityWarning represents a non-fatal issue that should be surfaced to the user.
	SeverityWarning ValidationSeverity = "warning"
)

// ValidationIssue describes a single validation finding.
type ValidationIssue struct {
	Field    string
	Value    interface{}
	Message  string
	Severity ValidationSeverity
}

func (i ValidationIssue) Error() string {
	if i.Value == nil {
		return fmt.Sprintf("[%s] %s: %s", i.Severity, i.Field, i.Message)
	}
	return fmt.Sprintf("[%s] %s: %s (value: %v)", i.Severity, i.Field, i.Message, i.Value)
}

// ValidationResult aggregates validation findings.
type ValidationResult struct {
	Errors   []ValidationIssue
	Warnings []ValidationIssue
}

// HasErrors reports whether blocking errors were found.
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// AddError appends a blocking validation error.
func (r *ValidationResult) AddError(field, message string, value interface{}) {
	r.Errors = append(r.Errors, ValidationIssue{
		Field:    field,
		Value:    value,
		Message:  message,
		Severity: SeverityError,
	})
}

// AddWarning appends a non-blocking validation warning.
func (r *ValidationResult) AddWarning(field, message string, value interface{}) {
	r.Warnings = append(r.Warnings, ValidationIssue{
		Field:    field,
		Value:    value,
		Message:  message,
		Severity: SeverityWarning,
	})
}

// Error returns an aggregated error when blocking issues exist.
func (r *ValidationResult) Error() error {
	if !r.HasErrors() {
		return nil
	}

	var builder strings.Builder
	builder.WriteString("configuration validation failed:")
	for _, issue := range r.Errors {
		builder.WriteString("\n  - ")
		builder.WriteString(issue.Error())
	}
	return errors.New(builder.String())
}

// Validate performs a comprehensive validation of the configuration.
func (cfg *Config) Validate() *ValidationResult {
	result := &ValidationResult{}

	cfg.validateServer(result)
	cfg.validateAI(result)
	cfg.validateRateLimit(result)
	cfg.validateRetry(result)
	cfg.validateCrossField(result)
	cfg.validateProviders(result)
	cfg.validateDatabase(result)
	cfg.validateLogging(result)

	return result
}

func (cfg *Config) validateServer(result *ValidationResult) {
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		result.AddError("server.port", "port must be between 1 and 65535", cfg.Server.Port)
	}

	if cfg.Server.Timeout.Duration <= 0 {
		result.AddError("server.timeout", "timeout must be greater than zero", cfg.Server.Timeout)
	}

	if cfg.Server.ReadTimeout.Duration <= 0 {
		result.AddError("server.read_timeout", "read_timeout must be greater than zero", cfg.Server.ReadTimeout)
	}

	if cfg.Server.WriteTimeout.Duration <= 0 {
		result.AddError("server.write_timeout", "write_timeout must be greater than zero", cfg.Server.WriteTimeout)
	}

	if cfg.Server.MaxConns < 1 {
		result.AddError("server.max_connections", "max_connections must be greater than zero", cfg.Server.MaxConns)
	}
}

func (cfg *Config) validateAI(result *ValidationResult) {
	if cfg.AI.Timeout.Duration <= 0 {
		result.AddError("ai.timeout", "timeout must be greater than zero", cfg.AI.Timeout)
	}

	if len(cfg.AI.Services) == 0 {
		result.AddError("ai.services", "at least one AI service must be configured", nil)
	}
}

func (cfg *Config) validateRateLimit(result *ValidationResult) {
	if !cfg.AI.RateLimit.Enabled {
		return
	}

	if cfg.AI.RateLimit.RequestsPerMinute <= 0 {
		result.AddError("ai.rate_limit.requests_per_minute", "requests_per_minute must be greater than zero", cfg.AI.RateLimit.RequestsPerMinute)
	}
	if cfg.AI.RateLimit.BurstSize <= 0 {
		result.AddError("ai.rate_limit.burst_size", "burst_size must be greater than zero", cfg.AI.RateLimit.BurstSize)
	}
	if cfg.AI.RateLimit.WindowSize.Duration <= 0 {
		result.AddError("ai.rate_limit.window_size", "window_size must be greater than zero", cfg.AI.RateLimit.WindowSize)
	}
}

func (cfg *Config) validateRetry(result *ValidationResult) {
	if !cfg.AI.Retry.Enabled {
		return
	}

	if cfg.AI.Retry.MaxAttempts <= 0 {
		result.AddError("ai.retry.max_attempts", "max_attempts must be greater than zero", cfg.AI.Retry.MaxAttempts)
	}
	if cfg.AI.Retry.InitialDelay.Duration < 0 {
		result.AddError("ai.retry.initial_delay", "initial_delay cannot be negative", cfg.AI.Retry.InitialDelay)
	}
	if cfg.AI.Retry.MaxDelay.Duration < 0 {
		result.AddError("ai.retry.max_delay", "max_delay cannot be negative", cfg.AI.Retry.MaxDelay)
	}
	if cfg.AI.Retry.Multiplier < 1 {
		result.AddWarning("ai.retry.multiplier", "multiplier below 1 disables exponential backoff", cfg.AI.Retry.Multiplier)
	}
}

func (cfg *Config) validateCrossField(result *ValidationResult) {
	if cfg.AI.DefaultService == "" {
		result.AddError("ai.default_service", "default_service must be configured", nil)
	} else if _, ok := cfg.AI.Services[cfg.AI.DefaultService]; !ok {
		result.AddError("ai.default_service", "default_service must reference an existing service", cfg.AI.DefaultService)
	} else if !cfg.AI.Services[cfg.AI.DefaultService].Enabled {
		result.AddWarning("ai.default_service", "default_service is disabled and will never be selected", cfg.AI.DefaultService)
	}

	seenFallback := make(map[string]struct{}, len(cfg.AI.Fallback))
	for idx, name := range cfg.AI.Fallback {
		key := strings.ToLower(strings.TrimSpace(name))
		if key == "" {
			result.AddWarning(fmt.Sprintf("ai.fallback_order[%d]", idx), "empty fallback entry ignored", name)
			continue
		}

		if _, ok := seenFallback[key]; ok {
			result.AddWarning(fmt.Sprintf("ai.fallback_order[%d]", idx), "duplicate fallback entry", name)
		}
		seenFallback[key] = struct{}{}

		if _, ok := cfg.AI.Services[name]; !ok {
			result.AddError(fmt.Sprintf("ai.fallback_order[%d]", idx), "fallback service does not exist", name)
		}
		if name == cfg.AI.DefaultService {
			result.AddWarning(fmt.Sprintf("ai.fallback_order[%d]", idx), "default service should not appear in fallback list", name)
		}
	}
}

func (cfg *Config) validateProviders(result *ValidationResult) {
	if len(cfg.AI.Services) == 0 {
		return
	}

	knownProviders := []string{"ollama", "openai", "claude", "deepseek", "custom"}
	providerRules := map[string]struct {
		requireAPIKey   bool
		requireEndpoint bool
	}{
		"ollama":   {requireEndpoint: true},
		"openai":   {requireAPIKey: true, requireEndpoint: true},
		"claude":   {requireAPIKey: true, requireEndpoint: true},
		"deepseek": {requireAPIKey: true, requireEndpoint: true},
		"custom":   {requireEndpoint: true},
	}

	for name, svc := range cfg.AI.Services {
		if !svc.Enabled {
			continue
		}

		fieldPrefix := fmt.Sprintf("ai.services.%s", name)
		provider := normalizeProviderName(svc.Provider)
		if provider == "" {
			result.AddError(fieldPrefix+".provider", "provider must be specified", svc.Provider)
			continue
		}

		rules, ok := providerRules[provider]
		if !ok {
			result.AddError(fieldPrefix+".provider", fmt.Sprintf("unknown provider (valid: %s)", strings.Join(knownProviders, ", ")), svc.Provider)
			continue
		}

		if rules.requireAPIKey && strings.TrimSpace(svc.APIKey) == "" {
			result.AddError(fieldPrefix+".api_key", fmt.Sprintf("%s provider requires an API key", provider), nil)
		}

		if rules.requireEndpoint {
			if strings.TrimSpace(svc.Endpoint) == "" {
				result.AddError(fieldPrefix+".endpoint", fmt.Sprintf("%s provider requires an endpoint", provider), nil)
			} else if !isValidEndpoint(svc.Endpoint) {
				result.AddWarning(fieldPrefix+".endpoint", "endpoint is not a valid URL", svc.Endpoint)
			}
		}

		if svc.MaxTokens <= 0 {
			result.AddWarning(fieldPrefix+".max_tokens", "max_tokens should be greater than zero", svc.MaxTokens)
		} else if svc.MaxTokens > 128000 {
			result.AddWarning(fieldPrefix+".max_tokens", "max_tokens exceeds typical limits (128000)", svc.MaxTokens)
		}

		if svc.Timeout.Duration <= 0 {
			result.AddWarning(fieldPrefix+".timeout", "timeout should be greater than zero", svc.Timeout)
		}

		if provider == "ollama" && strings.TrimSpace(svc.Model) == "" {
			result.AddWarning(fieldPrefix+".model", "model not specified for ollama provider", nil)
		}
	}
}

func (cfg *Config) validateDatabase(result *ValidationResult) {
	if !cfg.Database.Enabled {
		return
	}

	validDrivers := []string{"sqlite", "mysql", "postgresql"}
	if !containsFold(validDrivers, cfg.Database.Driver) {
		result.AddError("database.driver", fmt.Sprintf("driver must be one of %s", strings.Join(validDrivers, ", ")), cfg.Database.Driver)
	}

	if strings.TrimSpace(cfg.Database.DSN) == "" {
		result.AddError("database.dsn", "dsn must be provided when database integration is enabled", nil)
	}

	if cfg.Database.MaxConns < 0 {
		result.AddWarning("database.max_connections", "max_connections should be non-negative", cfg.Database.MaxConns)
	}
	if cfg.Database.MaxIdle < 0 {
		result.AddWarning("database.max_idle", "max_idle should be non-negative", cfg.Database.MaxIdle)
	}
	if cfg.Database.MaxLifetime.Duration < 0 {
		result.AddWarning("database.max_lifetime", "max_lifetime should not be negative", cfg.Database.MaxLifetime)
	}
}

func (cfg *Config) validateLogging(result *ValidationResult) {
	validFormats := []string{"json", "text"}
	if cfg.Logging.Format != "" && !containsFold(validFormats, cfg.Logging.Format) {
		result.AddError("logging.format", fmt.Sprintf("format must be one of %s", strings.Join(validFormats, ", ")), cfg.Logging.Format)
	}

	validOutputs := []string{"stdout", "stderr", "file"}
	if cfg.Logging.Output != "" && !containsFold(validOutputs, cfg.Logging.Output) {
		result.AddError("logging.output", fmt.Sprintf("output must be one of %s", strings.Join(validOutputs, ", ")), cfg.Logging.Output)
	}

	if strings.EqualFold(cfg.Logging.Output, "file") && strings.TrimSpace(cfg.Logging.File.Path) == "" {
		result.AddError("logging.file.path", "log file path is required when output is 'file'", nil)
	}
}

func normalizeProviderName(provider string) string {
	p := strings.ToLower(strings.TrimSpace(provider))
	if p == "local" {
		return "ollama"
	}
	return p
}

func containsFold(haystack []string, needle string) bool {
	for _, item := range haystack {
		if strings.EqualFold(item, needle) {
			return true
		}
	}
	return false
}

func isValidEndpoint(endpoint string) bool {
	if endpoint == "" {
		return false
	}
	parsed, err := url.ParseRequestURI(endpoint)
	if err != nil {
		return false
	}
	return parsed.Scheme != "" && parsed.Host != ""
}
