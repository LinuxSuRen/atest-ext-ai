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
	"fmt"
	"strings"
	"time"

	"github.com/linuxsuren/api-testing/pkg/server"
	"github.com/linuxsuren/api-testing/pkg/testing/remote"
	"github.com/linuxsuren/atest-ext-ai/pkg/ai"
	"github.com/linuxsuren/atest-ext-ai/pkg/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}

	if err := ctx.Err(); err != nil {
		return status.Error(codes.Canceled, err.Error())
	}

	return nil
}

type contextKey string

const apiKeyContextKey contextKey = "ai-plugin-runtime-api-key"

func withAPIKey(ctx context.Context, apiKey string) context.Context {
	if ctx == nil || apiKey == "" {
		return ctx
	}
	return context.WithValue(ctx, apiKeyContextKey, apiKey)
}

func apiKeyFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if value, ok := ctx.Value(apiKeyContextKey).(string); ok {
		return value
	}
	return ""
}

func extractAPIKeyFromMetadata(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	for key, values := range md {
		switch strings.ToLower(key) {
		case "auth", "x-auth", "x-ai-api-key", "authorization":
			for _, raw := range values {
				if normalized := normalizeAPIKeyValue(raw); normalized != "" {
					return normalized
				}
			}
		}
	}
	return ""
}

func normalizeAPIKeyValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(trimmed), "bearer ") {
		return strings.TrimSpace(trimmed[7:])
	}
	return trimmed
}

// AIPluginService implements the Loader gRPC service for AI functionality
type AIPluginService struct {
	remote.UnimplementedLoaderServer
	aiEngine           ai.Engine
	config             *config.Config
	capabilityDetector *ai.CapabilityDetector
	aiManager          *ai.Manager
	inputValidator     InputValidator
	queryHandlers      map[string]queryHandler
}

type queryHandler func(context.Context, *server.DataQuery) (*server.DataQueryResult, error)

// Lifecycle construction lives in lifecycle.go to keep request handling focused here.

// Query handles AI query requests from the main API testing system
func (s *AIPluginService) Query(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	logger := loggerFromContext(ctx)
	logger.Debug("Query received",
		"type", req.Type,
		"key", req.Key,
		"sql_length", len(req.Sql))

	ctx = withAPIKey(ctx, extractAPIKeyFromMetadata(ctx))

	// Accept both empty type (for backward compatibility) and explicit "ai" type
	// The main project doesn't always send the type field
	if req.Type != "" && req.Type != "ai" {
		logger.Warn("Unsupported query type", "type", req.Type)
		return nil, status.Errorf(codes.InvalidArgument, "unsupported query type: %s", req.Type)
	}

	if handler, ok := s.queryHandlers[req.Key]; ok {
		return handler(ctx, req)
	}

	if err := s.requireEngineAvailable(
		"AI query requested but AI engine is not available",
		"AI service is currently unavailable.",
		"Please check AI provider configuration and connectivity."); err != nil {
		return nil, err
	}
	return s.handleLegacyQuery(ctx, req)
}

// GetVersion returns the plugin version information
func (s *AIPluginService) GetVersion(ctx context.Context, _ *server.Empty) (*server.Version, error) {
	logger := loggerFromContext(ctx)
	logger.Debug("GetVersion called")

	if err := contextError(ctx); err != nil {
		return nil, err
	}

	return &server.Version{
		Version: fmt.Sprintf("%s (API: %s, gRPC: %s)", PluginVersion, APIVersion, GRPCInterfaceVersion),
		Commit:  "HEAD", // Could be set during build time via ldflags
		Date:    time.Now().Format(time.RFC3339),
	}, nil
}

func (s *AIPluginService) registerQueryHandlers() {
	s.queryHandlers = map[string]queryHandler{
		"generate": s.withEngineRequirement(
			"AI generation requested but AI engine is not available",
			"AI generation service is currently unavailable.",
			"Please check AI provider configuration and connectivity.",
			s.handleAIGenerate,
		),
		"capabilities":    s.handleAICapabilities,
		"providers":       s.withManagerRequirement("Provider discovery requested but AI manager is not available", "AI provider discovery is currently unavailable.", s.handleGetProviders),
		"models":          s.withManagerRequirement("Model listing requested but AI manager is not available", "AI model listing is currently unavailable.", s.handleGetModels),
		"models_catalog":  s.handleGetModelCatalog,
		"test_connection": s.withManagerRequirement("Connection test requested but AI manager is not available", "AI connection testing is currently unavailable.", s.handleTestConnection),
		"health_check":    s.handleHealthCheck,
		"update_config":   s.withManagerRequirement("Config update requested but AI manager is not available", "AI configuration update is currently unavailable.", s.handleUpdateConfig),
	}
}

func (s *AIPluginService) withEngineRequirement(operation, baseMessage, fallback string, handler queryHandler) queryHandler {
	return func(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
		if err := s.requireEngineAvailable(operation, baseMessage, fallback); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func (s *AIPluginService) withManagerRequirement(operation, baseMessage string, handler queryHandler) queryHandler {
	return func(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
		if err := s.requireManagerAvailable(operation, baseMessage); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
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
