package llm

import (
	"github.com/myothuko98/git-explain/internal/config"
)

type openAIProvider struct{ *openAICompatProvider }

func NewOpenAI(cfg config.OpenAIConfig) Provider {
	return &openAIProvider{&openAICompatProvider{
		name:    "openai",
		baseURL: "https://api.openai.com/v1",
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
	}}
}
