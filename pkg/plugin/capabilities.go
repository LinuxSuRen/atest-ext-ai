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
	"strings"

	"github.com/linuxsuren/api-testing/pkg/server"
	"github.com/linuxsuren/atest-ext-ai/pkg/ai"
	apperrors "github.com/linuxsuren/atest-ext-ai/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// handleCapabilitiesQuery handles requests for AI plugin capabilities.
func (s *AIPluginService) handleCapabilitiesQuery(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	logger := loggerFromContext(ctx)
	logger.Info("Handling capabilities query", "key", req.Key)

	capReq := &ai.CapabilitiesRequest{
		IncludeModels:    true,
		IncludeDatabases: true,
		IncludeFeatures:  true,
		CheckHealth:      false,
	}

	if req.Sql != "" {
		var params map[string]bool
		if err := json.Unmarshal([]byte(req.Sql), &params); err == nil {
			if includeModels, ok := params["include_models"]; ok {
				capReq.IncludeModels = includeModels
			}
			if includeDatabases, ok := params["include_databases"]; ok {
				capReq.IncludeDatabases = includeDatabases
			}
			if includeFeatures, ok := params["include_features"]; ok {
				capReq.IncludeFeatures = includeFeatures
			}
			if checkHealth, ok := params["check_health"]; ok {
				capReq.CheckHealth = checkHealth
			}
		} else {
			logger.Error("Failed to parse capability request parameters", "error", err)
		}
	}

	if strings.Contains(req.Key, ".") {
		parts := strings.Split(req.Key, ".")
		if len(parts) >= 2 {
			subQuery := parts[len(parts)-1]
			switch subQuery {
			case "metadata":
				return nil, status.Errorf(codes.Unimplemented, "metadata query not supported")
			case "models":
				capReq.IncludeModels = true
				capReq.IncludeDatabases = false
				capReq.IncludeFeatures = false
			case "databases":
				capReq.IncludeModels = false
				capReq.IncludeDatabases = true
				capReq.IncludeFeatures = false
			case "features":
				capReq.IncludeModels = false
				capReq.IncludeDatabases = false
				capReq.IncludeFeatures = true
			case "health":
				capReq.CheckHealth = true
			}
		}
	}

	if s.capabilityDetector == nil {
		return nil, status.Errorf(codes.Internal, "capability detector not initialized")
	}

	capabilities, err := s.capabilityDetector.GetCapabilities(ctx, capReq)
	if err != nil {
		logger.Error("Failed to get capabilities", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get capabilities: %v", err)
	}

	capabilitiesJSON, err := json.Marshal(capabilities)
	if err != nil {
		logger.Error("Failed to marshal capabilities", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to serialize capabilities: %v", err)
	}

	result := &server.DataQueryResult{
		Data: []*server.Pair{
			{
				Key:   "capabilities",
				Value: string(capabilitiesJSON),
			},
			{
				Key:   "version",
				Value: capabilities.Version,
			},
			{
				Key:   "last_updated",
				Value: capabilities.LastUpdated.Format("2006-01-02T15:04:05Z"),
			},
			{
				Key:   "model_count",
				Value: fmt.Sprintf("%d", len(capabilities.Models)),
			},
			{
				Key:   "database_count",
				Value: fmt.Sprintf("%d", len(capabilities.Databases)),
			},
			{
				Key:   "feature_count",
				Value: fmt.Sprintf("%d", len(capabilities.Features)),
			},
			{
				Key:   "overall_health",
				Value: fmt.Sprintf("%t", capabilities.Health.Overall),
			},
		},
	}

	logger.Info("Capabilities query completed successfully",
		"models", len(capabilities.Models), "databases", len(capabilities.Databases), "features", len(capabilities.Features))

	return result, nil
}

// handleAICapabilities handles ai.capabilities calls.
func (s *AIPluginService) handleAICapabilities(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}

	capReq := &ai.CapabilitiesRequest{
		IncludeModels:   true,
		IncludeFeatures: true,
		CheckHealth:     false,
	}

	if req != nil && req.Sql != "" {
		var params map[string]bool
		if err := json.Unmarshal([]byte(req.Sql), &params); err == nil {
			if includeModels, ok := params["include_models"]; ok {
				capReq.IncludeModels = includeModels
			}
			if includeFeatures, ok := params["include_features"]; ok {
				capReq.IncludeFeatures = includeFeatures
			}
			if checkHealth, ok := params["check_health"]; ok {
				capReq.CheckHealth = checkHealth
			}
		} else {
			logger := loggerFromContext(ctx)
			logger.Warn("Failed to parse capabilities request overrides", "error", err)
		}
	}

	if s.capabilityDetector == nil {
		logger := loggerFromContext(ctx)
		logger.Warn("Capability detector not available - returning minimal capabilities")
		fallback := CapabilitySummary{
			PluginReady:   true,
			AIAvailable:   false,
			DegradedMode:  true,
			PluginVersion: PluginVersion,
			APIVersion:    APIVersion,
		}
		capsJSON, _ := json.Marshal(fallback)
		return &server.DataQueryResult{
			Data: []*server.Pair{
				{Key: "api_version", Value: APIVersion},
				{Key: "capabilities", Value: string(capsJSON)},
				{Key: "models", Value: "[]"},
				{Key: "features", Value: "[]"},
				{Key: "description", Value: "AI Extension Plugin (degraded mode - AI services unavailable)"},
				{Key: "version", Value: PluginVersion},
				{Key: "success", Value: "true"},
				{Key: "warning", Value: "AI services are currently unavailable"},
			},
		}, nil
	}

	capabilities, err := s.capabilityDetector.GetCapabilities(ctx, capReq)
	if err != nil {
		logger := loggerFromContext(ctx)
		logger.Error("Failed to get capabilities", "error", err)
		return nil, apperrors.ToGRPCErrorf(apperrors.ErrProviderNotAvailable, "failed to retrieve capabilities: %v", err)
	}

	capabilitiesJSON, _ := json.Marshal(capabilities)
	modelsJSON, _ := json.Marshal(capabilities.Models)
	featuresJSON, _ := json.Marshal(capabilities.Features)

	return &server.DataQueryResult{
		Data: []*server.Pair{
			{Key: "api_version", Value: APIVersion},
			{Key: "capabilities", Value: string(capabilitiesJSON)},
			{Key: "models", Value: string(modelsJSON)},
			{Key: "features", Value: string(featuresJSON)},
			{Key: "description", Value: "AI Extension Plugin for intelligent SQL generation"},
			{Key: "version", Value: PluginVersion},
			{Key: "success", Value: "true"},
		},
	}, nil
}
