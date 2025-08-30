package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Linuxsuren/atest-ext-ai/internal/ai"
	"github.com/Linuxsuren/atest-ext-ai/internal/config"
	"github.com/Linuxsuren/atest-ext-ai/internal/server"
	pb "github.com/linuxsuren/api-testing/pkg/server"
)

// TestAIPluginIntegration tests the complete AI plugin integration
func TestAIPluginIntegration(t *testing.T) {
	// Set up test environment variables
	os.Setenv("AI_PROVIDER", "mock")
	os.Setenv("AI_API_KEY", "test_key")
	os.Setenv("AI_MODEL", "test_model")
	os.Setenv("AI_TIMEOUT", "30")
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("LOG_LEVEL", "info")

	// Clean up environment variables after test
	defer func() {
		os.Unsetenv("AI_PROVIDER")
		os.Unsetenv("AI_API_KEY")
		os.Unsetenv("AI_MODEL")
		os.Unsetenv("AI_TIMEOUT")
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("LOG_LEVEL")
	}()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Configuration validation failed: %v", err)
	}

	// Create AI service
	aiService, err := ai.NewService(cfg)
	if err != nil {
		t.Fatalf("Failed to create AI service: %v", err)
	}

	// Create AI plugin server
	pluginServer := server.NewAIPluginServer(cfg, aiService)
	if pluginServer == nil {
		t.Fatal("Failed to create AI plugin server")
	}

	// Test different scenarios
	testScenarios := []struct {
		name          string
		action        string
		requestBody   string
		expectSuccess bool
	}{
		{
			name:          "Convert to SQL - Users query",
			action:        "convert_to_sql",
			requestBody:   `{"query": "show me all users", "provider": "mock", "model": "test_model"}`,
			expectSuccess: true,
		},
		{
			name:          "Convert to SQL - Orders query",
			action:        "convert_to_sql",
			requestBody:   `{"query": "find all orders from last month", "provider": "mock", "model": "test_model"}`,
			expectSuccess: true,
		},
		{
			name:          "Convert to SQL - Products query",
			action:        "convert_to_sql",
			requestBody:   `{"query": "list all products with price", "provider": "mock", "model": "test_model"}`,
			expectSuccess: true,
		},
		{
			name:          "Convert to SQL - Invalid JSON",
			action:        "convert_to_sql",
			requestBody:   `{invalid json}`,
			expectSuccess: false,
		},
		{
			name:          "Ping request",
			action:        "ping",
			requestBody:   `{}`,
			expectSuccess: true,
		},
		{
			name:          "Health check request",
			action:        "health_check",
			requestBody:   `{}`,
			expectSuccess: true,
		},
		{
			name:          "Get model info request",
			action:        "get_model_info",
			requestBody:   `{}`,
			expectSuccess: true,
		},
		{
			name:          "Unsupported action",
			action:        "unsupported_action",
			requestBody:   `{}`,
			expectSuccess: false,
		},
	}

	for _, scenario := range testScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Create test request
			request := &pb.TestSuiteWithCase{
				Suite: &pb.TestSuite{
					Name: "integration_test_suite",
					Spec: &pb.APISpec{
						Url: scenario.action,
					},
				},
				Case: &pb.TestCase{
					Name: scenario.name,
					Request: &pb.Request{
						Body: scenario.requestBody,
					},
				},
			}

			// Execute request with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			response, err := pluginServer.Run(ctx, request)

			// Verify no error occurred
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify response is not nil
			if response == nil {
				t.Error("Response should not be nil")
				return
			}

			// Verify success expectation
			if response.Success != scenario.expectSuccess {
				t.Errorf("Expected success=%v, got success=%v, message=%s",
					scenario.expectSuccess, response.Success, response.Message)
			}

			// Verify response has a message
			if response.Message == "" {
				t.Error("Response message should not be empty")
			}

			// Additional verification for successful SQL conversion
			if scenario.action == "convert_to_sql" && scenario.expectSuccess {
				if response.Message == "" {
					t.Error("SQL conversion response message should not be empty")
				}
			}
		})
	}
}

// TestAIPluginPerformance tests the performance of the AI plugin
func TestAIPluginPerformance(t *testing.T) {
	// Set up test environment
	os.Setenv("AI_PROVIDER", "mock")
	os.Setenv("AI_API_KEY", "test_key")
	os.Setenv("AI_MODEL", "test_model")
	os.Setenv("AI_TIMEOUT", "30")

	defer func() {
		os.Unsetenv("AI_PROVIDER")
		os.Unsetenv("AI_API_KEY")
		os.Unsetenv("AI_MODEL")
		os.Unsetenv("AI_TIMEOUT")
	}()

	// Load configuration and create services
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	aiService, err := ai.NewService(cfg)
	if err != nil {
		t.Fatalf("Failed to create AI service: %v", err)
	}

	pluginServer := server.NewAIPluginServer(cfg, aiService)

	// Performance test parameters
	numRequests := 100
	maxDuration := 5 * time.Second

	// Create test request
	request := &pb.TestSuiteWithCase{
		Suite: &pb.TestSuite{
			Name: "performance_test_suite",
			Spec: &pb.APISpec{
				Url: "convert_to_sql",
			},
		},
		Case: &pb.TestCase{
			Name: "performance_test",
			Request: &pb.Request{
				Body: `{"query": "show me all users", "provider": "mock", "model": "test_model"}`,
			},
		},
	}

	// Measure performance
	start := time.Now()
	successCount := 0

	for i := 0; i < numRequests; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		response, err := pluginServer.Run(ctx, request)
		cancel()

		if err == nil && response != nil && response.Success {
			successCount++
		}

		// Check if we're taking too long
		if time.Since(start) > maxDuration {
			t.Logf("Performance test stopped early after %d requests due to time limit", i+1)
			break
		}
	}

	duration := time.Since(start)
	successRate := float64(successCount) / float64(numRequests) * 100
	avgDuration := duration / time.Duration(numRequests)

	t.Logf("Performance Results:")
	t.Logf("  Total Requests: %d", numRequests)
	t.Logf("  Successful Requests: %d", successCount)
	t.Logf("  Success Rate: %.2f%%", successRate)
	t.Logf("  Total Duration: %v", duration)
	t.Logf("  Average Duration per Request: %v", avgDuration)

	// Performance assertions
	if successRate < 95.0 {
		t.Errorf("Success rate too low: %.2f%% (expected >= 95%%)", successRate)
	}

	if avgDuration > 100*time.Millisecond {
		t.Errorf("Average duration too high: %v (expected <= 100ms)", avgDuration)
	}
}

// TestAIPluginConcurrency tests concurrent access to the AI plugin
func TestAIPluginConcurrency(t *testing.T) {
	// Set up test environment
	os.Setenv("AI_PROVIDER", "mock")
	os.Setenv("AI_API_KEY", "test_key")
	os.Setenv("AI_MODEL", "test_model")
	os.Setenv("AI_TIMEOUT", "30")

	defer func() {
		os.Unsetenv("AI_PROVIDER")
		os.Unsetenv("AI_API_KEY")
		os.Unsetenv("AI_MODEL")
		os.Unsetenv("AI_TIMEOUT")
	}()

	// Load configuration and create services
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	aiService, err := ai.NewService(cfg)
	if err != nil {
		t.Fatalf("Failed to create AI service: %v", err)
	}

	pluginServer := server.NewAIPluginServer(cfg, aiService)

	// Concurrency test parameters
	numGoroutines := 10
	numRequestsPerGoroutine := 10

	// Channel to collect results
	results := make(chan bool, numGoroutines*numRequestsPerGoroutine)

	// Start concurrent goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < numRequestsPerGoroutine; j++ {
				request := &pb.TestSuiteWithCase{
					Suite: &pb.TestSuite{
						Name: "concurrency_test_suite",
						Spec: &pb.APISpec{
							Url: "convert_to_sql",
						},
					},
					Case: &pb.TestCase{
						Name: "concurrency_test",
						Request: &pb.Request{
							Body: `{"query": "show me all users", "provider": "mock", "model": "test_model"}`,
						},
					},
				}

				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				response, err := pluginServer.Run(ctx, request)
				cancel()

				success := err == nil && response != nil && response.Success
				results <- success
			}
		}(i)
	}

	// Collect results
	totalRequests := numGoroutines * numRequestsPerGoroutine
	successCount := 0

	for i := 0; i < totalRequests; i++ {
		if <-results {
			successCount++
		}
	}

	successRate := float64(successCount) / float64(totalRequests) * 100

	t.Logf("Concurrency Test Results:")
	t.Logf("  Goroutines: %d", numGoroutines)
	t.Logf("  Requests per Goroutine: %d", numRequestsPerGoroutine)
	t.Logf("  Total Requests: %d", totalRequests)
	t.Logf("  Successful Requests: %d", successCount)
	t.Logf("  Success Rate: %.2f%%", successRate)

	// Concurrency assertions
	if successRate < 95.0 {
		t.Errorf("Concurrency success rate too low: %.2f%% (expected >= 95%%)", successRate)
	}
}
