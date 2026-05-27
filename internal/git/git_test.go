package git

import (
	"fmt"
	"strings"
	"testing"
)

// mockRunner replaces the package-level runner for a test, and restores it on cleanup.
func mockRunner(t *testing.T, responses map[string]string) {
	t.Helper()
	orig := runner
	runner = func(name string, args ...string) (string, error) {
		key := name + " " + strings.Join(args, " ")
		// Exact match first
		if v, ok := responses[key]; ok {
			return v, nil
		}
		// Prefix match (useful for dynamic args like -L40,55)
		for k, v := range responses {
			if strings.HasPrefix(key, k) {
				return v, nil
			}
		}
		return "", fmt.Errorf("mock: no response for %q", key)
	}
	t.Cleanup(func() { runner = orig })
}

// ── Blame ─────────────────────────────────────────────────────────────────────

const sampleBlameOutput = `abc1234567890123456789012345678901234567890 1 1 1
author Alice
author-mail <alice@example.com>
summary fix: correct token expiry check
	if token.ExpiresAt < now {
`

func TestBlame(t *testing.T) {
	mockRunner(t, map[string]string{
		"git blame -p -L1,1 -- src/auth.go": sampleBlameOutput,
	})

	res, err := Blame("src/auth.go", 1)
	if err != nil {
		t.Fatalf("Blame: %v", err)
	}
	if res.Author != "Alice" {
		t.Errorf("Author = %q, want %q", res.Author, "Alice")
	}
	if res.AuthorMail != "alice@example.com" {
		t.Errorf("AuthorMail = %q, want %q", res.AuthorMail, "alice@example.com")
	}
	if res.Summary != "fix: correct token expiry check" {
		t.Errorf("Summary = %q, want %q", res.Summary, "fix: correct token expiry check")
	}
	if !strings.Contains(res.LineText, "token.ExpiresAt") {
		t.Errorf("LineText = %q, want it to contain 'token.ExpiresAt'", res.LineText)
	}
}

func TestBlame_NotGitRepo(t *testing.T) {
	mockRunner(t, map[string]string{}) // no responses → always errors
	_, err := Blame("src/auth.go", 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ── BlameRange ────────────────────────────────────────────────────────────────

const sampleBlameRangeOutput = `abc1234567890123456789012345678901234567890 1 1 2
author Bob
author-mail <bob@example.com>
summary refactor: extract pool manager
	db.pool = newPool(cfg)
abc1234567890123456789012345678901234567890 2 2 2
author Bob
author-mail <bob@example.com>
summary refactor: extract pool manager
	db.pool.SetMaxConns(20)
`

func TestBlameRange(t *testing.T) {
	mockRunner(t, map[string]string{
		"git blame -p -L10,11 -- internal/db/conn.go": sampleBlameRangeOutput,
	})

	results, err := BlameRange("internal/db/conn.go", 10, 11)
	if err != nil {
		t.Fatalf("BlameRange: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Author != "Bob" {
		t.Errorf("results[0].Author = %q, want %q", results[0].Author, "Bob")
	}
	if results[0].LineNo != 10 {
		t.Errorf("results[0].LineNo = %d, want 10", results[0].LineNo)
	}
	if results[1].LineNo != 11 {
		t.Errorf("results[1].LineNo = %d, want 11", results[1].LineNo)
	}
}

// ── LogAll ────────────────────────────────────────────────────────────────────

const sampleLogOutput = `abc12345	Alice	2024-11-01	feat: add JWT auth
def56789	Bob	2024-10-28	fix: nil pointer in handler
`

func TestLogAll(t *testing.T) {
	mockRunner(t, map[string]string{
		"git log --format=%H\t%an\t%ad\t%s --date=short -n10": sampleLogOutput,
	})

	entries, err := LogAll(10)
	if err != nil {
		t.Fatalf("LogAll: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Author != "Alice" {
		t.Errorf("entries[0].Author = %q", entries[0].Author)
	}
	if entries[1].Subject != "fix: nil pointer in handler" {
		t.Errorf("entries[1].Subject = %q", entries[1].Subject)
	}
}

func TestLogAll_Empty(t *testing.T) {
	mockRunner(t, map[string]string{
		"git log --format=%H\t%an\t%ad\t%s --date=short -n5": "",
	})
	entries, err := LogAll(5)
	if err != nil {
		t.Fatalf("LogAll empty: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

// ── LogRange ──────────────────────────────────────────────────────────────────

func TestLogRange(t *testing.T) {
	mockRunner(t, map[string]string{
		"git log --format=%H\t%an\t%ad\t%s --date=short HEAD~2..HEAD": sampleLogOutput,
	})

	entries, err := LogRange("HEAD~2..HEAD")
	if err != nil {
		t.Fatalf("LogRange: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

// ── Diff ──────────────────────────────────────────────────────────────────────

func TestDiff_WorkingTree(t *testing.T) {
	mockRunner(t, map[string]string{
		"git diff HEAD": "diff --git a/foo.go b/foo.go\n+// new comment\n",
	})

	out, err := Diff("")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if !strings.Contains(out, "foo.go") {
		t.Errorf("expected foo.go in diff output, got %q", out)
	}
}

func TestDiff_Range(t *testing.T) {
	mockRunner(t, map[string]string{
		"git diff HEAD~3..HEAD": "diff --git a/bar.go b/bar.go\n-old\n+new\n",
	})

	out, err := Diff("HEAD~3..HEAD")
	if err != nil {
		t.Fatalf("Diff range: %v", err)
	}
	if !strings.Contains(out, "bar.go") {
		t.Errorf("expected bar.go in diff output, got %q", out)
	}
}

func TestDiff_NoChanges(t *testing.T) {
	mockRunner(t, map[string]string{
		"git diff HEAD":     "",
		"git diff --cached": "",
	})

	_, err := Diff("")
	if err == nil {
		t.Fatal("expected error for no changes, got nil")
	}
}

// ── TopDir ────────────────────────────────────────────────────────────────────

func TestTopDir(t *testing.T) {
	mockRunner(t, map[string]string{
		"git rev-parse --show-toplevel": "/home/user/myproject\n",
	})

	dir, err := TopDir()
	if err != nil {
		t.Fatalf("TopDir: %v", err)
	}
	if dir != "/home/user/myproject" {
		t.Errorf("TopDir = %q, want %q", dir, "/home/user/myproject")
	}
}

func TestTopDir_NotGitRepo(t *testing.T) {
	mockRunner(t, map[string]string{}) // always errors
	_, err := TopDir()
	if err == nil {
		t.Fatal("expected error outside git repo")
	}
}

// ── isHex ─────────────────────────────────────────────────────────────────────

func TestIsHex(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"abc1234567890abcdef", true},
		{"ABCDEF", false}, // uppercase not valid git sha
		{"xyz", false},
		{"0000000000000000", true},
	}
	for _, c := range cases {
		got := isHex(c.input)
		if got != c.want {
			t.Errorf("isHex(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}
