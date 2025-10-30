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
	"strings"

	"github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
)

// OpenAIStrategy implements ProviderStrategy for OpenAI-compatible APIs
// This includes: openai, deepseek, custom, and other OpenAI-compatible providers
type OpenAIStrategy struct {
	provider string
}

// BuildRequest builds an OpenAI-compatible request
func (s *OpenAIStrategy) BuildRequest(req *interfaces.GenerateRequest, config *Config) (any, error) {
	model := req.Model
	if model == "" {
		model = config.Model
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = config.MaxTokens
	}

	// Build messages
	messages := []map[string]string{}

	if req.SystemPrompt != "" {
		messages = append(messages, map[string]string{
			"role":    "system",
			"content": req.SystemPrompt,
		})
	}

	// Add context
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

	request := map[string]any{
		"model":      model,
		"messages":   messages,
		"max_tokens": maxTokens,
		"stream":     req.Stream,
	}

	// Add any additional parameters from config
	for k, v := range config.Parameters {
		if _, exists := request[k]; !exists {
			request[k] = v
		}
	}

	return request, nil
}

// ParseResponse parses an OpenAI-compatible API response
func (s *OpenAIStrategy) ParseResponse(body io.Reader, requestedModel string) (*interfaces.GenerateResponse, error) {
	var resp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	if resp.Model == "" && requestedModel != "" {
		resp.Model = requestedModel
	}

	return &interfaces.GenerateResponse{
		Text:      resp.Choices[0].Message.Content,
		Model:     resp.Model,
		RequestID: resp.ID,
		Metadata: map[string]any{
			"finish_reason": resp.Choices[0].FinishReason,
			// Token usage information available in metadata if needed
			"prompt_tokens":     resp.Usage.PromptTokens,
			"completion_tokens": resp.Usage.CompletionTokens,
			"total_tokens":      resp.Usage.TotalTokens,
		},
	}, nil
}

// ParseModels parses OpenAI's model list response
func (s *OpenAIStrategy) ParseModels(body io.Reader, maxTokens int) ([]interfaces.ModelInfo, error) {
	var resp struct {
		Data []struct {
			ID      string `json:"id"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}

	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, err
	}

	models := make([]interfaces.ModelInfo, 0, len(resp.Data))
	for _, m := range resp.Data {
		// Include models that are likely to be chat/completion models
		if isValidChatModel(m.ID) {
			models = append(models, interfaces.ModelInfo{
				ID:          m.ID,
				Name:        m.ID,
				Description: fmt.Sprintf("AI model (owner: %s)", m.OwnedBy),
				MaxTokens:   maxTokens,
			})
		}
	}

	return models, nil
}

// GetDefaultPaths returns default API paths for OpenAI-compatible providers
func (s *OpenAIStrategy) GetDefaultPaths() ProviderPaths {
	return ProviderPaths{
		CompletionPath: "/v1/chat/completions",
		ModelsPath:     "/v1/models",
		HealthPath:     "/v1/models",
	}
}

// GetDefaultModels returns default models for specific providers
func (s *OpenAIStrategy) GetDefaultModels(maxTokens int) []interfaces.ModelInfo {
	switch s.provider {
	case "deepseek":
		return []interfaces.ModelInfo{
			{
				ID:          "deepseek-chat",
				Name:        "DeepSeek Chat",
				Description: "DeepSeek's flagship conversational AI model",
				MaxTokens:   32768,
			},
			{
				ID:          "deepseek-reasoner",
				Name:        "DeepSeek Reasoner",
				Description: "DeepSeek's reasoning model with thinking capabilities",
				MaxTokens:   32768,
			},
		}
	case "openai":
		return []interfaces.ModelInfo{
			{
				ID:          "gpt-5",
				Name:        "GPT-5",
				Description: "OpenAI's flagship GPT-5 model",
				MaxTokens:   200000,
			},
			{
				ID:          "gpt-5-mini",
				Name:        "GPT-5 Mini",
				Description: "Optimized GPT-5 model for latency-sensitive workloads",
				MaxTokens:   80000,
			},
			{
				ID:          "gpt-5-nano",
				Name:        "GPT-5 Nano",
				Description: "Cost efficient GPT-5 variant for lightweight tasks",
				MaxTokens:   40000,
			},
			{
				ID:          "gpt-5-pro",
				Name:        "GPT-5 Pro",
				Description: "High performance GPT-5 model with extended reasoning",
				MaxTokens:   240000,
			},
			{
				ID:          "gpt-4.1",
				Name:        "GPT-4.1",
				Description: "Balanced GPT-4 series model with strong multimodal support",
				MaxTokens:   128000,
			},
		}
	default:
		// Generic fallback
		return []interfaces.ModelInfo{
			{
				ID:          "default",
				Name:        "Default Model",
				Description: "Default model for this provider",
				MaxTokens:   maxTokens,
			},
		}
	}
}

// SupportsStreaming indicates if this provider supports streaming
func (s *OpenAIStrategy) SupportsStreaming() bool {
	return true
}

// isValidChatModel determines if a model ID represents a valid chat/completion model
func isValidChatModel(modelID string) bool {
	modelID = strings.ToLower(modelID)

	// Common patterns for chat/completion models
	chatKeywords := []string{
		"gpt", "chat", "turbo", "instruct",
		"deepseek", "moonshot", "glm", "chatglm",
		"baichuan", "qwen", "claude", "llama",
		"yi", "internlm", "mistral", "gemma",
		"codeqwen", "codechat", "assistant",
		"completion", "text", "dialogue",
	}

	for _, keyword := range chatKeywords {
		if strings.Contains(modelID, keyword) {
			return true
		}
	}

	// Exclude models that are clearly not for chat/completion
	excludeKeywords := []string{
		"embedding", "whisper", "dall-e", "tts",
		"moderation", "edit", "similarity",
		"search", "classification", "fine-tune",
	}

	for _, keyword := range excludeKeywords {
		if strings.Contains(modelID, keyword) {
			return false
		}
	}

	// If no specific patterns match, include by default for compatibility
	return true
}
