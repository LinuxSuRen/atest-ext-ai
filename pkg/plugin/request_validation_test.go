package plugin

import "testing"

func TestValidatePromptLimit(t *testing.T) {
	validator := InputValidator{MaxPromptChars: 5}

	if err := validator.ValidatePrompt("short"); err != nil {
		t.Fatalf("expected short prompt to pass, got %v", err)
	}
	if err := validator.ValidatePrompt("toolong"); err == nil {
		t.Fatalf("expected prompt length validation error")
	}
}

func TestValidateDatabaseTypeLimit(t *testing.T) {
	validator := InputValidator{MaxDatabaseType: 4}

	if err := validator.ValidateDatabaseType("pg"); err != nil {
		t.Fatalf("expected short type to pass: %v", err)
	}
	if err := validator.ValidateDatabaseType("postgresql"); err == nil {
		t.Fatalf("expected database type length error")
	}
}

func TestValidateContextSize(t *testing.T) {
	validator := InputValidator{MaxContextBytes: 10}

	err := validator.ValidateContext(map[string]string{
		"k1": "abc",
	})
	if err != nil {
		t.Fatalf("expected context under limit to pass: %v", err)
	}

	err = validator.ValidateContext(map[string]string{
		"key": "0123456789",
	})
	if err == nil {
		t.Fatalf("expected context size error")
	}
}

func TestValidateEndpoint(t *testing.T) {
	validator := DefaultInputValidator()

	if err := validator.ValidateEndpoint("https://api.example.com/v1"); err != nil {
		t.Fatalf("expected valid endpoint to pass: %v", err)
	}
	if err := validator.ValidateEndpoint("://invalid"); err == nil {
		t.Fatalf("expected invalid endpoint to fail")
	}
}
