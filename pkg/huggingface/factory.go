package huggingface

import (
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/huggingface"
)

func New(token string) (llms.Model, error) {
	llm, err := huggingface.New(huggingface.WithToken(token))

	return llm, err
}
