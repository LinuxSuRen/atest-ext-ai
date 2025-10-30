package models

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
	"github.com/linuxsuren/atest-ext-ai/pkg/logging"
	"gopkg.in/yaml.v3"
)

const (
	// EnvCatalogPath allows overriding the embedded catalog with an external YAML file.
	EnvCatalogPath = "ATEST_EXT_AI_MODEL_CATALOG"

	// defaultCatalogPath is the path checked on disk when no explicit override is provided.
	defaultCatalogPath = "config/models.yaml"
)

//go:embed catalog.yaml
var embeddedCatalogFS embed.FS

var (
	catalogOnce sync.Once
	catalog     *Catalog
	catalogErr  error
)

// Catalog represents the AI model catalog loaded from YAML.
type Catalog struct {
	providers map[string]*Provider
}

// Provider describes a provider entry in the catalog.
type Provider struct {
	Name           string
	DisplayName    string
	Category       string
	Endpoint       string
	RequiresAPIKey bool
	Models         []interfaces.ModelInfo
	Tags           []string
}

type catalogFile struct {
	Providers map[string]catalogProvider `yaml:"providers"`
}

type catalogProvider struct {
	DisplayName    string            `yaml:"display_name"`
	Category       string            `yaml:"category"`
	Endpoint       string            `yaml:"endpoint"`
	RequiresAPIKey bool              `yaml:"requires_api_key"`
	Tags           []string          `yaml:"tags"`
	Models         []catalogModel    `yaml:"models"`
	Metadata       map[string]string `yaml:"metadata"`
}

type catalogModel struct {
	ID             string   `yaml:"id"`
	Name           string   `yaml:"name"`
	Description    string   `yaml:"description"`
	MaxTokens      int      `yaml:"max_tokens"`
	InputCostPerK  float64  `yaml:"input_cost_per_1k"`
	OutputCostPerK float64  `yaml:"output_cost_per_1k"`
	Capabilities   []string `yaml:"capabilities"`
	Tags           []string `yaml:"tags"`
}

// GetCatalog returns the singleton catalog instance.
func GetCatalog() (*Catalog, error) {
	catalogOnce.Do(func() {
		catalog, catalogErr = loadCatalog()
	})
	return catalog, catalogErr
}

// ReloadCatalog forces the catalog to be reloaded. Primarily used in tests.
func ReloadCatalog() (*Catalog, error) {
	catalogOnce = sync.Once{}
	catalog = nil
	catalogErr = nil
	return GetCatalog()
}

// ProviderNames returns the list of provider identifiers in the catalog.
func (c *Catalog) ProviderNames() []string {
	names := make([]string, 0, len(c.providers))
	for name := range c.providers {
		names = append(names, name)
	}
	return names
}

// Provider returns the provider entry for the given name (case-insensitive).
func (c *Catalog) Provider(name string) (*Provider, bool) {
	key := normalizeName(name)
	provider, ok := c.providers[key]
	return provider, ok
}

// ModelsForProvider returns the catalog models for a specific provider or nil if unknown.
func (c *Catalog) ModelsForProvider(name string) []interfaces.ModelInfo {
	if provider, ok := c.Provider(name); ok {
		return provider.Models
	}
	return nil
}

func loadCatalog() (*Catalog, error) {
	data, err := readCatalogSource()
	if err != nil {
		return nil, fmt.Errorf("failed to load model catalog: %w", err)
	}

	var file catalogFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("failed to parse model catalog: %w", err)
	}

	if len(file.Providers) == 0 {
		return nil, errors.New("model catalog is empty")
	}

	providers := make(map[string]*Provider, len(file.Providers))
	for rawName, rawProvider := range file.Providers {
		name := normalizeName(rawName)
		if name == "" {
			continue
		}

		models := make([]interfaces.ModelInfo, 0, len(rawProvider.Models))
		for _, model := range rawProvider.Models {
			if model.ID == "" || model.Name == "" {
				logging.Logger.Warn("Skipping invalid model entry in catalog", "provider", name, "model_id", model.ID)
				continue
			}
			models = append(models, interfaces.ModelInfo{
				ID:              model.ID,
				Name:            model.Name,
				Description:     model.Description,
				MaxTokens:       model.MaxTokens,
				InputCostPer1K:  model.InputCostPerK,
				OutputCostPer1K: model.OutputCostPerK,
				Capabilities:    model.Capabilities,
			})
		}

		if len(models) == 0 {
			logging.Logger.Warn("Provider in model catalog has no valid models", "provider", name)
		}

		providers[name] = &Provider{
			Name:           name,
			DisplayName:    firstNonEmpty(rawProvider.DisplayName, strings.Title(name)),
			Category:       firstNonEmpty(rawProvider.Category, "cloud"),
			Endpoint:       strings.TrimSpace(rawProvider.Endpoint),
			RequiresAPIKey: rawProvider.RequiresAPIKey,
			Models:         models,
			Tags:           rawProvider.Tags,
		}
	}

	return &Catalog{
		providers: providers,
	}, nil
}

func readCatalogSource() ([]byte, error) {
	if envPath := strings.TrimSpace(os.Getenv(EnvCatalogPath)); envPath != "" {
		if data, err := os.ReadFile(envPath); err == nil {
			return data, nil
		} else {
			return nil, fmt.Errorf("failed to read catalog from %s: %w", envPath, err)
		}
	}

	if data, err := os.ReadFile(defaultCatalogPath); err == nil {
		return data, nil
	}

	file, err := embeddedCatalogFS.Open("catalog.yaml")
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(file); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// EndpointForProvider returns the preferred endpoint for a provider, or an empty string if unknown.
func EndpointForProvider(name string) string {
	catalog, err := GetCatalog()
	if err != nil {
		return ""
	}
	if provider, ok := catalog.Provider(name); ok {
		return provider.Endpoint
	}
	return ""
}

// RequiresAPIKey reports whether the catalog marks the provider as requiring an API key.
func RequiresAPIKey(name string) bool {
	catalog, err := GetCatalog()
	if err != nil {
		return true
	}
	if provider, ok := catalog.Provider(name); ok {
		return provider.RequiresAPIKey
	}
	return true
}

// ProviderCatalogEntry encapsulates provider metadata for API responses.
type ProviderCatalogEntry struct {
	DisplayName    string                 `json:"display_name"`
	Category       string                 `json:"category"`
	Endpoint       string                 `json:"endpoint"`
	RequiresAPIKey bool                   `json:"requires_api_key"`
	Models         []interfaces.ModelInfo `json:"models"`
	Tags           []string               `json:"tags,omitempty"`
}

// CatalogSnapshot returns a serializable snapshot of the catalog.
func CatalogSnapshot(provider string) map[string]ProviderCatalogEntry {
	catalog, err := GetCatalog()
	if err != nil {
		return map[string]ProviderCatalogEntry{}
	}

	result := make(map[string]ProviderCatalogEntry)
	if provider != "" {
		if entry, ok := catalog.Provider(provider); ok {
			result[entry.Name] = providerToEntry(entry)
		}
		return result
	}

	for name, entry := range catalog.providers {
		result[name] = providerToEntry(entry)
	}
	return result
}

func providerToEntry(provider *Provider) ProviderCatalogEntry {
	return ProviderCatalogEntry{
		DisplayName:    provider.DisplayName,
		Category:       provider.Category,
		Endpoint:       provider.Endpoint,
		RequiresAPIKey: provider.RequiresAPIKey,
		Models:         provider.Models,
		Tags:           provider.Tags,
	}
}

// CatalogFilePath returns the resolved catalog file path when an external file is used.
func CatalogFilePath() string {
	if envPath := strings.TrimSpace(os.Getenv(EnvCatalogPath)); envPath != "" {
		return envPath
	}
	if _, err := os.Stat(defaultCatalogPath); err == nil {
		return filepath.Clean(defaultCatalogPath)
	}
	return ""
}
