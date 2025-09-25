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
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/linuxsuren/api-testing/pkg/server"
	"github.com/linuxsuren/api-testing/pkg/testing/remote"
	"github.com/linuxsuren/atest-ext-ai/pkg/ai"
	"github.com/linuxsuren/atest-ext-ai/pkg/ai/providers/universal"
	"github.com/linuxsuren/atest-ext-ai/pkg/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//go:embed assets/ai-chat.js
var aiChatJS string

//go:embed assets/ai-chat.css
var aiChatCSS string

// AIPluginService implements the Loader gRPC service for AI functionality
type AIPluginService struct {
	remote.UnimplementedLoaderServer
	aiEngine           ai.Engine
	config             *config.Config
	capabilityDetector *ai.CapabilityDetector
	providerManager    *ai.ProviderManager
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

	// Initialize provider manager
	providerManager := ai.NewProviderManager()
	log.Printf("Provider manager initialized")

	// Auto-discover providers in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if providers, err := providerManager.DiscoverProviders(ctx); err == nil {
			log.Printf("Discovered %d AI providers", len(providers))
			for _, p := range providers {
				log.Printf("  - %s: %d models available", p.Name, len(p.Models))
			}
		}
	}()

	service := &AIPluginService{
		aiEngine:           aiEngine,
		config:             cfg,
		capabilityDetector: capabilityDetector,
		providerManager:    providerManager,
	}

	log.Println("AI plugin service creation completed")
	return service, nil
}

// Query handles AI query requests from the main API testing system
func (s *AIPluginService) Query(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	log.Printf("Received query request: type=%s, key=%s, sql_length=%d", req.Type, req.Key, len(req.Sql))

	// Accept both empty type (for backward compatibility) and explicit "ai" type
	// The main project doesn't always send the type field
	if req.Type != "" && req.Type != "ai" {
		log.Printf("Unsupported query type: %s", req.Type)
		return nil, status.Errorf(codes.InvalidArgument, "unsupported query type: %s", req.Type)
	}

	// Handle new AI interface standard
	switch req.Key {
	case "generate":
		return s.handleAIGenerate(ctx, req)
	case "capabilities":
		return s.handleAICapabilities(ctx, req)
	case "providers":
		return s.handleGetProviders(ctx, req)
	case "models":
		return s.handleGetModels(ctx, req)
	case "test_connection":
		return s.handleTestConnection(ctx, req)
	case "update_config":
		return s.handleUpdateConfig(ctx, req)
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
		if err := json.Unmarshal([]byte(params.Config), &configMap); err != nil {
			log.Printf("Warning: failed to parse config JSON: %v", err)
		}
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

// GetMenus returns the menu entries for AI plugin UI
func (s *AIPluginService) GetMenus(ctx context.Context, req *server.Empty) (*server.MenuList, error) {
	log.Printf("🎯🎯🎯 AI PLUGIN GetMenus CALLED 🎯🎯🎯")
	fmt.Printf("🎯🎯🎯 AI PLUGIN GetMenus CALLED 🎯🎯🎯\n")

	return &server.MenuList{
		Data: []*server.Menu{
			{
				Name:    "AI Assistant",
				Index:   "ai-chat",
				Icon:    "ChatDotRound",
				Version: 1,
			},
		},
	}, nil
}

// GetPageOfJS returns the JavaScript code for AI plugin UI
func (s *AIPluginService) GetPageOfJS(ctx context.Context, req *server.SimpleName) (*server.CommonResult, error) {
	log.Printf("🎯🎯🎯 AI PLUGIN GetPageOfJS CALLED FOR: %s 🎯🎯🎯", req.Name)
	fmt.Printf("🎯🎯🎯 AI PLUGIN GetPageOfJS CALLED FOR: %s 🎯🎯🎯\n", req.Name)

	if req.Name != "ai-chat" {
		return &server.CommonResult{
			Success: false,
			Message: fmt.Sprintf("Unknown AI plugin page: %s", req.Name),
		}, nil
	}

	// Use embedded JavaScript file for clean separation of concerns
	jsCode := aiChatJS

	return &server.CommonResult{
		Success: true,
		Message: jsCode,
	}, nil
}

// GetPageOfCSS returns the CSS styles for AI plugin UI
func (s *AIPluginService) GetPageOfCSS(ctx context.Context, req *server.SimpleName) (*server.CommonResult, error) {
	log.Printf("Serving CSS for AI plugin: %s", req.Name)

	if req.Name != "ai-chat" {
		return &server.CommonResult{
			Success: false,
			Message: fmt.Sprintf("Unknown AI plugin page: %s", req.Name),
		}, nil
	}

	// Embedded CSS for AI Chat UI
	cssCode := aiChatCSS

	// Old embedded CSS replaced with external file:
	_ = `
.ai-chat-container {
    height: 100vh;
    display: flex;
    flex-direction: column;
    background: var(--el-bg-color);
    font-family: var(--el-font-family);
}

.ai-chat-header {
    padding: 20px;
    border-bottom: 1px solid var(--el-border-color);
    background: var(--el-bg-color-page);
}

.ai-chat-header h2 {
    margin: 0 0 8px 0;
    color: var(--el-text-color-primary);
    font-size: 20px;
    display: flex;
    align-items: center;
    gap: 8px;
}

.ai-chat-header p {
    margin: 0;
    color: var(--el-text-color-regular);
    font-size: 14px;
}

.ai-chat-messages {
    flex: 1;
    overflow-y: auto;
    padding: 20px;
    background: var(--el-bg-color);
}

.ai-message, .user-message {
    margin-bottom: 16px;
    max-width: 80%;
}

.user-message {
    margin-left: auto;
}

.ai-message-content, .user-message-content {
    padding: 12px 16px;
    border-radius: 8px;
    font-size: 14px;
    line-height: 1.5;
}

.ai-message-content {
    background: var(--el-bg-color-page);
    border: 1px solid var(--el-border-color);
    color: var(--el-text-color-primary);
}

.user-message-content {
    background: var(--el-color-primary);
    color: white;
    text-align: right;
}

.ai-message.loading .ai-message-content {
    background: var(--el-color-info-light-9);
    border-color: var(--el-color-info-light-7);
}

.ai-message.error .ai-message-content {
    background: var(--el-color-danger-light-9);
    border-color: var(--el-color-danger-light-7);
    color: var(--el-color-danger);
}

.sql-result {
    margin-top: 8px;
}

.sql-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
}

.copy-btn {
    background: var(--el-color-primary);
    color: white;
    border: none;
    padding: 4px 8px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 12px;
    display: flex;
    align-items: center;
    gap: 4px;
}

.copy-btn:hover {
    background: var(--el-color-primary-light-3);
}

.sql-code {
    background: var(--el-fill-color-darker);
    padding: 12px;
    border-radius: 4px;
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    font-size: 13px;
    line-height: 1.4;
    overflow-x: auto;
    border: 1px solid var(--el-border-color);
    color: var(--el-text-color-primary);
    white-space: pre-wrap;
    word-break: break-all;
}

.confidence, .model {
    font-size: 12px;
    color: var(--el-text-color-regular);
    margin-top: 4px;
}

.ai-chat-input-area {
    padding: 20px;
    border-top: 1px solid var(--el-border-color);
    background: var(--el-bg-color-page);
}

.ai-input-container {
    display: flex;
    gap: 12px;
    align-items: flex-end;
}

.ai-input-container textarea {
    flex: 1;
    padding: 12px;
    border: 1px solid var(--el-border-color);
    border-radius: 6px;
    font-size: 14px;
    font-family: inherit;
    resize: vertical;
    min-height: 60px;
    background: var(--el-bg-color);
    color: var(--el-text-color-primary);
}

.ai-input-container textarea:focus {
    outline: none;
    border-color: var(--el-color-primary);
    box-shadow: 0 0 0 2px var(--el-color-primary-light-9);
}

.ai-send-button {
    background: var(--el-color-primary);
    color: white;
    border: none;
    padding: 12px 20px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 14px;
    display: flex;
    align-items: center;
    gap: 6px;
    white-space: nowrap;
    min-height: 44px;
}

.ai-send-button:hover:not(:disabled) {
    background: var(--el-color-primary-light-3);
}

.ai-send-button:disabled {
    background: var(--el-color-info);
    cursor: not-allowed;
}

.ai-options {
    margin-top: 12px;
    display: flex;
    align-items: center;
    gap: 16px;
}

.ai-options label {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 14px;
    color: var(--el-text-color-regular);
    cursor: pointer;
}

.ai-options input[type="checkbox"] {
    margin: 0;
}

/* Scrollbar styling */
.ai-chat-messages::-webkit-scrollbar {
    width: 6px;
}

.ai-chat-messages::-webkit-scrollbar-track {
    background: var(--el-fill-color-lighter);
}

.ai-chat-messages::-webkit-scrollbar-thumb {
    background: var(--el-border-color-darker);
    border-radius: 3px;
}

.ai-chat-messages::-webkit-scrollbar-thumb:hover {
    background: var(--el-border-color-extra-light);
}

/* Animation for loading */
@keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
}

.el-icon-loading {
    animation: spin 1s linear infinite;
}

/* Dark mode adjustments */
html.dark .ai-chat-container {
    background: var(--el-bg-color);
}

html.dark .sql-code {
    background: var(--el-fill-color-dark);
    color: var(--el-text-color-primary);
}
`

	return &server.CommonResult{
		Success: true,
		Message: cssCode,
	}, nil
}

// GetPageOfStatic returns static files for AI plugin UI (not implemented)
func (s *AIPluginService) GetPageOfStatic(ctx context.Context, req *server.SimpleName) (*server.CommonResult, error) {
	return &server.CommonResult{
		Success: false,
		Message: "Static files not supported",
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

// handleGetProviders returns the list of available AI providers
func (s *AIPluginService) handleGetProviders(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	log.Printf("Getting AI providers list")

	// Ensure providers are discovered
	providers, err := s.providerManager.DiscoverProviders(ctx)
	if err != nil {
		log.Printf("Failed to discover providers: %v", err)
		// Continue with cached providers
		providers = s.providerManager.GetProviders()
	}

	// Convert to JSON
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

// handleGetModels returns models for a specific provider
func (s *AIPluginService) handleGetModels(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	// Parse provider name from SQL field
	var params struct {
		Provider string `json:"provider"`
	}

	if req.Sql != "" {
		if err := json.Unmarshal([]byte(req.Sql), &params); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid parameters: %v", err)
		}
	}

	if params.Provider == "" {
		// If no provider specified, return all models from all providers
		allModels := make(map[string][]interface{})
		providers := s.providerManager.GetProviders()

		for _, provider := range providers {
			if models, err := s.providerManager.GetModels(ctx, provider.Name); err == nil {
				modelList := make([]interface{}, len(models))
				for i, m := range models {
					modelList[i] = m
				}
				allModels[provider.Name] = modelList
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

	// Get models for specific provider
	models, err := s.providerManager.GetModels(ctx, params.Provider)
	if err != nil {
		return &server.DataQueryResult{
			Data: []*server.Pair{
				{Key: "error", Value: err.Error()},
				{Key: "success", Value: "false"},
			},
		}, nil
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

// handleTestConnection tests a connection with provided configuration
func (s *AIPluginService) handleTestConnection(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	// Parse configuration from SQL field
	var config universal.Config
	if req.Sql != "" {
		if err := json.Unmarshal([]byte(req.Sql), &config); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid configuration: %v", err)
		}
	}

	// Test the connection
	result, err := s.providerManager.TestConnection(ctx, &config)
	if err != nil {
		return &server.DataQueryResult{
			Data: []*server.Pair{
				{Key: "error", Value: err.Error()},
				{Key: "success", Value: "false"},
			},
		}, nil
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

// handleUpdateConfig updates the configuration for a provider
func (s *AIPluginService) handleUpdateConfig(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	// Parse update request from SQL field
	var updateReq struct {
		Provider string              `json:"provider"`
		Config   *universal.Config   `json:"config"`
	}

	if req.Sql != "" {
		if err := json.Unmarshal([]byte(req.Sql), &updateReq); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid update request: %v", err)
		}
	}

	if updateReq.Provider == "" || updateReq.Config == nil {
		return nil, status.Errorf(codes.InvalidArgument, "provider and config are required")
	}

	// Update the configuration
	err := s.providerManager.UpdateConfig(ctx, updateReq.Provider, updateReq.Config)
	if err != nil {
		return &server.DataQueryResult{
			Data: []*server.Pair{
				{Key: "error", Value: err.Error()},
				{Key: "success", Value: "false"},
			},
		}, nil
	}

	return &server.DataQueryResult{
		Data: []*server.Pair{
			{Key: "provider", Value: updateReq.Provider},
			{Key: "message", Value: "Configuration updated successfully"},
			{Key: "success", Value: "true"},
		},
	}, nil
}
