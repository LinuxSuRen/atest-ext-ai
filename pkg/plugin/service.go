/*
Copyright 2023-2025 API Testing Authors.

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
	"fmt"

	"github.com/linuxsuren/atest-ext-ai/pkg/ai"
	"github.com/linuxsuren/atest-ext-ai/pkg/config"
	"github.com/linuxsuren/api-testing/pkg/server"
	"github.com/linuxsuren/api-testing/pkg/testing/remote"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AIPluginService implements the Loader gRPC service for AI functionality
type AIPluginService struct {
	remote.UnimplementedLoaderServer
	aiEngine ai.Engine
	config   *config.Config
}

// NewAIPluginService creates a new AI plugin service instance
func NewAIPluginService() (*AIPluginService, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	aiEngine, err := ai.NewEngine(cfg.AI)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AI engine: %w", err)
	}

	return &AIPluginService{
		aiEngine: aiEngine,
		config:   cfg,
	}, nil
}

// Query handles AI query requests from the main API testing system
func (s *AIPluginService) Query(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	if req.Type != "ai" {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported query type: %s", req.Type)
	}

	// Validate natural language input
	if req.NaturalLanguage == "" {
		return nil, status.Errorf(codes.InvalidArgument, "natural_language field is required for AI queries")
	}

	// Generate SQL using AI engine
	sqlResult, err := s.aiEngine.GenerateSQL(ctx, &ai.GenerateSQLRequest{
		NaturalLanguage: req.NaturalLanguage,
		DatabaseType:    req.DatabaseType,
		Context:         req.AiContext,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate SQL: %v", err)
	}

	// Create response with AI processing info
	result := &server.DataQueryResult{
		AiInfo: &server.AIProcessingInfo{
			RequestId:        sqlResult.RequestID,
			ProcessingTimeMs: float32(sqlResult.ProcessingTime.Milliseconds()),
			ModelUsed:        sqlResult.ModelUsed,
			ConfidenceScore:  sqlResult.ConfidenceScore,
			DebugInfo:        sqlResult.DebugInfo,
		},
	}

	// Add generated SQL and explanation to result
	result.Data = append(result.Data, &server.Pair{
		Key:   "generated_sql",
		Value: sqlResult.SQL,
	})
	result.Data = append(result.Data, &server.Pair{
		Key:   "explanation",
		Value: sqlResult.Explanation,
	})
	result.Data = append(result.Data, &server.Pair{
		Key:   "confidence_score",
		Value: fmt.Sprintf("%.2f", sqlResult.ConfidenceScore),
	})

	return result, nil
}

// Verify returns the plugin status for health checks
func (s *AIPluginService) Verify(ctx context.Context, req *server.Empty) (*server.ExtensionStatus, error) {
	status := &server.ExtensionStatus{
		Ready:    s.aiEngine.IsHealthy(),
		ReadOnly: false,
		Version:  "1.0.0",
		Message:  "AI Plugin ready",
	}

	if !status.Ready {
		status.Message = "AI engine not available"
	}

	return status, nil
}

// GetAICapabilities returns information about AI plugin capabilities
func (s *AIPluginService) GetAICapabilities(ctx context.Context, req *server.Empty) (*server.AICapabilitiesResponse, error) {
	capabilities := s.aiEngine.GetCapabilities()
	
	response := &server.AICapabilitiesResponse{
		SupportedDatabases: capabilities.SupportedDatabases,
		Version:            "1.0.0",
		Status:             server.HealthStatus_HEALTHY,
		Features:           make([]*server.AIFeature, 0),
	}

	if !s.aiEngine.IsHealthy() {
		response.Status = server.HealthStatus_UNHEALTHY
	}

	// Add feature capabilities
	for _, feature := range capabilities.Features {
		response.Features = append(response.Features, &server.AIFeature{
			Name:        feature.Name,
			Enabled:     feature.Enabled,
			Description: feature.Description,
			Parameters:  feature.Parameters,
		})
	}

	return response, nil
}

// Shutdown gracefully stops the AI plugin service
func (s *AIPluginService) Shutdown() {
	if s.aiEngine != nil {
		s.aiEngine.Close()
	}
}