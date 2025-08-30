package server

import (
	"context"
	"testing"

	"github.com/Linuxsuren/atest-ext-ai/internal/ai"
	"github.com/Linuxsuren/atest-ext-ai/internal/config"
	pb "github.com/linuxsuren/api-testing/pkg/server"
)

func TestNewAIPluginServer(t *testing.T) {
	cfg := &config.Config{
		AI: config.AIConfig{
			DefaultModel: "test_model",
		},
	}

	aiService, err := ai.NewService(cfg)
	if err != nil {
		t.Fatalf("Failed to create AI service: %v", err)
	}

	server := NewAIPluginServer(cfg, aiService)
	if server == nil {
		t.Fatal("NewAIPluginServer() returned nil")
	}

	if server.aiService == nil {
		t.Error("AIPluginServer.aiService is nil")
	}
}

func TestAIPluginServerRun_ConvertToSQL(t *testing.T) {
	cfg := &config.Config{
		AI: config.AIConfig{
			DefaultModel: "test_model",
		},
	}

	aiService, err := ai.NewService(cfg)
	if err != nil {
		t.Fatalf("Failed to create AI service: %v", err)
	}

	server := NewAIPluginServer(cfg, aiService)

	tests := []struct {
		name    string
		request *pb.TestSuiteWithCase
		wantErr bool
	}{
		{
			name: "convert to sql request",
			request: &pb.TestSuiteWithCase{
				Suite: &pb.TestSuite{
					Name: "test_suite",
					Spec: &pb.APISpec{
						Url: "convert_to_sql",
					},
				},
				Case: &pb.TestCase{
					Name: "test_case",
					Request: &pb.Request{
						Body: `{"query": "show me all users", "provider": "mock", "model": "test_model"}`,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "ping request",
			request: &pb.TestSuiteWithCase{
				Suite: &pb.TestSuite{
					Name: "test_suite",
					Spec: &pb.APISpec{
						Url: "ping",
					},
				},
				Case: &pb.TestCase{
					Name: "test_case",
					Request: &pb.Request{
						Body: `{}`,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "health check request",
			request: &pb.TestSuiteWithCase{
				Suite: &pb.TestSuite{
					Name: "test_suite",
					Spec: &pb.APISpec{
						Url: "health_check",
					},
				},
				Case: &pb.TestCase{
					Name: "test_case",
					Request: &pb.Request{
						Body: `{}`,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "get model info request",
			request: &pb.TestSuiteWithCase{
				Suite: &pb.TestSuite{
					Name: "test_suite",
					Spec: &pb.APISpec{
						Url: "get_model_info",
					},
				},
				Case: &pb.TestCase{
					Name: "test_case",
					Request: &pb.Request{
						Body: `{}`,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "unsupported action",
			request: &pb.TestSuiteWithCase{
				Suite: &pb.TestSuite{
					Name: "test_suite",
					Spec: &pb.APISpec{
						Url: "unsupported_action",
					},
				},
				Case: &pb.TestCase{
					Name: "test_case",
					Request: &pb.Request{
						Body: `{}`,
					},
				},
			},
			wantErr: false, // Should not error, but return failure in response
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			response, err := server.Run(ctx, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("AIPluginServer.Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if response == nil {
				t.Error("AIPluginServer.Run() returned nil response")
				return
			}

			// For unsupported actions, expect success=false
			if tt.request.Suite.Spec.Url == "unsupported_action" {
				if response.Success {
					t.Error("Expected success=false for unsupported action")
				}
			} else {
				// For supported actions, expect success=true
				if !response.Success {
					t.Errorf("Expected success=true for action %s, got message: %s", tt.request.Suite.Spec.Url, response.Message)
				}
			}

			if response.Message == "" {
				t.Error("Expected non-empty message in response")
			}
		})
	}
}

func TestAIPluginServerRun_InvalidJSON(t *testing.T) {
	cfg := &config.Config{
		AI: config.AIConfig{
			DefaultModel: "test_model",
		},
	}

	aiService, err := ai.NewService(cfg)
	if err != nil {
		t.Fatalf("Failed to create AI service: %v", err)
	}

	server := NewAIPluginServer(cfg, aiService)

	request := &pb.TestSuiteWithCase{
		Suite: &pb.TestSuite{
			Name: "test_suite",
			Spec: &pb.APISpec{
				Url: "convert_to_sql",
			},
		},
		Case: &pb.TestCase{
			Name: "test_case",
			Request: &pb.Request{
				Body: `{invalid json}`,
			},
		},
	}

	ctx := context.Background()
	response, err := server.Run(ctx, request)

	if err != nil {
		t.Errorf("AIPluginServer.Run() should not return error for invalid JSON, got: %v", err)
	}

	if response == nil {
		t.Fatal("AIPluginServer.Run() returned nil response")
	}

	if response.Success {
		t.Error("Expected success=false for invalid JSON")
	}

	if response.Message == "" {
		t.Error("Expected non-empty error message for invalid JSON")
	}
}

func TestAIPluginServerRun_NilRequest(t *testing.T) {
	cfg := &config.Config{
		AI: config.AIConfig{
			DefaultModel: "test_model",
		},
	}

	aiService, err := ai.NewService(cfg)
	if err != nil {
		t.Fatalf("Failed to create AI service: %v", err)
	}

	server := NewAIPluginServer(cfg, aiService)

	ctx := context.Background()
	response, err := server.Run(ctx, nil)

	if err != nil {
		t.Errorf("AIPluginServer.Run() should not return error for nil request, got: %v", err)
	}

	if response == nil {
		t.Fatal("AIPluginServer.Run() returned nil response")
	}

	if response.Success {
		t.Error("Expected success=false for nil request")
	}

	if response.Message == "" {
		t.Error("Expected non-empty error message for nil request")
	}
}
