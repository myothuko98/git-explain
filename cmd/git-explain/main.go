package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/myothuko98/git-explain/internal/analyze"
	gitpkg "github.com/myothuko98/git-explain/internal/git"
	"github.com/myothuko98/git-explain/internal/llm"
	"github.com/myothuko98/git-explain/internal/config"
	"github.com/myothuko98/git-explain/internal/render"
)

func main() {
	root := &cobra.Command{
		Use:   "git-explain",
		Short: "AI-powered git history explainer",
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
		patternsCmd(),
		setupCmd(),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// ── blame ─────────────────────────────────────────────────────────────────────

func blameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "blame <file>:<line>",
		Short: "Explain why a specific line exists",
		Args:  cobra.ExactArgs(1),
		Example: `  git-explain blame src/auth.go:42
  git-explain blame internal/db/conn.go:15`,
		RunE: func(cmd *cobra.Command, args []string) error {
			parts := strings.SplitN(args[0], ":", 2)
			if len(parts) != 2 {
				return fmt.Errorf("expected <file>:<line>, got %q", args[0])
			}
			file := parts[0]
			lineNo, err := strconv.Atoi(parts[1])
			if err != nil {
				return fmt.Errorf("invalid line number: %q", parts[1])
			}

			if _, err := gitpkg.TopDir(); err != nil {
				return err
			}

			blame, err := gitpkg.Blame(file, lineNo)
			if err != nil {
				return err
			}

			commit, err := gitpkg.Show(blame.SHA)
			if err != nil {
				return err
			}

			cfg, _ := config.Load()
			prompt := fmt.Sprintf(`You are an expert software engineer. Explain why this line of code exists and what change introduced it.

File: %s
Line %d: %s

This line was last changed in commit %s by %s.
Commit message: %s
%s

Diff (truncated to 3000 chars):
%s

Explain in 3-5 sentences: what the commit did, why this line is the way it is, and any important context.`,
				file, lineNo, blame.LineText,
				blame.SHA[:8], blame.Author,
				commit.Subject, commit.Body,
				truncate(commit.Diff, 3000),
			)

			ctx := context.Background()
			explanation, providerName, err := llm.Explain(ctx, cfg, prompt)
			if err != nil {
				return err
			}

			meta := fmt.Sprintf("%s:%d  ·  commit %s  ·  %s", file, lineNo, blame.SHA[:8], blame.Author)
			render.ExplainResult("Blame Explanation", meta, providerName, explanation)
			return nil
		},
	}
}

// ── log ───────────────────────────────────────────────────────────────────────

func logCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "log <commit-sha>",
		Short: "Explain what a commit changed and why",
		Args:  cobra.ExactArgs(1),
		Example: `  git-explain log a3f9b2c
  git-explain log HEAD~3`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := gitpkg.TopDir(); err != nil {
				return err
			}

			commit, err := gitpkg.Show(args[0])
			if err != nil {
				return err
			}

			cfg, _ := config.Load()
			prompt := fmt.Sprintf(`You are an expert software engineer. Explain this git commit clearly and concisely.

Commit: %s
Author: %s
Date: %s
Subject: %s
Body: %s

Diff:
%s

Explain in 4-6 sentences: what changed, why it was needed, what problem it solves, and any notable implementation decisions.`,
				commit.SHA[:8], commit.Author, commit.Date,
				commit.Subject, commit.Body,
				truncate(commit.Diff, 4000),
			)

			ctx := context.Background()
			explanation, providerName, err := llm.Explain(ctx, cfg, prompt)
			if err != nil {
				return err
			}

			meta := fmt.Sprintf("commit %s  ·  %s  ·  %s", commit.SHA[:8], commit.Author, commit.Date)
			render.ExplainResult(commit.Subject, meta, providerName, explanation)
			return nil
		},
	}
}

// ── pr ────────────────────────────────────────────────────────────────────────

func prCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pr <number>",
		Short: "Summarise a GitHub pull request",
		Args:  cobra.ExactArgs(1),
		Example: `  git-explain pr 42
  git-explain pr 100`,
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
			prompt := fmt.Sprintf(`You are an expert software engineer. Summarise this GitHub pull request.

PR metadata (JSON):
%s

Diff (truncated to 4000 chars):
%s

Provide a concise summary (5-8 sentences): purpose of the PR, what was changed, key design decisions, and potential risks or things reviewers should pay attention to.`,
				view, truncate(diff, 4000),
			)

			ctx := context.Background()
			explanation, providerName, err := llm.Explain(ctx, cfg, prompt)
			if err != nil {
				return err
			}

			render.ExplainResult(fmt.Sprintf("PR #%d Summary", num), "", providerName, explanation)
			return nil
		},
	}
}

// ── patterns ──────────────────────────────────────────────────────────────────

func patternsCmd() *cobra.Command {
	var author string
	var limit int

	cmd := &cobra.Command{
		Use:   "patterns",
		Short: "Detect team coding habits from git history",
		Example: `  git-explain patterns
  git-explain patterns --author=alice
  git-explain patterns --limit=500`,
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

			render.Header(fmt.Sprintf("Team Patterns  (%d commits analysed)", len(entries)))
			for _, p := range patterns {
				render.Plain(analyze.FormatPattern(p))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&author, "author", "", "Filter to a specific author name")
	cmd.Flags().IntVar(&limit, "limit", 1000, "Max commits to analyse")
	return cmd
}

// ── setup ─────────────────────────────────────────────────────────────────────

func setupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Interactive setup wizard",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.DefaultConfig()

			render.Header("git-explain setup")
			fmt.Println()

			// Detect Ollama
			ctx := context.Background()
			ollamaOK := llm.NewOllama(cfg.Ollama).Available(ctx)
			if ollamaOK {
				fmt.Println("  ✓ Ollama detected at " + cfg.Ollama.URL)
				fmt.Println("    Using model: " + cfg.Ollama.Model)
			} else {
				fmt.Println("  ✗ Ollama not found (install from https://ollama.com for free local LLM)")
			}
			fmt.Println()

			// Prompt for API keys
			fmt.Print("  OpenAI API key (leave blank to skip): ")
			cfg.OpenAI.APIKey = readLine()

			fmt.Print("  Anthropic API key (leave blank to skip): ")
			cfg.Anthropic.APIKey = readLine()

			fmt.Print("  Gemini API key (leave blank to skip): ")
			cfg.Gemini.APIKey = readLine()

			cfg.Provider = "auto"

			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Println()
			fmt.Println("  ✓ Config saved to " + config.ConfigPath())
			fmt.Println()

			// Show active chain
			render.Header("Active provider chain")
			for _, p := range llm.Chain(cfg) {
				avail := p.Available(ctx)
				status := "✗"
				if avail {
					status = "✓"
				}
				fmt.Printf("  %s  %s\n", status, p.Name())
			}
			fmt.Println()
			return nil
		},
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n... (truncated)"
}

func readLine() string {
	var s string
	fmt.Scanln(&s)
	return strings.TrimSpace(s)
}
