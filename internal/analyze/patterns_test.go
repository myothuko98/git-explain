package analyze_test

import (
	"testing"

	"github.com/myothuko98/git-explain/internal/analyze"
	gitpkg "github.com/myothuko98/git-explain/internal/git"
)

var sampleEntries = []gitpkg.LogEntry{
	{SHA: "a1", Author: "Alice", Date: "2024-01-01", Subject: "fix: memory leak in handler"},
	{SHA: "a2", Author: "Alice", Date: "2024-01-02", Subject: "fix: nil pointer in auth"},
	{SHA: "a3", Author: "Alice", Date: "2024-01-03", Subject: "feat: add user search"},
	{SHA: "b1", Author: "Bob", Date: "2024-01-01", Subject: "refactor: clean up DB layer"},
	{SHA: "b2", Author: "Bob", Date: "2024-01-02", Subject: "docs: update README"},
	{SHA: "b3", Author: "Bob", Date: "2024-01-03", Subject: "refactor: rename user model"},
}

func TestTeamPatterns_AllAuthors(t *testing.T) {
	patterns := analyze.TeamPatterns(sampleEntries, "")
	if len(patterns) != 2 {
		t.Fatalf("expected 2 authors, got %d", len(patterns))
	}
	// Alice has most commits
	if patterns[0].Author != "Alice" {
		t.Errorf("expected Alice first, got %s", patterns[0].Author)
	}
}

func TestTeamPatterns_FilterAuthor(t *testing.T) {
	patterns := analyze.TeamPatterns(sampleEntries, "Bob")
	if len(patterns) != 1 {
		t.Fatalf("expected 1 author, got %d", len(patterns))
	}
	if patterns[0].Author != "Bob" {
		t.Errorf("expected Bob, got %s", patterns[0].Author)
	}
}

func TestTeamPatterns_FixRatio(t *testing.T) {
	patterns := analyze.TeamPatterns(sampleEntries, "Alice")
	alice := patterns[0]
	// 2 out of 3 commits are fixes
	if alice.FixRatio < 0.6 || alice.FixRatio > 0.7 {
		t.Errorf("expected Alice fix ratio ~0.67, got %.2f", alice.FixRatio)
	}
}

func TestTeamPatterns_RefactorRatio(t *testing.T) {
	patterns := analyze.TeamPatterns(sampleEntries, "Bob")
	bob := patterns[0]
	// 2 out of 3 are refactors
	if bob.RefactorRatio < 0.6 || bob.RefactorRatio > 0.7 {
		t.Errorf("expected Bob refactor ratio ~0.67, got %.2f", bob.RefactorRatio)
	}
}

func TestFormatPattern(t *testing.T) {
	p := analyze.AuthorPattern{
		Author:        "Alice",
		TotalCommits:  3,
		FixRatio:      0.67,
		RefactorRatio: 0.0,
		TopKeywords:   []string{"fix", "memory", "auth"},
	}
	out := analyze.FormatPattern(p)
	if out == "" {
		t.Fatal("expected non-empty format output")
	}
	if len(out) < 10 {
		t.Errorf("format output too short: %q", out)
	}
}
