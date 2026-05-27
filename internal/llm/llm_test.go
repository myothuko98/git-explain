package llm_test

import (
	"context"
	"testing"

	"github.com/myothuko98/git-explain/internal/config"
	"github.com/myothuko98/git-explain/internal/llm"
)

func TestRuleBasedAlwaysAvailable(t *testing.T) {
	p := llm.NewRuleBased()
	if !p.Available(context.Background()) {
		t.Fatal("rule-based provider must always be available")
	}
}

func TestRuleBasedName(t *testing.T) {
	p := llm.NewRuleBased()
	if p.Name() != "rule-based" {
		t.Fatalf("expected 'rule-based', got %q", p.Name())
	}
}

func TestRuleBasedClassifiesFix(t *testing.T) {
	p := llm.NewRuleBased()
	out, err := p.Explain(context.Background(), "fix memory leak in connection pool")
	if err != nil {
		t.Fatal(err)
	}
	if out == "" {
		t.Fatal("expected non-empty output")
	}
	// Should mention Bug fix
	if !contains(out, "Bug fix") {
		t.Errorf("expected 'Bug fix' in output, got: %s", out)
	}
}

func TestRuleBasedClassifiesRefactor(t *testing.T) {
	p := llm.NewRuleBased()
	out, _ := p.Explain(context.Background(), "refactor auth middleware to reduce duplication")
	if !contains(out, "Refactor") {
		t.Errorf("expected 'Refactor' in output, got: %s", out)
	}
}

func TestRuleBasedClassifiesFeat(t *testing.T) {
	p := llm.NewRuleBased()
	out, _ := p.Explain(context.Background(), "feat: add OAuth2 support")
	if !contains(out, "feature") {
		t.Errorf("expected 'feature' in output, got: %s", out)
	}
}

func TestRuleBasedClassifiesRevert(t *testing.T) {
	p := llm.NewRuleBased()
	out, _ := p.Explain(context.Background(), "revert 'add broken feature'")
	if !contains(out, "Revert") {
		t.Errorf("expected 'Revert' in output, got: %s", out)
	}
}

func TestOllamaUnavailableWithoutServer(t *testing.T) {
	p := llm.NewOllama(config.OllamaConfig{URL: "http://localhost:19999", Model: "llama3"})
	if p.Available(context.Background()) {
		t.Skip("Ollama unexpectedly running on port 19999 — skipping")
	}
}

func TestOpenAIUnavailableWithoutKey(t *testing.T) {
	p := llm.NewOpenAI(config.OpenAIConfig{APIKey: "", Model: "gpt-4o-mini"})
	if p.Available(context.Background()) {
		t.Fatal("OpenAI should not be available without API key")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
