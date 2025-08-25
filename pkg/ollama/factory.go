package ollama

import (
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"os"
)

func New(model, server string) (llms.Model, error) {
	if model == "" {
		model = "llama3.2:1b"
	}
	if v := os.Getenv("OLLAMA_TEST_MODEL"); v != "" {
		model = v
	}

	llm, err := ollama.New(
		ollama.WithModel(model),
		ollama.WithServerURL(server),
		ollama.WithFormat("json"),
	)
	return llm, err
}
