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

	"github.com/linuxsuren/atest-ext-ai/pkg/ai"
	"github.com/linuxsuren/atest-ext-ai/pkg/config"
	"github.com/linuxsuren/atest-ext-ai/pkg/logging"
)

// NewAIPluginService creates a new AI plugin service instance. The plugin is
// allowed to start in a degraded mode so that configuration and UI surfaces
// remain available even when AI providers cannot be reached.
func NewAIPluginService() (*AIPluginService, error) {
	logging.Logger.Info("Initializing AI plugin service...")
	logVersionMetadata()

	cfg, err := config.LoadConfig()
	if err != nil {
		logging.Logger.Error("Failed to load configuration", "error", err)
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	logging.Logger.Info("Configuration loaded successfully")

	service := &AIPluginService{
		config:         cfg,
		inputValidator: DefaultInputValidator(),
	}

	service.aiEngine = initializeAIEngine(cfg)
	service.aiManager, service.capabilityDetector = initializeManager(cfg)
	service.registerQueryHandlers()

	launchProviderDiscovery(service.aiManager)
	logStartupStatus(service)

	return service, nil
}

func logVersionMetadata() {
	logging.Logger.Info("Plugin version information",
		"plugin_version", PluginVersion,
		"api_version", APIVersion,
		"grpc_interface_version", GRPCInterfaceVersion,
		"min_api_testing_version", MinCompatibleAPITestingVersion)
	logging.Logger.Info("Compatibility note: This plugin requires api-testing >= " + MinCompatibleAPITestingVersion)
}

func initializeAIEngine(cfg *config.Config) ai.Engine {
	aiEngine, err := ai.NewEngine(cfg.AI)
	if err != nil {
		logging.Logger.Warn("AI engine initialization failed - plugin will start in degraded mode",
			"error", err,
			"impact", "AI generation features will be unavailable until AI service is available")

		initErr := InitializationError{
			Component: "AI Engine",
			Reason:    err.Error(),
			Details: map[string]string{
				"default_service": cfg.AI.DefaultService,
				"provider_count":  fmt.Sprintf("%d", len(cfg.AI.Services)),
			},
		}
		if cfg.AI.DefaultService != "" {
			if svc, ok := cfg.AI.Services[cfg.AI.DefaultService]; ok {
				initErr.Details["provider_endpoint"] = svc.Endpoint
				initErr.Details["provider_model"] = svc.Model
			}
		}
		initErrors = append(initErrors, initErr)
		return nil
	}

	logging.Logger.Info("AI engine initialized successfully")
	return aiEngine
}

func initializeManager(cfg *config.Config) (*ai.Manager, *ai.CapabilityDetector) {
	aiManager, err := ai.NewAIManager(cfg.AI)
	if err != nil {
		logging.Logger.Warn("AI manager initialization failed - plugin will start in degraded mode",
			"error", err,
			"impact", "Provider discovery and model listing will be unavailable")

		initErr := InitializationError{
			Component: "AI Manager",
			Reason:    err.Error(),
			Details: map[string]string{
				"default_service":  cfg.AI.DefaultService,
				"configured_count": fmt.Sprintf("%d", len(cfg.AI.Services)),
			},
		}
		if len(cfg.AI.Services) > 0 {
			var services []string
			for name := range cfg.AI.Services {
				services = append(services, name)
			}
			initErr.Details["configured_services"] = strings.Join(services, ", ")
		}
		initErrors = append(initErrors, initErr)
		return nil, nil
	}

	logging.Logger.Info("AI manager initialized successfully")
	capabilityDetector := ai.NewCapabilityDetector(cfg.AI, aiManager)
	logging.Logger.Info("Capability detector initialized")

	return aiManager, capabilityDetector
}

func launchProviderDiscovery(manager *ai.Manager) {
	if manager == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		providers, err := manager.DiscoverProviders(ctx)
		if err != nil {
			logging.Logger.Warn("Provider discovery failed during startup", "error", err)
			return
		}

		logging.Logger.Info("Discovered AI providers", "count", len(providers))
		for _, provider := range providers {
			logging.Logger.Debug("Provider models available", "provider", provider.Name, "model_count", len(provider.Models))
		}
	}()
}

func logStartupStatus(service *AIPluginService) {
	if service.aiEngine != nil && service.aiManager != nil {
		logging.Logger.Info("AI plugin service fully operational")
		return
	}

	logging.Logger.Warn("AI plugin service started in degraded mode - some features unavailable",
		"ai_engine_available", service.aiEngine != nil,
		"ai_manager_available", service.aiManager != nil)
}

// Shutdown gracefully stops the AI plugin service.
func (s *AIPluginService) Shutdown() {
	logging.Logger.Info("Shutting down AI plugin service...")

	if s.aiEngine != nil {
		logging.Logger.Info("Closing AI engine...")
		s.aiEngine.Close()
		logging.Logger.Info("AI engine closed successfully")
	}

	logging.Logger.Info("AI plugin service shutdown complete")
}
