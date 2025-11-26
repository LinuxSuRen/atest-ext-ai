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
	"time"

	"github.com/linuxsuren/api-testing/pkg/server"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Verify returns the plugin status for health checks.
func (s *AIPluginService) Verify(ctx context.Context, _ *server.Empty) (*server.ExtensionStatus, error) {
	logger := loggerFromContext(ctx)
	logger.Info("Health check requested")

	if err := contextError(ctx); err != nil {
		return nil, err
	}

	isReady := s.config != nil

	var message string
	if !isReady {
		message = "Configuration not loaded - plugin cannot start"
		logger.Error("Health check failed: configuration missing")
	} else {
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
			logger.Info("Health check passed: plugin fully operational")
		} else {
			message = fmt.Sprintf("AI Plugin ready (degraded mode: AI engine=%s, AI manager=%s)",
				aiEngineStatus, aiManagerStatus)
			logger.Warn("Health check passed but plugin in degraded mode",
				"ai_engine", aiEngineStatus,
				"ai_manager", aiManagerStatus)
		}
	}

	versionInfo := fmt.Sprintf("%s (API: %s, gRPC: %s, requires api-testing >= %s)",
		PluginVersion, APIVersion, GRPCInterfaceVersion, MinCompatibleAPITestingVersion)

	statusResp := &server.ExtensionStatus{
		Ready:    isReady,
		ReadOnly: true,
		Version:  versionInfo,
		Message:  message,
	}

	logger.Debug("Verify response",
		"ready", isReady,
		"version", versionInfo,
		"message", message)

	return statusResp, nil
}

// handleHealthCheck performs health check on specific AI service.
func (s *AIPluginService) handleHealthCheck(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
	logger := loggerFromContext(ctx)
	logger.Debug("AI service health check requested")

	var params struct {
		Provider string `json:"provider"`
		Timeout  int    `json:"timeout"`
	}

	if req.Sql != "" {
		if err := json.Unmarshal([]byte(req.Sql), &params); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid parameters: %v", err)
		}
	}

	if params.Timeout == 0 {
		params.Timeout = 5
	}

	provider := params.Provider
	if provider == "" {
		provider = s.config.AI.DefaultService
	}

	checkCtx, cancel := context.WithTimeout(ctx, time.Duration(params.Timeout)*time.Second)
	defer cancel()

	var healthy bool
	var errorMsg string

	if s.aiEngine != nil {
		if s.aiEngine.IsHealthy() {
			healthy = true
		} else {
			errorMsg = "AI service is not available"
		}
	} else {
		errorMsg = "AI engine not initialized"
	}

	select {
	case <-checkCtx.Done():
		if checkCtx.Err() == context.DeadlineExceeded {
			healthy = false
			errorMsg = "Health check timeout"
		}
	default:
	}

	return &server.DataQueryResult{
		Data: []*server.Pair{
			{Key: "healthy", Value: fmt.Sprintf("%t", healthy)},
			{Key: "provider", Value: provider},
			{Key: "error", Value: errorMsg},
			{Key: "timestamp", Value: time.Now().Format(time.RFC3339)},
			{Key: "success", Value: "true"},
		},
	}, nil
}
