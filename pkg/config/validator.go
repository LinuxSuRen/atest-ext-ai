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
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

// Validator handles configuration validation
type Validator struct {
	validator *validator.Validate
}

// NewValidator creates a new configuration validator
func NewValidator() *Validator {
	validate := validator.New()

	// Register custom validation functions
	registerCustomValidations(validate)

	return &Validator{
		validator: validate,
	}
}

// ValidateConfig validates the complete configuration
func (v *Validator) ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	if err := v.validator.Struct(config); err != nil {
		return v.formatValidationErrors(err)
	}

	// Additional custom validations
	if err := v.validateBusinessLogic(config); err != nil {
		return err
	}

	return nil
}

// ValidateStruct validates any struct with validation tags
func (v *Validator) ValidateStruct(s interface{}) error {
	if err := v.validator.Struct(s); err != nil {
		return v.formatValidationErrors(err)
	}
	return nil
}

// formatValidationErrors converts validator errors to our custom error format
func (v *Validator) formatValidationErrors(err error) error {
	var validationErrors ValidationErrors

	if validatorErrs, ok := err.(validator.ValidationErrors); ok {
		for _, fieldErr := range validatorErrs {
			validationErrors = append(validationErrors, ValidationError{
				Field:   fieldErr.Field(),
				Value:   fieldErr.Value(),
				Tag:     fieldErr.Tag(),
				Message: v.generateErrorMessage(fieldErr),
			})
		}
	}

	return validationErrors
}

// generateErrorMessage generates human-readable error messages
func (v *Validator) generateErrorMessage(err validator.FieldError) string {
	field := err.Field()
	value := fmt.Sprintf("%v", err.Value())
	tag := err.Tag()
	param := err.Param()

	switch tag {
	case "required":
		return fmt.Sprintf("field '%s' is required", field)
	case "min":
		return fmt.Sprintf("field '%s' must be at least %s", field, param)
	case "max":
		return fmt.Sprintf("field '%s' must be at most %s", field, param)
	case "oneof":
		return fmt.Sprintf("field '%s' must be one of: %s", field, param)
	case "url":
		return fmt.Sprintf("field '%s' must be a valid URL", field)
	case "hostname_rfc1123":
		return fmt.Sprintf("field '%s' must be a valid hostname", field)
	case "file":
		return fmt.Sprintf("field '%s' must be a valid file path", field)
	case "semver":
		return fmt.Sprintf("field '%s' must be a valid semantic version", field)
	case "duration":
		return fmt.Sprintf("field '%s' must be a valid duration", field)
	case "ai_service_config":
		return fmt.Sprintf("AI service configuration for '%s' is invalid", field)
	case "log_level":
		return fmt.Sprintf("field '%s' must be a valid log level (debug, info, warn, error)", field)
	default:
		return fmt.Sprintf("field '%s' with value '%s' failed validation for tag '%s'", field, value, tag)
	}
}

// validateBusinessLogic performs additional business logic validations
func (v *Validator) validateBusinessLogic(config *Config) error {
	var errors ValidationErrors

	// Validate AI service configuration consistency
	if err := v.validateAIServices(config.AI); err != nil {
		if valErrors, ok := err.(ValidationErrors); ok {
			errors = append(errors, valErrors...)
		} else {
			errors = append(errors, ValidationError{
				Field:   "ai",
				Message: err.Error(),
			})
		}
	}

	// Validate database configuration
	if config.Database.Enabled {
		if err := v.validateDatabaseConfig(config.Database); err != nil {
			if valErrors, ok := err.(ValidationErrors); ok {
				errors = append(errors, valErrors...)
			} else {
				errors = append(errors, ValidationError{
					Field:   "database",
					Message: err.Error(),
				})
			}
		}
	}

	// Validate logging configuration
	if err := v.validateLoggingConfig(config.Logging); err != nil {
		if valErrors, ok := err.(ValidationErrors); ok {
			errors = append(errors, valErrors...)
		} else {
			errors = append(errors, ValidationError{
				Field:   "logging",
				Message: err.Error(),
			})
		}
	}

	// Validate security settings
	if err := v.validateSecurityConfig(config.AI.Security); err != nil {
		if valErrors, ok := err.(ValidationErrors); ok {
			errors = append(errors, valErrors...)
		} else {
			errors = append(errors, ValidationError{
				Field:   "ai.security",
				Message: err.Error(),
			})
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// validateAIServices validates AI service configurations
func (v *Validator) validateAIServices(ai AIConfig) error {
	var errors ValidationErrors

	// Check if default service exists and is enabled
	if ai.DefaultService != "" {
		if service, exists := ai.Services[ai.DefaultService]; !exists {
			errors = append(errors, ValidationError{
				Field:   "ai.default_service",
				Value:   ai.DefaultService,
				Message: fmt.Sprintf("default service '%s' is not defined in services", ai.DefaultService),
			})
		} else if !service.Enabled {
			errors = append(errors, ValidationError{
				Field:   "ai.default_service",
				Value:   ai.DefaultService,
				Message: fmt.Sprintf("default service '%s' is not enabled", ai.DefaultService),
			})
		}
	}

	// Validate each enabled AI service
	for name, service := range ai.Services {
		if service.Enabled {
			if err := v.validateAIService(name, service); err != nil {
				if valErrors, ok := err.(ValidationErrors); ok {
					errors = append(errors, valErrors...)
				} else {
					errors = append(errors, ValidationError{
						Field:   fmt.Sprintf("ai.services.%s", name),
						Message: err.Error(),
					})
				}
			}
		}
	}

	// Validate fallback order
	for _, serviceName := range ai.Fallback {
		if _, exists := ai.Services[serviceName]; !exists {
			errors = append(errors, ValidationError{
				Field:   "ai.fallback_order",
				Value:   serviceName,
				Message: fmt.Sprintf("fallback service '%s' is not defined in services", serviceName),
			})
		}
	}

	// Validate rate limiting configuration
	if ai.RateLimit.Enabled {
		if ai.RateLimit.RequestsPerMinute <= 0 {
			errors = append(errors, ValidationError{
				Field:   "ai.rate_limit.requests_per_minute",
				Value:   ai.RateLimit.RequestsPerMinute,
				Message: "requests per minute must be greater than 0",
			})
		}
		if ai.RateLimit.BurstSize <= 0 {
			errors = append(errors, ValidationError{
				Field:   "ai.rate_limit.burst_size",
				Value:   ai.RateLimit.BurstSize,
				Message: "burst size must be greater than 0",
			})
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// validateAIService validates a single AI service configuration
func (v *Validator) validateAIService(name string, service AIService) error {
	var errors ValidationErrors

	// Provider-specific validations
	switch service.Provider {
	case "ollama":
		if service.Endpoint == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("ai.services.%s.endpoint", name),
				Message: "endpoint is required for Ollama provider",
			})
		} else if !isValidURL(service.Endpoint) {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("ai.services.%s.endpoint", name),
				Value:   service.Endpoint,
				Message: "endpoint must be a valid URL",
			})
		}

	case "openai", "claude":
		if service.APIKey == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("ai.services.%s.api_key", name),
				Message: fmt.Sprintf("API key is required for %s provider", service.Provider),
			})
		}
	}

	// Validate temperature range
	if service.Temperature < 0 || service.Temperature > 2 {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("ai.services.%s.temperature", name),
			Value:   service.Temperature,
			Message: "temperature must be between 0 and 2",
		})
	}

	// Validate top_p range
	if service.TopP < 0 || service.TopP > 1 {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("ai.services.%s.top_p", name),
			Value:   service.TopP,
			Message: "top_p must be between 0 and 1",
		})
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// validateDatabaseConfig validates database configuration
func (v *Validator) validateDatabaseConfig(db DatabaseConfig) error {
	var errors ValidationErrors

	if db.DSN == "" {
		errors = append(errors, ValidationError{
			Field:   "database.dsn",
			Message: "DSN is required when database is enabled",
		})
	}

	if db.MaxConns <= 0 {
		errors = append(errors, ValidationError{
			Field:   "database.max_connections",
			Value:   db.MaxConns,
			Message: "max connections must be greater than 0",
		})
	}

	if db.MaxIdle <= 0 || db.MaxIdle > db.MaxConns {
		errors = append(errors, ValidationError{
			Field:   "database.max_idle",
			Value:   db.MaxIdle,
			Message: "max idle must be between 1 and max connections",
		})
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// validateLoggingConfig validates logging configuration
func (v *Validator) validateLoggingConfig(logging LoggingConfig) error {
	var errors ValidationErrors

	// Validate file logging configuration
	if logging.Output == "file" {
		if logging.File.Path == "" {
			errors = append(errors, ValidationError{
				Field:   "logging.file.path",
				Message: "file path is required when output is 'file'",
			})
		}

		if logging.File.MaxSize != "" {
			if !isValidSize(logging.File.MaxSize) {
				errors = append(errors, ValidationError{
					Field:   "logging.file.max_size",
					Value:   logging.File.MaxSize,
					Message: "max_size must be a valid size (e.g., '100MB', '1GB')",
				})
			}
		}
	}

	// Validate log rotation
	if logging.Rotation.Enabled {
		if logging.Rotation.Size != "" && !isValidSize(logging.Rotation.Size) {
			errors = append(errors, ValidationError{
				Field:   "logging.rotation.size",
				Value:   logging.Rotation.Size,
				Message: "rotation size must be a valid size (e.g., '100MB', '1GB')",
			})
		}

		if logging.Rotation.Age != "" && !isValidDuration(logging.Rotation.Age) {
			errors = append(errors, ValidationError{
				Field:   "logging.rotation.age",
				Value:   logging.Rotation.Age,
				Message: "rotation age must be a valid duration (e.g., '30d', '1h')",
			})
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// validateSecurityConfig validates security configuration
func (v *Validator) validateSecurityConfig(security SecurityConfig) error {
	var errors ValidationErrors

	// Validate TLS configuration
	if security.TLSEnabled {
		if security.CertFile == "" {
			errors = append(errors, ValidationError{
				Field:   "ai.security.cert_file",
				Message: "certificate file is required when TLS is enabled",
			})
		} else if !fileExists(security.CertFile) {
			errors = append(errors, ValidationError{
				Field:   "ai.security.cert_file",
				Value:   security.CertFile,
				Message: "certificate file does not exist",
			})
		}

		if security.KeyFile == "" {
			errors = append(errors, ValidationError{
				Field:   "ai.security.key_file",
				Message: "key file is required when TLS is enabled",
			})
		} else if !fileExists(security.KeyFile) {
			errors = append(errors, ValidationError{
				Field:   "ai.security.key_file",
				Value:   security.KeyFile,
				Message: "key file does not exist",
			})
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// registerCustomValidations registers custom validation functions
func registerCustomValidations(validate *validator.Validate) {
	// Register semver validation
	if err := validate.RegisterValidation("semver", validateSemVer); err != nil {
		// Log error but continue - custom validations are not critical
		log.Printf("Failed to register semver validation: %v", err)
	}

	// Register file validation
	if err := validate.RegisterValidation("file", validateFile); err != nil {
		// Log error but continue - custom validations are not critical
		log.Printf("Failed to register file validation: %v", err)
	}

	// Register duration validation
	if err := validate.RegisterValidation("duration", validateDuration); err != nil {
		// Log error but continue - custom validations are not critical
		log.Printf("Failed to register duration validation: %v", err)
	}

	// Register log level validation
	if err := validate.RegisterValidation("log_level", validateLogLevel); err != nil {
		// Log error but continue - custom validations are not critical
		log.Printf("Failed to register log level validation: %v", err)
	}
}

// validateSemVer validates semantic versioning
func validateSemVer(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // Allow empty values, required tag will catch it if needed
	}

	// Simplified semver regex
	semverRegex := regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)
	return semverRegex.MatchString(value)
}

// validateFile validates file path
func validateFile(fl validator.FieldLevel) bool {
	path := fl.Field().String()
	if path == "" {
		return true // Allow empty values
	}
	return fileExists(path)
}

// validateDuration validates duration string
func validateDuration(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true // Allow empty values
	}
	return isValidDuration(value)
}

// validateLogLevel validates log level
func validateLogLevel(fl validator.FieldLevel) bool {
	level := fl.Field().String()
	validLevels := []string{"debug", "info", "warn", "error"}
	for _, valid := range validLevels {
		if level == valid {
			return true
		}
	}
	return false
}

// Helper functions

// isValidURL checks if a string is a valid URL
func isValidURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// isValidSize validates size format (e.g., "100MB", "1GB")
func isValidSize(size string) bool {
	if size == "" {
		return true
	}
	sizeRegex := regexp.MustCompile(`^(\d+(?:\.\d+)?)(B|KB|MB|GB|TB)$`)
	return sizeRegex.MatchString(strings.ToUpper(size))
}

// isValidDuration validates duration format
func isValidDuration(duration string) bool {
	if duration == "" {
		return true
	}

	// Try parsing as Go duration
	if _, err := time.ParseDuration(duration); err == nil {
		return true
	}

	// Try parsing as custom format (e.g., "30d", "1y")
	customDurationRegex := regexp.MustCompile(`^(\d+)(s|m|h|d|w|M|y)$`)
	if customDurationRegex.MatchString(duration) {
		matches := customDurationRegex.FindStringSubmatch(duration)
		if len(matches) == 3 {
			if _, err := strconv.Atoi(matches[1]); err == nil {
				return true
			}
		}
	}

	return false
}
