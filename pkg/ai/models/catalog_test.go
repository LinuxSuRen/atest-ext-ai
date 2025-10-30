package models

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetCatalog(t *testing.T) {
	catalog, err := GetCatalog()
	if err != nil {
		t.Fatalf("failed to load catalog: %v", err)
	}

	if len(catalog.ProviderNames()) == 0 {
		t.Fatalf("expected catalog to contain providers")
	}

	openai, ok := catalog.Provider("openai")
	if !ok {
		t.Fatalf("expected openai provider in catalog")
	}
	if len(openai.Models) == 0 {
		t.Fatalf("expected openai to have models")
	}
}

func TestCatalogSnapshot(t *testing.T) {
	snapshot := CatalogSnapshot("")
	if len(snapshot) == 0 {
		t.Fatalf("expected catalog snapshot to have entries")
	}

	entry, ok := snapshot["openai"]
	if !ok {
		t.Fatalf("expected snapshot to contain openai entry")
	}
	if entry.Endpoint == "" {
		t.Fatalf("expected openai endpoint to be populated")
	}
}

func TestEndpointForProvider(t *testing.T) {
	if endpoint := EndpointForProvider("openai"); endpoint == "" {
		t.Fatalf("expected endpoint for openai")
	}
}

func TestReloadWithExternalFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.yaml")

	content := []byte(`
providers:
  test:
    display_name: "Test Provider"
    category: "cloud"
    endpoint: "https://example.com"
    requires_api_key: false
    models:
      - id: "test-model"
        name: "Test Model"
        max_tokens: 1024
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("failed to write temp catalog: %v", err)
	}

	t.Setenv(EnvCatalogPath, path)
	catalog, err := ReloadCatalog()
	if err != nil {
		t.Fatalf("failed to reload catalog: %v", err)
	}

	if _, ok := catalog.Provider("test"); !ok {
		t.Fatalf("expected test provider from external catalog")
	}
}
