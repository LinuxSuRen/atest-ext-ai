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
	log.Printf("Received query request: type=%s, key=%s, sql_length=%d", req.Type, req.Key, len(req.Sql))

	if req.Type != "ai" {
		log.Printf("Unsupported query type: %s", req.Type)
		return nil, status.Errorf(codes.InvalidArgument, "unsupported query type: %s", req.Type)
	}

	// For AI queries, we use the 'key' field as the natural language input
	// and 'sql' field for any additional context or existing SQL
	if req.Key == "" {
		log.Printf("Missing key field (natural language query) in request")
		return nil, status.Errorf(codes.InvalidArgument, "key field is required for AI queries (natural language input)")
	}

	// Generate SQL using AI engine
	queryPreview := req.Key
	if len(queryPreview) > 100 {
		queryPreview = queryPreview[:100] + "..."
	}
	log.Printf("Generating SQL for natural language query: %s", queryPreview)

	// Create context map from available information
	contextMap := make(map[string]string)
	if req.Sql != "" {
		contextMap["existing_sql"] = req.Sql
	}

	sqlResult, err := s.aiEngine.GenerateSQL(ctx, &ai.GenerateSQLRequest{
		NaturalLanguage: req.Key,
		DatabaseType:    "mysql", // Default database type
		Context:         contextMap,
	})
	if err != nil {
		log.Printf("Failed to generate SQL: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to generate SQL: %v", err)
	}

	// Create response with basic data structure
	result := &server.DataQueryResult{
		Data: []*server.Pair{
			{
				Key:   "generated_sql",
				Value: sqlResult.SQL,
			},
			{
				Key:   "explanation",
				Value: sqlResult.Explanation,
			},
			{
				Key:   "confidence_score",
				Value: fmt.Sprintf("%.2f", sqlResult.ConfidenceScore),
			},
			{
				Key:   "request_id",
				Value: sqlResult.RequestID,
			},
			{
				Key:   "processing_time_ms",
				Value: fmt.Sprintf("%d", sqlResult.ProcessingTime.Milliseconds()),
			},
			{
				Key:   "model_used",
				Value: sqlResult.ModelUsed,
			},
		},
	}

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