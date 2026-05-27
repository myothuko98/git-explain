package llm

import (
	"github.com/myothuko98/git-explain/internal/config"
)

type qwenProvider struct{ *openAICompatProvider }

// NewQwen creates a Qwen provider using Alibaba DashScope's OpenAI-compatible endpoint.
// Get an API key at https://dashscope.aliyuncs.com/
func NewQwen(cfg config.QwenConfig) Provider {
	return &qwenProvider{&openAICompatProvider{
		name:    "qwen",
		baseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
	}}
}
