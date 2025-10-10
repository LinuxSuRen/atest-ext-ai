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

package discovery

import (
	"context"
	"fmt"
	"testing"
)

// TestOllamaDiscovery tests Ollama auto-discovery functionality
func TestOllamaDiscovery(t *testing.T) {
	// Skip if Ollama is not available
	discovery := NewOllamaDiscovery("http://localhost:11434")
	ctx := context.Background()

	if !discovery.IsAvailable(ctx) {
		t.Skip("Ollama is not available, skipping test")
	}

	// Test getting models
	models, err := discovery.GetModels(ctx)
	if err != nil {
		t.Errorf("Failed to get models: %v", err)
		return
	}

	fmt.Printf("Found %d Ollama models:\n", len(models))
	for _, model := range models {
		fmt.Printf("  - %s: %s\n", model.ID, model.Description)
	}
}
