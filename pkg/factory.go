package pkg

import (
	"github.com/linuxsuren/atest-ext-ai/pkg/huggingface"
	"github.com/tmc/langchaingo/llms"
)

var providers = map[string]func(string) (llms.Model, error){
	"huggingface": huggingface.New,
}
