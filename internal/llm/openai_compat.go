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
)

// openAICompatProvider implements both Provider and Streamer for any API that
// follows the OpenAI chat-completions protocol (same request/response shape,
// same SSE streaming format).  Qwen, Moonshot, and OpenAI itself all use this.
type openAICompatProvider struct {
	name    string
	baseURL string
	apiKey  string
	model   string
}

func (p *openAICompatProvider) Name() string { return p.name }

func (p *openAICompatProvider) Available(_ context.Context) bool {
	return p.apiKey != ""
}

func (p *openAICompatProvider) Explain(ctx context.Context, prompt string) (string, error) {
	body, err := json.Marshal(map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are an expert software engineer explaining git history. Be concise and precise."},
			{"role": "user", "content": prompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("%s: marshal: %w", p.name, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := apiClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("%s: HTTP %d: %s", p.name, resp.StatusCode, raw)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("%s: read: %w", p.name, err)
	}
	var res struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return "", fmt.Errorf("%s: %w", p.name, err)
	}
	if res.Error != nil {
		return "", fmt.Errorf("%s: %s", p.name, res.Error.Message)
	}
	if len(res.Choices) == 0 {
		return "", fmt.Errorf("%s: no choices returned", p.name)
	}
	return res.Choices[0].Message.Content, nil
}

// Stream implements Streamer using OpenAI-compatible SSE.
func (p *openAICompatProvider) Stream(ctx context.Context, prompt string, w io.Writer) error {
	body, err := json.Marshal(map[string]any{
		"model":  p.model,
		"stream": true,
		"messages": []map[string]string{
			{"role": "system", "content": "You are an expert software engineer explaining git history. Be concise and precise."},
			{"role": "user", "content": prompt},
		},
	})
	if err != nil {
		return fmt.Errorf("%s: marshal: %w", p.name, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := streamClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%s: HTTP %d: %s", p.name, resp.StatusCode, raw)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // allow up to 1MB per line
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk struct {
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if chunk.Error != nil {
			return fmt.Errorf("%s: %s", p.name, chunk.Error.Message)
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			_, _ = fmt.Fprint(w, chunk.Choices[0].Delta.Content)
		}
	}
	return scanner.Err()
}
