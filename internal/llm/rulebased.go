package llm

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// ── public interface ──────────────────────────────────────────────────────────

type ruleBasedProvider struct{}

func NewRuleBased() Provider { return &ruleBasedProvider{} }

func (r *ruleBasedProvider) Name() string                     { return "rule-based" }
func (r *ruleBasedProvider) Available(_ context.Context) bool { return true }

func (r *ruleBasedProvider) Explain(_ context.Context, prompt string) (string, error) {
	subject := firstRelevantLine(prompt)
	cc := parseConventionalCommit(subject)
	kind := scoreChangeType(strings.ToLower(subject))
	scopes := collectScopes(strings.ToLower(prompt), cc.scope)
	breaking := cc.breaking || isBreakingChange(strings.ToLower(prompt))
	return buildOutput(kind, scopes, breaking), nil
}

// ── conventional commit parser ────────────────────────────────────────────────

// conventionalCommit holds parsed fields from a conventional commit subject.
type conventionalCommit struct {
	kind     string // feat, fix, chore, …
	scope    string // optional (scope) part
	desc     string // description after colon
	breaking bool   // true when ! suffix or BREAKING CHANGE present
}

// ccRe matches: type[!][(scope)][!]: description
var ccRe = regexp.MustCompile(`^([a-zA-Z]+)(\([^)]*\))?(!)?:\s*(.*)$`)

func parseConventionalCommit(subject string) conventionalCommit {
	m := ccRe.FindStringSubmatch(subject)
	if m == nil {
		return conventionalCommit{}
	}
	scope := strings.Trim(m[2], "()")
	return conventionalCommit{
		kind:     strings.ToLower(m[1]),
		scope:    scope,
		breaking: m[3] == "!",
		desc:     m[4],
	}
}

func isBreakingChange(text string) bool {
	return contains(text, "breaking change", "breaking-change") ||
		(contains(text, "remove") && contains(text, "api")) ||
		contains(text, "drop support", "deprecat")
}

// ── change type catalog ─────────────────────────────────────────────────────

type changeKind struct {
	name        string
	risk        string   // emoji + word
	description string   // 2-3 sentence explanation
	sideEffects string   // optional warning
	causes      []string // common reasons this change is made
	checklist   []string // review items
}

var changeKinds = map[string]changeKind{
	"bug-fix": {
		name:        "Bug Fix",
		risk:        "🟡 Medium",
		description: "Corrects incorrect or unexpected behavior in the codebase. Bug fixes address logic errors, nil/null dereferences, race conditions, resource leaks, or edge cases not covered by the original implementation.",
		causes: []string{
			"Edge case not handled in original implementation",
			"State mutation causing unexpected side effects",
			"Missing error-path cleanup (defer / Close / Rollback)",
			"Off-by-one or boundary condition error",
			"Concurrent access without proper synchronization",
		},
		checklist: []string{
			"Regression test that reproduces the original bug?",
			"All error paths handled, not just the happy path?",
			"Could this change affect behavior elsewhere in the system?",
			"Root cause fixed, not just symptom masked?",
		},
		sideEffects: "Bug fixes can change behavior that callers depended on (even if incorrectly). Verify downstream consumers.",
	},
	"new-feature": {
		name:        "New Feature",
		risk:        "🟡 Medium",
		description: "Introduces new functionality or capabilities that did not exist before. May add new API surfaces, endpoints, commands, UI flows, or integration points.",
		causes: []string{
			"Product requirement or user story",
			"Developer productivity or tooling improvement",
			"Integration with a new external service or library",
			"Extending existing functionality to cover a new use case",
			"Response to user feedback or community request",
		},
		checklist: []string{
			"Unit and integration tests cover the new feature?",
			"Error and edge cases handled gracefully?",
			"Feature is backward-compatible or behind a flag?",
			"Documentation updated (README, API docs, CHANGELOG)?",
		},
		sideEffects: "New features can increase binary size, startup time, or memory usage. Benchmark performance-sensitive paths.",
	},
	"refactor": {
		name:        "Refactor / Cleanup",
		risk:        "🟢 Low",
		description: "Restructures existing code without changing observable behavior. Improves readability, maintainability, or internal design without adding features or fixing bugs.",
		causes: []string{
			"Code duplication identified (DRY principle)",
			"Long function or file split for single-responsibility",
			"Rename to better reflect intent or domain language",
			"Extract reusable helper or abstraction",
			"Prepare codebase for an upcoming feature",
		},
		checklist: []string{
			"All existing tests still pass without modification?",
			"No observable behavior change (same inputs → same outputs)?",
			"Public API / exported symbols unchanged?",
			"Performance not regressed (run benchmarks if applicable)?",
		},
		sideEffects: "Refactors are generally safe but subtle logic bugs can hide inside renames or extractions. Review carefully.",
	},
	"test": {
		name:        "Test Change",
		risk:        "🟢 Low",
		description: "Adds, modifies, or removes tests. This includes unit tests, integration tests, end-to-end tests, benchmarks, fuzz tests, and test helpers.",
		causes: []string{
			"Increasing code coverage for existing behavior",
			"Regression test for a recently fixed bug",
			"Test added as part of TDD before implementation",
			"Flaky or non-deterministic test being stabilized",
			"Test infrastructure or fixture refactoring",
		},
		checklist: []string{
			"Tests actually assert meaningful behavior (not just coverage%)?",
			"Tests are deterministic and not time-dependent?",
			"No production code changed inadvertently?",
			"Test names clearly describe the scenario being tested?",
		},
		sideEffects: "Removing tests can silently reduce coverage. Confirm intent if tests are deleted.",
	},
	"documentation": {
		name:        "Documentation",
		risk:        "🟢 Low",
		description: "Updates comments, README, API docs, changelogs, or other documentation. No functional code changes are expected.",
		causes: []string{
			"New feature or API requires documentation",
			"Existing docs are outdated or incorrect",
			"Improving onboarding experience for contributors",
			"Adding examples or usage instructions",
			"Fixing typos or improving clarity",
		},
		checklist: []string{
			"No accidental production code changes sneaked in?",
			"Code examples in docs are correct and tested?",
			"Links and references are valid?",
		},
		sideEffects: "Generally no side effects. Watch for embedded code examples that may drift from reality.",
	},
	"dependency": {
		name:        "Dependency / Version Bump",
		risk:        "🟡 Medium",
		description: "Updates, adds, or removes a third-party dependency. May include lock-file updates, module version bumps, or security patches.",
		causes: []string{
			"Security vulnerability in current version (CVE fix)",
			"Bug fix or performance improvement in upstream library",
			"New feature in upstream needed by the project",
			"Keeping dependencies current to avoid drift",
			"Removing an unused or problematic dependency",
		},
		checklist: []string{
			"Changelog for the updated dependency reviewed?",
			"Breaking changes in the new version handled?",
			"Full test suite run against updated dependency?",
			"Lock file (go.sum / package-lock.json) updated consistently?",
		},
		sideEffects: "Even patch-level dependency updates can introduce subtle behavior changes. Run the full test suite.",
	},
	"ci-build": {
		name:        "CI / Build Change",
		risk:        "🟡 Medium",
		description: "Modifies the build system, CI/CD pipeline, deployment scripts, or tooling configuration. Does not change production application logic directly.",
		causes: []string{
			"Adding a new CI check (lint, security scan, coverage gate)",
			"Fixing a broken or flaky pipeline step",
			"Migrating CI provider or runner configuration",
			"Optimizing build time (caching, parallelism)",
			"Adding automated release or deployment step",
		},
		checklist: []string{
			"Pipeline runs successfully end-to-end?",
			"Secrets / environment variables correctly scoped?",
			"Build is reproducible (no non-deterministic steps)?",
			"Rollback procedure defined if deployment fails?",
		},
		sideEffects: "CI changes can silently disable checks. Verify all required gates are still enforced.",
	},
	"performance": {
		name:        "Performance Improvement",
		risk:        "🟡 Medium",
		description: "Optimizes speed, memory usage, throughput, or resource efficiency. Changes are often micro-level (algorithm, data structure) or macro-level (caching, batching).",
		causes: []string{
			"Profiling identified a hot path or allocation bottleneck",
			"User-reported latency or timeout issues",
			"Scaling requirement (more load, larger data sets)",
			"Replacing O(n²) algorithm with O(n log n)",
			"Adding or tuning a cache layer",
		},
		checklist: []string{
			"Benchmark results before and after documented?",
			"No correctness regression (perf changes often trade safety for speed)?",
			"Memory trade-offs acceptable (space vs time)?",
			"Works correctly under concurrent load?",
		},
		sideEffects: "Performance optimizations can introduce subtle correctness bugs. Prioritize correctness validation alongside benchmarks.",
	},
	"revert": {
		name:        "Revert",
		risk:        "🔴 High",
		description: "Undoes a previous commit or set of changes. Reverts are typically urgent responses to regressions, broken builds, or failed deployments.",
		causes: []string{
			"Regression introduced by the reverted change",
			"Failed deployment or production incident",
			"Accidental merge of unfinished work",
			"Breaking change that needs more design work",
		},
		checklist: []string{
			"Root cause of the original problem understood?",
			"All downstream changes that built on the reverted commit also reverted?",
			"Post-mortem or follow-up issue created?",
			"Original author notified?",
		},
		sideEffects: "Reverts in a shared branch can cause merge conflicts for in-flight work. Coordinate with the team.",
	},
	"chore": {
		name:        "Chore / Misc",
		risk:        "🟢 Low",
		description: "Routine maintenance tasks that do not directly affect production behavior. Includes formatting, tooling config, generated file updates, and housekeeping.",
		causes: []string{
			"Auto-generated code or proto files updated",
			"Code formatter applied (gofmt, prettier, black)",
			"Config file cleaned up or normalized",
			"Unused imports, variables, or dead code removed",
			"Repository housekeeping (gitignore, editor config)",
		},
		checklist: []string{
			"No unintended logic changes hidden in formatting noise?",
			"Generated files regenerated correctly from source?",
		},
		sideEffects: "Usually no risk, but large formatting diffs can obscure real logic changes during review.",
	},
}

// ── keyword scoring ───────────────────────────────────────────────────────────

// scoreMap maps keywords to change-type keys with a weight.
var scoreMap = []struct {
	key      string
	keywords []string
	weight   int
}{
	{key: "revert", keywords: []string{"revert", "undo", "rollback"}, weight: 10},
	{key: "bug-fix", keywords: []string{"fix", "bug", "hotfix", "patch", "correct", "regression", "broken", "crash", "nil pointer", "null pointer", "memory leak", "panic"}, weight: 3},
	{key: "new-feature", keywords: []string{"feat", "add", "new", "implement", "support", "introduc", "creat", "enable", "allow"}, weight: 2},
	{key: "refactor", keywords: []string{"refactor", "clean", "tidy", "rename", "reorgani", "restructur", "extract", "move", "split", "consolidat", "simplif"}, weight: 3},
	{key: "test", keywords: []string{"test", "spec", "coverage", "assert", "mock", "stub", "fixture", "benchmark", "fuzz"}, weight: 3},
	{key: "documentation", keywords: []string{"doc", "comment", "readme", "changelog", "license", "typo", "spell", "grammar"}, weight: 3},
	{key: "dependency", keywords: []string{"bump", "upgrade", "update", "version", "depend", "go.mod", "package.json", "requirements", "gemfile", "cargo.toml", "security", "cve", "vulnerabilit"}, weight: 3},
	{key: "ci-build", keywords: []string{"ci", "cd", "workflow", "action", "pipeline", "deploy", "build", "makefile", "dockerfile", "docker", "k8s", "kubernetes", "helm", "terraform", "ansible"}, weight: 3},
	{key: "performance", keywords: []string{"perf", "optim", "speed", "slow", "fast", "latency", "throughput", "memory", "cache", "allocat", "benchmark", "profil"}, weight: 3},
	{key: "chore", keywords: []string{"chore", "misc", "minor", "format", "lint", "generate", "generat", "proto", "tidy", "cleanup"}, weight: 2},
}

func scoreChangeType(text string) changeKind {
	scores := make(map[string]int, len(scoreMap))
	for _, entry := range scoreMap {
		for _, kw := range entry.keywords {
			if strings.Contains(text, kw) {
				scores[entry.key] += entry.weight
			}
		}
	}
	// Also honor conventional commit type directly
	cc := parseConventionalCommit(text)
	switch cc.kind {
	case "fix", "bugfix":
		scores["bug-fix"] += 10
	case "feat", "feature":
		scores["new-feature"] += 10
	case "refactor":
		scores["refactor"] += 10
	case "test", "tests":
		scores["test"] += 10
	case "docs", "doc":
		scores["documentation"] += 10
	case "chore":
		scores["chore"] += 10
	case "perf":
		scores["performance"] += 10
	case "ci", "build":
		scores["ci-build"] += 10
	case "revert":
		scores["revert"] += 10
	}

	best, bestScore := "chore", 0
	for k, s := range scores {
		if s > bestScore {
			best, bestScore = k, s
		}
	}
	if bestScore == 0 {
		// No match — default to "new-feature" heuristic
		best = "new-feature"
	}
	k, ok := changeKinds[best]
	if !ok {
		k = changeKinds["chore"]
	}
	return k
}

// ── scope detection ───────────────────────────────────────────────────────────

var fileExtScopes = []struct {
	ext  string
	desc string
}{
	{".go", "Go source"},
	{".ts", "TypeScript"},
	{".tsx", "TypeScript/React"},
	{".js", "JavaScript"},
	{".jsx", "JavaScript/React"},
	{".py", "Python"},
	{".rs", "Rust"},
	{".java", "Java"},
	{".rb", "Ruby"},
	{".php", "PHP"},
	{".swift", "Swift"},
	{".kt", "Kotlin"},
	{".cs", "C#"},
	{"_test.go", "Go tests"},
	{".spec.ts", "TypeScript tests"},
	{".spec.js", "JavaScript tests"},
	{"_test.py", "Python tests"},
	{".sql", "SQL / migrations"},
	{".proto", "Protobuf definitions"},
	{"dockerfile", "Docker"},
	{".yaml", "YAML config"},
	{".yml", "YAML config"},
	{".json", "JSON config"},
	{".toml", "TOML config"},
	{".tf", "Terraform"},
}

var domainScopes = []struct {
	desc     string
	keywords []string
}{
	{desc: "Auth", keywords: []string{"auth", "oauth", "jwt", "login", "logout", "session", "token"}},
	{desc: "API", keywords: []string{"api", "endpoint", "handler", "route", "controller"}},
	{desc: "Database", keywords: []string{"db", "database", "sql", "migration", "schema", "orm", "query"}},
	{desc: "Frontend", keywords: []string{"ui", "frontend", "component", "render", "style", "css", "html"}},
	{desc: "Backend", keywords: []string{"backend", "server", "service", "middleware"}},
	{desc: "Cache", keywords: []string{"cache", "redis", "memcache"}},
	{desc: "Queue/Async", keywords: []string{"queue", "worker", "job", "async", "celery", "kafka", "rabbitmq", "pubsub"}},
	{desc: "Config", keywords: []string{"config", "env", "settings", "flag"}},
	{desc: "Infrastructure", keywords: []string{"infra", "terraform", "k8s", "kubernetes", "helm", "docker", "deploy"}},
	{desc: "CI/CD", keywords: []string{"ci", "cd", "pipeline", "workflow", "action", "github"}},
	{desc: "Tests", keywords: []string{"test", "spec", "mock", "fixture", "coverage"}},
}

func collectScopes(text, ccScope string) []string {
	seen := make(map[string]bool)
	var scopes []string

	add := func(s string) {
		if !seen[s] {
			seen[s] = true
			scopes = append(scopes, s)
		}
	}

	// Scope from conventional commit e.g. feat(auth): ...
	if ccScope != "" {
		add(capitalise(ccScope))
	}

	// Domain keywords
	for _, ds := range domainScopes {
		for _, kw := range ds.keywords {
			if strings.Contains(text, kw) {
				add(ds.desc)
				break
			}
		}
	}

	// File extension hints
	for _, fe := range fileExtScopes {
		if strings.Contains(text, strings.ToLower(fe.ext)) {
			add(fe.desc)
			break // only first file type to avoid noise
		}
	}

	if len(scopes) == 0 {
		return []string{"General"}
	}
	return scopes
}

// ── output builder ────────────────────────────────────────────────────────────

func buildOutput(kind changeKind, scopes []string, breaking bool) string {
	var b strings.Builder

	breakStr := "No"
	if breaking {
		breakStr = "⚠  YES — this may be a breaking change"
	}

	b.WriteString("[rule-based analysis — no LLM configured]\n")
	b.WriteString(strings.Repeat("─", 62) + "\n")
	fmt.Fprintf(&b, "  Change type:  %s\n", kind.name)
	fmt.Fprintf(&b, "  Risk level:   %s\n", kind.risk)
	fmt.Fprintf(&b, "  Scope:        %s\n", strings.Join(scopes, " · "))
	fmt.Fprintf(&b, "  Breaking:     %s\n", breakStr)
	b.WriteString("\n")

	b.WriteString("  What this change likely does:\n")
	for _, line := range wordWrap(kind.description, 60) {
		fmt.Fprintf(&b, "  %s\n", line)
	}
	b.WriteString("\n")

	b.WriteString("  Common causes:\n")
	for _, c := range kind.causes {
		fmt.Fprintf(&b, "  • %s\n", c)
	}
	b.WriteString("\n")

	b.WriteString("  Review checklist:\n")
	for _, item := range kind.checklist {
		fmt.Fprintf(&b, "  ✓ %s\n", item)
	}

	if kind.sideEffects != "" {
		b.WriteString("\n")
		b.WriteString("  ⚠  Side-effects:\n")
		for _, line := range wordWrap(kind.sideEffects, 58) {
			fmt.Fprintf(&b, "  %s\n", line)
		}
	}

	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 62) + "\n")
	b.WriteString("  Tip: run `git-explain setup` to configure an LLM\n")
	b.WriteString("       for richer, context-aware explanations.\n")

	return b.String()
}

// wordWrap splits a string into lines of at most width runes.
func wordWrap(s string, width int) []string {
	words := strings.Fields(s)
	var lines []string
	current := ""
	for _, w := range words {
		if current == "" {
			current = w
		} else if len(current)+1+len(w) <= width {
			current += " " + w
		} else {
			lines = append(lines, current)
			current = w
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

// ── helpers ───────────────────────────────────────────────────────────────────

// firstRelevantLine returns the Subject: line if present, otherwise the first
// non-empty, non-instruction line from the prompt.
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

func contains(s string, keywords ...string) bool {
	for _, k := range keywords {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}

// capitalise uppercases the first rune of s.
func capitalise(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
