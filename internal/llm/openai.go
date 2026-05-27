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

type openAIProvider struct {
	cfg config.OpenAIConfig
}

func NewOpenAI(cfg config.OpenAIConfig) Provider {
	return &openAIProvider{cfg: cfg}
}

func (o *openAIProvider) Name() string { return "openai" }

func (o *openAIProvider) Available(_ context.Context) bool {
	return o.cfg.APIKey != ""
}

func (o *openAIProvider) Explain(ctx context.Context, prompt string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model": o.cfg.Model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are an expert software engineer explaining git history. Be concise and precise."},
			{"role": "user", "content": prompt},
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.cfg.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var res struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return "", fmt.Errorf("openai: %w", err)
	}
	if res.Error != nil {
		return "", fmt.Errorf("openai: %s", res.Error.Message)
	}
	if len(res.Choices) == 0 {
		return "", fmt.Errorf("openai: no choices returned")
	}
	return res.Choices[0].Message.Content, nil
}
