package analyze_test

import (
	"strings"
	"testing"

	"github.com/myothuko98/git-explain/internal/analyze"
	gitpkg "github.com/myothuko98/git-explain/internal/git"
)

var sampleEntries = []gitpkg.LogEntry{
	{SHA: "a1", Author: "Alice", Date: "2024-01-01", Subject: "fix: memory leak in handler"},
	{SHA: "a2", Author: "Alice", Date: "2024-01-02", Subject: "fix: nil pointer in auth"},
	{SHA: "a3", Author: "Alice", Date: "2024-01-03", Subject: "feat: add user search"},
	{SHA: "a4", Author: "Alice", Date: "2024-01-04", Subject: "refactor: split handler"},
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
	// 2 out of 4 commits are fixes
	if alice.FixRatio < 0.49 || alice.FixRatio > 0.51 {
		t.Errorf("expected Alice fix ratio ~0.50, got %.2f", alice.FixRatio)
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
		TotalCommits:  4,
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

// ── file hotspot tests ────────────────────────────────────────────────────────

var hotspotEntries = []gitpkg.LogEntry{
	{SHA: "h1", Author: "Alice", Date: "2024-01-01", Subject: "fix: nil pointer in handler.go"},
	{SHA: "h2", Author: "Alice", Date: "2024-01-02", Subject: "fix: race in handler.go"},
	{SHA: "h3", Author: "Bob", Date: "2024-01-03", Subject: "refactor: clean handler.go"},
	{SHA: "h4", Author: "Alice", Date: "2024-01-04", Subject: "feat: add cache in redis.go"},
	{SHA: "h5", Author: "Bob", Date: "2024-01-05", Subject: "fix: nil in auth.go"},
}

func TestHotspotFilesTopFile(t *testing.T) {
	hotspots := analyze.HotspotFiles(hotspotEntries, 5)
	if len(hotspots) == 0 {
		t.Fatal("expected at least one hotspot")
	}
	if hotspots[0].Path != "handler.go" {
		t.Errorf("expected handler.go as top hotspot, got %q", hotspots[0].Path)
	}
	if hotspots[0].Changes != 3 {
		t.Errorf("expected 3 commits for handler.go, got %d", hotspots[0].Changes)
	}
}

func TestHotspotFilesLimit(t *testing.T) {
	hotspots := analyze.HotspotFiles(hotspotEntries, 1)
	if len(hotspots) != 1 {
		t.Errorf("expected 1 hotspot with limit=1, got %d", len(hotspots))
	}
}

func TestHotspotFilesEmpty(t *testing.T) {
	hotspots := analyze.HotspotFiles([]gitpkg.LogEntry{}, 5)
	if len(hotspots) != 0 {
		t.Errorf("expected no hotspots for empty entries, got %d", len(hotspots))
	}
}

func TestFormatHotspots(t *testing.T) {
	hotspots := []analyze.FileHotspot{
		{Path: "handler.go", Changes: 3},
		{Path: "auth.go", Changes: 1},
	}
	out := analyze.FormatHotspots(hotspots)
	if !strings.Contains(out, "handler.go") {
		t.Errorf("expected handler.go in hotspot output, got: %s", out)
	}
	if !strings.Contains(out, "3 commits") {
		t.Errorf("expected commit count in hotspot output, got: %s", out)
	}
}

func TestFormatHotspotsEmpty(t *testing.T) {
	out := analyze.FormatHotspots(nil)
	if out != "" {
		t.Errorf("expected empty string for nil hotspots, got: %q", out)
	}
}
