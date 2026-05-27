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
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		g.cfg.Model, g.cfg.APIKey,
	)
	body, _ := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": prompt}}},
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var res struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
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
