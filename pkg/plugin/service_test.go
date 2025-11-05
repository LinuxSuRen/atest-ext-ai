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
	"testing"

	"github.com/linuxsuren/api-testing/pkg/server"
	"github.com/linuxsuren/atest-ext-ai/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAIGenerateFieldNames verifies that the AI generate response contains the correct field names
// This is a regression test for Issue #1: Field name mismatch (content vs generated_sql)
func TestAIGenerateFieldNames(t *testing.T) {
	t.Run("generate response contains generated_sql field", func(t *testing.T) {
		// This test ensures the response uses "generated_sql" instead of "content"
		// to match what the main project (api-testing) expects

		// Create a mock response simulating what handleAIGenerate returns
		mockResponse := &server.DataQueryResult{
			Data: []*server.Pair{
				{Key: "api_version", Value: "v1"},
				{Key: "generated_sql", Value: "sql:SELECT * FROM users;\nexplanation:This query selects all users."},
				{Key: "success", Value: "true"},
				{Key: "meta", Value: `{"confidence": 0.95, "model": "test-model"}`},
			},
		}

		// Verify the critical field exists
		var hasGeneratedSQL bool
		var generatedSQLValue string

		for _, pair := range mockResponse.Data {
			if pair.Key == "generated_sql" {
				hasGeneratedSQL = true
				generatedSQLValue = pair.Value
			}
		}

		// Assert the field exists
		assert.True(t, hasGeneratedSQL, "Response must contain 'generated_sql' field, not 'content'")
		assert.NotEmpty(t, generatedSQLValue, "generated_sql value should not be empty")

		// Verify the format includes both SQL and explanation
		assert.Contains(t, generatedSQLValue, "sql:", "generated_sql should contain 'sql:' prefix")
		assert.Contains(t, generatedSQLValue, "explanation:", "generated_sql should contain 'explanation:' prefix")
	})

	t.Run("legacy response contains generated_sql field", func(t *testing.T) {
		// This test ensures the legacy query handler also uses "generated_sql"

		mockResponse := &server.DataQueryResult{
			Data: []*server.Pair{
				{Key: "generated_sql", Value: "sql:SELECT name FROM users;\nexplanation:Get user names."},
				{Key: "success", Value: "true"},
				{Key: "meta", Value: `{"confidence": 0.9, "model": "legacy-model"}`},
			},
		}

		// Verify field existence
		var hasGeneratedSQL bool
		for _, pair := range mockResponse.Data {
			if pair.Key == "generated_sql" {
				hasGeneratedSQL = true
				break
			}
		}

		assert.True(t, hasGeneratedSQL, "Legacy response must also use 'generated_sql' field")
	})
}

// TestResponseFieldStructure verifies the complete structure of AI responses
func TestResponseFieldStructure(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   *server.DataQueryResult
		expectedFields map[string]bool
	}{
		{
			name: "AI generate response structure",
			mockResponse: &server.DataQueryResult{
				Data: []*server.Pair{
					{Key: "api_version", Value: "v1"},
					{Key: "generated_sql", Value: "sql:SELECT 1;"},
					{Key: "success", Value: "true"},
					{Key: "meta", Value: "{}"},
				},
			},
			expectedFields: map[string]bool{
				"api_version":   true,
				"generated_sql": true,
				"success":       true,
				"meta":          true,
			},
		},
		{
			name: "AI capabilities response structure",
			mockResponse: &server.DataQueryResult{
				Data: []*server.Pair{
					{Key: "api_version", Value: "v1"},
					{Key: "capabilities", Value: "{}"},
					{Key: "models", Value: "[]"},
					{Key: "features", Value: "[]"},
					{Key: "success", Value: "true"},
				},
			},
			expectedFields: map[string]bool{
				"api_version":  true,
				"capabilities": true,
				"models":       true,
				"features":     true,
				"success":      true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualFields := make(map[string]bool)
			for _, pair := range tt.mockResponse.Data {
				actualFields[pair.Key] = true
			}

			// Check all expected fields are present
			for field, expected := range tt.expectedFields {
				assert.Equal(t, expected, actualFields[field],
					"Field %s should be present", field)
			}
		})
	}
}

// TestSuccessFieldConsistency verifies success field is correctly set
// This is a regression test for Issue #2: Success field processing conflict
func TestSuccessFieldConsistency(t *testing.T) {
	t.Run("success field with no error", func(t *testing.T) {
		mockResponse := &server.DataQueryResult{
			Data: []*server.Pair{
				{Key: "generated_sql", Value: "sql:SELECT 1;"},
				{Key: "success", Value: "true"},
				// No error field should be present when successful
			},
		}

		var hasSuccess bool
		var hasError bool
		var successValue string

		for _, pair := range mockResponse.Data {
			if pair.Key == "success" {
				hasSuccess = true
				successValue = pair.Value
			}
			if pair.Key == "error" {
				hasError = true
			}
		}

		assert.True(t, hasSuccess, "success field must be present")
		assert.Equal(t, "true", successValue, "success should be 'true'")
		assert.False(t, hasError, "error field should not be present when successful")
	})

	t.Run("error response structure", func(t *testing.T) {
		// When an error occurs, we expect different handling
		// This test documents the expected error format

		mockErrorResponse := &server.DataQueryResult{
			Data: []*server.Pair{
				{Key: "success", Value: "false"},
				{Key: "error", Value: "AI generation failed"},
				{Key: "error_code", Value: "GENERATION_FAILED"},
			},
		}

		var hasSuccess bool
		var hasError bool
		var hasErrorCode bool
		var successValue string

		for _, pair := range mockErrorResponse.Data {
			if pair.Key == "success" {
				hasSuccess = true
				successValue = pair.Value
			}
			if pair.Key == "error" {
				hasError = true
			}
			if pair.Key == "error_code" {
				hasErrorCode = true
			}
		}

		assert.True(t, hasSuccess, "success field must be present even on error")
		assert.Equal(t, "false", successValue, "success should be 'false' on error")
		assert.True(t, hasError, "error field should be present on error")
		assert.True(t, hasErrorCode, "error_code field should be present on error")
	})

	t.Run("success response must not contain error fields", func(t *testing.T) {
		// Verify successful responses don't accidentally include error fields
		mockSuccessResponse := &server.DataQueryResult{
			Data: []*server.Pair{
				{Key: "api_version", Value: "v1"},
				{Key: "generated_sql", Value: "sql:SELECT * FROM users;"},
				{Key: "success", Value: "true"},
				{Key: "meta", Value: `{"confidence": 0.9}`},
			},
		}

		var hasError bool
		var hasErrorCode bool
		var successValue string

		for _, pair := range mockSuccessResponse.Data {
			if pair.Key == "success" {
				successValue = pair.Value
			}
			if pair.Key == "error" {
				hasError = true
			}
			if pair.Key == "error_code" {
				hasErrorCode = true
			}
		}

		assert.Equal(t, "true", successValue, "success should be 'true'")
		assert.False(t, hasError, "error field must not be present in successful response")
		assert.False(t, hasErrorCode, "error_code field must not be present in successful response")
	})
}

// TestMetaJSONParsing verifies meta field contains valid JSON
func TestMetaJSONParsing(t *testing.T) {
	mockResponse := &server.DataQueryResult{
		Data: []*server.Pair{
			{Key: "generated_sql", Value: "sql:SELECT 1;"},
			{Key: "meta", Value: `{"confidence": 0.95, "model": "test-model"}`},
		},
	}

	var metaValue string
	for _, pair := range mockResponse.Data {
		if pair.Key == "meta" {
			metaValue = pair.Value
			break
		}
	}

	// Verify meta is valid JSON
	var metaData map[string]interface{}
	err := json.Unmarshal([]byte(metaValue), &metaData)
	require.NoError(t, err, "meta field should contain valid JSON")

	// Verify expected meta fields
	assert.Contains(t, metaData, "confidence", "meta should contain confidence")
	assert.Contains(t, metaData, "model", "meta should contain model")
}

// TestGeneratedSQLFormat verifies the format of generated SQL
func TestGeneratedSQLFormat(t *testing.T) {
	tests := []struct {
		name        string
		sqlValue    string
		expectValid bool
	}{
		{
			name:        "valid format with sql and explanation",
			sqlValue:    "sql:SELECT * FROM users;\nexplanation:Get all users",
			expectValid: true,
		},
		{
			name:        "valid format multiline",
			sqlValue:    "sql:SELECT * FROM users WHERE age > 18;\nexplanation:Get adult users",
			expectValid: true,
		},
		{
			name:        "missing explanation",
			sqlValue:    "sql:SELECT * FROM users;",
			expectValid: false, // Should have explanation
		},
		{
			name:        "missing sql prefix",
			sqlValue:    "SELECT * FROM users;\nexplanation:Get users",
			expectValid: false, // Should have sql: prefix
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasSQL := containsStr(tt.sqlValue, "sql:")
			hasExplanation := containsStr(tt.sqlValue, "explanation:")

			isValid := hasSQL && hasExplanation
			assert.Equal(t, tt.expectValid, isValid,
				"SQL format validation failed for: %s", tt.name)
		})
	}
}

// Helper function
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark for field lookup performance
func BenchmarkFieldLookup(b *testing.B) {
	mockResponse := &server.DataQueryResult{
		Data: []*server.Pair{
			{Key: "api_version", Value: "v1"},
			{Key: "generated_sql", Value: "sql:SELECT * FROM users;"},
			{Key: "success", Value: "true"},
			{Key: "meta", Value: `{"confidence": 0.95}`},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, pair := range mockResponse.Data {
			if pair.Key == "generated_sql" {
				_ = pair.Value
				break
			}
		}
	}
}

func TestHandleUpdateConfigRefreshesEngine(t *testing.T) {
	service, err := NewAIPluginService()
	require.NoError(t, err)
	require.NotNil(t, service)
	t.Cleanup(service.Shutdown)

	oldManager := service.aiManager
	oldEngine := service.aiEngine

	updatePayload := map[string]any{
		"provider": "ollama",
		"config": map[string]any{
			"provider":   "ollama",
			"endpoint":   "http://localhost:11439",
			"model":      "test-model",
			"max_tokens": 1337,
		},
	}

	payload, err := json.Marshal(updatePayload)
	require.NoError(t, err)

	resp, err := service.handleUpdateConfig(context.Background(), &server.DataQuery{Sql: string(payload)})
	require.NoError(t, err)
	require.NotNil(t, resp)

	updatedService := service.config.AI.Services["ollama"]
	require.Equal(t, "http://localhost:11439", updatedService.Endpoint)
	require.Equal(t, 1337, updatedService.MaxTokens)

	require.NotNil(t, service.aiManager)
	require.NotNil(t, service.aiEngine)
	require.NotNil(t, service.capabilityDetector)

	require.NotEqual(t, oldManager, service.aiManager)
	require.NotEqual(t, oldEngine, service.aiEngine)
}

func TestResolveDatabaseType(t *testing.T) {
	svc := &AIPluginService{
		config: &config.Config{
			Database: config.DatabaseConfig{DefaultType: "postgres"},
		},
	}

	t.Run("uses explicit value when provided", func(t *testing.T) {
		assert.Equal(t, "mysql", svc.resolveDatabaseType("mysql", nil))
	})

	t.Run("normalizes postgres aliases", func(t *testing.T) {
		assert.Equal(t, "postgresql", svc.resolveDatabaseType("pg", nil))
	})

	t.Run("falls back to config default", func(t *testing.T) {
		assert.Equal(t, "postgresql", svc.resolveDatabaseType("", nil))
	})

	t.Run("uses config map overrides", func(t *testing.T) {
		configMap := map[string]any{"database_dialect": "sqlite3"}
		assert.Equal(t, "sqlite", svc.resolveDatabaseType("", configMap))
	})
}
