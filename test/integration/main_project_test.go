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

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	server "github.com/linuxsuren/api-testing/pkg/server"
	"github.com/linuxsuren/api-testing/pkg/testing/remote"
)

// TestMainProjectIntegration tests integration from the atest-ext-ai perspective
// This test verifies that the AI plugin properly implements the AI interface standards
// expected by the main api-testing project
func TestMainProjectIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping main project integration test in short mode")
	}

	// Start the AI plugin
	plugin := startLocalAIPlugin(t)
	defer plugin.Stop()

	// Wait for the plugin to be ready
	require.True(t, waitForPlugin(plugin.SocketPath, 30*time.Second), "AI plugin failed to start")

	// Connect to the plugin using gRPC
	conn := connectToPlugin(t, plugin.SocketPath)
	defer conn.Close()

	client := remote.NewLoaderClient(conn)

	// Test the AI interface standard compliance
	t.Run("TestAIInterfaceStandardCompliance", func(t *testing.T) {
		testAIInterfaceCompliance(t, client)
	})

	t.Run("TestMainProjectCompatibility", func(t *testing.T) {
		testMainProjectCompatibility(t, client)
	})

	t.Run("TestErrorHandling", func(t *testing.T) {
		testErrorHandling(t, client)
	})

	t.Run("TestPerformanceRequirements", func(t *testing.T) {
		testPerformanceRequirements(t, client)
	})
}

// testAIInterfaceCompliance tests that the plugin correctly implements the AI interface standards
func testAIInterfaceCompliance(t *testing.T, client remote.LoaderClient) {
	ctx := context.Background()

	// Test ai.capabilities method
	t.Run("AICapabilitiesMethod", func(t *testing.T) {
		req := &server.DataQuery{
			Type: "ai",
			Key:  "capabilities",
			Sql:  "",
		}

		resp, err := client.Query(ctx, req)
		require.NoError(t, err, "ai.capabilities should not return gRPC error")
		require.NotNil(t, resp, "Response should not be nil")

		// Verify response format
		pairs := pairListToMap(resp.Data)
		assert.Equal(t, "true", pairs["success"], "Capabilities should succeed")
		assert.NotEmpty(t, pairs["capabilities"], "Should return capabilities JSON")

		// Validate capabilities structure
		var capabilities map[string]interface{}
		err = json.Unmarshal([]byte(pairs["capabilities"]), &capabilities)
		assert.NoError(t, err, "Capabilities should be valid JSON")

		// Check for required capability fields
		assert.Contains(t, pairs, "description", "Should include plugin description")
		assert.Contains(t, pairs, "version", "Should include plugin version")
	})

	// Test ai.generate method
	t.Run("AIGenerateMethod", func(t *testing.T) {
		// Encode parameters as JSON in SQL field (as per AI interface standard)
		params := map[string]string{
			"model":  "gpt-4",
			"prompt": "Create a simple SELECT statement for users table",
			"config": `{"temperature": 0.7}`,
		}
		paramsJSON, _ := json.Marshal(params)

		req := &server.DataQuery{
			Type: "ai",
			Key:  "generate",
			Sql:  string(paramsJSON),
		}

		resp, err := client.Query(ctx, req)
		require.NoError(t, err, "ai.generate should not return gRPC error")
		require.NotNil(t, resp, "Response should not be nil")

		// Verify response format
		pairs := pairListToMap(resp.Data)
		assert.Equal(t, "true", pairs["success"], "Generate should succeed")
		assert.NotEmpty(t, pairs["content"], "Should return generated content")

		// Validate generated content
		content := pairs["content"]
		assert.Contains(t, strings.ToUpper(content), "SELECT", "Generated content should contain SELECT")
		assert.Contains(t, strings.ToLower(content), "users", "Generated content should reference users table")
	})

	// Test legacy natural language query support
	t.Run("LegacyNaturalLanguageQuery", func(t *testing.T) {
		req := &server.DataQuery{
			Type: "ai",
			Key:  "Create a users table with id and name columns",
			Sql:  "",
		}

		resp, err := client.Query(ctx, req)
		require.NoError(t, err, "Legacy query should not return gRPC error")
		require.NotNil(t, resp, "Response should not be nil")

		// Should still work for backward compatibility
		pairs := pairListToMap(resp.Data)
		if pairs["success"] == "true" {
			assert.NotEmpty(t, pairs["content"], "Should return generated content for legacy queries")
		}
	})
}

// testMainProjectCompatibility tests that the plugin works with main project expectations
func testMainProjectCompatibility(t *testing.T, client remote.LoaderClient) {
	ctx := context.Background()

	// Test non-AI query handling
	t.Run("NonAIQueryHandling", func(t *testing.T) {
		req := &server.DataQuery{
			Type: "database",
			Key:  "test",
			Sql:  "SELECT 1",
		}

		resp, err := client.Query(ctx, req)
		// This should either:
		// 1. Return an error indicating unsupported type
		// 2. Handle it gracefully
		// It should not crash or panic
		if err != nil {
			assert.Contains(t, err.Error(), "unsupported", "Should indicate unsupported query type")
		} else {
			pairs := pairListToMap(resp.Data)
			// If it handles it, it should return a proper response
			assert.NotNil(t, pairs, "Should return valid response structure")
		}
	})

	// Test store configuration compatibility
	t.Run("StoreConfigurationCompatibility", func(t *testing.T) {
		// Test that the plugin responds to basic store verification
		req := &server.Empty{}

		resp, err := client.Verify(ctx, req)
		require.NoError(t, err, "Verify should not return error")
		require.NotNil(t, resp, "Verify response should not be nil")

		// Should indicate it's ready and not read-only for AI operations
		assert.True(t, resp.Ready, "Plugin should be ready")
		assert.False(t, resp.ReadOnly, "AI plugin should not be read-only")
		assert.NotEmpty(t, resp.Version, "Should report version")
	})
}

// testErrorHandling tests error scenarios
func testErrorHandling(t *testing.T, client remote.LoaderClient) {
	ctx := context.Background()

	// Test invalid AI method
	t.Run("InvalidAIMethod", func(t *testing.T) {
		req := &server.DataQuery{
			Type: "ai",
			Key:  "invalid_method",
			Sql:  "",
		}

		resp, err := client.Query(ctx, req)
		require.NoError(t, err, "Should not return gRPC error for invalid method")
		require.NotNil(t, resp, "Response should not be nil")

		pairs := pairListToMap(resp.Data)
		assert.Equal(t, "false", pairs["success"], "Invalid method should return success=false")
		assert.NotEmpty(t, pairs["error"], "Should return error message")
	})

	// Test malformed parameters
	t.Run("MalformedParameters", func(t *testing.T) {
		req := &server.DataQuery{
			Type: "ai",
			Key:  "generate",
			Sql:  "{invalid json}",
		}

		resp, err := client.Query(ctx, req)
		require.NoError(t, err, "Should not return gRPC error for malformed params")
		require.NotNil(t, resp, "Response should not be nil")

		pairs := pairListToMap(resp.Data)
		assert.Equal(t, "false", pairs["success"], "Malformed params should return success=false")
		assert.NotEmpty(t, pairs["error"], "Should return error message")
	})

	// Test missing required parameters
	t.Run("MissingRequiredParameters", func(t *testing.T) {
		// Empty parameters for generate method
		params := map[string]string{}
		paramsJSON, _ := json.Marshal(params)

		req := &server.DataQuery{
			Type: "ai",
			Key:  "generate",
			Sql:  string(paramsJSON),
		}

		resp, err := client.Query(ctx, req)
		require.NoError(t, err, "Should not return gRPC error for missing params")
		require.NotNil(t, resp, "Response should not be nil")

		pairs := pairListToMap(resp.Data)
		assert.Equal(t, "false", pairs["success"], "Missing params should return success=false")
		assert.NotEmpty(t, pairs["error"], "Should return error message about missing prompt")
	})
}

// testPerformanceRequirements tests that the plugin meets performance requirements
func testPerformanceRequirements(t *testing.T, client remote.LoaderClient) {
	ctx := context.Background()

	// Test capabilities response time
	t.Run("CapabilitiesResponseTime", func(t *testing.T) {
		start := time.Now()

		req := &server.DataQuery{
			Type: "ai",
			Key:  "capabilities",
			Sql:  "",
		}

		resp, err := client.Query(ctx, req)
		duration := time.Since(start)

		require.NoError(t, err)
		pairs := pairListToMap(resp.Data)
		require.Equal(t, "true", pairs["success"])

		assert.Less(t, duration, 5*time.Second, "Capabilities should respond within 5 seconds")
	})

	// Test generation response time
	t.Run("GenerationResponseTime", func(t *testing.T) {
		start := time.Now()

		params := map[string]string{
			"prompt": "SELECT * FROM users LIMIT 10",
		}
		paramsJSON, _ := json.Marshal(params)

		req := &server.DataQuery{
			Type: "ai",
			Key:  "generate",
			Sql:  string(paramsJSON),
		}

		resp, err := client.Query(ctx, req)
		duration := time.Since(start)

		require.NoError(t, err)
		pairs := pairListToMap(resp.Data)
		require.Equal(t, "true", pairs["success"])

		assert.Less(t, duration, 10*time.Second, "Generation should respond within 10 seconds")
	})
}

// TestAIPluginStandaloneOperation tests the plugin in standalone mode
func TestAIPluginStandaloneOperation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping standalone operation test in short mode")
	}

	plugin := startLocalAIPlugin(t)
	defer plugin.Stop()

	require.True(t, waitForPlugin(plugin.SocketPath, 30*time.Second), "AI plugin failed to start")

	conn := connectToPlugin(t, plugin.SocketPath)
	defer conn.Close()

	client := remote.NewLoaderClient(conn)
	ctx := context.Background()

	// Test plugin startup and health
	t.Run("PluginStartupHealth", func(t *testing.T) {
		resp, err := client.Verify(ctx, &server.Empty{})
		require.NoError(t, err)
		assert.True(t, resp.Ready, "Plugin should be ready after startup")
	})

	// Test concurrent access
	t.Run("ConcurrentAccess", func(t *testing.T) {
		const numRequests = 5
		results := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			go func(id int) {
				req := &server.DataQuery{
					Type: "ai",
					Key:  "capabilities",
					Sql:  "",
				}

				resp, err := client.Query(ctx, req)
				if err != nil {
					results <- err
					return
				}

				pairs := pairListToMap(resp.Data)
				if pairs["success"] != "true" {
					results <- fmt.Errorf("request %d failed: %s", id, pairs["error"])
					return
				}

				results <- nil
			}(i)
		}

		// Collect results
		for i := 0; i < numRequests; i++ {
			select {
			case err := <-results:
				assert.NoError(t, err, "Concurrent request %d failed", i)
			case <-time.After(30 * time.Second):
				t.Fatal("Concurrent request timed out")
			}
		}
	})
}

// LocalAIPlugin represents a running local AI plugin instance
type LocalAIPlugin struct {
	Process    *os.Process
	SocketPath string
}

// Stop terminates the AI plugin process
func (p *LocalAIPlugin) Stop() {
	if p.Process != nil {
		p.Process.Signal(syscall.SIGTERM)
		p.Process.Wait()
	}
	// Clean up socket file
	os.Remove(p.SocketPath)
}

// startLocalAIPlugin starts the AI plugin locally for testing
func startLocalAIPlugin(t *testing.T) *LocalAIPlugin {
	// Create socket path
	socketPath := "/tmp/atest-ext-ai-main-project-test.sock"
	os.Remove(socketPath) // Clean up any existing socket

	// Find the plugin binary
	binaryPath := findPluginBinary()
	if binaryPath == "" {
		t.Skip("AI plugin binary not found, skipping integration test")
	}

	// Set up environment
	env := []string{
		"AI_PLUGIN_SOCKET_PATH=" + socketPath,
		"CONFIG_PATH=" + findConfigPath(),
		"LOG_LEVEL=debug",
	}

	// Start the plugin
	cmd := exec.Command(binaryPath)
	cmd.Env = append(os.Environ(), env...)

	err := cmd.Start()
	require.NoError(t, err, "Failed to start AI plugin")

	return &LocalAIPlugin{
		Process:    cmd.Process,
		SocketPath: socketPath,
	}
}


// waitForPlugin waits for the plugin to become available
func waitForPlugin(socketPath string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := net.Dial("unix", socketPath); err == nil {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// connectToPlugin connects to the plugin via gRPC over Unix socket
func connectToPlugin(t *testing.T, socketPath string) *grpc.ClientConn {
	conn, err := grpc.Dial(
		"unix://"+socketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err, "Failed to connect to AI plugin")
	return conn
}

// pairListToMap converts a list of server.Pair to a map
func pairListToMap(pairs []*server.Pair) map[string]string {
	result := make(map[string]string)
	for _, pair := range pairs {
		result[pair.Key] = pair.Value
	}
	return result
}

// BenchmarkMainProjectIntegration benchmarks the integration from main project perspective
func BenchmarkMainProjectIntegration(b *testing.B) {
	plugin := startLocalAIPluginForBench(b)
	defer plugin.Stop()

	if !waitForPlugin(plugin.SocketPath, 30*time.Second) {
		b.Fatal("AI plugin failed to start")
	}

	conn := connectToPluginForBench(b, plugin.SocketPath)
	defer conn.Close()

	client := remote.NewLoaderClient(conn)
	ctx := context.Background()

	b.ResetTimer()

	// Benchmark capabilities call through the main project interface
	b.Run("Capabilities", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			req := &server.DataQuery{
				Type: "ai",
				Key:  "capabilities",
				Sql:  "",
			}

			resp, err := client.Query(ctx, req)
			if err != nil {
				b.Fatalf("Capabilities call failed: %v", err)
			}

			pairs := pairListToMap(resp.Data)
			if pairs["success"] != "true" {
				b.Fatalf("Capabilities call returned error: %s", pairs["error"])
			}
		}
	})

	// Benchmark generation call through the main project interface
	b.Run("Generation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			params := map[string]string{
				"prompt": fmt.Sprintf("SELECT * FROM users WHERE id = %d", i),
			}
			paramsJSON, _ := json.Marshal(params)

			req := &server.DataQuery{
				Type: "ai",
				Key:  "generate",
				Sql:  string(paramsJSON),
			}

			resp, err := client.Query(ctx, req)
			if err != nil {
				b.Fatalf("Generation call failed: %v", err)
			}

			pairs := pairListToMap(resp.Data)
			if pairs["success"] != "true" {
				b.Fatalf("Generation call returned error: %s", pairs["error"])
			}
		}
	})
}

// Helper functions for benchmarks
func startLocalAIPluginForBench(b *testing.B) *LocalAIPlugin {
	socketPath := "/tmp/atest-ext-ai-bench-main.sock"
	os.Remove(socketPath)

	binaryPath := findPluginBinary()
	if binaryPath == "" {
		b.Skip("AI plugin binary not found, skipping benchmark")
	}

	env := []string{
		"AI_PLUGIN_SOCKET_PATH=" + socketPath,
		"LOG_LEVEL=error", // Reduce logging for benchmarks
	}

	cmd := exec.Command(binaryPath)
	cmd.Env = append(os.Environ(), env...)

	err := cmd.Start()
	if err != nil {
		b.Fatalf("Failed to start AI plugin: %v", err)
	}

	return &LocalAIPlugin{
		Process:    cmd.Process,
		SocketPath: socketPath,
	}
}

func connectToPluginForBench(b *testing.B, socketPath string) *grpc.ClientConn {
	conn, err := grpc.Dial(
		"unix://"+socketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		b.Fatalf("Failed to connect to AI plugin: %v", err)
	}
	return conn
}