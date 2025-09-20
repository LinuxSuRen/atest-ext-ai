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
	"errors"
	"testing"
	"time"
)

func TestLoadBalancer_NewDefaultLoadBalancer(t *testing.T) {
	config := LoadBalancerConfig{
		Strategy:            "round_robin",
		HealthCheckInterval: 30 * time.Second,
	}

	lb := NewDefaultLoadBalancer(config)
	if lb == nil {
		t.Fatal("Load balancer is nil")
	}

	stats := lb.GetStats()
	if stats.TotalRequests != 0 {
		t.Errorf("Expected 0 total requests initially, got %d", stats.TotalRequests)
	}
}

func TestLoadBalancer_RegisterUnregisterClient(t *testing.T) {
	config := LoadBalancerConfig{
		Strategy:            "round_robin",
		HealthCheckInterval: 30 * time.Second,
	}

	lb := NewDefaultLoadBalancer(config)
	mockClient := &mockAIClient{name: "test-client", healthy: true}

	// Cast to access registration methods
	if lbImpl, ok := lb.(*defaultLoadBalancer); ok {
		// Register client
		lbImpl.RegisterClient("test-client", mockClient)

		healthyClients := lb.GetHealthyClients()
		if len(healthyClients) != 1 {
			t.Errorf("Expected 1 healthy client, got %d", len(healthyClients))
		}
		if healthyClients[0] != "test-client" {
			t.Errorf("Expected 'test-client', got %s", healthyClients[0])
		}

		// Unregister client
		lbImpl.UnregisterClient("test-client")

		healthyClients = lb.GetHealthyClients()
		if len(healthyClients) != 0 {
			t.Errorf("Expected 0 healthy clients after unregistration, got %d", len(healthyClients))
		}
	} else {
		t.Fatal("Could not cast to defaultLoadBalancer")
	}
}

func TestLoadBalancer_SelectClient_NoHealthyClients(t *testing.T) {
	config := LoadBalancerConfig{
		Strategy:            "round_robin",
		HealthCheckInterval: 30 * time.Second,
	}

	lb := NewDefaultLoadBalancer(config)

	req := &GenerateRequest{Prompt: "test"}
	_, err := lb.SelectClient(req)

	if !errors.Is(err, ErrNoHealthyClients) {
		t.Errorf("Expected ErrNoHealthyClients, got: %v", err)
	}
}

func TestLoadBalancer_SelectClient_RoundRobin(t *testing.T) {
	config := LoadBalancerConfig{
		Strategy:            "round_robin",
		HealthCheckInterval: 30 * time.Second,
	}

	lb := NewDefaultLoadBalancer(config)

	// Register multiple clients
	if lbImpl, ok := lb.(*defaultLoadBalancer); ok {
		clients := []string{"client1", "client2", "client3"}
		for _, name := range clients {
			mockClient := &mockAIClient{name: name, healthy: true}
			lbImpl.RegisterClient(name, mockClient)
		}

		req := &GenerateRequest{Prompt: "test"}

		// Test round-robin selection
		selectedClients := make([]string, 6) // Two full rounds
		for i := 0; i < 6; i++ {
			client, err := lb.SelectClient(req)
			if err != nil {
				t.Fatalf("SelectClient failed: %v", err)
			}

			// Get client name from mock
			if mockClient, ok := client.(*mockAIClient); ok {
				selectedClients[i] = mockClient.name
			} else {
				t.Fatalf("Expected mockAIClient, got %T", client)
			}
		}

		// Verify round-robin pattern
		expected := []string{"client1", "client2", "client3", "client1", "client2", "client3"}
		for i, expectedName := range expected {
			if selectedClients[i] != expectedName {
				t.Errorf("Round %d: expected %s, got %s", i, expectedName, selectedClients[i])
			}
		}
	} else {
		t.Fatal("Could not cast to defaultLoadBalancer")
	}
}

func TestLoadBalancer_SelectClient_Weighted(t *testing.T) {
	config := LoadBalancerConfig{
		Strategy:            "weighted",
		HealthCheckInterval: 30 * time.Second,
	}

	lb := NewDefaultLoadBalancer(config)

	if lbImpl, ok := lb.(*defaultLoadBalancer); ok {
		// Register clients with different success rates (simulated via RecordSuccess/RecordFailure)
		client1 := &mockAIClient{name: "client1", healthy: true}
		client2 := &mockAIClient{name: "client2", healthy: true}

		lbImpl.RegisterClient("client1", client1)
		lbImpl.RegisterClient("client2", client2)

		// Simulate different success rates
		lbImpl.RecordSuccess("client1", 10*time.Millisecond)
		lbImpl.RecordSuccess("client1", 10*time.Millisecond)
		lbImpl.RecordSuccess("client2", 10*time.Millisecond)
		lbImpl.RecordFailure("client2")

		req := &GenerateRequest{Prompt: "test"}

		// client1 should be preferred due to better success rate
		client, err := lb.SelectClient(req)
		if err != nil {
			t.Fatalf("SelectClient failed: %v", err)
		}

		if mockClient, ok := client.(*mockAIClient); ok {
			if mockClient.name != "client1" {
				t.Errorf("Expected client1 to be selected (better success rate), got %s", mockClient.name)
			}
		} else {
			t.Fatal("Expected mockAIClient")
		}
	} else {
		t.Fatal("Could not cast to defaultLoadBalancer")
	}
}

func TestLoadBalancer_SelectClient_LeastConnections(t *testing.T) {
	config := LoadBalancerConfig{
		Strategy:            "least_connections",
		HealthCheckInterval: 30 * time.Second,
	}

	lb := NewDefaultLoadBalancer(config)

	if lbImpl, ok := lb.(*defaultLoadBalancer); ok {
		client1 := &mockAIClient{name: "client1", healthy: true}
		client2 := &mockAIClient{name: "client2", healthy: true}

		lbImpl.RegisterClient("client1", client1)
		lbImpl.RegisterClient("client2", client2)

		// Simulate different request counts
		lbImpl.RecordSuccess("client1", 10*time.Millisecond)
		lbImpl.RecordSuccess("client1", 10*time.Millisecond)
		// client2 has fewer requests

		req := &GenerateRequest{Prompt: "test"}

		// client2 should be selected (least used)
		client, err := lb.SelectClient(req)
		if err != nil {
			t.Fatalf("SelectClient failed: %v", err)
		}

		if mockClient, ok := client.(*mockAIClient); ok {
			if mockClient.name != "client2" {
				t.Errorf("Expected client2 to be selected (least connections), got %s", mockClient.name)
			}
		} else {
			t.Fatal("Expected mockAIClient")
		}
	} else {
		t.Fatal("Could not cast to defaultLoadBalancer")
	}
}

func TestLoadBalancer_SelectClient_Failover(t *testing.T) {
	config := LoadBalancerConfig{
		Strategy:            "failover",
		HealthCheckInterval: 30 * time.Second,
	}

	lb := NewDefaultLoadBalancer(config)

	if lbImpl, ok := lb.(*defaultLoadBalancer); ok {
		client1 := &mockAIClient{name: "client1", healthy: true}
		client2 := &mockAIClient{name: "client2", healthy: true}

		lbImpl.RegisterClient("client1", client1)
		lbImpl.RegisterClient("client2", client2)

		req := &GenerateRequest{Prompt: "test"}

		// For failover, first healthy client should be selected
		client, err := lb.SelectClient(req)
		if err != nil {
			t.Fatalf("SelectClient failed: %v", err)
		}

		if mockClient, ok := client.(*mockAIClient); ok {
			// Should select the first healthy client consistently
			expectedName := mockClient.name

			// Test multiple selections to ensure consistency
			for i := 0; i < 5; i++ {
				client, err := lb.SelectClient(req)
				if err != nil {
					t.Fatalf("SelectClient failed on iteration %d: %v", i, err)
				}
				if mockClient, ok := client.(*mockAIClient); ok {
					if mockClient.name != expectedName {
						t.Errorf("Failover should be consistent, expected %s, got %s", expectedName, mockClient.name)
					}
				}
			}
		} else {
			t.Fatal("Expected mockAIClient")
		}
	} else {
		t.Fatal("Could not cast to defaultLoadBalancer")
	}
}

func TestLoadBalancer_UpdateHealth(t *testing.T) {
	config := LoadBalancerConfig{
		Strategy:            "round_robin",
		HealthCheckInterval: 30 * time.Second,
	}

	lb := NewDefaultLoadBalancer(config)

	if lbImpl, ok := lb.(*defaultLoadBalancer); ok {
		client1 := &mockAIClient{name: "client1", healthy: true}
		client2 := &mockAIClient{name: "client2", healthy: true}

		lbImpl.RegisterClient("client1", client1)
		lbImpl.RegisterClient("client2", client2)

		// Initially both should be healthy
		healthyClients := lb.GetHealthyClients()
		if len(healthyClients) != 2 {
			t.Errorf("Expected 2 healthy clients, got %d", len(healthyClients))
		}

		// Mark one as unhealthy
		lb.UpdateHealth("client1", false)

		healthyClients = lb.GetHealthyClients()
		if len(healthyClients) != 1 {
			t.Errorf("Expected 1 healthy client after health update, got %d", len(healthyClients))
		}
		if healthyClients[0] != "client2" {
			t.Errorf("Expected client2 to be healthy, got %s", healthyClients[0])
		}

		// Mark it as healthy again
		lb.UpdateHealth("client1", true)

		healthyClients = lb.GetHealthyClients()
		if len(healthyClients) != 2 {
			t.Errorf("Expected 2 healthy clients after recovery, got %d", len(healthyClients))
		}
	} else {
		t.Fatal("Could not cast to defaultLoadBalancer")
	}
}

func TestLoadBalancer_GetStats(t *testing.T) {
	config := LoadBalancerConfig{
		Strategy:            "round_robin",
		HealthCheckInterval: 30 * time.Second,
	}

	lb := NewDefaultLoadBalancer(config)

	if lbImpl, ok := lb.(*defaultLoadBalancer); ok {
		client1 := &mockAIClient{name: "client1", healthy: true}
		lbImpl.RegisterClient("client1", client1)

		// Make some requests to generate stats
		req := &GenerateRequest{Prompt: "test"}
		lb.SelectClient(req)
		lb.SelectClient(req)

		// Record some successes and failures
		lbImpl.RecordSuccess("client1", 10*time.Millisecond)
		lbImpl.RecordFailure("client1")

		stats := lb.GetStats()

		if stats.TotalRequests != 2 {
			t.Errorf("Expected 2 total requests, got %d", stats.TotalRequests)
		}

		if clientStats, exists := stats.ClientStats["client1"]; exists {
			if clientStats.Requests != 2 {
				t.Errorf("Expected 2 requests for client1, got %d", clientStats.Requests)
			}
			if clientStats.Successes != 1 {
				t.Errorf("Expected 1 success for client1, got %d", clientStats.Successes)
			}
			if clientStats.Failures != 1 {
				t.Errorf("Expected 1 failure for client1, got %d", clientStats.Failures)
			}
			if clientStats.AverageResponseTime != 10*time.Millisecond {
				t.Errorf("Expected 10ms average response time, got %v", clientStats.AverageResponseTime)
			}
		} else {
			t.Error("Expected stats for client1")
		}
	} else {
		t.Fatal("Could not cast to defaultLoadBalancer")
	}
}

func TestLoadBalancer_RecordSuccessFailure(t *testing.T) {
	config := LoadBalancerConfig{
		Strategy:            "round_robin",
		HealthCheckInterval: 30 * time.Second,
	}

	lb := NewDefaultLoadBalancer(config)

	if lbImpl, ok := lb.(*defaultLoadBalancer); ok {
		client1 := &mockAIClient{name: "client1", healthy: true}
		lbImpl.RegisterClient("client1", client1)

		// Record multiple successes to test average response time calculation
		lbImpl.RecordSuccess("client1", 10*time.Millisecond)
		lbImpl.RecordSuccess("client1", 20*time.Millisecond)

		stats := lb.GetStats()
		if clientStats, exists := stats.ClientStats["client1"]; exists {
			if clientStats.Successes != 2 {
				t.Errorf("Expected 2 successes, got %d", clientStats.Successes)
			}
			// Average should be (10 + 20) / 2 = 15ms
			expectedAvg := 15 * time.Millisecond
			if clientStats.AverageResponseTime != expectedAvg {
				t.Errorf("Expected average response time %v, got %v", expectedAvg, clientStats.AverageResponseTime)
			}
		} else {
			t.Error("Expected stats for client1")
		}

		// Record failures
		lbImpl.RecordFailure("client1")
		lbImpl.RecordFailure("client1")

		stats = lb.GetStats()
		if clientStats, exists := stats.ClientStats["client1"]; exists {
			if clientStats.Failures != 2 {
				t.Errorf("Expected 2 failures, got %d", clientStats.Failures)
			}
		}
	} else {
		t.Fatal("Could not cast to defaultLoadBalancer")
	}
}

func TestLoadBalancer_SelectClient_UnknownStrategy(t *testing.T) {
	config := LoadBalancerConfig{
		Strategy:            "unknown_strategy",
		HealthCheckInterval: 30 * time.Second,
	}

	lb := NewDefaultLoadBalancer(config)

	if lbImpl, ok := lb.(*defaultLoadBalancer); ok {
		client1 := &mockAIClient{name: "client1", healthy: true}
		lbImpl.RegisterClient("client1", client1)

		req := &GenerateRequest{Prompt: "test"}

		// Unknown strategy should fall back to round robin
		client, err := lb.SelectClient(req)
		if err != nil {
			t.Fatalf("SelectClient failed: %v", err)
		}

		if mockClient, ok := client.(*mockAIClient); ok {
			if mockClient.name != "client1" {
				t.Errorf("Expected client1, got %s", mockClient.name)
			}
		} else {
			t.Fatal("Expected mockAIClient")
		}
	} else {
		t.Fatal("Could not cast to defaultLoadBalancer")
	}
}
