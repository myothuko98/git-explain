package llm

import (
	"bufio"
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.cfg.URL, nil)
	if err != nil {
		return false
	}
	c := &http.Client{Timeout: 2 * time.Second}
	resp, err := c.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

func (o *ollamaProvider) Explain(ctx context.Context, prompt string) (string, error) {
	body, err := json.Marshal(map[string]any{
		"model":  o.cfg.Model,
		"prompt": prompt,
		"stream": false,
	})
	if err != nil {
		return "", fmt.Errorf("ollama: marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.cfg.URL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := apiClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("ollama: HTTP %d: %s", resp.StatusCode, raw)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("ollama: read: %w", err)
	}
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

// Stream implements Streamer — writes tokens to w as they arrive.
func (o *ollamaProvider) Stream(ctx context.Context, prompt string, w io.Writer) error {
	body, err := json.Marshal(map[string]any{
		"model":  o.cfg.Model,
		"prompt": prompt,
		"stream": true,
	})
	if err != nil {
		return fmt.Errorf("ollama: marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.cfg.URL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := streamClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("ollama: HTTP %d: %s", resp.StatusCode, raw)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // allow up to 1MB per line
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var chunk struct {
			Response string `json:"response"`
			Error    string `json:"error"`
			Done     bool   `json:"done"`
		}
		if err := json.Unmarshal(line, &chunk); err != nil {
			continue
		}
		if chunk.Error != "" {
			return fmt.Errorf("ollama: %s", chunk.Error)
		}
		if chunk.Response != "" {
			fmt.Fprint(w, chunk.Response)
		}
		if chunk.Done {
			break
		}
	}
	return scanner.Err()
}
