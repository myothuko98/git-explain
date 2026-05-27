package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/myothuko98/git-explain/internal/analyze"
	"github.com/myothuko98/git-explain/internal/cache"
	"github.com/myothuko98/git-explain/internal/config"
	gitpkg "github.com/myothuko98/git-explain/internal/git"
	"github.com/myothuko98/git-explain/internal/llm"
	"github.com/myothuko98/git-explain/internal/render"
)

// version is set at build time via -ldflags "-X main.version=v1.0.0".
var version = "dev"

func main() {
	root := &cobra.Command{
		Use:     "git-explain",
		Short:   "AI-powered git history explainer",
		Version: version,
		Long: `git-explain — understand why code changed, not just when.

Explains git blame lines, commits, pull requests, and team patterns
using your local LLM (Ollama) or any configured API provider.
Zero config required — falls back to rule-based analysis if no LLM is available.`,
		SilenceUsage: true,
	}

	root.AddCommand(
		blameCmd(),
		logCmd(),
		prCmd(),
		diffCmd(),
		patternsCmd(),
		setupCmd(),
		cacheCmd(),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// ── common flags ──────────────────────────────────────────────────────────────

type commonFlags struct {
	model   string
	jsonOut bool
}

func addCommonFlags(cmd *cobra.Command, f *commonFlags) {
	cmd.Flags().BoolVar(&f.jsonOut, "json", false, "Output as JSON instead of formatted text")
	cmd.Flags().StringVar(&f.model, "model", "", "Override LLM model for this request (e.g. gpt-4o, llama3.2)")
}

func applyModelOverride(cfg *config.Config, model string) {
	if model == "" {
		return
	}
	cfg.Ollama.Model = model
	cfg.OpenAI.Model = model
	cfg.Anthropic.Model = model
	cfg.Gemini.Model = model
	cfg.Qwen.Model = model
	cfg.Moonshot.Model = model
}

// explainAndRender calls the LLM (streaming when TTY) and renders result.
func explainAndRender(ctx context.Context, cfg config.Config, prompt, title, meta string, asJSON bool) error {
	if asJSON {
		explanation, providerName, err := llm.Explain(ctx, cfg, prompt)
		if err != nil {
			return err
		}
		render.ExplainJSON(title, meta, providerName, explanation)
		return nil
	}
	// Streaming mode
	render.StreamHeader(title, meta)
	providerName, err := llm.ExplainStream(ctx, cfg, prompt, render.StreamWriter())
	if err != nil {
		return err
	}
	render.StreamFooter(providerName)
	return nil
}

// ── blame ─────────────────────────────────────────────────────────────────────

func blameCmd() *cobra.Command {
	var f commonFlags
	cmd := &cobra.Command{
		Use:   "blame <file>:<line>[‑end]",
		Short: "Explain why a specific line (or range) exists",
		Args:  cobra.ExactArgs(1),
		Example: `  git-explain blame src/auth.go:42
  git-explain blame src/auth.go:40-55
  git-explain blame internal/db/conn.go:15 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			parts := strings.SplitN(args[0], ":", 2)
			if len(parts) != 2 {
				return fmt.Errorf("expected <file>:<line> or <file>:<start>-<end>, got %q", args[0])
			}
			file := parts[0]

			if _, err := gitpkg.TopDir(); err != nil {
				return err
			}

			cfg, _ := config.Load()
			applyModelOverride(&cfg, f.model)

			// Range or single line?
			lineSpec := parts[1]
			if strings.Contains(lineSpec, "-") {
				return blameRange(cmd.Context(), cfg, file, lineSpec, f.jsonOut)
			}
			lineNo, err := strconv.Atoi(lineSpec)
			if err != nil {
				return fmt.Errorf("invalid line: %q", lineSpec)
			}
			return blameSingle(cmd.Context(), cfg, file, lineNo, f.jsonOut)
		},
	}
	addCommonFlags(cmd, &f)
	return cmd
}

func blameSingle(ctx context.Context, cfg config.Config, file string, lineNo int, asJSON bool) error {
	blame, err := gitpkg.Blame(file, lineNo)
	if err != nil {
		return err
	}
	commit, err := gitpkg.Show(blame.SHA)
	if err != nil {
		return err
	}
	prompt := fmt.Sprintf(`You are an expert software engineer. Explain why this line of code exists.

File: %s
Line %d: %s

Subject: %s
Author: %s  Commit: %s
Body: %s

Diff (truncated):
%s

Explain in 3-5 sentences: what the commit did, why this line is the way it is, and any important context.`,
		file, lineNo, blame.LineText,
		commit.Subject, blame.Author, blame.SHA[:8],
		commit.Body, truncate(commit.Diff, 3000))

	title := fmt.Sprintf("Blame: %s:%d", file, lineNo)
	meta := fmt.Sprintf("commit %s  ·  %s", blame.SHA[:8], blame.Author)
	return explainAndRender(ctx, cfg, prompt, title, meta, asJSON)
}

func blameRange(ctx context.Context, cfg config.Config, file, lineSpec string, asJSON bool) error {
	rangeParts := strings.SplitN(lineSpec, "-", 2)
	if len(rangeParts) != 2 {
		return fmt.Errorf("invalid range: %q", lineSpec)
	}
	start, err := strconv.Atoi(rangeParts[0])
	if err != nil {
		return fmt.Errorf("invalid start line: %q", rangeParts[0])
	}
	end, err := strconv.Atoi(rangeParts[1])
	if err != nil {
		return fmt.Errorf("invalid end line: %q", rangeParts[1])
	}
	if start <= 0 || end <= 0 {
		return fmt.Errorf("line numbers must be positive (got %d-%d)", start, end)
	}
	if start > end {
		return fmt.Errorf("start line (%d) must not exceed end line (%d)", start, end)
	}

	results, err := gitpkg.BlameRange(file, start, end)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		return fmt.Errorf("no blame results for %s:%s", file, lineSpec)
	}

	// Collect unique SHAs
	seen := map[string]bool{}
	var lines []string
	for _, r := range results {
		lines = append(lines, fmt.Sprintf("  line %d [%s]: %s", r.LineNo, r.SHA[:8], r.LineText))
		seen[r.SHA] = true
	}

	// Get first commit detail for context
	first := results[0]
	commit, err := gitpkg.Show(first.SHA)
	if err != nil {
		return fmt.Errorf("failed to get commit details for %s: %w", first.SHA[:8], err)
	}

	prompt := fmt.Sprintf(`You are an expert software engineer. Explain why this block of code exists.

File: %s  Lines: %s

Code block:
%s

Primary commit: %s by %s
Subject: %s
Body: %s

Explain in 4-6 sentences: what this block does, why it was written this way, and what problem it solves.`,
		file, lineSpec,
		strings.Join(lines, "\n"),
		first.SHA[:8], first.Author,
		commit.Subject, commit.Body)

	title := fmt.Sprintf("Blame: %s:%s", file, lineSpec)
	meta := fmt.Sprintf("%d lines  ·  %d unique commits", len(results), len(seen))
	return explainAndRender(ctx, cfg, prompt, title, meta, asJSON)
}

// ── log ───────────────────────────────────────────────────────────────────────

func logCmd() *cobra.Command {
	var f commonFlags
	var rangeSpec string

	cmd := &cobra.Command{
		Use:   "log [commit-sha]",
		Short: "Explain what a commit (or commit range) changed and why",
		Args:  cobra.MaximumNArgs(1),
		Example: `  git-explain log a3f9b2c
  git-explain log HEAD~3
  git-explain log --range HEAD~5..HEAD
  git-explain log HEAD --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := gitpkg.TopDir(); err != nil {
				return err
			}
			cfg, _ := config.Load()
			applyModelOverride(&cfg, f.model)

			if rangeSpec != "" {
				return logRange(cmd.Context(), cfg, rangeSpec, f.jsonOut)
			}
			sha := "HEAD"
			if len(args) > 0 {
				sha = args[0]
			}
			return logSingle(cmd.Context(), cfg, sha, f.jsonOut)
		},
	}
	addCommonFlags(cmd, &f)
	cmd.Flags().StringVar(&rangeSpec, "range", "", "Explain a range of commits, e.g. HEAD~5..HEAD")
	return cmd
}

func logSingle(ctx context.Context, cfg config.Config, sha string, asJSON bool) error {
	commit, err := gitpkg.Show(sha)
	if err != nil {
		return err
	}
	prompt := fmt.Sprintf(`You are an expert software engineer. Explain this git commit.

Commit: %s  Author: %s  Date: %s
Subject: %s
Body: %s

Diff:
%s

Explain in 4-6 sentences: what changed, why it was needed, what problem it solves, and notable implementation decisions.`,
		commit.SHA[:8], commit.Author, commit.Date,
		commit.Subject, commit.Body,
		truncate(commit.Diff, 4000))

	title := commit.Subject
	meta := fmt.Sprintf("commit %s  ·  %s  ·  %s", commit.SHA[:8], commit.Author, commit.Date)
	return explainAndRender(ctx, cfg, prompt, title, meta, asJSON)
}

func logRange(ctx context.Context, cfg config.Config, rangeSpec string, asJSON bool) error {
	entries, err := gitpkg.LogRange(rangeSpec)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		render.Plain("No commits in range.")
		return nil
	}
	var summary []string
	for _, e := range entries {
		summary = append(summary, fmt.Sprintf("  %s  %s  (%s)", e.SHA[:8], e.Subject, e.Author))
	}
	prompt := fmt.Sprintf(`You are an expert software engineer. Summarize what happened across these %d commits.

Range: %s

Commits (newest first):
%s

Write a cohesive narrative (6-10 sentences): what was worked on, the overall goal, key decisions, and the end state after all these commits.`,
		len(entries), rangeSpec, strings.Join(summary, "\n"))

	title := fmt.Sprintf("Log range: %s (%d commits)", rangeSpec, len(entries))
	return explainAndRender(ctx, cfg, prompt, title, "", asJSON)
}

// ── pr ────────────────────────────────────────────────────────────────────────

func prCmd() *cobra.Command {
	var f commonFlags
	cmd := &cobra.Command{
		Use:   "pr <number>",
		Short: "Summarize a GitHub pull request",
		Args:  cobra.ExactArgs(1),
		Example: `  git-explain pr 42
  git-explain pr 100 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			num, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %q", args[0])
			}
			view, err := gitpkg.PRView(num)
			if err != nil {
				return err
			}
			diff, err := gitpkg.PRDiff(num)
			if err != nil {
				return err
			}
			cfg, _ := config.Load()
			applyModelOverride(&cfg, f.model)

			prompt := fmt.Sprintf(`You are an expert software engineer. Summarize this GitHub pull request.

PR metadata (JSON):
%s

Diff (truncated):
%s

5-8 sentences: purpose, what changed, key decisions, and what reviewers should focus on.`,
				view, truncate(diff, 4000))

			title := fmt.Sprintf("PR #%d", num)
			return explainAndRender(cmd.Context(), cfg, prompt, title, "", f.jsonOut)
		},
	}
	addCommonFlags(cmd, &f)
	return cmd
}

// ── diff ──────────────────────────────────────────────────────────────────────

func diffCmd() *cobra.Command {
	var f commonFlags
	var rangeSpec string

	cmd := &cobra.Command{
		Use:   "diff [range]",
		Short: "Explain the current working-tree diff or a commit range",
		Args:  cobra.MaximumNArgs(1),
		Example: `  git-explain diff
  git-explain diff HEAD~3..HEAD
  git-explain diff main..feature-branch --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := gitpkg.TopDir(); err != nil {
				return err
			}
			r := rangeSpec
			if len(args) > 0 {
				r = args[0]
			}
			diff, err := gitpkg.Diff(r)
			if err != nil {
				return err
			}
			cfg, _ := config.Load()
			applyModelOverride(&cfg, f.model)

			label := "working-tree changes"
			if r != "" {
				label = r
			}
			prompt := fmt.Sprintf(`You are an expert software engineer. Explain what these code changes do.

Range/context: %s

Diff:
%s

5-7 sentences: what was changed, why these changes make sense together, potential side-effects, and anything that needs attention.`,
				label, truncate(diff, 5000))

			title := "Diff explanation"
			meta := label
			return explainAndRender(cmd.Context(), cfg, prompt, title, meta, f.jsonOut)
		},
	}
	addCommonFlags(cmd, &f)
	cmd.Flags().StringVar(&rangeSpec, "range", "", "Commit range, e.g. HEAD~3..HEAD")
	return cmd
}

// ── patterns ──────────────────────────────────────────────────────────────────

func patternsCmd() *cobra.Command {
	var author string
	var limit int
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "patterns",
		Short: "Detect team coding habits from git history",
		Example: `  git-explain patterns
  git-explain patterns --author=alice
  git-explain patterns --limit=500 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := gitpkg.TopDir(); err != nil {
				return err
			}
			entries, err := gitpkg.LogAll(limit)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				render.Plain("No commits found.")
				return nil
			}
			patterns := analyze.TeamPatterns(entries, author)
			if len(patterns) == 0 {
				render.Plain("No matching authors found.")
				return nil
			}
			if asJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(patterns)
			}
			render.Header(fmt.Sprintf("Team Patterns  (%d commits analyzed)", len(entries)))
			for _, p := range patterns {
				render.Plain(analyze.FormatPattern(p))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&author, "author", "", "Filter to a specific author name")
	cmd.Flags().IntVar(&limit, "limit", 1000, "Max commits to analyze")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")
	return cmd
}

// ── setup ─────────────────────────────────────────────────────────────────────

// knownModels lists well-known models per provider for the setup picker.
var knownModels = map[string][]string{
	"openai": {
		"gpt-4o", "gpt-4o-mini", "o4-mini", "o3-mini",
		"o1", "o1-mini", "gpt-4-turbo", "gpt-3.5-turbo",
	},
	"anthropic": {
		"claude-opus-4-5", "claude-sonnet-4-5", "claude-haiku-4-5",
		"claude-3-5-sonnet-latest", "claude-3-5-haiku-latest",
	},
	"gemini": {
		"gemini-2.5-pro", "gemini-2.5-flash",
		"gemini-2.0-flash", "gemini-1.5-pro", "gemini-1.5-flash",
	},
	"qwen": {
		"qwen-max", "qwen-plus", "qwen-turbo",
		"qwen2.5-72b-instruct", "qwen2.5-coder-32b-instruct",
	},
	"moonshot": {
		"moonshot-v1-128k", "moonshot-v1-32k", "moonshot-v1-8k",
	},
}

// selectModel shows a numbered picker and returns the chosen model.
// models is the list to display; current is highlighted as default.
func selectModel(providerName string, models []string, current string) string {
	fmt.Printf("\n  Available models for %s:\n", providerName)
	for i, m := range models {
		marker := " "
		if m == current {
			marker = "*"
		}
		fmt.Printf("    %s %d. %s\n", marker, i+1, m)
	}
	fmt.Printf("    %s %d. Enter custom model name\n", " ", len(models)+1)
	fmt.Printf("  Select [1-%d] or press Enter to keep (%s): ", len(models)+1, current)

	line := readLine()
	if line == "" {
		return current
	}

	// Numeric selection
	var choice int
	if _, err := fmt.Sscanf(line, "%d", &choice); err == nil {
		if choice >= 1 && choice <= len(models) {
			return models[choice-1]
		}
		if choice == len(models)+1 {
			fmt.Print("  Custom model name: ")
			return readLine()
		}
	}

	// Non-numeric → treat as literal model name
	return line
}

func setupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Interactive setup wizard",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Start from existing config so re-runs don't wipe keys.
			cfg, err := config.Load()
			if err != nil {
				cfg = config.DefaultConfig()
			}

			render.Header("git-explain setup")
			fmt.Println()

			ctx := context.Background()

			// ── Ollama ──────────────────────────────────────────────────────
			fmt.Println("  ── Ollama (local, free) ──")
			ollamaOK := llm.NewOllama(cfg.Ollama).Available(ctx)
			if ollamaOK {
				fmt.Printf("  ✓ Ollama detected at %s\n", cfg.Ollama.URL)
				models := llm.ListModels(ctx, cfg.Ollama.URL)
				if len(models) > 0 {
					cfg.Ollama.Model = selectModel("Ollama", models, cfg.Ollama.Model)
				} else {
					fmt.Println("  (no models installed — run: ollama pull <model>)")
				}
			} else {
				fmt.Println("  ✗ Ollama not running (https://ollama.com)")
			}

			// ── OpenAI ──────────────────────────────────────────────────────
			fmt.Println("\n  ── OpenAI ──")
			fmt.Printf("  API key (Enter to keep existing%s): ",
				maskKey(cfg.OpenAI.APIKey))
			if k := readLine(); k != "" {
				cfg.OpenAI.APIKey = k
			}
			if cfg.OpenAI.APIKey != "" {
				cfg.OpenAI.Model = selectModel("OpenAI", knownModels["openai"], cfg.OpenAI.Model)
			}

			// ── Anthropic ───────────────────────────────────────────────────
			fmt.Println("\n  ── Anthropic ──")
			fmt.Printf("  API key (Enter to keep existing%s): ",
				maskKey(cfg.Anthropic.APIKey))
			if k := readLine(); k != "" {
				cfg.Anthropic.APIKey = k
			}
			if cfg.Anthropic.APIKey != "" {
				cfg.Anthropic.Model = selectModel("Anthropic", knownModels["anthropic"], cfg.Anthropic.Model)
			}

			// ── Gemini ──────────────────────────────────────────────────────
			fmt.Println("\n  ── Gemini ──")
			fmt.Printf("  API key (Enter to keep existing%s): ",
				maskKey(cfg.Gemini.APIKey))
			if k := readLine(); k != "" {
				cfg.Gemini.APIKey = k
			}
			if cfg.Gemini.APIKey != "" {
				cfg.Gemini.Model = selectModel("Gemini", knownModels["gemini"], cfg.Gemini.Model)
			}

			// ── Qwen ────────────────────────────────────────────────────────
			fmt.Println("\n  ── Qwen (Alibaba DashScope) ──")
			fmt.Printf("  API key (Enter to keep existing%s): ",
				maskKey(cfg.Qwen.APIKey))
			if k := readLine(); k != "" {
				cfg.Qwen.APIKey = k
			}
			if cfg.Qwen.APIKey != "" {
				cfg.Qwen.Model = selectModel("Qwen", knownModels["qwen"], cfg.Qwen.Model)
			}

			// ── Moonshot ────────────────────────────────────────────────────
			fmt.Println("\n  ── Moonshot (Kimi) ──")
			fmt.Printf("  API key (Enter to keep existing%s): ",
				maskKey(cfg.Moonshot.APIKey))
			if k := readLine(); k != "" {
				cfg.Moonshot.APIKey = k
			}
			if cfg.Moonshot.APIKey != "" {
				cfg.Moonshot.Model = selectModel("Moonshot", knownModels["moonshot"], cfg.Moonshot.Model)
			}

			cfg.Provider = "auto"
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			fmt.Println("\n  ✓ Config saved to " + config.ConfigPath())
			fmt.Println()
			render.Header("Active provider chain")
			for _, p := range llm.Chain(cfg) {
				status := "✗"
				if p.Available(ctx) {
					status = "✓"
				}
				fmt.Printf("  %s  %s\n", status, p.Name())
			}
			fmt.Println()
			return nil
		},
	}
}

// maskKey returns " (set)" if key is non-empty, else "" — never reveals key chars.
func maskKey(key string) string {
	if key != "" {
		return " (set)"
	}
	return ""
}

// ── cache ─────────────────────────────────────────────────────────────────────

func cacheCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cache clear",
		Short: "Manage the local response cache",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "clear":
				if err := clearCache(); err != nil {
					return err
				}
				fmt.Println("✓ Cache cleared.")
			default:
				return fmt.Errorf("unknown cache subcommand %q (available: clear)", args[0])
			}
			return nil
		},
	}
}

func clearCache() error {
	return cache.Clear()
}

// ── helpers ───────────────────────────────────────────────────────────────────

var stdinReader = bufio.NewReader(os.Stdin)

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n... (truncated)"
}

func readLine() string {
	s, _ := stdinReader.ReadString('\n')
	return strings.TrimSpace(s)
}
