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

package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
)

// OllamaDiscovery handles Ollama service discovery and model management
type OllamaDiscovery struct {
	baseURL    string
	httpClient *http.Client
}

// OllamaModel represents a model available in Ollama
type OllamaModel struct {
	Name       string    `json:"name"`
	Model      string    `json:"model"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
	Details    struct {
		ParameterSize     string   `json:"parameter_size"`
		QuantizationLevel string   `json:"quantization_level"`
		Families          []string `json:"families"`
		Family            string   `json:"family"`
		Format            string   `json:"format"`
	} `json:"details"`
}

// OllamaListResponse represents the response from /api/tags endpoint
type OllamaListResponse struct {
	Models []OllamaModel `json:"models"`
}

// NewOllamaDiscovery creates a new Ollama discovery instance
func NewOllamaDiscovery(baseURL string) *OllamaDiscovery {
	if baseURL == "" {
		panic("baseURL is required for Ollama discovery - set OLLAMA_ENDPOINT environment variable")
	}

	return &OllamaDiscovery{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// IsAvailable checks if Ollama service is running
func (od *OllamaDiscovery) IsAvailable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", od.baseURL+"/api/tags", nil)
	if err != nil {
		return false
	}

	resp, err := od.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GetModels retrieves the list of available models from Ollama
func (od *OllamaDiscovery) GetModels(ctx context.Context) ([]interfaces.ModelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", od.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := od.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	var listResp OllamaListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to ModelInfo format
	models := make([]interfaces.ModelInfo, 0, len(listResp.Models))
	for _, m := range listResp.Models {
		model := interfaces.ModelInfo{
			ID:          m.Name,
			Name:        m.Model,
			Description: fmt.Sprintf("Ollama model: %s (Size: %.2f GB)", m.Name, float64(m.Size)/(1024*1024*1024)),
			MaxTokens:   4096, // Default for most models
			Capabilities: []string{
				"text-generation",
				"conversation",
			},
		}

		// Add parameter size if available
		if m.Details.ParameterSize != "" {
			model.Description += fmt.Sprintf(", Parameters: %s", m.Details.ParameterSize)
		}

		// Add special capabilities based on model families
		for _, family := range m.Details.Families {
			switch family {
			case "llama":
				model.Capabilities = append(model.Capabilities, "code-generation")
			case "code":
				model.Capabilities = append(model.Capabilities, "code-generation", "code-completion")
			}
		}

		models = append(models, model)
	}

	return models, nil
}

// TestModel tests if a specific model is available and working
func (od *OllamaDiscovery) TestModel(ctx context.Context, modelName string) error {
	// Test with a simple generation request
	reqBody := map[string]interface{}{
		"model":  modelName,
		"prompt": "Hello",
		"stream": false,
		"options": map[string]interface{}{
			"num_predict": 1,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", od.baseURL+"/api/generate", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := od.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to test model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("model test failed with status %d", resp.StatusCode)
	}

	return nil
}

// GetModelInfo retrieves detailed information about a specific model
func (od *OllamaDiscovery) GetModelInfo(ctx context.Context, modelName string) (*interfaces.ModelInfo, error) {
	// First get all models
	models, err := od.GetModels(ctx)
	if err != nil {
		return nil, err
	}

	// Find the specific model
	for _, model := range models {
		if model.ID == modelName || model.Name == modelName {
			return &model, nil
		}
	}

	return nil, fmt.Errorf("model %s not found", modelName)
}

// PullModel pulls a model from the Ollama registry
func (od *OllamaDiscovery) PullModel(ctx context.Context, modelName string) error {
	reqBody := map[string]interface{}{
		"name": modelName,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", od.baseURL+"/api/pull", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := od.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to pull model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to pull model, status %d", resp.StatusCode)
	}

	// Note: This is a streaming endpoint, we're just checking if it starts successfully
	// In production, you might want to handle the streaming response
	return nil
}

// GetBaseURL returns the configured Ollama base URL
func (od *OllamaDiscovery) GetBaseURL() string {
	return od.baseURL
}