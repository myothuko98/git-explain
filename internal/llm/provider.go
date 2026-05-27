package llm

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/myothuko98/git-explain/internal/cache"
	"github.com/myothuko98/git-explain/internal/config"
)

// Provider is a single LLM backend.
type Provider interface {
	Name() string
	Available(ctx context.Context) bool
	Explain(ctx context.Context, prompt string) (string, error)
}

// Streamer is an optional interface for providers that support token streaming.
type Streamer interface {
	Stream(ctx context.Context, prompt string, w io.Writer) error
}

// Chain returns providers in fallback order.
func Chain(cfg config.Config) []Provider {
	return []Provider{
		NewOllama(cfg.Ollama),
		NewOpenAI(cfg.OpenAI),
		NewAnthropic(cfg.Anthropic),
		NewGemini(cfg.Gemini),
		NewQwen(cfg.Qwen),
		NewMoonshot(cfg.Moonshot),
		NewRuleBased(),
	}
}

// Explain runs the prompt through the first available provider, with caching.
// Returns the explanation, the provider name, and any error.
func Explain(ctx context.Context, cfg config.Config, prompt string) (result, providerName string, err error) {
	chain := Chain(cfg)

	if cfg.Provider != "auto" {
		for _, p := range chain {
			if p.Name() == cfg.Provider {
				r, e := explainWithCache(ctx, p, prompt)
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
		r, e := explainWithCache(ctx, p, prompt)
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

// ExplainStream writes tokens to w as they arrive (if provider supports streaming),
// or writes the full result at once. Returns the provider name used.
func ExplainStream(ctx context.Context, cfg config.Config, prompt string, w io.Writer) (providerName string, err error) {
	chain := Chain(cfg)

	try := func(p Provider) (string, error) {
		cacheKey := cache.Key(p.Name(), prompt)
		if cached, ok := cache.Get(cacheKey, 0); ok {
			_, _ = fmt.Fprint(w, cached)
			return p.Name(), nil
		}
		if s, ok := p.(Streamer); ok {
			// For streaming providers, we tee output to cache after completion
			var buf strings.Builder
			tw := io.MultiWriter(w, &buf)
			if e := s.Stream(ctx, prompt, tw); e != nil {
				return "", e
			}
			_ = cache.Set(cacheKey, buf.String())
			return p.Name(), nil
		}
		// Non-streaming: get full result, write it, cache it
		result, e := p.Explain(ctx, prompt)
		if e != nil {
			return "", e
		}
		_, _ = fmt.Fprint(w, result)
		_ = cache.Set(cacheKey, result)
		return p.Name(), nil
	}

	if cfg.Provider != "auto" {
		for _, p := range chain {
			if p.Name() == cfg.Provider {
				return try(p)
			}
		}
		return "", fmt.Errorf("provider %q not found or configured", cfg.Provider)
	}

	var lastErr error
	for _, p := range chain {
		if !p.Available(ctx) {
			continue
		}
		name, e := try(p)
		if e == nil {
			return name, nil
		}
		lastErr = e
	}
	if lastErr != nil {
		return "", fmt.Errorf("all providers failed: %w", lastErr)
	}
	return "", fmt.Errorf("no provider available")
}

func explainWithCache(ctx context.Context, p Provider, prompt string) (string, error) {
	key := cache.Key(p.Name(), prompt)
	if cached, ok := cache.Get(key, 0); ok {
		return cached, nil
	}
	result, err := p.Explain(ctx, prompt)
	if err != nil {
		return "", err
	}
	_ = cache.Set(key, result)
	return result, nil
}
