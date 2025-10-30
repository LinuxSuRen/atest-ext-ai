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
	"strings"
	"time"

	"github.com/linuxsuren/api-testing/pkg/server"
	"github.com/linuxsuren/api-testing/pkg/testing/remote"
	"github.com/linuxsuren/atest-ext-ai/pkg/ai"
	"github.com/linuxsuren/atest-ext-ai/pkg/ai/providers/universal"
	"github.com/linuxsuren/atest-ext-ai/pkg/config"
	apperrors "github.com/linuxsuren/atest-ext-ai/pkg/errors"
	"github.com/linuxsuren/atest-ext-ai/pkg/logging"
	"github.com/linuxsuren/atest-ext-ai/pkg/metrics"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//go:embed assets/ai-chat.js
var aiChatJS string

//go:embed assets/ai-chat.css
var aiChatCSS string

// InitializationError captures detailed context about component initialization failures
// This allows us to provide comprehensive diagnostic information in error responses
type InitializationError struct {
	Component string            // Component that failed (e.g., "AI Engine", "AI Manager")
	Reason    string            // Error message explaining the failure
	Details   map[string]string // Additional diagnostic information
}

// Global initialization error tracking for enhanced error messages
// This is populated during service initialization and used to provide detailed context
// when operations fail due to unavailable services
var initErrors []InitializationError

func clearInitErrorsFor(components ...string) {
	if len(initErrors) == 0 || len(components) == 0 {
		return
	}

	componentSet := make(map[string]struct{}, len(components))
	for _, name := range components {
		componentSet[name] = struct{}{}
	}

	filtered := initErrors[:0]
	for _, initErr := range initErrors {
		if _, drop := componentSet[initErr.Component]; drop {
			continue
		}
		filtered = append(filtered, initErr)
	}
	initErrors = filtered
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}

	if err := ctx.Err(); err != nil {
		return status.Error(codes.Canceled, err.Error())
	}

	return nil
}

func normalizeDurationField(payload map[string]any, key string) {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return
	}

	switch value := raw.(type) {
	case string:
		if value == "" {
			return
		}
		duration, err := time.ParseDuration(value)
		if err != nil {
			logging.Logger.Warn("Invalid duration string", "field", key, "value", value, "error", err)
			return
		}
		payload[key] = duration.Nanoseconds()
	case float64:
		if value == 0 {
			return
		}
		if value < float64(time.Second) {
			payload[key] = int64(value * float64(time.Second))
		} else {
			payload[key] = int64(value)
		}
	}
}

// AIPluginService implements the Loader gRPC service for AI functionality
type AIPluginService struct {
	remote.UnimplementedLoaderServer
	aiEngine           ai.Engine
	config             *config.Config
	capabilityDetector *ai.CapabilityDetector
	aiManager          *ai.Manager
}

// NewAIPluginService creates a new AI plugin service instance
// This function implements graceful degradation - the plugin will start successfully
// even if AI services are temporarily unavailable, allowing configuration and UI features to work.
func NewAIPluginService() (*AIPluginService, error) {
	logging.Logger.Info("Initializing AI plugin service...")

	// Log version information for debugging and compatibility verification
	logging.Logger.Info("Plugin version information",
		"plugin_version", PluginVersion,
		"api_version", APIVersion,
		"grpc_interface_version", GRPCInterfaceVersion,
		"min_api_testing_version", MinCompatibleAPITestingVersion)
	logging.Logger.Info("Compatibility note: This plugin requires api-testing >= " + MinCompatibleAPITestingVersion)

	cfg, err := config.LoadConfig()
	if err != nil {
		logging.Logger.Error("Failed to load configuration", "error", err)
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	logging.Logger.Info("Configuration loaded successfully")

	service := &AIPluginService{
		config: cfg,
	}

	// Try to initialize AI engine - but allow plugin to start if it fails
	aiEngine, err := ai.NewEngine(cfg.AI)
	if err != nil {
		logging.Logger.Warn("AI engine initialization failed - plugin will start in degraded mode",
			"error", err,
			"impact", "AI generation features will be unavailable until AI service is available")

		// Collect detailed initialization error for diagnostic messages
		initErr := InitializationError{
			Component: "AI Engine",
			Reason:    err.Error(),
			Details: map[string]string{
				"default_service": cfg.AI.DefaultService,
				"provider_count":  fmt.Sprintf("%d", len(cfg.AI.Services)),
			},
		}
		// Add specific provider details if available
		if cfg.AI.DefaultService != "" {
			if svc, ok := cfg.AI.Services[cfg.AI.DefaultService]; ok {
				initErr.Details["provider_endpoint"] = svc.Endpoint
				initErr.Details["provider_model"] = svc.Model
			}
		}
		initErrors = append(initErrors, initErr)
		service.aiEngine = nil
	} else {
		logging.Logger.Info("AI engine initialized successfully")
		service.aiEngine = aiEngine
	}

	// Try to initialize unified AI manager - but allow plugin to start if it fails
	aiManager, err := ai.NewAIManager(cfg.AI)
	if err != nil {
		logging.Logger.Warn("AI manager initialization failed - plugin will start in degraded mode",
			"error", err,
			"impact", "Provider discovery and model listing will be unavailable")

		// Collect detailed initialization error for diagnostic messages
		initErr := InitializationError{
			Component: "AI Manager",
			Reason:    err.Error(),
			Details: map[string]string{
				"default_service":  cfg.AI.DefaultService,
				"configured_count": fmt.Sprintf("%d", len(cfg.AI.Services)),
			},
		}
		// Add list of configured services
		if len(cfg.AI.Services) > 0 {
			var services []string
			for name := range cfg.AI.Services {
				services = append(services, name)
			}
			initErr.Details["configured_services"] = strings.Join(services, ", ")
		}
		initErrors = append(initErrors, initErr)
		service.aiManager = nil
	} else {
		logging.Logger.Info("AI manager initialized successfully")
		service.aiManager = aiManager

		// Initialize capability detector only if AI manager is available
		capabilityDetector := ai.NewCapabilityDetector(cfg.AI, aiManager)
		logging.Logger.Info("Capability detector initialized")
		service.capabilityDetector = capabilityDetector

		// Auto-discover providers in background only if manager is available
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if providers, err := aiManager.DiscoverProviders(ctx); err == nil {
				logging.Logger.Info("Discovered AI providers", "count", len(providers))
				for _, p := range providers {
					logging.Logger.Debug("Provider models available", "provider", p.Name, "model_count", len(p.Models))
				}
			} else {
				logging.Logger.Warn("Provider discovery failed", "error", err)
			}
		}()
	}

	// Log final status
	if service.aiEngine != nil && service.aiManager != nil {
		logging.Logger.Info("AI plugin service fully operational")
	} else {
		logging.Logger.Warn("AI plugin service started in degraded mode - some features unavailable",
			"ai_engine_available", service.aiEngine != nil,
			"ai_manager_available", service.aiManager != nil)
	}

	return service, nil
}

// Query handles AI query requests from the main API testing system
func (s *AIPluginService) Query(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	logging.Logger.Debug("Query received",
		"type", req.Type,
		"key", req.Key,
		"sql_length", len(req.Sql))

	// Accept both empty type (for backward compatibility) and explicit "ai" type
	// The main project doesn't always send the type field
	if req.Type != "" && req.Type != "ai" {
		logging.Logger.Warn("Unsupported query type", "type", req.Type)
		return nil, status.Errorf(codes.InvalidArgument, "unsupported query type: %s", req.Type)
	}

	// Handle new AI interface standard
	switch req.Key {
	case "generate":
		// Check AI engine availability for generation requests
		if s.aiEngine == nil {
			logging.Logger.Error("AI generation requested but AI engine is not available")

			// Build enhanced error message with initialization details
			errMsg := "AI generation service is currently unavailable."

			// Add specific initialization error information if available
			if len(initErrors) > 0 {
				errMsg += " Initialization errors:"
				for _, initErr := range initErrors {
					errMsg += fmt.Sprintf("\n- %s: %s", initErr.Component, initErr.Reason)
					// Add relevant details
					if len(initErr.Details) > 0 {
						for key, value := range initErr.Details {
							errMsg += fmt.Sprintf("\n  %s: %s", key, value)
						}
					}
				}
			} else {
				errMsg += " Please check AI provider configuration and connectivity."
			}

			return nil, status.Errorf(codes.FailedPrecondition, errMsg)
		}
		return s.handleAIGenerate(ctx, req)
	case "capabilities":
		return s.handleAICapabilities(ctx, req)
	case "providers":
		// Check AI manager availability for provider operations
		if s.aiManager == nil {
			logging.Logger.Error("Provider discovery requested but AI manager is not available")

			// Build enhanced error message with initialization details
			errMsg := "AI provider discovery is currently unavailable."
			if len(initErrors) > 0 {
				errMsg += " Initialization errors:"
				for _, initErr := range initErrors {
					if initErr.Component == "AI Manager" {
						errMsg += fmt.Sprintf("\n- %s: %s", initErr.Component, initErr.Reason)
						if len(initErr.Details) > 0 {
							for key, value := range initErr.Details {
								errMsg += fmt.Sprintf("\n  %s: %s", key, value)
							}
						}
					}
				}
			} else {
				errMsg += " Please check AI service configuration."
			}

			return nil, status.Errorf(codes.FailedPrecondition, errMsg)
		}
		return s.handleGetProviders(ctx, req)
	case "models":
		// Check AI manager availability for model operations
		if s.aiManager == nil {
			logging.Logger.Error("Model listing requested but AI manager is not available")

			// Build enhanced error message with initialization details
			errMsg := "AI model listing is currently unavailable."
			if len(initErrors) > 0 {
				errMsg += " Initialization errors:"
				for _, initErr := range initErrors {
					if initErr.Component == "AI Manager" {
						errMsg += fmt.Sprintf("\n- %s: %s", initErr.Component, initErr.Reason)
						if len(initErr.Details) > 0 {
							for key, value := range initErr.Details {
								errMsg += fmt.Sprintf("\n  %s: %s", key, value)
							}
						}
					}
				}
			} else {
				errMsg += " Please check AI service configuration."
			}

			return nil, status.Errorf(codes.FailedPrecondition, errMsg)
		}
		return s.handleGetModels(ctx, req)
	case "test_connection":
		// Connection testing can work even without initialized services
		if s.aiManager == nil {
			logging.Logger.Error("Connection test requested but AI manager is not available")

			// Build enhanced error message with initialization details
			errMsg := "AI connection testing is currently unavailable."
			if len(initErrors) > 0 {
				errMsg += " Initialization errors:"
				for _, initErr := range initErrors {
					if initErr.Component == "AI Manager" {
						errMsg += fmt.Sprintf("\n- %s: %s", initErr.Component, initErr.Reason)
						if len(initErr.Details) > 0 {
							for key, value := range initErr.Details {
								errMsg += fmt.Sprintf("\n  %s: %s", key, value)
							}
						}
					}
				}
			} else {
				errMsg += " Please check AI service configuration."
			}

			return nil, status.Errorf(codes.FailedPrecondition, errMsg)
		}
		return s.handleTestConnection(ctx, req)
	case "health_check":
		return s.handleHealthCheck(ctx, req)
	case "update_config":
		if s.aiManager == nil {
			logging.Logger.Error("Config update requested but AI manager is not available")

			// Build enhanced error message with initialization details
			errMsg := "AI configuration update is currently unavailable."
			if len(initErrors) > 0 {
				errMsg += " Initialization errors:"
				for _, initErr := range initErrors {
					if initErr.Component == "AI Manager" {
						errMsg += fmt.Sprintf("\n- %s: %s", initErr.Component, initErr.Reason)
						if len(initErr.Details) > 0 {
							for key, value := range initErr.Details {
								errMsg += fmt.Sprintf("\n  %s: %s", key, value)
							}
						}
					}
				}
			} else {
				errMsg += " Please check AI service configuration."
			}

			return nil, status.Errorf(codes.FailedPrecondition, errMsg)
		}
		return s.handleUpdateConfig(ctx, req)
	default:
		// Backward compatibility: support legacy natural language queries
		// Check AI engine availability for legacy queries
		if s.aiEngine == nil {
			logging.Logger.Error("AI query requested but AI engine is not available")

			// Build enhanced error message with initialization details
			errMsg := "AI service is currently unavailable."

			// Add specific initialization error information if available
			if len(initErrors) > 0 {
				errMsg += " Initialization errors:"
				for _, initErr := range initErrors {
					errMsg += fmt.Sprintf("\n- %s: %s", initErr.Component, initErr.Reason)
					// Add relevant details
					if len(initErr.Details) > 0 {
						for key, value := range initErr.Details {
							errMsg += fmt.Sprintf("\n  %s: %s", key, value)
						}
					}
				}
			} else {
				errMsg += " Please check AI provider configuration and connectivity."
			}

			return nil, status.Errorf(codes.FailedPrecondition, errMsg)
		}
		return s.handleLegacyQuery(ctx, req)
	}
}

// handleCapabilitiesQuery handles requests for AI plugin capabilities
func (s *AIPluginService) handleCapabilitiesQuery(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	logging.Logger.Info("Handling capabilities query", "key", req.Key)

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
			logging.Logger.Error("Failed to parse capability request parameters", "error", err)
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
		logging.Logger.Error("Failed to get capabilities", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get capabilities: %v", err)
	}

	// Convert capabilities to JSON
	capabilitiesJSON, err := json.Marshal(capabilities)
	if err != nil {
		logging.Logger.Error("Failed to marshal capabilities", "error", err)
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

	logging.Logger.Info("Capabilities query completed successfully",
		"models", len(capabilities.Models), "databases", len(capabilities.Databases), "features", len(capabilities.Features))

	return result, nil
}

// Verify returns the plugin status for health checks
// This implements graceful degradation: the plugin is considered "Ready" if the core
// configuration is loaded, even if AI services are temporarily unavailable.
// AI service status is reported in the message field for diagnostic purposes.
func (s *AIPluginService) Verify(ctx context.Context, _ *server.Empty) (*server.ExtensionStatus, error) {
	logging.Logger.Info("Health check requested")

	if err := contextError(ctx); err != nil {
		return nil, err
	}

	// Plugin Ready check: only require core configuration to be loaded
	// This allows UI and configuration features to work even if AI services are down
	isReady := s.config != nil

	var message string
	if !isReady {
		message = "Configuration not loaded - plugin cannot start"
		logging.Logger.Error("Health check failed: configuration missing")
	} else {
		// Build detailed status message
		aiEngineStatus := "unavailable"
		if s.aiEngine != nil {
			aiEngineStatus = "operational"
		}
		aiManagerStatus := "unavailable"
		if s.aiManager != nil {
			aiManagerStatus = "operational"
		}

		if s.aiEngine != nil && s.aiManager != nil {
			message = "AI Plugin fully operational"
			logging.Logger.Info("Health check passed: plugin fully operational")
		} else {
			message = fmt.Sprintf("AI Plugin ready (degraded mode: AI engine=%s, AI manager=%s)",
				aiEngineStatus, aiManagerStatus)
			logging.Logger.Warn("Health check passed but plugin in degraded mode",
				"ai_engine", aiEngineStatus,
				"ai_manager", aiManagerStatus)
		}
	}

	// Include detailed version information for diagnostics
	versionInfo := fmt.Sprintf("%s (API: %s, gRPC: %s, requires api-testing >= %s)",
		PluginVersion, APIVersion, GRPCInterfaceVersion, MinCompatibleAPITestingVersion)

	status := &server.ExtensionStatus{
		Ready:    isReady,
		ReadOnly: true, // AI plugin is read-only - only provides AI query and UI features, not data storage
		Version:  versionInfo,
		Message:  message,
	}

	logging.Logger.Debug("Verify response",
		"ready", isReady,
		"version", versionInfo,
		"message", message)

	return status, nil
}

// Shutdown gracefully stops the AI plugin service
func (s *AIPluginService) Shutdown() {
	logging.Logger.Info("Shutting down AI plugin service...")

	if s.aiEngine != nil {
		logging.Logger.Info("Closing AI engine...")
		s.aiEngine.Close()
		logging.Logger.Info("AI engine closed successfully")
	}

	logging.Logger.Info("AI plugin service shutdown complete")
}

// GetVersion returns the plugin version information
func (s *AIPluginService) GetVersion(ctx context.Context, _ *server.Empty) (*server.Version, error) {
	logging.Logger.Debug("GetVersion called")

	if err := contextError(ctx); err != nil {
		return nil, err
	}

	return &server.Version{
		Version: fmt.Sprintf("%s (API: %s, gRPC: %s)", PluginVersion, APIVersion, GRPCInterfaceVersion),
		Commit:  "HEAD", // Could be set during build time via ldflags
		Date:    time.Now().Format(time.RFC3339),
	}, nil
}

var (
	// APIVersion is the current API version for the AI plugin
	APIVersion = "v1"
	// PluginVersion is the plugin implementation version (resolved at build time or via module metadata)
	PluginVersion = detectPluginVersion()
	// GRPCInterfaceVersion is the expected gRPC interface version from api-testing.
	// This helps detect incompatibilities between plugin and main project.
	GRPCInterfaceVersion = detectAPITestingVersion()
	// MinCompatibleAPITestingVersion is the minimum api-testing version required.
	MinCompatibleAPITestingVersion = GRPCInterfaceVersion
)

// handleAIGenerate handles ai.generate calls
func (s *AIPluginService) handleAIGenerate(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	start := time.Now()
	provider := s.config.AI.DefaultService

	defer func() {
		duration := time.Since(start).Seconds()
		metrics.RecordDuration("generate", provider, duration)
	}()

	// Parse parameters from SQL field
	var params struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
		Config string `json:"config"`
	}

	if req.Sql != "" {
		if err := json.Unmarshal([]byte(req.Sql), &params); err != nil {
			return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidRequest, "failed to parse AI parameters: %v", err)
		}
	}

	if params.Prompt == "" {
		return nil, apperrors.ToGRPCError(apperrors.ErrInvalidRequest)
	}

	// Parse optional config
	var configMap map[string]interface{}
	if params.Config != "" {
		if err := json.Unmarshal([]byte(params.Config), &configMap); err != nil {
			logging.Logger.Warn("Failed to parse config JSON", "error", err)
		}
	}

	logging.Logger.Debug("AI generate parameters",
		"model", params.Model,
		"prompt_length", len(params.Prompt),
		"has_config", params.Config != "")

	// Generate using AI engine
	context := map[string]string{}
	if params.Model != "" {
		context["preferred_model"] = params.Model
		logging.Logger.Debug("Setting preferred model", "model", params.Model)
	}
	if params.Config != "" {
		context["config"] = params.Config
	}

	// Get database type from configuration, fallback to mysql if not configured
	databaseType := "mysql"
	if s.config.Database.DefaultType != "" {
		databaseType = s.config.Database.DefaultType
	}

	sqlResult, err := s.aiEngine.GenerateSQL(ctx, &ai.GenerateSQLRequest{
		NaturalLanguage: params.Prompt,
		DatabaseType:    databaseType,
		Context:         context,
	})
	if err != nil {
		metrics.RecordRequest("generate", provider, "error")

		logging.Logger.Error("SQL generation failed",
			"error", err,
			"database_type", databaseType,
			"prompt_length", len(params.Prompt))

		// Business logic error: return error in response data, not as gRPC error
		// This allows the main project to handle it gracefully
		return &server.DataQueryResult{
			Data: []*server.Pair{
				{Key: "api_version", Value: APIVersion},
				{Key: "success", Value: "false"},
				{Key: "error", Value: err.Error()},
				{Key: "error_code", Value: "GENERATION_FAILED"},
			},
		}, nil
	}

	// Return in simplified format with line break
	simpleFormat := fmt.Sprintf("sql:%s\nexplanation:%s", sqlResult.SQL, sqlResult.Explanation)

	// Build minimal meta information for UI display
	metaData := map[string]interface{}{
		"confidence": sqlResult.ConfidenceScore,
		"model":      sqlResult.ModelUsed,
	}
	metaJSON, err := json.Marshal(metaData)
	if err != nil {
		metaJSON = []byte(fmt.Sprintf(`{"confidence": %f, "model": "%s"}`,
			sqlResult.ConfidenceScore, sqlResult.ModelUsed))
	}

	logging.Logger.Debug("Returning SQL generation result",
		"confidence", sqlResult.ConfidenceScore,
		"model", sqlResult.ModelUsed,
		"sql_length", len(sqlResult.SQL))

	metrics.RecordRequest("generate", provider, "success")

	return &server.DataQueryResult{
		Data: []*server.Pair{
			{Key: "api_version", Value: APIVersion},
			{Key: "generated_sql", Value: simpleFormat},
			{Key: "success", Value: "true"},
			{Key: "meta", Value: string(metaJSON)},
		},
	}, nil
}

// handleAICapabilities handles ai.capabilities calls
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
			logging.Logger.Warn("Failed to parse capabilities request overrides", "error", err)
		}
	}

	// Check if capability detector is available
	if s.capabilityDetector == nil {
		logging.Logger.Warn("Capability detector not available - returning minimal capabilities")
		// Return minimal capabilities when detector is not available
		minimalCaps := map[string]interface{}{
			"plugin_ready":   true,
			"ai_available":   false,
			"degraded_mode":  true,
			"plugin_version": PluginVersion,
			"api_version":    APIVersion,
		}
		capsJSON, _ := json.Marshal(minimalCaps)
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
		logging.Logger.Error("Failed to get capabilities", "error", err)
		return nil, apperrors.ToGRPCErrorf(apperrors.ErrProviderNotAvailable, "failed to retrieve capabilities: %v", err)
	}

	// Convert to JSON strings for AI interface standard
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

// GetMenus returns the menu entries for AI plugin UI
func (s *AIPluginService) GetMenus(ctx context.Context, _ *server.Empty) (*server.MenuList, error) {
	logging.Logger.Debug("AI plugin GetMenus called")

	if err := contextError(ctx); err != nil {
		return nil, err
	}

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
	logging.Logger.Debug("AI plugin GetPageOfJS called", "name", req.Name)

	if err := contextError(ctx); err != nil {
		return nil, err
	}

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
	logging.Logger.Debug("Serving CSS for AI plugin", "name", req.Name)

	if err := contextError(ctx); err != nil {
		return nil, err
	}

	if req.Name != "ai-chat" {
		return &server.CommonResult{
			Success: false,
			Message: fmt.Sprintf("Unknown AI plugin page: %s", req.Name),
		}, nil
	}

	return &server.CommonResult{
		Success: true,
		Message: aiChatCSS,
	}, nil
}

// GetPageOfStatic returns static files for AI plugin UI (not implemented)
func (s *AIPluginService) GetPageOfStatic(ctx context.Context, _ *server.SimpleName) (*server.CommonResult, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}

	result := &server.CommonResult{
		Success: false,
		Message: "Static files not supported",
	}
	return result, nil
}

// GetThemes returns the list of available themes (AI plugin doesn't provide themes)
func (s *AIPluginService) GetThemes(ctx context.Context, _ *server.Empty) (*server.SimpleList, error) {
	logging.Logger.Debug("GetThemes called - AI plugin does not provide themes")

	if err := contextError(ctx); err != nil {
		return nil, err
	}

	return &server.SimpleList{
		Data: []*server.Pair{}, // Empty list - AI plugin doesn't provide themes
	}, nil
}

// GetTheme returns a specific theme (AI plugin doesn't provide themes)
func (s *AIPluginService) GetTheme(ctx context.Context, req *server.SimpleName) (*server.CommonResult, error) {
	logging.Logger.Debug("GetTheme called", "theme", req.Name)

	if err := contextError(ctx); err != nil {
		return nil, err
	}

	return &server.CommonResult{
		Success: false,
		Message: "AI plugin does not provide themes",
	}, nil
}

// GetBindings returns the list of available bindings (AI plugin doesn't provide bindings)
func (s *AIPluginService) GetBindings(ctx context.Context, _ *server.Empty) (*server.SimpleList, error) {
	logging.Logger.Debug("GetBindings called - AI plugin does not provide bindings")

	if err := contextError(ctx); err != nil {
		return nil, err
	}

	return &server.SimpleList{
		Data: []*server.Pair{}, // Empty list - AI plugin doesn't provide bindings
	}, nil
}

// GetBinding returns a specific binding (AI plugin doesn't provide bindings)
func (s *AIPluginService) GetBinding(ctx context.Context, req *server.SimpleName) (*server.CommonResult, error) {
	logging.Logger.Debug("GetBinding called", "binding", req.Name)

	if err := contextError(ctx); err != nil {
		return nil, err
	}

	return &server.CommonResult{
		Success: false,
		Message: "AI plugin does not provide bindings",
	}, nil
}

// PProf returns profiling data for the AI plugin
func (s *AIPluginService) PProf(ctx context.Context, req *server.PProfRequest) (*server.PProfData, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "profiling request cannot be nil")
	}

	logging.Logger.Debug("PProf called", "profile_type", req.Name)

	// For now, return empty profiling data
	// In the future, this could be extended to provide actual profiling information
	result := &server.PProfData{
		Data: []byte{}, // Empty profiling data
	}
	return result, nil
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
		logging.Logger.Warn("Missing key field (natural language query) in request")
		return nil, status.Errorf(codes.InvalidArgument, "key field is required for AI queries (natural language input)")
	}

	// Generate SQL using AI engine
	queryPreview := req.Key
	if len(queryPreview) > 100 {
		queryPreview = queryPreview[:100] + "..."
	}
	logging.Logger.Info("Generating SQL for natural language query", "query_preview", queryPreview)

	// Create context map from available information
	contextMap := make(map[string]string)
	if req.Sql != "" {
		contextMap["existing_sql"] = req.Sql
	}

	// Get database type from configuration, fallback to mysql if not configured
	databaseType := "mysql"
	if s.config.Database.DefaultType != "" {
		databaseType = s.config.Database.DefaultType
	}

	sqlResult, err := s.aiEngine.GenerateSQL(ctx, &ai.GenerateSQLRequest{
		NaturalLanguage: req.Key,
		DatabaseType:    databaseType,
		Context:         contextMap,
	})
	if err != nil {
		logging.Logger.Error("Failed to generate SQL", "error", err)

		// Business logic error: return error in response data, not as gRPC error
		return &server.DataQueryResult{
			Data: []*server.Pair{
				{Key: "success", Value: "false"},
				{Key: "error", Value: err.Error()},
				{Key: "error_code", Value: "GENERATION_FAILED"},
			},
		}, nil
	}

	// Create response in simplified format with line break
	simpleFormat := fmt.Sprintf("sql:%s\nexplanation:%s", sqlResult.SQL, sqlResult.Explanation)

	// Build minimal meta information for UI display
	metaData := map[string]interface{}{
		"confidence": sqlResult.ConfidenceScore,
		"model":      sqlResult.ModelUsed,
	}
	metaJSON, err := json.Marshal(metaData)
	if err != nil {
		metaJSON = []byte(fmt.Sprintf(`{"confidence": %f, "model": "%s"}`,
			sqlResult.ConfidenceScore, sqlResult.ModelUsed))
	}

	logging.Logger.Debug("Legacy query result",
		"confidence", sqlResult.ConfidenceScore,
		"model", sqlResult.ModelUsed,
		"request_id", sqlResult.RequestID)

	result := &server.DataQueryResult{
		Data: []*server.Pair{
			{
				Key:   "generated_sql",
				Value: simpleFormat,
			},
			{
				Key:   "success",
				Value: "true",
			},
			{
				Key:   "meta",
				Value: string(metaJSON),
			},
		},
	}

	logging.Logger.Info("AI query completed successfully",
		"request_id", sqlResult.RequestID, "confidence", sqlResult.ConfidenceScore, "processing_time_ms", sqlResult.ProcessingTime.Milliseconds())

	return result, nil
}

// handleGetProviders returns the list of available AI providers
func (s *AIPluginService) handleGetProviders(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	logging.Logger.Debug("Getting AI providers list")

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
			logging.Logger.Warn("Failed to parse provider query overrides", "error", err)
		}
	}

	// Discover providers
	providers, err := s.aiManager.DiscoverProviders(ctx)
	if err != nil {
		logging.Logger.Error("Failed to discover providers", "error", err)
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
			return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidRequest, "invalid parameters: %v", err)
		}
	}

	if params.Provider == "" {
		// If no provider specified, return all models from all providers
		allModels := make(map[string][]interface{})

		// Get all configured clients
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

	// Get models for specific provider
	// Map frontend category names to backend provider names
	providerName := params.Provider
	switch params.Provider {
	case "local":
		providerName = "ollama"
	case "online":
		// Map "online" to default online provider (can be configured)
		providerName = "deepseek"
	}

	models, err := s.aiManager.GetModels(ctx, providerName)
	if err != nil {
		logging.Logger.Error("Failed to get models", "provider", providerName, "error", err)
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

// handleTestConnection tests a connection with provided configuration
func (s *AIPluginService) handleTestConnection(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	logging.Logger.Debug("Handling test connection request", "sql_length", len(req.Sql))

	// Parse configuration from SQL field
	var config universal.Config
	if req.Sql != "" {
		var payload map[string]any
		if err := json.Unmarshal([]byte(req.Sql), &payload); err != nil {
			logging.Logger.Error("Failed to parse connection config", "error", err)
			return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidConfig, "invalid configuration: %v", err)
		}

		normalizeDurationField(payload, "timeout")

		normalizedPayload, err := json.Marshal(payload)
		if err != nil {
			logging.Logger.Error("Failed to normalize connection config", "error", err)
			return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidConfig, "invalid configuration: %v", err)
		}

		if err := json.Unmarshal(normalizedPayload, &config); err != nil {
			logging.Logger.Error("Failed to decode normalized connection config", "error", err)
			return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidConfig, "invalid configuration: %v", err)
		}
	}

	// Map "local" to "ollama" for backward compatibility
	if config.Provider == "local" {
		config.Provider = "ollama"
	}

	// Log configuration for debugging (mask API key)
	apiKeyDisplay := "***masked***"
	if config.APIKey != "" && len(config.APIKey) > 4 {
		apiKeyDisplay = config.APIKey[:4] + "***"
	}
	logging.Logger.Debug("Testing connection",
		"provider", config.Provider,
		"api_key_prefix", apiKeyDisplay,
		"model", config.Model)

	// Test the connection
	result, err := s.aiManager.TestConnection(ctx, &config)
	if err != nil {
		logging.Logger.Error("Connection test failed",
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

// handleUpdateConfig updates the configuration for a provider
func (s *AIPluginService) handleUpdateConfig(_ context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	logging.Logger.Debug("Handling update config request", "sql_length", len(req.Sql))

	// Parse update request from SQL field
	var updateReq struct {
		Provider string            `json:"provider"`
		Config   *universal.Config `json:"config"`
	}

	if req.Sql != "" {
		var payload map[string]any
		if err := json.Unmarshal([]byte(req.Sql), &payload); err != nil {
			logging.Logger.Error("Failed to parse update request", "error", err)
			return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidRequest, "invalid update request: %v", err)
		}

		if configPayload, ok := payload["config"].(map[string]any); ok {
			normalizeDurationField(configPayload, "timeout")
			payload["config"] = configPayload
		}

		normalizedPayload, err := json.Marshal(payload)
		if err != nil {
			logging.Logger.Error("Failed to normalize update config payload", "error", err)
			return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidRequest, "invalid update request: %v", err)
		}

		if err := json.Unmarshal(normalizedPayload, &updateReq); err != nil {
			logging.Logger.Error("Failed to decode normalized update request", "error", err)
			return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidRequest, "invalid update request: %v", err)
		}
	}

	if updateReq.Provider == "" || updateReq.Config == nil {
		return nil, apperrors.ToGRPCError(apperrors.ErrInvalidRequest)
	}

	// Map "local" to "ollama" for backward compatibility
	if updateReq.Provider == "local" {
		updateReq.Provider = "ollama"
	}
	if updateReq.Config.Provider == "local" {
		updateReq.Config.Provider = "ollama"
	}

	logging.Logger.Debug("Updating provider config", "provider", updateReq.Provider)

	// Update the configuration by adding/updating the client
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
		logging.Logger.Error("Failed to rebuild AI manager",
			"provider", updateReq.Provider,
			"error", err)
		return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidConfig, "failed to rebuild AI manager: %v", err)
	}

	engine, err := ai.NewEngineWithManager(manager, newAIConfig)
	if err != nil {
		logging.Logger.Error("Failed to rebuild AI engine",
			"provider", updateReq.Provider,
			"error", err)
		_ = manager.Close()
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

// handleHealthCheck performs health check on specific AI service
// This is separate from the plugin's Verify method, which only checks if the plugin is ready
func (s *AIPluginService) handleHealthCheck(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	logging.Logger.Debug("AI service health check requested")

	// Parse request parameters
	var params struct {
		Provider string `json:"provider"`
		Timeout  int    `json:"timeout"` // seconds
	}

	if req.Sql != "" {
		if err := json.Unmarshal([]byte(req.Sql), &params); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid parameters: %v", err)
		}
	}

	// Default timeout 5 seconds
	if params.Timeout == 0 {
		params.Timeout = 5
	}

	// Check specified provider or default provider
	provider := params.Provider
	if provider == "" {
		provider = s.config.AI.DefaultService
	}

	// Execute health check with timeout
	checkCtx, cancel := context.WithTimeout(ctx, time.Duration(params.Timeout)*time.Second)
	defer cancel()

	var healthy bool
	var errorMsg string

	// Perform health check on the AI engine
	if s.aiEngine != nil {
		if s.aiEngine.IsHealthy() {
			healthy = true
			errorMsg = ""
		} else {
			healthy = false
			errorMsg = "AI service is not available"
		}
	} else {
		healthy = false
		errorMsg = "AI engine not initialized"
	}

	// Wait for context timeout if still checking
	select {
	case <-checkCtx.Done():
		if checkCtx.Err() == context.DeadlineExceeded {
			healthy = false
			errorMsg = "Health check timeout"
		}
	default:
		// Check completed
	}

	return &server.DataQueryResult{
		Data: []*server.Pair{
			{Key: "healthy", Value: fmt.Sprintf("%t", healthy)},
			{Key: "provider", Value: provider},
			{Key: "error", Value: errorMsg},
			{Key: "timestamp", Value: time.Now().Format(time.RFC3339)},
			{Key: "success", Value: "true"}, // API call succeeded even if service is unhealthy
		},
	}, nil
}
