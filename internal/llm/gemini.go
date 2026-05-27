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

type geminiProvider struct {
	cfg config.GeminiConfig
}

func NewGemini(cfg config.GeminiConfig) Provider {
	return &geminiProvider{cfg: cfg}
}

func (g *geminiProvider) Name() string { return "gemini" }

func (g *geminiProvider) Available(_ context.Context) bool {
	return g.cfg.APIKey != ""
}

func (g *geminiProvider) Explain(ctx context.Context, prompt string) (string, error) {
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent",
		g.cfg.Model,
	)
	body, err := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": prompt}}},
		},
	})
	if err != nil {
		return "", fmt.Errorf("gemini: marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", g.cfg.APIKey)
	resp, err := apiClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("gemini: HTTP %d: %s", resp.StatusCode, raw)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("gemini: read: %w", err)
	}
	var res struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return "", fmt.Errorf("gemini: %w", err)
	}
	if res.Error != nil {
		return "", fmt.Errorf("gemini: %s", res.Error.Message)
	}
	if len(res.Candidates) == 0 || len(res.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: no content returned")
	}
	return res.Candidates[0].Content.Parts[0].Text, nil
}
