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
	"fmt"
	"sync"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
)

// defaultLoadBalancer implements the LoadBalancer interface
type defaultLoadBalancer struct {
	strategy      string
	clients       map[string]interfaces.AIClient
	healthStatus  map[string]bool
	stats         *LoadBalancerStats
	roundRobinIdx int
	mu            sync.RWMutex
	config        LoadBalancerConfig
}

// NewDefaultLoadBalancer creates a new default load balancer
func NewDefaultLoadBalancer(config LoadBalancerConfig) LoadBalancer {
	return &defaultLoadBalancer{
		strategy:     config.Strategy,
		clients:      make(map[string]interfaces.AIClient),
		healthStatus: make(map[string]bool),
		stats: &LoadBalancerStats{
			ClientStats: make(map[string]*ClientStats),
			LastUpdated: time.Now(),
		},
		config: config,
	}
}

// SelectClient selects the best available client for the given request
func (lb *defaultLoadBalancer) SelectClient(req *GenerateRequest) (interfaces.AIClient, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	healthyClients := lb.getHealthyClientNames()
	if len(healthyClients) == 0 {
		return nil, ErrNoHealthyClients
	}

	var selectedClient string
	switch lb.strategy {
	case "round_robin":
		selectedClient = lb.selectRoundRobin(healthyClients)
	case "weighted":
		selectedClient = lb.selectWeighted(healthyClients)
	case "least_connections":
		selectedClient = lb.selectLeastConnections(healthyClients)
	case "failover":
		selectedClient = lb.selectFailover(healthyClients)
	default:
		selectedClient = lb.selectRoundRobin(healthyClients)
	}

	client, exists := lb.clients[selectedClient]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrClientNotFound, selectedClient)
	}

	// Update stats
	lb.updateClientStats(selectedClient)

	return client, nil
}

// UpdateHealth updates the health status of a client
func (lb *defaultLoadBalancer) UpdateHealth(clientID string, healthy bool) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.healthStatus[clientID] = healthy

	// Update client stats
	if stats, exists := lb.stats.ClientStats[clientID]; exists {
		stats.HealthStatus = healthy
	}
}

// GetHealthyClients returns a list of currently healthy clients
func (lb *defaultLoadBalancer) GetHealthyClients() []string {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	return lb.getHealthyClientNames()
}

// GetStats returns load balancing statistics
func (lb *defaultLoadBalancer) GetStats() *LoadBalancerStats {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	// Create a deep copy of stats
	statsCopy := &LoadBalancerStats{
		TotalRequests: lb.stats.TotalRequests,
		ClientStats:   make(map[string]*ClientStats),
		LastUpdated:   time.Now(),
	}

	for name, stats := range lb.stats.ClientStats {
		statsCopy.ClientStats[name] = &ClientStats{
			Requests:            stats.Requests,
			Successes:           stats.Successes,
			Failures:            stats.Failures,
			AverageResponseTime: stats.AverageResponseTime,
			LastUsed:            stats.LastUsed,
			HealthStatus:        stats.HealthStatus,
		}
	}

	return statsCopy
}

// RegisterClient registers a new client with the load balancer
func (lb *defaultLoadBalancer) RegisterClient(name string, client interfaces.AIClient) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.clients[name] = client
	lb.healthStatus[name] = true // Assume healthy initially

	// Initialize stats for the client
	if _, exists := lb.stats.ClientStats[name]; !exists {
		lb.stats.ClientStats[name] = &ClientStats{
			HealthStatus: true,
		}
	}
}

// UnregisterClient removes a client from the load balancer
func (lb *defaultLoadBalancer) UnregisterClient(name string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	delete(lb.clients, name)
	delete(lb.healthStatus, name)
	delete(lb.stats.ClientStats, name)
}

// getHealthyClientNames returns the names of healthy clients (caller must hold read lock)
func (lb *defaultLoadBalancer) getHealthyClientNames() []string {
	var healthy []string
	// Use a deterministic order by iterating through sorted client names
	var allNames []string
	for name := range lb.healthStatus {
		allNames = append(allNames, name)
	}

	// Sort to ensure deterministic order
	for i := 0; i < len(allNames); i++ {
		for j := i + 1; j < len(allNames); j++ {
			if allNames[i] > allNames[j] {
				allNames[i], allNames[j] = allNames[j], allNames[i]
			}
		}
	}

	for _, name := range allNames {
		if lb.healthStatus[name] {
			healthy = append(healthy, name)
		}
	}
	return healthy
}

// selectRoundRobin selects a client using round-robin strategy
func (lb *defaultLoadBalancer) selectRoundRobin(healthyClients []string) string {
	if len(healthyClients) == 0 {
		return ""
	}

	selected := healthyClients[lb.roundRobinIdx%len(healthyClients)]
	lb.roundRobinIdx++
	return selected
}

// selectWeighted selects a client using weighted strategy
func (lb *defaultLoadBalancer) selectWeighted(healthyClients []string) string {
	if len(healthyClients) == 0 {
		return ""
	}

	// For now, implement simple weighted selection based on success rate
	var bestClient string
	var bestScore float64

	for _, name := range healthyClients {
		stats := lb.stats.ClientStats[name]
		if stats == nil {
			continue
		}

		// Calculate success rate
		totalRequests := stats.Successes + stats.Failures
		if totalRequests == 0 {
			// No history, give it a chance
			bestClient = name
			break
		}

		successRate := float64(stats.Successes) / float64(totalRequests)
		if successRate > bestScore {
			bestScore = successRate
			bestClient = name
		}
	}

	if bestClient == "" && len(healthyClients) > 0 {
		bestClient = healthyClients[0]
	}

	return bestClient
}

// selectLeastConnections selects a client with the least active connections
func (lb *defaultLoadBalancer) selectLeastConnections(healthyClients []string) string {
	if len(healthyClients) == 0 {
		return ""
	}

	// For now, use the client with the least total activity (successes + failures)
	var leastUsedClient string
	var leastActivity int64 = -1

	for _, name := range healthyClients {
		stats := lb.stats.ClientStats[name]
		if stats == nil {
			// No stats, this client hasn't been used
			return name
		}

		totalActivity := stats.Successes + stats.Failures
		if leastActivity == -1 || totalActivity < leastActivity {
			leastActivity = totalActivity
			leastUsedClient = name
		}
	}

	return leastUsedClient
}

// selectFailover selects the highest priority healthy client
func (lb *defaultLoadBalancer) selectFailover(healthyClients []string) string {
	if len(healthyClients) == 0 {
		return ""
	}

	// For failover, consistently select the first client alphabetically
	// This ensures consistent behavior across calls
	return healthyClients[0]
}

// updateClientStats updates statistics for a client (caller must hold read lock)
func (lb *defaultLoadBalancer) updateClientStats(clientName string) {
	lb.stats.TotalRequests++

	stats, exists := lb.stats.ClientStats[clientName]
	if !exists {
		stats = &ClientStats{
			HealthStatus: true,
		}
		lb.stats.ClientStats[clientName] = stats
	}

	stats.Requests++
	stats.LastUsed = time.Now()
}

// RecordSuccess records a successful request for a client
func (lb *defaultLoadBalancer) RecordSuccess(clientName string, responseTime time.Duration) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	stats, exists := lb.stats.ClientStats[clientName]
	if !exists {
		stats = &ClientStats{
			HealthStatus: true,
		}
		lb.stats.ClientStats[clientName] = stats
	}

	stats.Successes++

	// Update average response time
	if stats.AverageResponseTime == 0 {
		stats.AverageResponseTime = responseTime
	} else {
		// Simple moving average
		stats.AverageResponseTime = (stats.AverageResponseTime + responseTime) / 2
	}
}

// RecordFailure records a failed request for a client
func (lb *defaultLoadBalancer) RecordFailure(clientName string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	stats, exists := lb.stats.ClientStats[clientName]
	if !exists {
		return
	}

	stats.Failures++
}
