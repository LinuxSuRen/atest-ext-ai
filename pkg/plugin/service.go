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
	"log"

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
	log.Println("Initializing AI plugin service...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("Failed to load configuration: %v", err)
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	log.Printf("Configuration loaded successfully")

	aiEngine, err := ai.NewEngine(cfg.AI)
	if err != nil {
		log.Printf("Failed to initialize AI engine: %v", err)
		return nil, fmt.Errorf("failed to initialize AI engine: %w", err)
	}
	log.Printf("AI engine initialized successfully")

	service := &AIPluginService{
		aiEngine: aiEngine,
		config:   cfg,
	}

	log.Println("AI plugin service creation completed")
	return service, nil
}

// Query handles AI query requests from the main API testing system
func (s *AIPluginService) Query(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	log.Printf("Received query request: type=%s", req.Type)

	if req.Type != "ai" {
		log.Printf("Unsupported query type: %s", req.Type)
		return nil, status.Errorf(codes.InvalidArgument, "unsupported query type: %s", req.Type)
	}

	// Validate natural language input
	if req.NaturalLanguage == "" {
		log.Printf("Missing natural language field in request")
		return nil, status.Errorf(codes.InvalidArgument, "natural_language field is required for AI queries")
	}

	log.Printf("Generating SQL for natural language query: %s", req.NaturalLanguage)

	// Generate SQL using AI engine
	sqlResult, err := s.aiEngine.GenerateSQL(ctx, &ai.GenerateSQLRequest{
		NaturalLanguage: req.NaturalLanguage,
		DatabaseType:    req.DatabaseType,
		Context:         req.AiContext,
	})
	if err != nil {
		log.Printf("Failed to generate SQL: %v", err)
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

	log.Printf("AI query completed successfully: request_id=%s, confidence=%.2f, processing_time=%dms",
		sqlResult.RequestID, sqlResult.ConfidenceScore, sqlResult.ProcessingTime.Milliseconds())

	return result, nil
}

// Verify returns the plugin status for health checks
func (s *AIPluginService) Verify(ctx context.Context, req *server.Empty) (*server.ExtensionStatus, error) {
	log.Printf("Health check requested")

	engineHealthy := s.aiEngine.IsHealthy()
	status := &server.ExtensionStatus{
		Ready:    engineHealthy,
		ReadOnly: false,
		Version:  "1.0.0",
		Message:  "AI Plugin ready",
	}

	if !status.Ready {
		status.Message = "AI engine not available"
		log.Printf("Health check failed: AI engine not healthy")
	} else {
		log.Printf("Health check passed: AI plugin is ready")
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
	log.Println("Shutting down AI plugin service...")

	if s.aiEngine != nil {
		log.Println("Closing AI engine...")
		s.aiEngine.Close()
		log.Println("AI engine closed successfully")
	}

	log.Println("AI plugin service shutdown complete")
}