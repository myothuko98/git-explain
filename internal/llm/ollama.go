package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/myothuko98/git-explain/internal/config"
)

type ollamaProvider struct {
	cfg config.OllamaConfig
}

func NewOllama(cfg config.OllamaConfig) Provider {
	return &ollamaProvider{cfg: cfg}
}

func (o *ollamaProvider) Name() string { return "ollama" }

func (o *ollamaProvider) Available(ctx context.Context) bool {
	c := &http.Client{Timeout: 2 * time.Second}
	resp, err := c.Get(o.cfg.URL)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

func (o *ollamaProvider) Explain(ctx context.Context, prompt string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model":  o.cfg.Model,
		"prompt": prompt,
		"stream": false,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.cfg.URL+"/api/generate", bytes.NewReader(body))
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
		Response string `json:"response"`
		Error    string `json:"error"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return "", fmt.Errorf("ollama: %w", err)
	}
	if res.Error != "" {
		return "", fmt.Errorf("ollama: %s", res.Error)
	}
	return res.Response, nil
}
