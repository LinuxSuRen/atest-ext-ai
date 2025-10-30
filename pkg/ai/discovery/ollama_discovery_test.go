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

// TestOllamaDiscovery tests Ollama service discovery functionality
func TestOllamaDiscovery(t *testing.T) {
	discovery := NewOllamaDiscovery("http://localhost:11434")
	ctx := context.Background()

	// Test IsAvailable - this is a connectivity test, may fail if Ollama is not running
	available := discovery.IsAvailable(ctx)
	if !available {
		t.Log("Ollama is not available - this is expected if Ollama is not running")
	} else {
		fmt.Println("Ollama service is available")
	}

	// Test GetBaseURL
	baseURL := discovery.GetBaseURL()
	if baseURL != "http://localhost:11434" {
		t.Errorf("Expected base URL 'http://localhost:11434', got '%s'", baseURL)
	}
}

// TestOllamaDiscoveryWithCustomEndpoint tests discovery with custom endpoint
func TestOllamaDiscoveryWithCustomEndpoint(t *testing.T) {
	customEndpoint := "http://custom-host:8080"
	discovery := NewOllamaDiscovery(customEndpoint)

	baseURL := discovery.GetBaseURL()
	if baseURL != customEndpoint {
		t.Errorf("Expected base URL '%s', got '%s'", customEndpoint, baseURL)
	}
}
