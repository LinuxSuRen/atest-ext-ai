package ai

import (
	"context"
	"strings"
	"testing"

	"github.com/Linuxsuren/atest-ext-ai/internal/config"
	"github.com/Linuxsuren/atest-ext-ai/pkg/models"
)

func TestNewService(t *testing.T) {
	cfg := &config.Config{
		AI: config.AIConfig{
			DefaultModel: "test_model",
		},
	}

	service, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() failed: %v", err)
	}

	if service == nil {
		t.Fatal("NewService() returned nil service")
	}

	// Check that cache is initialized
	if service.cache == nil {
		t.Error("Service cache is nil")
	}

	// Check that clients map is initialized
	if service.clients == nil {
		t.Error("Service clients map is nil")
	}

	// Check that mock client is registered
	if _, exists := service.clients["mock"]; !exists {
		t.Error("Mock client not registered")
	}
}

func TestServiceConvertToSQL(t *testing.T) {
	cfg := &config.Config{
		AI: config.AIConfig{
			DefaultModel: "test_model",
		},
	}

	service, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() failed: %v", err)
	}

	tests := []struct {
		name    string
		request *models.SQLConversionRequest
		wantSQL string
		wantErr bool
	}{
		{
			name: "user query",
			request: &models.SQLConversionRequest{
				Query:   "show me all users",
				Context: "test database",
				Dialect: "mysql",
			},
			wantSQL: "SELECT * FROM users",
			wantErr: false,
		},
		{
			name: "order query",
			request: &models.SQLConversionRequest{
				Query:   "get all orders",
				Context: "test database",
				Dialect: "mysql",
			},
			wantSQL: "SELECT * FROM orders",
			wantErr: false,
		},
		{
			name: "product query",
			request: &models.SQLConversionRequest{
				Query:   "list all products",
				Context: "test database",
				Dialect: "mysql",
			},
			wantSQL: "SELECT * FROM products",
			wantErr: false,
		},
		{
			name: "generic query",
			request: &models.SQLConversionRequest{
				Query:   "some random query",
				Context: "test database",
				Dialect: "mysql",
			},
			wantSQL: "SELECT 1",
			wantErr: false,
		},
		{
			name: "empty query",
			request: &models.SQLConversionRequest{
				Query:   "",
				Context: "test database",
				Dialect: "mysql",
			},
			wantSQL: "SELECT 1",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			response, err := service.ConvertToSQL(ctx, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("Service.ConvertToSQL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if response == nil {
					t.Error("Service.ConvertToSQL() returned nil response")
					return
				}

				if response.SQL != tt.wantSQL {
					t.Errorf("Service.ConvertToSQL() SQL = %v, want %v", response.SQL, tt.wantSQL)
				}

				if !response.Success {
					t.Error("Service.ConvertToSQL() Success should be true")
				}
			}
		})
	}
}

func TestServiceConvertToSQLWithCache(t *testing.T) {
	cfg := &config.Config{
		AI: config.AIConfig{
			DefaultModel: "test_model",
		},
	}

	service, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() failed: %v", err)
	}

	request := &models.SQLConversionRequest{
		Query:   "show me all users",
		Context: "test database",
		Dialect: "mysql",
	}

	ctx := context.Background()

	// First call - should hit the AI client
	response1, err := service.ConvertToSQL(ctx, request)
	if err != nil {
		t.Fatalf("First ConvertToSQL() failed: %v", err)
	}

	// Second call - should hit the cache
	response2, err := service.ConvertToSQL(ctx, request)
	if err != nil {
		t.Fatalf("Second ConvertToSQL() failed: %v", err)
	}

	// Both responses should be identical
	if response1.SQL != response2.SQL {
		t.Errorf("Cached response SQL differs: %v vs %v", response1.SQL, response2.SQL)
	}

	if response1.Success != response2.Success {
		t.Errorf("Cached response Success differs: %v vs %v", response1.Success, response2.Success)
	}
}

func TestMockAIClientConvertToSQL(t *testing.T) {
	client := &MockAIClient{}

	tests := []struct {
		name    string
		request *models.SQLConversionRequest
		wantSQL string
		wantErr bool
	}{
		{
			name: "user keyword",
			request: &models.SQLConversionRequest{
				Query: "show me all users",
			},
			wantSQL: "SELECT * FROM users",
			wantErr: false,
		},
		{
			name: "order keyword",
			request: &models.SQLConversionRequest{
				Query: "get all orders",
			},
			wantSQL: "SELECT * FROM orders",
			wantErr: false,
		},
		{
			name: "product keyword",
			request: &models.SQLConversionRequest{
				Query: "list all products",
			},
			wantSQL: "SELECT * FROM products",
			wantErr: false,
		},
		{
			name: "case insensitive user",
			request: &models.SQLConversionRequest{
				Query: "Show me all USERS",
			},
			wantSQL: "SELECT * FROM users",
			wantErr: false,
		},
		{
			name: "no matching keyword",
			request: &models.SQLConversionRequest{
				Query: "some random query",
			},
			wantSQL: "SELECT 1",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			response, err := client.ConvertToSQL(ctx, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("MockAIClient.ConvertToSQL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if response == nil {
					t.Error("MockAIClient.ConvertToSQL() returned nil response")
					return
				}

				if response.SQL != tt.wantSQL {
					t.Errorf("MockAIClient.ConvertToSQL() SQL = %v, want %v", response.SQL, tt.wantSQL)
				}

				if !response.Success {
					t.Error("MockAIClient.ConvertToSQL() Success should be true")
				}

				if response.Model == "" {
					t.Error("MockAIClient.ConvertToSQL() Model should not be empty")
				}

				if response.Provider == "" {
					t.Error("MockAIClient.ConvertToSQL() Provider should not be empty")
				}
			}
		})
	}
}

func TestServiceConvertToSQLWithUnsupportedProvider(t *testing.T) {
	cfg := &config.Config{
		AI: config.AIConfig{
			DefaultModel: "test_model",
		},
	}

	service, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService() failed: %v", err)
	}

	request := &models.SQLConversionRequest{
		Query:       "show me all users",
		Context:     "test database",
		TableSchema: map[string]string{"users": "id, name, email"},
		Dialect:     "mysql",
	}

	ctx := context.Background()
	response, err := service.ConvertToSQL(ctx, request)

	if err == nil {
		t.Error("Expected error for unsupported provider, got nil")
	}

	if response != nil {
		t.Error("Expected nil response for unsupported provider")
	}

	if !strings.Contains(err.Error(), "unsupported AI provider") {
		t.Errorf("Expected error message to contain 'unsupported AI provider', got: %v", err.Error())
	}
}
