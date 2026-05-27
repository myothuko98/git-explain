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

func TestRuleBasedOutputHasRiskLevel(t *testing.T) {
	p := llm.NewRuleBased()
	out, _ := p.Explain(context.Background(), "fix: nil pointer in auth middleware")
	if !contains(out, "Risk level") {
		t.Errorf("expected 'Risk level' section in output, got: %s", out)
	}
}

func TestRuleBasedOutputHasCauses(t *testing.T) {
	p := llm.NewRuleBased()
	out, _ := p.Explain(context.Background(), "fix: nil pointer in auth middleware")
	if !contains(out, "Common causes") {
		t.Errorf("expected 'Common causes' section in output, got: %s", out)
	}
}

func TestRuleBasedOutputHasChecklist(t *testing.T) {
	p := llm.NewRuleBased()
	out, _ := p.Explain(context.Background(), "fix: nil pointer in auth middleware")
	if !contains(out, "Review checklist") {
		t.Errorf("expected 'Review checklist' section in output, got: %s", out)
	}
}

func TestRuleBasedOutputHasScope(t *testing.T) {
	p := llm.NewRuleBased()
	out, _ := p.Explain(context.Background(), "fix: nil pointer in auth middleware")
	if !contains(out, "Scope") {
		t.Errorf("expected 'Scope' line in output, got: %s", out)
	}
}

func TestRuleBasedOutputScopeDetectsAuth(t *testing.T) {
	p := llm.NewRuleBased()
	out, _ := p.Explain(context.Background(), "fix: nil pointer in auth middleware")
	if !contains(out, "Auth") {
		t.Errorf("expected 'Auth' scope in output, got: %s", out)
	}
}

func TestRuleBasedBreakingChangeDetected(t *testing.T) {
	p := llm.NewRuleBased()
	out, _ := p.Explain(context.Background(), "feat!: remove legacy login API")
	if !contains(out, "YES") {
		t.Errorf("expected breaking change flag in output, got: %s", out)
	}
}

func TestRuleBasedNoBreakingFlagForNormal(t *testing.T) {
	p := llm.NewRuleBased()
	out, _ := p.Explain(context.Background(), "fix: correct timeout handling")
	if !contains(out, "Breaking:     No") {
		t.Errorf("expected 'Breaking: No' in output, got: %s", out)
	}
}

func TestRuleBasedConventionalCommitScope(t *testing.T) {
	p := llm.NewRuleBased()
	out, _ := p.Explain(context.Background(), "feat(payments): add Stripe integration")
	if !contains(out, "Payments") {
		t.Errorf("expected 'Payments' scope from conventional commit, got: %s", out)
	}
}

func TestRuleBasedPerformanceClassification(t *testing.T) {
	p := llm.NewRuleBased()
	out, _ := p.Explain(context.Background(), "perf: optimize database query with index")
	if !contains(out, "Performance") {
		t.Errorf("expected 'Performance' change type, got: %s", out)
	}
}

func TestRuleBasedDependencyClassification(t *testing.T) {
	p := llm.NewRuleBased()
	out, _ := p.Explain(context.Background(), "chore: bump golang.org/x/net from 0.15 to 0.17")
	// bump is scored for dependency; chore CC type → chore. Both valid.
	if !contains(out, "Dependency") && !contains(out, "Chore") {
		t.Errorf("expected Dependency or Chore classification, got: %s", out)
	}
}

func TestRuleBasedOutputContainsTip(t *testing.T) {
	p := llm.NewRuleBased()
	out, _ := p.Explain(context.Background(), "fix: something")
	if !contains(out, "git-explain setup") {
		t.Errorf("expected setup tip in output, got: %s", out)
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

func TestQwenUnavailableWithoutKey(t *testing.T) {
	p := llm.NewQwen(config.QwenConfig{APIKey: "", Model: "qwen-turbo"})
	if p.Available(context.Background()) {
		t.Fatal("Qwen should not be available without API key")
	}
}

func TestQwenAvailableWithKey(t *testing.T) {
	p := llm.NewQwen(config.QwenConfig{APIKey: "fake-key", Model: "qwen-turbo"})
	if !p.Available(context.Background()) {
		t.Fatal("Qwen should be available when API key is set")
	}
}

func TestQwenName(t *testing.T) {
	p := llm.NewQwen(config.QwenConfig{APIKey: "k", Model: "qwen-turbo"})
	if p.Name() != "qwen" {
		t.Fatalf("expected 'qwen', got %q", p.Name())
	}
}

func TestMoonshotUnavailableWithoutKey(t *testing.T) {
	p := llm.NewMoonshot(config.MoonshotConfig{APIKey: "", Model: "moonshot-v1-8k"})
	if p.Available(context.Background()) {
		t.Fatal("Moonshot should not be available without API key")
	}
}

func TestMoonshotAvailableWithKey(t *testing.T) {
	p := llm.NewMoonshot(config.MoonshotConfig{APIKey: "fake-key", Model: "moonshot-v1-8k"})
	if !p.Available(context.Background()) {
		t.Fatal("Moonshot should be available when API key is set")
	}
}

func TestMoonshotName(t *testing.T) {
	p := llm.NewMoonshot(config.MoonshotConfig{APIKey: "k", Model: "moonshot-v1-8k"})
	if p.Name() != "moonshot" {
		t.Fatalf("expected 'moonshot', got %q", p.Name())
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
