package plugin

import (
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"
)

const (
	defaultMaxPromptChars  = 4000
	defaultMaxContextBytes = 16 * 1024
	defaultMaxDatabaseType = 64
)

// InputValidator enforces limits on incoming AI generation requests.
type InputValidator struct {
	MaxPromptChars  int
	MaxContextBytes int
	MaxDatabaseType int
}

// DefaultInputValidator returns an InputValidator with sane defaults.
func DefaultInputValidator() InputValidator {
	return InputValidator{
		MaxPromptChars:  defaultMaxPromptChars,
		MaxContextBytes: defaultMaxContextBytes,
		MaxDatabaseType: defaultMaxDatabaseType,
	}
}

// ValidatePrompt ensures the prompt does not exceed the configured length.
func (v InputValidator) ValidatePrompt(prompt string) error {
	if v.MaxPromptChars <= 0 {
		return nil
	}
	length := utf8.RuneCountInString(prompt)
	if length > v.MaxPromptChars {
		return fmt.Errorf("prompt exceeds maximum length of %d characters", v.MaxPromptChars)
	}
	return nil
}

// ValidateDatabaseType ensures database type identifiers remain within limits.
func (v InputValidator) ValidateDatabaseType(databaseType string) error {
	if databaseType == "" || v.MaxDatabaseType <= 0 {
		return nil
	}
	length := utf8.RuneCountInString(databaseType)
	if length > v.MaxDatabaseType {
		return fmt.Errorf("database type exceeds maximum length of %d characters", v.MaxDatabaseType)
	}
	return nil
}

// ValidateContext ensures the serialized context payload stays within bounds.
func (v InputValidator) ValidateContext(ctx map[string]string) error {
	if ctx == nil || v.MaxContextBytes <= 0 {
		return nil
	}
	total := 0
	for key, value := range ctx {
		total += len(key) + len(value)
		if total > v.MaxContextBytes {
			return fmt.Errorf("context payload exceeds maximum size of %d bytes", v.MaxContextBytes)
		}
	}
	return nil
}

// ValidateEndpoint ensures runtime-supplied endpoints are well-formed URLs.
func (v InputValidator) ValidateEndpoint(endpoint string) error {
	if strings.TrimSpace(endpoint) == "" {
		return nil
	}
	parsed, err := url.ParseRequestURI(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("endpoint must be a valid absolute URL")
	}
	return nil
}
