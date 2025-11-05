package ai

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRuntimeClientReuseAndClose(t *testing.T) {
	generator := &SQLGenerator{
		runtimeClients: make(map[string]*runtimeClientEntry),
	}

	options := &GenerateOptions{
		Provider:  "ollama",
		APIKey:    "test-key",
		Endpoint:  "http://localhost:11434",
		MaxTokens: 512,
	}

	client1, reused1, err := generator.getOrCreateRuntimeClient(options)
	require.NoError(t, err)
	require.False(t, reused1)
	require.NotNil(t, client1)

	client2, reused2, err := generator.getOrCreateRuntimeClient(options)
	require.NoError(t, err)
	require.True(t, reused2)
	require.Equal(t, client1, client2)

	generator.Close()

	_, reused3, err := generator.getOrCreateRuntimeClient(options)
	require.NoError(t, err)
	require.False(t, reused3)
}
