package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/myothuko98/git-explain/internal/config"
)

type ollamaProvider struct {
	cfg           config.OllamaConfig
	resolvedModel string // set by Available(); may differ from cfg.Model
}

func NewOllama(cfg config.OllamaConfig) Provider {
	return &ollamaProvider{cfg: cfg}
}

// ListModels returns the names of all models currently installed in Ollama.
// Returns nil if Ollama is unreachable or returns an error.
func ListModels(ctx context.Context, baseURL string) []string {
	c := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/tags", nil)
	if err != nil {
		return nil
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil
	}
	names := make([]string, 0, len(payload.Models))
	for _, m := range payload.Models {
		names = append(names, m.Name)
	}
	return names
}

func (o *ollamaProvider) Name() string { return "ollama" }

// model returns the resolved model name (auto-detected if configured model is absent).
func (o *ollamaProvider) model() string {
	if o.resolvedModel != "" {
		return o.resolvedModel
	}
	return o.cfg.Model
}

// ensureModel resolves the effective model name if not yet done.
// Called at the start of Explain/Stream so explicit --provider ollama also benefits.
func (o *ollamaProvider) ensureModel(ctx context.Context) {
	if o.resolvedModel == "" {
		_ = o.Available(ctx)
	}
}

// Available checks that Ollama is reachable and at least one chat model is
// present. If the configured model is not installed it auto-selects the first
// non-embedding model that is available.
func (o *ollamaProvider) Available(ctx context.Context) bool {
	c := &http.Client{Timeout: 2 * time.Second}

	// Quick liveness check
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.cfg.URL, nil)
	if err != nil {
		return false
	}
	resp, err := c.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		return false
	}

	// Fetch model list and resolve the effective model.
	tagsReq, err := http.NewRequestWithContext(ctx, http.MethodGet, o.cfg.URL+"/api/tags", nil)
	if err != nil {
		return true // liveness OK; proceed optimistically
	}
	tagsResp, err := c.Do(tagsReq)
	if err != nil {
		return true
	}
	defer tagsResp.Body.Close()

	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(tagsResp.Body).Decode(&payload); err != nil || len(payload.Models) == 0 {
		return false // Ollama running but no models installed
	}

	// Check if the configured model is available.
	for _, m := range payload.Models {
		if m.Name == o.cfg.Model || strings.HasPrefix(m.Name, o.cfg.Model+":") {
			o.resolvedModel = m.Name
			return true
		}
	}

	// Configured model not found — auto-select first non-embedding model.
	embeddingFamilies := []string{"nomic-bert", "bert", "embed"}
	for _, m := range payload.Models {
		isEmbed := false
		lower := strings.ToLower(m.Name)
		for _, e := range embeddingFamilies {
			if strings.Contains(lower, e) {
				isEmbed = true
				break
			}
		}
		if !isEmbed {
			o.resolvedModel = m.Name
			return true
		}
	}

	return false
}

func (o *ollamaProvider) Explain(ctx context.Context, prompt string) (string, error) {
	o.ensureModel(ctx)
	body, err := json.Marshal(map[string]any{
		"model":  o.model(),
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
	o.ensureModel(ctx)
	body, err := json.Marshal(map[string]any{
		"model":  o.model(),
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
			_, _ = fmt.Fprint(w, chunk.Response)
		}
		if chunk.Done {
			break
		}
	}
	return scanner.Err()
}
