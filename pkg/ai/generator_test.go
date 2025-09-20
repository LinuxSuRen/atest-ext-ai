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

package ai

import (
	"context"
	"testing"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/config"
	"github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
)

// mockGeneratorAIClient is a mock implementation of interfaces.AIClient for generator testing
type mockGeneratorAIClient struct {
	generateFunc     func(ctx context.Context, req *interfaces.GenerateRequest) (*interfaces.GenerateResponse, error)
	capabilitiesFunc func(ctx context.Context) (*interfaces.Capabilities, error)
	healthCheckFunc  func(ctx context.Context) (*interfaces.HealthStatus, error)
	closeFunc        func() error
}

func (m *mockGeneratorAIClient) Generate(ctx context.Context, req *interfaces.GenerateRequest) (*interfaces.GenerateResponse, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, req)
	}
	return &interfaces.GenerateResponse{
		Text:  "SELECT * FROM users WHERE id = 1;",
		Model: "mock-model",
		Usage: interfaces.TokenUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
		ProcessingTime:  100 * time.Millisecond,
		ConfidenceScore: 0.9,
	}, nil
}

func (m *mockGeneratorAIClient) GetCapabilities(ctx context.Context) (*interfaces.Capabilities, error) {
	if m.capabilitiesFunc != nil {
		return m.capabilitiesFunc(ctx)
	}
	return &interfaces.Capabilities{
		Provider: "mock",
		Models: []interfaces.ModelInfo{
			{ID: "mock-model", Name: "Mock Model", MaxTokens: 4096},
		},
		Features: []interfaces.Feature{
			{Name: "text-generation", Enabled: true},
		},
		MaxTokens: 4096,
	}, nil
}

func (m *mockGeneratorAIClient) HealthCheck(ctx context.Context) (*interfaces.HealthStatus, error) {
	if m.healthCheckFunc != nil {
		return m.healthCheckFunc(ctx)
	}
	return &interfaces.HealthStatus{
		Healthy:      true,
		Status:       "OK",
		ResponseTime: 50 * time.Millisecond,
		LastChecked:  time.Now(),
	}, nil
}

func (m *mockGeneratorAIClient) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func TestNewSQLGenerator(t *testing.T) {
	tests := []struct {
		name      string
		aiClient  interfaces.AIClient
		config    config.AIConfig
		wantError bool
	}{
		{
			name:     "valid client and config",
			aiClient: &mockGeneratorAIClient{},
			config: config.AIConfig{
				DefaultService: "mock",
			},
			wantError: false,
		},
		{
			name:      "nil AI client",
			aiClient:  nil,
			config:    config.AIConfig{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator, err := NewSQLGenerator(tt.aiClient, tt.config)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if generator == nil {
				t.Errorf("Expected generator but got nil")
				return
			}

			// Verify capabilities are initialized
			caps := generator.GetCapabilities()
			if caps == nil {
				t.Errorf("Expected capabilities but got nil")
				return
			}

			if len(caps.SupportedDatabases) == 0 {
				t.Errorf("Expected supported databases but got none")
			}

			// Verify SQL dialects are initialized
			if len(generator.sqlDialects) == 0 {
				t.Errorf("Expected SQL dialects but got none")
			}
		})
	}
}

func TestSQLGenerator_Generate(t *testing.T) {
	mockClient := &mockGeneratorAIClient{
		generateFunc: func(ctx context.Context, req *interfaces.GenerateRequest) (*interfaces.GenerateResponse, error) {
			// Return a mock SQL response based on the request
			return &interfaces.GenerateResponse{
				Text:  "SELECT name, email FROM users WHERE age > 18;",
				Model: "mock-model",
				Usage: interfaces.TokenUsage{
					PromptTokens:     25,
					CompletionTokens: 15,
					TotalTokens:      40,
				},
				ProcessingTime:  150 * time.Millisecond,
				ConfidenceScore: 0.85,
			}, nil
		},
	}

	generator, err := NewSQLGenerator(mockClient, config.AIConfig{})
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	tests := []struct {
		name         string
		naturalLang  string
		options      *GenerateOptions
		wantError    bool
		expectedSQL  string
		expectedConf float64
	}{
		{
			name:        "basic query",
			naturalLang: "Get all users older than 18",
			options: &GenerateOptions{
				DatabaseType:       "mysql",
				ValidateSQL:        false,
				OptimizeQuery:      false,
				IncludeExplanation: true,
				SafetyMode:         true,
			},
			wantError:    false,
			expectedSQL:  "SELECT name, email FROM users WHERE age > 18;",
			expectedConf: 0.8,
		},
		{
			name:        "empty natural language",
			naturalLang: "",
			options: &GenerateOptions{
				DatabaseType: "mysql",
			},
			wantError: true,
		},
		{
			name:        "unsupported database",
			naturalLang: "Get all users",
			options: &GenerateOptions{
				DatabaseType: "oracle",
			},
			wantError: true,
		},
		{
			name:        "nil options (should use defaults)",
			naturalLang: "Get all users",
			options:     nil,
			wantError:   false,
		},
		{
			name:        "with schema context",
			naturalLang: "Get user information",
			options: &GenerateOptions{
				DatabaseType: "postgresql",
				Schema: map[string]Table{
					"users": {
						Name: "users",
						Columns: []Column{
							{Name: "id", Type: "INTEGER", Nullable: false},
							{Name: "name", Type: "VARCHAR", MaxLength: 100},
							{Name: "email", Type: "VARCHAR", MaxLength: 255},
							{Name: "created_at", Type: "TIMESTAMP"},
						},
						PrimaryKey: []string{"id"},
					},
				},
				Context:            []string{"Include email in results"},
				ValidateSQL:        true,
				OptimizeQuery:      false,
				IncludeExplanation: true,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := generator.Generate(ctx, tt.naturalLang, tt.options)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("Expected result but got nil")
				return
			}

			// Verify SQL is not empty
			if result.SQL == "" {
				t.Errorf("Expected SQL but got empty string")
			}

			// Verify confidence score is reasonable
			if result.ConfidenceScore < 0 || result.ConfidenceScore > 1 {
				t.Errorf("Invalid confidence score: %f", result.ConfidenceScore)
			}

			// Verify metadata
			if result.Metadata.RequestID == "" {
				t.Errorf("Expected request ID but got empty string")
			}

			if result.Metadata.ProcessingTime <= 0 {
				t.Errorf("Expected positive processing time but got: %v", result.Metadata.ProcessingTime)
			}

			// If specific SQL is expected, check it
			if tt.expectedSQL != "" && result.SQL != tt.expectedSQL {
				t.Errorf("Expected SQL: %s, got: %s", tt.expectedSQL, result.SQL)
			}

			// If specific confidence is expected, check it's within range
			if tt.expectedConf > 0 {
				if result.ConfidenceScore < tt.expectedConf-0.1 || result.ConfidenceScore > tt.expectedConf+0.1 {
					t.Errorf("Expected confidence around %f, got: %f", tt.expectedConf, result.ConfidenceScore)
				}
			}
		})
	}
}

func TestSQLGenerator_DetectQueryType(t *testing.T) {
	generator := &SQLGenerator{}

	tests := []struct {
		name     string
		sql      string
		expected string
	}{
		{
			name:     "SELECT query",
			sql:      "SELECT * FROM users",
			expected: "SELECT",
		},
		{
			name:     "INSERT query",
			sql:      "INSERT INTO users (name) VALUES ('test')",
			expected: "INSERT",
		},
		{
			name:     "UPDATE query",
			sql:      "UPDATE users SET name = 'test' WHERE id = 1",
			expected: "UPDATE",
		},
		{
			name:     "DELETE query",
			sql:      "DELETE FROM users WHERE id = 1",
			expected: "DELETE",
		},
		{
			name:     "CREATE query",
			sql:      "CREATE TABLE test (id INT)",
			expected: "CREATE",
		},
		{
			name:     "DROP query",
			sql:      "DROP TABLE test",
			expected: "DROP",
		},
		{
			name:     "ALTER query",
			sql:      "ALTER TABLE test ADD COLUMN name VARCHAR(50)",
			expected: "ALTER",
		},
		{
			name:     "lowercase SELECT",
			sql:      "select * from users",
			expected: "SELECT",
		},
		{
			name:     "unknown query",
			sql:      "SHOW TABLES",
			expected: "UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.detectQueryType(tt.sql)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestSQLGenerator_ExtractTableNames(t *testing.T) {
	generator := &SQLGenerator{}

	tests := []struct {
		name     string
		sql      string
		expected []string
	}{
		{
			name:     "simple SELECT",
			sql:      "SELECT * FROM users",
			expected: []string{"USERS"},
		},
		{
			name:     "JOIN query",
			sql:      "SELECT * FROM users JOIN orders ON users.id = orders.user_id",
			expected: []string{"USERS", "ORDERS"},
		},
		{
			name:     "INSERT query",
			sql:      "INSERT INTO products (name) VALUES ('test')",
			expected: []string{"PRODUCTS"},
		},
		{
			name:     "UPDATE query",
			sql:      "UPDATE customers SET name = 'test' WHERE id = 1",
			expected: []string{"CUSTOMERS"},
		},
		{
			name:     "multiple tables",
			sql:      "SELECT u.name, o.total FROM users u JOIN orders o ON u.id = o.user_id JOIN products p ON o.product_id = p.id",
			expected: []string{"USERS", "ORDERS", "PRODUCTS"},
		},
		{
			name:     "no tables",
			sql:      "SELECT 1 as test",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.extractTableNames(tt.sql)

			// Convert expected to map for easy comparison (order doesn't matter)
			expectedMap := make(map[string]bool)
			for _, table := range tt.expected {
				expectedMap[table] = true
			}

			// Check that all expected tables are found
			for _, table := range tt.expected {
				found := false
				for _, resultTable := range result {
					if resultTable == table {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected table %s not found in result: %v", table, result)
				}
			}

			// Check that no unexpected tables are found
			for _, resultTable := range result {
				if !expectedMap[resultTable] {
					t.Errorf("Unexpected table %s found in result: %v", resultTable, result)
				}
			}
		})
	}
}

func TestSQLGenerator_AssessComplexity(t *testing.T) {
	generator := &SQLGenerator{}

	tests := []struct {
		name     string
		sql      string
		expected string
	}{
		{
			name:     "simple SELECT",
			sql:      "SELECT * FROM users",
			expected: "simple",
		},
		{
			name:     "SELECT with WHERE",
			sql:      "SELECT * FROM users WHERE age > 18",
			expected: "simple",
		},
		{
			name:     "JOIN query",
			sql:      "SELECT * FROM users JOIN orders ON users.id = orders.user_id",
			expected: "moderate",
		},
		{
			name:     "GROUP BY query",
			sql:      "SELECT country, COUNT(*) FROM users GROUP BY country",
			expected: "moderate",
		},
		{
			name:     "complex query with multiple features",
			sql:      "SELECT u.name, COUNT(o.id) FROM users u LEFT JOIN orders o ON u.id = o.user_id GROUP BY u.name HAVING COUNT(o.id) > 5",
			expected: "complex",
		},
		{
			name:     "very complex query",
			sql:      "WITH user_stats AS (SELECT user_id, COUNT(*) as order_count FROM orders GROUP BY user_id) SELECT u.name, us.order_count, SUM(o.total) OVER (PARTITION BY u.country) FROM users u JOIN user_stats us ON u.id = us.user_id JOIN orders o ON u.id = o.user_id",
			expected: "very_complex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.assessComplexity(tt.sql)
			if result != tt.expected {
				t.Errorf("Expected complexity %s, got %s for SQL: %s", tt.expected, result, tt.sql)
			}
		})
	}
}

func TestSQLGenerator_GetCapabilities(t *testing.T) {
	mockClient := &mockGeneratorAIClient{}
	generator, err := NewSQLGenerator(mockClient, config.AIConfig{})
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	capabilities := generator.GetCapabilities()

	if capabilities == nil {
		t.Errorf("Expected capabilities but got nil")
		return
	}

	expectedDatabases := []string{"mysql", "postgresql", "sqlite"}
	for _, expected := range expectedDatabases {
		found := false
		for _, supported := range capabilities.SupportedDatabases {
			if supported == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected database %s not found in supported databases", expected)
		}
	}

	if len(capabilities.Features) == 0 {
		t.Errorf("Expected features but got none")
	}

	// Verify key features are present
	expectedFeatures := []string{"Natural Language to SQL", "Multi-dialect Support", "Schema-aware Generation"}
	for _, expectedFeature := range expectedFeatures {
		found := false
		for _, feature := range capabilities.Features {
			if feature.Name == expectedFeature && feature.Enabled {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected feature %s not found or not enabled", expectedFeature)
		}
	}
}

func TestSQLGenerator_Integration(t *testing.T) {
	// Integration test that tests the full workflow
	mockClient := &mockGeneratorAIClient{
		generateFunc: func(ctx context.Context, req *interfaces.GenerateRequest) (*interfaces.GenerateResponse, error) {
			// Simulate AI response with JSON-like structured SQL
			response := "SELECT users.name, users.email, COUNT(orders.id) as order_count FROM users LEFT JOIN orders ON users.id = orders.user_id WHERE users.age >= 18 GROUP BY users.id, users.name, users.email ORDER BY order_count DESC LIMIT 10;"

			return &interfaces.GenerateResponse{
				Text:  response,
				Model: "mock-advanced-model",
				Usage: interfaces.TokenUsage{
					PromptTokens:     50,
					CompletionTokens: 30,
					TotalTokens:      80,
				},
				ProcessingTime:  200 * time.Millisecond,
				ConfidenceScore: 0.92,
			}, nil
		},
	}

	generator, err := NewSQLGenerator(mockClient, config.AIConfig{
		DefaultService: "mock",
	})
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	// Define a realistic schema
	schema := map[string]Table{
		"users": {
			Name: "users",
			Columns: []Column{
				{Name: "id", Type: "INT", Nullable: false},
				{Name: "name", Type: "VARCHAR", MaxLength: 100, Nullable: false},
				{Name: "email", Type: "VARCHAR", MaxLength: 255, Nullable: false},
				{Name: "age", Type: "INT", Nullable: true},
				{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			},
			PrimaryKey: []string{"id"},
		},
		"orders": {
			Name: "orders",
			Columns: []Column{
				{Name: "id", Type: "INT", Nullable: false},
				{Name: "user_id", Type: "INT", Nullable: false},
				{Name: "total", Type: "DECIMAL", Precision: 10, Scale: 2, Nullable: false},
				{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			},
			PrimaryKey:  []string{"id"},
			ForeignKeys: []ForeignKey{{Name: "fk_user", Columns: []string{"user_id"}, ReferencedTable: "users", ReferencedColumns: []string{"id"}}},
		},
	}

	options := &GenerateOptions{
		DatabaseType:       "mysql",
		Schema:             schema,
		Context:            []string{"Include order statistics", "Focus on adult users"},
		ValidateSQL:        true,
		OptimizeQuery:      true,
		IncludeExplanation: true,
		SafetyMode:         true,
		Temperature:        0.2,
		MaxTokens:          1500,
	}

	ctx := context.Background()
	result, err := generator.Generate(ctx, "Show me the top 10 adult users with the most orders, including their order count", options)

	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify the result has all expected components
	if result.SQL == "" {
		t.Errorf("Expected SQL but got empty string")
	}

	if result.Explanation == "" {
		t.Errorf("Expected explanation but got empty string")
	}

	if result.ConfidenceScore <= 0 {
		t.Errorf("Expected positive confidence score, got: %f", result.ConfidenceScore)
	}

	if result.Metadata.RequestID == "" {
		t.Errorf("Expected request ID")
	}

	if result.Metadata.ProcessingTime <= 0 {
		t.Errorf("Expected positive processing time")
	}

	if result.Metadata.QueryType != "SELECT" {
		t.Errorf("Expected SELECT query type, got: %s", result.Metadata.QueryType)
	}

	if result.Metadata.Complexity == "" {
		t.Errorf("Expected complexity assessment")
	}

	// Verify tables involved
	expectedTables := []string{"USERS", "ORDERS"}
	for _, expected := range expectedTables {
		found := false
		for _, table := range result.Metadata.TablesInvolved {
			if table == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected table %s not found in tables involved: %v", expected, result.Metadata.TablesInvolved)
		}
	}

	t.Logf("Generated SQL: %s", result.SQL)
	t.Logf("Explanation: %s", result.Explanation)
	t.Logf("Confidence: %.2f", result.ConfidenceScore)
	t.Logf("Complexity: %s", result.Metadata.Complexity)
	t.Logf("Processing time: %v", result.Metadata.ProcessingTime)
}
