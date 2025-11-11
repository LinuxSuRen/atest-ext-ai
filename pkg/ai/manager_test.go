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

func TestManagerSelectHealthyClientPrefersDefault(t *testing.T) {
	manager := &Manager{
		clients: map[string]interfaces.AIClient{
			"primary":   &fakeAIClient{},
			"secondary": &fakeAIClient{},
		},
		config: config.AIConfig{
			DefaultService: "primary",
		},
	}

	client := manager.selectHealthyClient()
	require.NotNil(t, client)
	assert.Same(t, manager.clients["primary"], client)
}

func TestManagerGetModelsReturnsProviderModels(t *testing.T) {
	expectedModels := []interfaces.ModelInfo{
		{ID: "model-1", Name: "Test Model"},
	}
	manager := &Manager{
		clients: map[string]interfaces.AIClient{
			"ollama": &fakeAIClient{
				capabilities: &interfaces.Capabilities{
					Provider: "ollama",
					Models:   expectedModels,
				},
			},
		},
	}

	models, err := manager.GetModels(context.Background(), "local")
	require.NoError(t, err)
	assert.Equal(t, expectedModels, models)

	_, err = manager.GetModels(context.Background(), "missing")
	assert.Error(t, err)
}

func TestManagerHealthCheckAllAggregatesStatuses(t *testing.T) {
	manager := &Manager{
		clients: map[string]interfaces.AIClient{
			"healthy": &fakeAIClient{
				healthStatus: &interfaces.HealthStatus{Healthy: true, Status: "ok"},
			},
			"unhealthy": &fakeAIClient{
				healthStatus: &interfaces.HealthStatus{Healthy: false, Status: "offline"},
			},
		},
	}

	results := manager.HealthCheckAll(context.Background())
	require.Len(t, results, 2)
	assert.True(t, results["healthy"].Healthy)
	assert.False(t, results["unhealthy"].Healthy)
}

type fakeAIClient struct {
	capabilities *interfaces.Capabilities
	healthStatus *interfaces.HealthStatus
	healthErr    error
	closeErr     error
}

func (f *fakeAIClient) Generate(_ context.Context, _ *interfaces.GenerateRequest) (*interfaces.GenerateResponse, error) {
	return nil, nil
}

func (f *fakeAIClient) GetCapabilities(_ context.Context) (*interfaces.Capabilities, error) {
	if f.capabilities == nil {
		return &interfaces.Capabilities{}, nil
	}
	return f.capabilities, nil
}

func (f *fakeAIClient) HealthCheck(_ context.Context) (*interfaces.HealthStatus, error) {
	if f.healthErr != nil {
		return nil, f.healthErr
	}
	if f.healthStatus == nil {
		return &interfaces.HealthStatus{
			Healthy:      true,
			Status:       "ok",
			LastChecked:  time.Now(),
			ResponseTime: time.Millisecond,
		}, nil
	}
	return f.healthStatus, nil
}

func (f *fakeAIClient) Close() error {
	return f.closeErr
}
