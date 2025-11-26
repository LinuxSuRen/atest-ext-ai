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
	"time"

	"github.com/linuxsuren/api-testing/pkg/server"
	"github.com/linuxsuren/atest-ext-ai/pkg/ai"
	apperrors "github.com/linuxsuren/atest-ext-ai/pkg/errors"
	"github.com/linuxsuren/atest-ext-ai/pkg/logging"
	"github.com/linuxsuren/atest-ext-ai/pkg/metrics"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GenerationMetadata describes metadata returned with AI generation responses.
type GenerationMetadata struct {
	Confidence float32 `json:"confidence"`
	Model      string  `json:"model,omitempty"`
	Dialect    string  `json:"dialect"`
}

// CapabilitySummary is returned when the capability detector is unavailable.
type CapabilitySummary struct {
	PluginReady   bool   `json:"plugin_ready"`
	AIAvailable   bool   `json:"ai_available"`
	DegradedMode  bool   `json:"degraded_mode"`
	PluginVersion string `json:"plugin_version"`
	APIVersion    string `json:"api_version"`
}

// GenerationConfigOverrides captures optional generation configuration overrides.
type GenerationConfigOverrides struct {
	DatabaseTypePrimary string `json:"database_type"`
	DatabaseDialect     string `json:"databaseDialect"`
	DatabaseDialectAlt  string `json:"database_dialect"`
	Dialect             string `json:"dialect"`
	Provider            string `json:"provider"`
	Endpoint            string `json:"endpoint"`
}

func (g GenerationConfigOverrides) preferredDatabaseType() string {
	candidates := []string{
		g.DatabaseTypePrimary,
		g.DatabaseDialect,
		g.DatabaseDialectAlt,
		g.Dialect,
	}
	for _, candidate := range candidates {
		if normalized := normalizeDatabaseType(candidate); normalized != "" {
			return normalized
		}
	}
	return ""
}

// handleAIGenerate handles ai.generate calls.
func (s *AIPluginService) handleAIGenerate(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	logger := loggerFromContext(ctx)
	start := time.Now()
	provider := s.config.AI.DefaultService

	defer func() {
		duration := time.Since(start).Seconds()
		metrics.RecordDuration("generate", provider, duration)
	}()

	// Parse parameters from SQL field.
	var params struct {
		Model        string `json:"model"`
		Prompt       string `json:"prompt"`
		Config       string `json:"config"`
		DatabaseType string `json:"database_type"`
	}

	if req.Sql != "" {
		if err := json.Unmarshal([]byte(req.Sql), &params); err != nil {
			return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidRequest, "failed to parse AI parameters: %v", err)
		}
	}

	if params.Prompt == "" {
		return nil, apperrors.ToGRPCError(apperrors.ErrInvalidRequest)
	}

	// Parse optional config.
	var generationOverrides GenerationConfigOverrides
	if params.Config != "" {
		if err := json.Unmarshal([]byte(params.Config), &generationOverrides); err != nil {
			logger.Warn("Failed to parse config JSON", "error", err)
		}
	}

	if err := s.inputValidator.ValidatePrompt(params.Prompt); err != nil {
		return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidRequest, err.Error())
	}
	if err := s.inputValidator.ValidateDatabaseName(params.DatabaseType); err != nil {
		return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidRequest, err.Error())
	}
	if endpoint := generationOverrides.Endpoint; endpoint != "" {
		if err := s.inputValidator.ValidateEndpoint(endpoint); err != nil {
			return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidRequest, err.Error())
		}
	}

	logger.Debug("AI generate parameters",
		"model", params.Model,
		"prompt_length", len(params.Prompt),
		"has_config", params.Config != "")

	apiKey := apiKeyFromContext(ctx)

	// Generate using AI engine.
	context := map[string]string{}
	if params.Model != "" {
		context["preferred_model"] = params.Model
		logger.Debug("Setting preferred model", "model", params.Model)
	}
	if params.Config != "" {
		context["config"] = params.Config
	}

	// Get database type from configuration, fallback to mysql if not configured.
	databaseType := s.resolveDatabaseType(params.DatabaseType, generationOverrides)
	if err := s.inputValidator.ValidateDatabaseName(databaseType); err != nil {
		return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidRequest, err.Error())
	}
	context["database_type"] = databaseType

	if err := s.inputValidator.ValidateContext(context); err != nil {
		return nil, apperrors.ToGRPCErrorf(apperrors.ErrInvalidRequest, err.Error())
	}

	sqlResult, err := s.aiEngine.GenerateSQL(ctx, &ai.GenerateSQLRequest{
		NaturalLanguage: params.Prompt,
		DatabaseType:    databaseType,
		Context:         context,
		RuntimeAPIKey:   apiKey,
	})
	if err != nil {
		metrics.RecordRequest("generate", provider, "error")

		logger.Error("SQL generation failed",
			"error", err,
			"database_type", databaseType,
			"prompt_length", len(params.Prompt))

		// Business logic error: return error in response data, not as gRPC error.
		return &server.DataQueryResult{
			Data: []*server.Pair{
				{Key: "api_version", Value: APIVersion},
				{Key: "success", Value: "false"},
				{Key: "error", Value: err.Error()},
				{Key: "error_code", Value: "GENERATION_FAILED"},
			},
		}, nil
	}

	// Return in simplified format with line break.
	simpleFormat := fmt.Sprintf("sql:%s\nexplanation:%s", sqlResult.SQL, sqlResult.Explanation)

	// Build minimal meta information for UI display.
	meta := GenerationMetadata{
		Confidence: sqlResult.ConfidenceScore,
		Model:      sqlResult.ModelUsed,
		Dialect:    databaseType,
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		metaJSON = []byte(fmt.Sprintf(`{"confidence": %f, "model": "%s"}`,
			sqlResult.ConfidenceScore, sqlResult.ModelUsed))
	}

	logger.Debug("Returning SQL generation result",
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

// handleLegacyQuery maintains backward compatibility with the original implementation.
func (s *AIPluginService) handleLegacyQuery(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	logger := loggerFromContext(ctx)
	// Handle legacy capabilities query.
	if req.Key == "capabilities" || strings.HasPrefix(req.Key, "ai.capabilities") {
		return s.handleCapabilitiesQuery(ctx, req)
	}

	if req.Key == "" {
		logger.Warn("Missing key field (natural language query) in request")
		return nil, status.Errorf(codes.InvalidArgument, "key field is required for AI queries (natural language input)")
	}
	if err := s.inputValidator.ValidatePrompt(req.Key); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	// Generate SQL using AI engine.
	queryPreview := req.Key
	if len(queryPreview) > 100 {
		queryPreview = queryPreview[:100] + "..."
	}
	logger.Info("Generating SQL for natural language query", "query_preview", queryPreview)

	// Create context map from available information.
	contextMap := make(map[string]string)
	if req.Sql != "" {
		contextMap["existing_sql"] = req.Sql
	}

	// Get database type from configuration, fallback to mysql if not configured.
	databaseType := s.defaultDatabaseType()
	if err := s.inputValidator.ValidateDatabaseName(databaseType); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	contextMap["database_type"] = databaseType

	if err := s.inputValidator.ValidateContext(contextMap); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	sqlResult, err := s.aiEngine.GenerateSQL(ctx, &ai.GenerateSQLRequest{
		NaturalLanguage: req.Key,
		DatabaseType:    databaseType,
		Context:         contextMap,
	})
	if err != nil {
		logger.Error("Failed to generate SQL", "error", err)

		// Business logic error: return error in response data, not as gRPC error.
		return &server.DataQueryResult{
			Data: []*server.Pair{
				{Key: "success", Value: "false"},
				{Key: "error", Value: err.Error()},
				{Key: "error_code", Value: "GENERATION_FAILED"},
			},
		}, nil
	}

	// Create response in simplified format with line break.
	simpleFormat := fmt.Sprintf("sql:%s\nexplanation:%s", sqlResult.SQL, sqlResult.Explanation)

	// Build minimal meta information for UI display.
	meta := GenerationMetadata{
		Confidence: sqlResult.ConfidenceScore,
		Model:      sqlResult.ModelUsed,
		Dialect:    databaseType,
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		metaJSON = []byte(fmt.Sprintf(`{"confidence": %f, "model": "%s"}`,
			sqlResult.ConfidenceScore, sqlResult.ModelUsed))
	}

	logger.Debug("Legacy query result",
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

	logger.Info("AI query completed successfully",
		"request_id", sqlResult.RequestID, "confidence", sqlResult.ConfidenceScore, "processing_time_ms", sqlResult.ProcessingTime.Milliseconds())

	return result, nil
}

func normalizeDatabaseType(value string) string {
	dbType := strings.ToLower(strings.TrimSpace(value))
	switch dbType {
	case "mysql":
		return "mysql"
	case "postgres", "postgresql", "pg":
		return "postgresql"
	case "sqlite", "sqlite3":
		return "sqlite"
	default:
		return ""
	}
}

func (s *AIPluginService) defaultDatabaseType() string {
	if normalized := normalizeDatabaseType(s.config.Database.DefaultType); normalized != "" {
		return normalized
	}
	return "mysql"
}

func (s *AIPluginService) resolveDatabaseType(explicit string, overrides GenerationConfigOverrides) string {
	if normalized := normalizeDatabaseType(explicit); normalized != "" {
		return normalized
	}

	if fromConfig := overrides.preferredDatabaseType(); fromConfig != "" {
		return fromConfig
	}

	return s.defaultDatabaseType()
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
