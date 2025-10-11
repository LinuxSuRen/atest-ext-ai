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
	"context"
	"net/http"
	"time"
)

// OllamaDiscovery handles Ollama service discovery
// It only checks if Ollama is available, not model management.
// Model information should be retrieved through the AIClient interface.
type OllamaDiscovery struct {
	baseURL    string
	httpClient *http.Client
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
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode == http.StatusOK
}

// GetBaseURL returns the configured Ollama base URL
func (od *OllamaDiscovery) GetBaseURL() string {
	return od.baseURL
}
