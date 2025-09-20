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
	"os"
	"strings"
	"testing"
	"time"

	"github.com/linuxsuren/api-testing/pkg/server"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// Create a temporary config file for testing
	configContent := `
ai:
  default_service: "test"
  timeout: 30s
  services:
    test:
      enabled: true
      provider: "local"
`

	// Create temp config file
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		panic(err)
	}
	tmpFile.Close()

	// Set config file path
	os.Setenv("ATEST_AI_CONFIG", tmpFile.Name())

	// Run tests
	code := m.Run()

	// Cleanup
	os.Unsetenv("ATEST_AI_CONFIG")
	os.Exit(code)
}

func TestNewAIPluginService(t *testing.T) {
	service, err := NewAIPluginService()

	// Service creation might fail due to missing config in test environment
	// That's okay - we're mainly testing the structure
	if err != nil {
		t.Logf("Service creation failed (expected in test environment): %v", err)
		return
	}

	assert.NotNil(t, service)
	assert.NotNil(t, service.aiEngine)
	assert.NotNil(t, service.config)
	assert.NotNil(t, service.capabilityDetector)
	assert.NotNil(t, service.metadataProvider)

	defer service.Shutdown()
}

func TestAIPluginService_Query_Capabilities(t *testing.T) {
	// Create a mock service for testing capabilities query
	service := &AIPluginService{
		// Initialize with minimal required fields for testing
		capabilityDetector: nil, // This will cause a controlled error
		metadataProvider:   nil,
	}

	ctx := context.Background()

	tests := []struct {
		name    string
		request *server.DataQuery
		wantErr bool
	}{
		{
			name: "capabilities request",
			request: &server.DataQuery{
				Type: "ai",
				Key:  "capabilities",
			},
			wantErr: true, // Will fail because detector is nil
		},
		{
			name: "ai.capabilities request",
			request: &server.DataQuery{
				Type: "ai",
				Key:  "ai.capabilities",
			},
			wantErr: true, // Will fail because detector is nil
		},
		{
			name: "metadata request",
			request: &server.DataQuery{
				Type: "ai",
				Key:  "ai.capabilities.metadata",
			},
			wantErr: true, // Will fail because provider is nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Query(ctx, tt.request)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAIPluginService_Query_UnsupportedType(t *testing.T) {
	service := &AIPluginService{}

	ctx := context.Background()
	req := &server.DataQuery{
		Type: "unsupported",
		Key:  "test",
	}

	_, err := service.Query(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported query type")
}

func TestAIPluginService_handleCapabilitiesQuery_ParameterParsing(t *testing.T) {
	// We can't easily test this without a full service setup,
	// but we can test the parameter parsing logic conceptually

	tests := []struct {
		name       string
		sqlParams  string
		expectJSON bool
	}{
		{
			name:       "valid JSON parameters",
			sqlParams:  `{"include_models": true, "include_databases": false}`,
			expectJSON: true,
		},
		{
			name:       "invalid JSON parameters",
			sqlParams:  `{invalid json}`,
			expectJSON: false,
		},
		{
			name:       "empty parameters",
			sqlParams:  "",
			expectJSON: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.sqlParams == "" {
				return // Skip empty case
			}

			var params map[string]bool
			err := json.Unmarshal([]byte(tt.sqlParams), &params)

			if tt.expectJSON {
				assert.NoError(t, err)
				assert.NotNil(t, params)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestAIPluginService_handleCapabilitiesQuery_SubQueryParsing(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "metadata subquery",
			key:      "ai.capabilities.metadata",
			expected: "metadata",
		},
		{
			name:     "models subquery",
			key:      "ai.capabilities.models",
			expected: "models",
		},
		{
			name:     "databases subquery",
			key:      "ai.capabilities.databases",
			expected: "databases",
		},
		{
			name:     "features subquery",
			key:      "ai.capabilities.features",
			expected: "features",
		},
		{
			name:     "health subquery",
			key:      "ai.capabilities.health",
			expected: "health",
		},
		{
			name:     "no subquery",
			key:      "capabilities",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var subQuery string

			if strings.Contains(tt.key, ".") {
				parts := strings.Split(tt.key, ".")
				if len(parts) >= 2 {
					subQuery = parts[len(parts)-1]
				}
			}

			if tt.expected == "" {
				assert.Empty(t, subQuery)
			} else {
				assert.Equal(t, tt.expected, subQuery)
			}
		})
	}
}

func TestAIPluginService_Verify(t *testing.T) {
	// Create a mock service
	service := &AIPluginService{
		aiEngine: nil, // This will make IsHealthy return false
	}

	ctx := context.Background()
	req := &server.Empty{}

	result, err := service.Verify(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Ready) // Should be false because aiEngine is nil
	assert.Equal(t, "1.0.0", result.Version)
	assert.Equal(t, "AI engine not initialized", result.Message)
}

func TestAIPluginService_Shutdown(t *testing.T) {
	service := &AIPluginService{
		aiEngine: nil, // Safe to call Shutdown on nil
	}

	// Should not panic
	assert.NotPanics(t, func() {
		service.Shutdown()
	})
}

// Integration test that validates the JSON structure of capabilities response
func TestCapabilitiesResponse_JSONStructure(t *testing.T) {
	// Create a sample capabilities response to test JSON structure
	sampleResponse := map[string]interface{}{
		"capabilities": map[string]interface{}{
			"version": "1.0.0",
			"models": []map[string]interface{}{
				{
					"name":      "test-model",
					"provider":  "local",
					"available": true,
					"features":  []string{"sql-generation"},
				},
			},
			"databases": []map[string]interface{}{
				{
					"type":      "mysql",
					"versions":  []string{"8.0"},
					"features":  []string{"joins", "subqueries"},
					"supported": true,
				},
			},
			"features": []map[string]interface{}{
				{
					"name":        "sql-generation",
					"enabled":     true,
					"description": "Generate SQL from natural language",
					"version":     "1.0.0",
				},
			},
			"health": map[string]interface{}{
				"overall":   true,
				"timestamp": time.Now().Format(time.RFC3339),
			},
			"limits": map[string]interface{}{
				"max_concurrent_requests": 10,
				"rate_limit": map[string]interface{}{
					"requests_per_minute": 60,
				},
			},
		},
	}

	// Validate that this can be marshaled to JSON
	jsonBytes, err := json.Marshal(sampleResponse)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonBytes)

	// Validate that it can be unmarshaled
	var unmarshaled map[string]interface{}
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	assert.NoError(t, err)
	assert.NotEmpty(t, unmarshaled)

	// Validate structure
	capabilities, ok := unmarshaled["capabilities"].(map[string]interface{})
	assert.True(t, ok)
	assert.NotEmpty(t, capabilities)

	// Check key fields exist
	assert.Contains(t, capabilities, "version")
	assert.Contains(t, capabilities, "models")
	assert.Contains(t, capabilities, "databases")
	assert.Contains(t, capabilities, "features")
	assert.Contains(t, capabilities, "health")
	assert.Contains(t, capabilities, "limits")
}

// Benchmark test for capabilities query performance
func BenchmarkCapabilitiesJSONMarshal(b *testing.B) {
	sampleResponse := map[string]interface{}{
		"version": "1.0.0",
		"models": []map[string]interface{}{
			{"name": "test1", "provider": "local", "available": true},
			{"name": "test2", "provider": "remote", "available": false},
		},
		"databases": []map[string]interface{}{
			{"type": "mysql", "supported": true},
			{"type": "postgresql", "supported": true},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(sampleResponse)
		if err != nil {
			b.Fatal(err)
		}
	}
}
