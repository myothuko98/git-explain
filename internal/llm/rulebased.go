package llm

import (
	"context"
	"fmt"
	"strings"
)

type ruleBasedProvider struct{}

func NewRuleBased() Provider { return &ruleBasedProvider{} }

func (r *ruleBasedProvider) Name() string                      { return "rule-based" }
func (r *ruleBasedProvider) Available(_ context.Context) bool  { return true }

func (r *ruleBasedProvider) Explain(_ context.Context, prompt string) (string, error) {
	// Extract the most relevant line for classification:
	// prefer "Subject:" line if present, otherwise use first non-empty line.
	classifyText := firstRelevantLine(prompt)
	kind := classifyChange(strings.ToLower(classifyText))
	scope := extractScope(strings.ToLower(prompt))
	return fmt.Sprintf("[rule-based — no LLM configured]\n\nChange type: %s\nScope: %s\n\nTip: run `git-explain setup` to configure an LLM for richer explanations.", kind, scope), nil
}

// firstRelevantLine returns the Subject line from the prompt if available,
// otherwise the first non-empty, non-instruction line.
func firstRelevantLine(prompt string) string {
	for _, line := range strings.Split(prompt, "\n") {
		if strings.HasPrefix(line, "Subject: ") {
			return strings.TrimPrefix(line, "Subject: ")
		}
	}
	for _, line := range strings.Split(prompt, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "You are") && !strings.HasPrefix(line, "Explain") {
			return line
		}
	}
	return prompt
}

func classifyChange(text string) string {
	switch {
	case contains(text, "revert"):
		return "Revert — undoes a previous change"
	case contains(text, "fix", "bug", "hotfix", "patch", "correct"):
		return "Bug fix"
	case contains(text, "refactor", "clean", "tidy", "rename", "reorgani"):
		return "Refactor / cleanup"
	case contains(text, "feat", "add", "new", "implement", "support", "introduc"):
		return "New feature"
	case contains(text, "test", "spec", "coverage"):
		return "Test change"
	case contains(text, "doc", "comment", "readme", "changelog"):
		return "Documentation"
	case contains(text, "bump", "upgrade", "update", "version", "depend"):
		return "Dependency / version bump"
	case contains(text, "ci", "workflow", "action", "pipeline", "deploy", "build"):
		return "CI / build change"
	case contains(text, "perf", "optim", "speed", "slow", "memory", "cache"):
		return "Performance improvement"
	case contains(text, "chore", "misc", "minor"):
		return "Chore / misc"
	default:
		return "General change"
	}
}

func extractScope(text string) string {
	// Look for common file extension context clues
	for _, ext := range []string{".go", ".ts", ".js", ".py", ".rs", ".java", ".rb", ".php"} {
		if strings.Contains(text, ext) {
			return "Code change (" + ext + " files)"
		}
	}
	for _, kw := range []string{"api", "auth", "db", "database", "ui", "frontend", "backend", "config", "infra"} {
		if strings.Contains(text, kw) {
			return strings.ToUpper(kw[:1]) + kw[1:] + " layer"
		}
	}
	return "Unknown scope"
}

func contains(s string, keywords ...string) bool {
	for _, k := range keywords {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}
