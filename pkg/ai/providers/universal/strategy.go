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

package universal

import (
	"io"

	"github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
)

// ProviderStrategy defines the interface for provider-specific implementations
// This strategy pattern eliminates hardcoded provider checks throughout the codebase
type ProviderStrategy interface {
	// BuildRequest builds provider-specific request body
	BuildRequest(req *interfaces.GenerateRequest, config *Config) (any, error)

	// ParseResponse parses provider-specific response
	ParseResponse(body io.Reader, requestedModel string) (*interfaces.GenerateResponse, error)

	// ParseModels parses provider-specific models list
	ParseModels(body io.Reader, maxTokens int) ([]interfaces.ModelInfo, error)

	// GetDefaultPaths returns default API paths for this provider
	GetDefaultPaths() ProviderPaths

	// GetDefaultModels returns default models when API call fails
	GetDefaultModels(maxTokens int) []interfaces.ModelInfo

	// SupportsStreaming indicates if this provider supports streaming
	SupportsStreaming() bool
}

// ProviderPaths contains provider-specific API paths
type ProviderPaths struct {
	CompletionPath string
	ModelsPath     string
	HealthPath     string
}

// GetStrategy returns the appropriate strategy for a provider
func GetStrategy(provider string) ProviderStrategy {
	switch provider {
	case "ollama":
		return &OllamaStrategy{}
	default:
		// OpenAI-compatible strategy for: openai, deepseek, custom, etc.
		return &OpenAIStrategy{provider: provider}
	}
}
