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

package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/linuxsuren/api-testing/pkg/server"
	"github.com/linuxsuren/atest-ext-ai/pkg/ai"
	"github.com/linuxsuren/atest-ext-ai/pkg/ai/models"
	"github.com/linuxsuren/atest-ext-ai/pkg/ai/providers/universal"
	"github.com/linuxsuren/atest-ext-ai/pkg/config"
	apperrors "github.com/linuxsuren/atest-ext-ai/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// handleGetProviders returns the list of available AI providers.
func (s *AIPluginService) handleGetProviders(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	logger := loggerFromContext(ctx)
	logger.Debug("Getting AI providers list")

	if err := contextError(ctx); err != nil {
		return nil, err
	}

	includeUnavailable := true
	if req != nil && req.Sql != "" {
		var params map[string]bool
		if err := json.Unmarshal([]byte(req.Sql), &params); err == nil {
			if includeUnavailableParam, ok := params["include_unavailable"]; ok {
				includeUnavailable = includeUnavailableParam
			}
		} else {
			logger.Warn("Failed to parse provider query overrides", "error", err)
		}
	}

	providers, err := s.aiManager.DiscoverProviders(ctx)
	if err != nil {
		logger.Error("Failed to discover providers", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to discover providers: %v", err)
	}

	if !includeUnavailable {
		filtered := providers[:0]
		for _, provider := range providers {
			if provider.Available {
				filtered = append(filtered, provider)
			}
		}
		providers = filtered
	}

	providersJSON, err := json.Marshal(providers)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to serialize providers: %v", err)
	}

	return &server.DataQueryResult{
		Data: []*server.Pair{
			{Key: "providers", Value: string(providersJSON)},
			{Key: "count", Value: fmt.Sprintf("%d", len(providers))},
			{Key: "success", Value: "true"},
		},
	}, nil
}

// handleGetModels returns models for a specific provider.
func (s *AIPluginService) handleGetModels(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	var params struct {
		Provider string `json:"provider"`
	}

	if req.Sql != "" {
		if err := json.Unmarshal([]byte(req.Sql), &params); err != nil {
			return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidRequest, "invalid parameters: %v", err)
		}
	}

	if params.Provider == "" {
		allModels := make(map[string][]interface{})

		clients := s.aiManager.GetAllClients()
		for providerName := range clients {
			if models, err := s.aiManager.GetModels(ctx, providerName); err == nil {
				modelList := make([]interface{}, len(models))
				for i, m := range models {
					modelList[i] = m
				}
				allModels[providerName] = modelList
			}
		}

		modelsJSON, _ := json.Marshal(allModels)
		return &server.DataQueryResult{
			Data: []*server.Pair{
				{Key: "models", Value: string(modelsJSON)},
				{Key: "success", Value: "true"},
			},
		}, nil
	}

	logger := loggerFromContext(ctx)
	providerName := params.Provider
	switch params.Provider {
	case "local":
		providerName = "ollama"
	case "online":
		providerName = "deepseek"
	}

	models, err := s.aiManager.GetModels(ctx, providerName)
	if err != nil {
		logger.Error("Failed to get models", "provider", providerName, "error", err)
		return nil, apperrors.ToGRPCErrorf(apperrors.ErrModelNotFound, "failed to get models for provider %s: %v", providerName, err)
	}

	modelsJSON, _ := json.Marshal(models)
	return &server.DataQueryResult{
		Data: []*server.Pair{
			{Key: "models", Value: string(modelsJSON)},
			{Key: "provider", Value: params.Provider},
			{Key: "count", Value: fmt.Sprintf("%d", len(models))},
			{Key: "success", Value: "true"},
		},
	}, nil
}

// handleGetModelCatalog returns the static model catalog (entire or provider-specific slice).
func (s *AIPluginService) handleGetModelCatalog(_ context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	var params struct {
		Provider string `json:"provider"`
	}

	if req.Sql != "" {
		if err := json.Unmarshal([]byte(req.Sql), &params); err != nil {
			return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidRequest, "invalid parameters: %v", err)
		}
	}

	catalogSnapshot := models.CatalogSnapshot(params.Provider)
	if len(catalogSnapshot) == 0 {
		return nil, apperrors.ToGRPCErrorf(apperrors.ErrModelNotFound, "no catalog entries found for provider %s", params.Provider)
	}

	snapshotJSON, err := json.Marshal(catalogSnapshot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal catalog: %v", err)
	}

	result := &server.DataQueryResult{
		Data: []*server.Pair{
			{Key: "catalog", Value: string(snapshotJSON)},
			{Key: "success", Value: "true"},
		},
	}

	if params.Provider != "" {
		result.Data = append(result.Data, &server.Pair{Key: "provider", Value: params.Provider})
	}

	return result, nil
}

// handleTestConnection tests a connection with provided configuration.
func (s *AIPluginService) handleTestConnection(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	logger := loggerFromContext(ctx)
	logger.Debug("Handling test connection request", "sql_length", len(req.Sql))

	var config universal.Config
	if req.Sql != "" {
		var payload map[string]any
		if err := json.Unmarshal([]byte(req.Sql), &payload); err != nil {
			logger.Error("Failed to parse connection config", "error", err)
			return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidConfig, "invalid configuration: %v", err)
		}

		normalizeDurationField(payload, "timeout")

		normalizedPayload, err := json.Marshal(payload)
		if err != nil {
			logger.Error("Failed to normalize connection config", "error", err)
			return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidConfig, "invalid configuration: %v", err)
		}

		if err := json.Unmarshal(normalizedPayload, &config); err != nil {
			logger.Error("Failed to decode normalized connection config", "error", err)
			return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidConfig, "invalid configuration: %v", err)
		}
	}

	if config.Provider == "local" {
		config.Provider = "ollama"
	}

	if config.APIKey == "" {
		if apiKey := apiKeyFromContext(ctx); apiKey != "" {
			config.APIKey = apiKey
		}
	}

	apiKeyDisplay := "***masked***"
	if config.APIKey != "" && len(config.APIKey) > 4 {
		apiKeyDisplay = config.APIKey[:4] + "***"
	}
	logger.Debug("Testing connection",
		"provider", config.Provider,
		"api_key_prefix", apiKeyDisplay,
		"model", config.Model)

	result, err := s.aiManager.TestConnection(ctx, &config)
	if err != nil {
		logger.Error("Connection test failed",
			"provider", config.Provider,
			"error", err)
		return nil, apperrors.ToGRPCErrorf(apperrors.ErrConnectionFailed, "connection test failed for provider %s: %v", config.Provider, err)
	}

	resultJSON, _ := json.Marshal(result)
	return &server.DataQueryResult{
		Data: []*server.Pair{
			{Key: "result", Value: string(resultJSON)},
			{Key: "success", Value: fmt.Sprintf("%t", result.Success)},
			{Key: "message", Value: result.Message},
			{Key: "response_time_ms", Value: fmt.Sprintf("%d", result.ResponseTime.Milliseconds())},
		},
	}, nil
}

// handleUpdateConfig updates provider configuration and reloads AI components.
func (s *AIPluginService) handleUpdateConfig(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	logger := loggerFromContext(ctx)
	logger.Debug("Handling update config request", "sql_length", len(req.Sql))

	if err := contextError(ctx); err != nil {
		return nil, err
	}

	var updateReq struct {
		Provider string            `json:"provider"`
		Config   *universal.Config `json:"config"`
	}

	if req.Sql != "" {
		var payload map[string]any
		if err := json.Unmarshal([]byte(req.Sql), &payload); err != nil {
			logger.Error("Failed to parse update request", "error", err)
			return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidRequest, "invalid update request: %v", err)
		}

		if configPayload, ok := payload["config"].(map[string]any); ok {
			normalizeDurationField(configPayload, "timeout")
			payload["config"] = configPayload
		}

		normalizedPayload, err := json.Marshal(payload)
		if err != nil {
			logger.Error("Failed to normalize update config payload", "error", err)
			return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidRequest, "invalid update request: %v", err)
		}

		if err := json.Unmarshal(normalizedPayload, &updateReq); err != nil {
			logger.Error("Failed to decode normalized update request", "error", err)
			return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidRequest, "invalid update request: %v", err)
		}
	}

	if updateReq.Provider == "" || updateReq.Config == nil {
		return nil, apperrors.ToGRPCError(apperrors.ErrInvalidRequest)
	}

	if updateReq.Config.APIKey == "" {
		if apiKey := apiKeyFromContext(ctx); apiKey != "" {
			updateReq.Config.APIKey = apiKey
		}
	}

	if updateReq.Provider == "local" {
		updateReq.Provider = "ollama"
	}
	if updateReq.Config.Provider == "local" {
		updateReq.Config.Provider = "ollama"
	}

	logger.Debug("Updating provider config", "provider", updateReq.Provider)

	serviceConfig := config.AIService{
		Enabled:   true,
		Provider:  updateReq.Config.Provider,
		Endpoint:  updateReq.Config.Endpoint,
		Model:     updateReq.Config.Model,
		APIKey:    updateReq.Config.APIKey,
		MaxTokens: updateReq.Config.MaxTokens,
	}
	if updateReq.Config.Timeout > 0 {
		serviceConfig.Timeout = config.Duration{Duration: updateReq.Config.Timeout}
	}

	oldEngine := s.aiEngine

	servicesCopy := make(map[string]config.AIService, len(s.config.AI.Services)+1)
	for name, svc := range s.config.AI.Services {
		servicesCopy[name] = svc
	}
	servicesCopy[updateReq.Provider] = serviceConfig

	newAIConfig := s.config.AI
	newAIConfig.Services = servicesCopy
	if newAIConfig.DefaultService == "" {
		newAIConfig.DefaultService = updateReq.Provider
	}

	manager, err := ai.NewAIManager(newAIConfig)
	if err != nil {
		logger.Error("Failed to rebuild AI manager",
			"provider", updateReq.Provider,
			"error", err)
		return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidConfig, "failed to rebuild AI manager: %v", err)
	}

	engine, err := ai.NewEngineWithManager(manager, newAIConfig)
	if err != nil {
		logger.Error("Failed to rebuild AI engine",
			"provider", updateReq.Provider,
			"error", err)
		if closeErr := manager.Close(); closeErr != nil {
			logger.Warn("Failed to close AI manager after rebuild error",
				"provider", updateReq.Provider,
				"error", closeErr)
		}
		return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidConfig, "failed to rebuild AI engine: %v", err)
	}

	capabilityDetector := ai.NewCapabilityDetector(newAIConfig, manager)

	s.config.AI = newAIConfig
	s.aiManager = manager
	s.aiEngine = engine
	s.capabilityDetector = capabilityDetector
	clearInitErrorsFor("AI Engine", "AI Manager")

	if oldEngine != nil {
		oldEngine.Close()
	}

	return &server.DataQueryResult{
		Data: []*server.Pair{
			{Key: "provider", Value: updateReq.Provider},
			{Key: "message", Value: "Configuration updated successfully"},
			{Key: "success", Value: "true"},
		},
	}, nil
}
