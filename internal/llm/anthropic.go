package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/myothuko98/git-explain/internal/config"
)

type anthropicProvider struct {
	cfg config.AnthropicConfig
}

func NewAnthropic(cfg config.AnthropicConfig) Provider {
	return &anthropicProvider{cfg: cfg}
}

func (a *anthropicProvider) Name() string { return "anthropic" }

func (a *anthropicProvider) Available(_ context.Context) bool {
	return a.cfg.APIKey != ""
}

func (a *anthropicProvider) Explain(ctx context.Context, prompt string) (string, error) {
	body, err := json.Marshal(map[string]any{
		"model":      a.cfg.Model,
		"max_tokens": 1024,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("anthropic: marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	resp, err := apiClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("anthropic: HTTP %d: %s", resp.StatusCode, raw)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("anthropic: read: %w", err)
	}
	var res struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return "", fmt.Errorf("anthropic: %w", err)
	}
	if res.Error != nil {
		return "", fmt.Errorf("anthropic: %s", res.Error.Message)
	}
	if len(res.Content) == 0 {
		return "", fmt.Errorf("anthropic: no content returned")
	}
	return res.Content[0].Text, nil
}
