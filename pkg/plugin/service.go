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
	"log"
	"strings"

	"github.com/linuxsuren/api-testing/pkg/server"
	"github.com/linuxsuren/api-testing/pkg/testing/remote"
	"github.com/linuxsuren/atest-ext-ai/pkg/ai"
	"github.com/linuxsuren/atest-ext-ai/pkg/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AIPluginService implements the Loader gRPC service for AI functionality
type AIPluginService struct {
	remote.UnimplementedLoaderServer
	aiEngine           ai.Engine
	config             *config.Config
	capabilityDetector *ai.CapabilityDetector
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

	// Create AI client for capability detection
	var aiClient *ai.Client
	aiClient, err = ai.NewClient(cfg.AI)
	if err != nil {
		log.Printf("Warning: Failed to create AI client for capabilities: %v", err)
		// Continue without AI client - capability detector will work with limited functionality
	}

	// Initialize capability detector
	capabilityDetector := ai.NewCapabilityDetector(cfg.AI, aiClient)
	log.Printf("Capability detector initialized")

	service := &AIPluginService{
		aiEngine:           aiEngine,
		config:             cfg,
		capabilityDetector: capabilityDetector,
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

	// Handle new AI interface standard
	switch req.Key {
	case "generate":
		return s.handleAIGenerate(ctx, req)
	case "capabilities":
		return s.handleAICapabilities(ctx, req)
	default:
		// Backward compatibility: support legacy natural language queries
		return s.handleLegacyQuery(ctx, req)
	}
}

// handleCapabilitiesQuery handles requests for AI plugin capabilities
func (s *AIPluginService) handleCapabilitiesQuery(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	log.Printf("Handling capabilities query: key=%s", req.Key)

	// Parse capability request parameters from SQL field (if provided)
	capReq := &ai.CapabilitiesRequest{
		IncludeModels:    true,
		IncludeDatabases: true,
		IncludeFeatures:  true,
		CheckHealth:      false, // Default to false for performance
	}

	// Parse parameters from SQL field if provided
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
			log.Printf("Failed to parse capability request parameters: %v", err)
		}
	}

	// Handle specific capability subqueries
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

	// Get capabilities
	if s.capabilityDetector == nil {
		return nil, status.Errorf(codes.Internal, "capability detector not initialized")
	}

	capabilities, err := s.capabilityDetector.GetCapabilities(ctx, capReq)
	if err != nil {
		log.Printf("Failed to get capabilities: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get capabilities: %v", err)
	}

	// Convert capabilities to JSON
	capabilitiesJSON, err := json.Marshal(capabilities)
	if err != nil {
		log.Printf("Failed to marshal capabilities: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to serialize capabilities: %v", err)
	}

	// Create response
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

	log.Printf("Capabilities query completed successfully: models=%d, databases=%d, features=%d",
		len(capabilities.Models), len(capabilities.Databases), len(capabilities.Features))

	return result, nil
}

// Verify returns the plugin status for health checks
func (s *AIPluginService) Verify(ctx context.Context, req *server.Empty) (*server.ExtensionStatus, error) {
	log.Printf("Health check requested")

	var engineHealthy bool
	if s.aiEngine != nil {
		engineHealthy = s.aiEngine.IsHealthy()
	}

	status := &server.ExtensionStatus{
		Ready:    engineHealthy,
		ReadOnly: false,
		Version:  "1.0.0",
		Message:  "AI Plugin ready",
	}

	if !status.Ready {
		if s.aiEngine == nil {
			status.Message = "AI engine not initialized"
		} else {
			status.Message = "AI engine not available"
		}
		log.Printf("Health check failed: %s", status.Message)
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

// handleAIGenerate handles ai.generate calls
func (s *AIPluginService) handleAIGenerate(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	// Parse parameters from SQL field
	var params struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
		Config string `json:"config"`
	}

	if req.Sql != "" {
		if err := json.Unmarshal([]byte(req.Sql), &params); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid AI parameters: %v", err)
		}
	}

	if params.Prompt == "" {
		return nil, status.Errorf(codes.InvalidArgument, "prompt is required for ai.generate")
	}

	// Parse optional config
	var configMap map[string]interface{}
	if params.Config != "" {
		json.Unmarshal([]byte(params.Config), &configMap)
	}

	log.Printf("Generating SQL with AI interface standard: model=%s, prompt_length=%d", params.Model, len(params.Prompt))

	// Generate using AI engine
	context := map[string]string{}
	if params.Model != "" {
		context["preferred_model"] = params.Model
	}
	if params.Config != "" {
		context["config"] = params.Config
	}

	sqlResult, err := s.aiEngine.GenerateSQL(ctx, &ai.GenerateSQLRequest{
		NaturalLanguage: params.Prompt,
		DatabaseType:    "mysql", // TODO: Make configurable
		Context:         context,
	})
	if err != nil {
		return &server.DataQueryResult{
			Data: []*server.Pair{
				{Key: "error", Value: err.Error()},
				{Key: "success", Value: "false"},
			},
		}, nil
	}

	// Return in AI interface standard format
	return &server.DataQueryResult{
		Data: []*server.Pair{
			{Key: "content", Value: sqlResult.SQL},
			{Key: "success", Value: "true"},
			{Key: "meta", Value: fmt.Sprintf(`{"confidence": %f, "model": "%s"}`,
				sqlResult.ConfidenceScore, sqlResult.ModelUsed)},
		},
	}, nil
}

// handleAICapabilities handles ai.capabilities calls
func (s *AIPluginService) handleAICapabilities(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	capabilities, err := s.capabilityDetector.GetCapabilities(ctx, &ai.CapabilitiesRequest{
		IncludeModels:   true,
		IncludeFeatures: true,
		CheckHealth:     false,
	})
	if err != nil {
		return &server.DataQueryResult{
			Data: []*server.Pair{
				{Key: "error", Value: err.Error()},
				{Key: "success", Value: "false"},
			},
		}, nil
	}

	// Convert to JSON strings for AI interface standard
	capabilitiesJSON, _ := json.Marshal(capabilities)
	modelsJSON, _ := json.Marshal(capabilities.Models)
	featuresJSON, _ := json.Marshal(capabilities.Features)

	return &server.DataQueryResult{
		Data: []*server.Pair{
			{Key: "capabilities", Value: string(capabilitiesJSON)},
			{Key: "models", Value: string(modelsJSON)},
			{Key: "features", Value: string(featuresJSON)},
			{Key: "description", Value: "AI Extension Plugin for intelligent SQL generation"},
			{Key: "version", Value: "1.0.0"},
			{Key: "success", Value: "true"},
		},
	}, nil
}

// handleLegacyQuery maintains backward compatibility with the original implementation
func (s *AIPluginService) handleLegacyQuery(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	// Handle legacy capabilities query
	if req.Key == "capabilities" || strings.HasPrefix(req.Key, "ai.capabilities") {
		return s.handleCapabilitiesQuery(ctx, req)
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
