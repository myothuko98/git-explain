package llm

import (
	"github.com/myothuko98/git-explain/internal/config"
)

type moonshotProvider struct{ *openAICompatProvider }

// NewMoonshot creates a Moonshot AI (Kimi) provider using its OpenAI-compatible endpoint.
// Get an API key at https://platform.moonshot.cn/
func NewMoonshot(cfg config.MoonshotConfig) Provider {
	return &moonshotProvider{&openAICompatProvider{
		name:    "moonshot",
		baseURL: "https://api.moonshot.cn/v1",
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
	}}
}
