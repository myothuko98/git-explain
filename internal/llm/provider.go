package llm

import (
	"context"
	"fmt"

	"github.com/myothuko98/git-explain/internal/config"
)

// Provider is a single LLM backend.
type Provider interface {
	Name() string
	Available(ctx context.Context) bool
	Explain(ctx context.Context, prompt string) (string, error)
}

// Chain tries providers in order, falling back to rule-based.
func Chain(cfg config.Config) []Provider {
	return []Provider{
		NewOllama(cfg.Ollama),
		NewOpenAI(cfg.OpenAI),
		NewAnthropic(cfg.Anthropic),
		NewGemini(cfg.Gemini),
		NewRuleBased(),
	}
}

// Explain runs the prompt through the first available provider, falling back on error.
// Returns the explanation, the name of the provider that answered, and any error.
func Explain(ctx context.Context, cfg config.Config, prompt string) (result, providerName string, err error) {
	chain := Chain(cfg)

	if cfg.Provider != "auto" {
		for _, p := range chain {
			if p.Name() == cfg.Provider {
				r, e := p.Explain(ctx, prompt)
				return r, p.Name(), e
			}
		}
		return "", "", fmt.Errorf("provider %q not found or configured", cfg.Provider)
	}

	var lastErr error
	for _, p := range chain {
		if !p.Available(ctx) {
			continue
		}
		r, e := p.Explain(ctx, prompt)
		if e == nil {
			return r, p.Name(), nil
		}
		lastErr = e
	}
	if lastErr != nil {
		return "", "", fmt.Errorf("all providers failed, last error: %w", lastErr)
	}
	return "", "", fmt.Errorf("no provider available")
}
