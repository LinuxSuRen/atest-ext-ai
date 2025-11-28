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
	"fmt"
	"sort"
	"strings"

	"github.com/linuxsuren/atest-ext-ai/pkg/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// InitializationError captures detailed context about component initialization failures
// This allows us to provide comprehensive diagnostic information in error responses.
type InitializationError struct {
	Component string            // Component that failed (e.g., "AI Engine", "AI Manager")
	Reason    string            // Error message explaining the failure
	Details   map[string]string // Additional diagnostic information
}

// Global initialization error tracking for enhanced error messages.
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

func formatInitErrors(filter func(InitializationError) bool) string {
	if len(initErrors) == 0 {
		return ""
	}

	var builder strings.Builder
	for _, initErr := range initErrors {
		if filter != nil && !filter(initErr) {
			continue
		}
		if builder.Len() == 0 {
			builder.WriteString(" Initialization errors:")
		}
		builder.WriteString(fmt.Sprintf("\n- %s: %s", initErr.Component, initErr.Reason))
		if len(initErr.Details) > 0 {
			keys := make([]string, 0, len(initErr.Details))
			for key := range initErr.Details {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				builder.WriteString(fmt.Sprintf("\n  %s: %s", key, initErr.Details[key]))
			}
		}
	}

	return builder.String()
}

func (s *AIPluginService) requireEngineAvailable(operation, baseMessage, fallback string) error {
	if s.aiEngine != nil {
		return nil
	}

	logging.Logger.Error(operation)
	errMsg := baseMessage
	if details := formatInitErrors(nil); details != "" {
		errMsg += details
	} else if fallback != "" {
		errMsg += " " + fallback
	}
	return status.Error(codes.FailedPrecondition, errMsg)
}

const managerFallbackMessage = "Please check AI service configuration."

func (s *AIPluginService) requireManagerAvailable(operation, baseMessage string) error {
	if s.aiManager != nil {
		return nil
	}

	logging.Logger.Error(operation)
	details := formatInitErrors(func(initErr InitializationError) bool {
		return initErr.Component == "AI Manager"
	})
	errMsg := baseMessage
	if details != "" {
		errMsg += details
	} else {
		errMsg += " " + managerFallbackMessage
	}
	return status.Error(codes.FailedPrecondition, errMsg)
}
