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
	"fmt"

	"github.com/linuxsuren/api-testing/pkg/server"
)

//go:embed assets/ai-chat.js
var aiChatJS string

//go:embed assets/ai-chat.css
var aiChatCSS string

// GetMenus returns the menu entries for AI plugin UI.
func (s *AIPluginService) GetMenus(ctx context.Context, _ *server.Empty) (*server.MenuList, error) {
	logger := loggerFromContext(ctx)
	logger.Debug("AI plugin GetMenus called")

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

// GetPageOfJS returns the JavaScript code for AI plugin UI.
func (s *AIPluginService) GetPageOfJS(ctx context.Context, req *server.SimpleName) (*server.CommonResult, error) {
	logger := loggerFromContext(ctx)
	logger.Debug("AI plugin GetPageOfJS called", "name", req.Name)

	if err := contextError(ctx); err != nil {
		return nil, err
	}

	if req.Name != "ai-chat" {
		return &server.CommonResult{
			Success: false,
			Message: fmt.Sprintf("Unknown AI plugin page: %s", req.Name),
		}, nil
	}

	jsCode := aiChatJS

	return &server.CommonResult{
		Success: true,
		Message: jsCode,
	}, nil
}

// GetPageOfCSS returns the CSS styles for AI plugin UI.
func (s *AIPluginService) GetPageOfCSS(ctx context.Context, req *server.SimpleName) (*server.CommonResult, error) {
	logger := loggerFromContext(ctx)
	logger.Debug("Serving CSS for AI plugin", "name", req.Name)

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

// GetPageOfStatic returns static files for AI plugin UI (not implemented).
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

// GetThemes returns the list of available themes (AI plugin doesn't provide themes).
func (s *AIPluginService) GetThemes(ctx context.Context, _ *server.Empty) (*server.SimpleList, error) {
	logger := loggerFromContext(ctx)
	logger.Debug("GetThemes called - AI plugin does not provide themes")

	if err := contextError(ctx); err != nil {
		return nil, err
	}

	return &server.SimpleList{
		Data: []*server.Pair{},
	}, nil
}

// GetTheme returns a specific theme (AI plugin doesn't provide themes).
func (s *AIPluginService) GetTheme(ctx context.Context, req *server.SimpleName) (*server.CommonResult, error) {
	logger := loggerFromContext(ctx)
	logger.Debug("GetTheme called", "theme", req.Name)

	if err := contextError(ctx); err != nil {
		return nil, err
	}

	return &server.CommonResult{
		Success: false,
		Message: "AI plugin does not provide themes",
	}, nil
}

// GetBindings returns the list of available bindings (AI plugin doesn't provide bindings).
func (s *AIPluginService) GetBindings(ctx context.Context, _ *server.Empty) (*server.SimpleList, error) {
	logger := loggerFromContext(ctx)
	logger.Debug("GetBindings called - AI plugin does not provide bindings")

	if err := contextError(ctx); err != nil {
		return nil, err
	}

	return &server.SimpleList{
		Data: []*server.Pair{},
	}, nil
}

// GetBinding returns a specific binding (AI plugin doesn't provide bindings).
func (s *AIPluginService) GetBinding(ctx context.Context, req *server.SimpleName) (*server.CommonResult, error) {
	logger := loggerFromContext(ctx)
	logger.Debug("GetBinding called", "binding", req.Name)

	if err := contextError(ctx); err != nil {
		return nil, err
	}

	return &server.CommonResult{
		Success: false,
		Message: "AI plugin does not provide bindings",
	}, nil
}
