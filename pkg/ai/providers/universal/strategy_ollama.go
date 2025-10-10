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
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
)

// OllamaStrategy implements ProviderStrategy for Ollama
type OllamaStrategy struct{}

// BuildRequest builds an Ollama-specific request
func (s *OllamaStrategy) BuildRequest(req *interfaces.GenerateRequest, config *Config) (any, error) {
	model := req.Model
	if model == "" {
		model = config.Model
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = config.MaxTokens
	}

	// Build messages for chat format
	messages := []map[string]string{}

	if req.SystemPrompt != "" {
		messages = append(messages, map[string]string{
			"role":    "system",
			"content": req.SystemPrompt,
		})
	}

	// Add context as previous messages
	for _, ctx := range req.Context {
		messages = append(messages, map[string]string{
			"role":    "assistant",
			"content": ctx,
		})
	}

	// Add the main prompt
	messages = append(messages, map[string]string{
		"role":    "user",
		"content": req.Prompt,
	})

	return map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   req.Stream,
		"options": map[string]any{
			"num_predict": maxTokens,
		},
	}, nil
}

// ParseResponse parses an Ollama API response
func (s *OllamaStrategy) ParseResponse(body io.Reader, requestedModel string) (*interfaces.GenerateResponse, error) {
	var resp struct {
		Model   string `json:"model"`
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Done               bool  `json:"done"`
		TotalDuration      int64 `json:"total_duration"`
		LoadDuration       int64 `json:"load_duration"`
		PromptEvalCount    int   `json:"prompt_eval_count"`
		PromptEvalDuration int64 `json:"prompt_eval_duration"`
		EvalCount          int   `json:"eval_count"`
		EvalDuration       int64 `json:"eval_duration"`
	}

	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, err
	}

	return &interfaces.GenerateResponse{
		Text:      resp.Message.Content,
		Model:     resp.Model,
		RequestID: fmt.Sprintf("ollama-%d", time.Now().Unix()),
		Metadata: map[string]any{
			"total_duration":   resp.TotalDuration,
			"load_duration":    resp.LoadDuration,
			"prompt_eval_time": resp.PromptEvalDuration,
			"eval_time":        resp.EvalDuration,
			// Token usage information available in metadata if needed
			"prompt_eval_count": resp.PromptEvalCount,
			"eval_count":        resp.EvalCount,
		},
	}, nil
}

// ParseModels parses Ollama's model list response
func (s *OllamaStrategy) ParseModels(body io.Reader, maxTokens int) ([]interfaces.ModelInfo, error) {
	var resp struct {
		Models []struct {
			Name       string `json:"name"`
			ModifiedAt string `json:"modified_at"`
			Size       int64  `json:"size"`
			Details    struct {
				ParameterSize string `json:"parameter_size"`
			} `json:"details"`
		} `json:"models"`
	}

	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, err
	}

	models := make([]interfaces.ModelInfo, 0, len(resp.Models))
	for _, m := range resp.Models {
		models = append(models, interfaces.ModelInfo{
			ID:          m.Name,
			Name:        m.Name,
			Description: fmt.Sprintf("Ollama model (size: %.2f GB)", float64(m.Size)/(1024*1024*1024)),
			MaxTokens:   maxTokens,
		})
	}

	return models, nil
}

// GetDefaultPaths returns default API paths for Ollama
func (s *OllamaStrategy) GetDefaultPaths() ProviderPaths {
	return ProviderPaths{
		CompletionPath: "/api/chat",
		ModelsPath:     "/api/tags",
		HealthPath:     "/api/tags",
	}
}

// GetDefaultModels returns default models when API call fails
func (s *OllamaStrategy) GetDefaultModels(maxTokens int) []interfaces.ModelInfo {
	// Ollama doesn't have predefined models - they're pulled on demand
	// Return empty list and let the system discover via API
	return []interfaces.ModelInfo{}
}

// SupportsStreaming indicates if Ollama supports streaming
func (s *OllamaStrategy) SupportsStreaming() bool {
	return true
}
