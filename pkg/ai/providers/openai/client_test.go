package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHealthCheckSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/models", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	t.Cleanup(server.Close)

	client := &Client{
		config: &Config{
			APIKey:  "test",
			BaseURL: server.URL,
			Timeout: time.Second,
		},
	}

	status, err := client.HealthCheck(context.Background())
	require.NoError(t, err)
	require.NotNil(t, status)
	require.True(t, status.Healthy)
	require.Equal(t, "OK", status.Status)
}

func TestHealthCheckFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	client := &Client{
		config: &Config{
			APIKey:  "test",
			BaseURL: server.URL,
		},
	}

	status, err := client.HealthCheck(context.Background())
	require.NoError(t, err)
	require.NotNil(t, status)
	require.False(t, status.Healthy)
}
