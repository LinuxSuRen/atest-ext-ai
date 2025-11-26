package ai

import (
	"context"
	"testing"
	"time"

	"github.com/linuxsuren/atest-ext-ai/pkg/config"
	"github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapabilityDetectorCachesResponses(t *testing.T) {
	manager := &Manager{
		clients: map[string]interfaces.AIClient{
			"alpha": &recordingClient{
				capabilities: &interfaces.Capabilities{
					Provider: "alpha",
					Models: []interfaces.ModelInfo{
						{ID: "model-a"},
					},
				},
			},
		},
	}

	detector := NewCapabilityDetector(config.AIConfig{}, manager)
	detector.SetCacheTTL(time.Hour)

	req := &CapabilitiesRequest{IncludeModels: true}
	resp, err := detector.GetCapabilities(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, resp.Models, 1)
	assert.Equal(t, "model-a", resp.Models[0].Name)

	client := manager.clients["alpha"].(*recordingClient)
	client.capabilities.Models = []interfaces.ModelInfo{{ID: "model-b"}}

	cachedResp, err := detector.GetCapabilities(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, resp.Models, cachedResp.Models)
	assert.Equal(t, 1, client.capabilityCalls)
}

func TestCapabilityDetectorInvalidateCache(t *testing.T) {
	manager := &Manager{
		clients: map[string]interfaces.AIClient{
			"alpha": &recordingClient{
				capabilities: &interfaces.Capabilities{
					Provider: "alpha",
					Models:   []interfaces.ModelInfo{{ID: "model-a"}},
				},
			},
		},
	}

	detector := NewCapabilityDetector(config.AIConfig{}, manager)
	detector.SetCacheTTL(time.Hour)

	req := &CapabilitiesRequest{IncludeModels: true}
	_, err := detector.GetCapabilities(context.Background(), req)
	require.NoError(t, err)

	client := manager.clients["alpha"].(*recordingClient)
	client.capabilities.Models = []interfaces.ModelInfo{{ID: "model-b"}}

	detector.InvalidateCache()
	resp, err := detector.GetCapabilities(context.Background(), req)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Models)
	assert.Equal(t, "model-b", resp.Models[0].Name)
	assert.Equal(t, 2, client.capabilityCalls)
}

type recordingClient struct {
	capabilities    *interfaces.Capabilities
	healthStatus    *interfaces.HealthStatus
	capabilityCalls int
}

func (r *recordingClient) Generate(_ context.Context, _ *interfaces.GenerateRequest) (*interfaces.GenerateResponse, error) {
	return nil, nil
}

func (r *recordingClient) GetCapabilities(_ context.Context) (*interfaces.Capabilities, error) {
	r.capabilityCalls++
	if r.capabilities == nil {
		return &interfaces.Capabilities{}, nil
	}
	return r.capabilities, nil
}

func (r *recordingClient) HealthCheck(_ context.Context) (*interfaces.HealthStatus, error) {
	if r.healthStatus == nil {
		return &interfaces.HealthStatus{Healthy: true}, nil
	}
	return r.healthStatus, nil
}

func (r *recordingClient) Close() error { return nil }
