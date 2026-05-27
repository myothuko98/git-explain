package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// BlameResult is the blame info for a single line.
type BlameResult struct {
	SHA        string
	Author     string
	AuthorMail string
	Summary    string
	LineNo     int
	LineText   string
}

// Blame returns blame info for <file> at <line> (1-indexed).
func Blame(file string, line int) (BlameResult, error) {
	out, err := run("git", "blame", "-p", fmt.Sprintf("-L%d,%d", line, line), "--", file)
	if err != nil {
		return BlameResult{}, fmt.Errorf("git blame: %w", err)
	}
	return parseBlame(out, line)
}

func parseBlame(out string, line int) (BlameResult, error) {
	lines := strings.Split(out, "\n")
	if len(lines) == 0 {
		return BlameResult{}, fmt.Errorf("empty blame output")
	}
	res := BlameResult{LineNo: line}
	// First line: <sha> <orig-line> <final-line> <count>
	fields := strings.Fields(lines[0])
	if len(fields) < 1 {
		return BlameResult{}, fmt.Errorf("unexpected blame output")
	}
	res.SHA = fields[0]

	for _, l := range lines[1:] {
		switch {
		case strings.HasPrefix(l, "author "):
			res.Author = strings.TrimPrefix(l, "author ")
		case strings.HasPrefix(l, "author-mail "):
			res.AuthorMail = strings.Trim(strings.TrimPrefix(l, "author-mail "), "<>")
		case strings.HasPrefix(l, "summary "):
			res.Summary = strings.TrimPrefix(l, "summary ")
		case strings.HasPrefix(l, "\t"):
			res.LineText = strings.TrimPrefix(l, "\t")
		}
	}
	return res, nil
}

// CommitDetail holds full info for a commit.
type CommitDetail struct {
	SHA     string
	Author  string
	Date    string
	Subject string
	Body    string
	Diff    string
}

// Show returns full detail for a commit SHA.
func Show(sha string) (CommitDetail, error) {
	// Format: sha\nauthor\ndate\nsubject\n\nbody\n---\ndiff
	format := "%H\n%an\n%ad\n%s\n\n%b"
	msgOut, err := run("git", "show", "--no-patch", fmt.Sprintf("--format=%s", format), sha)
	if err != nil {
		return CommitDetail{}, fmt.Errorf("git show: %w", err)
	}
	diffOut, err := run("git", "show", "--stat", "--patch", sha)
	if err != nil {
		return CommitDetail{}, fmt.Errorf("git show diff: %w", err)
	}

	parts := strings.SplitN(strings.TrimSpace(msgOut), "\n", 5)
	detail := CommitDetail{Diff: diffOut}
	if len(parts) > 0 {
		detail.SHA = parts[0]
	}
	if len(parts) > 1 {
		detail.Author = parts[1]
	}
	if len(parts) > 2 {
		detail.Date = parts[2]
	}
	if len(parts) > 3 {
		detail.Subject = parts[3]
	}
	if len(parts) > 4 {
		detail.Body = strings.TrimSpace(parts[4])
	}
	return detail, nil
}

// PRDiff returns the diff for a GitHub PR number using the `gh` CLI.
func PRDiff(number int) (string, error) {
	out, err := run("gh", "pr", "diff", strconv.Itoa(number))
	if err != nil {
		return "", fmt.Errorf("gh pr diff: %w\nMake sure `gh` is installed and authenticated", err)
	}
	return out, nil
}

// PRView returns the PR title + body.
func PRView(number int) (string, error) {
	out, err := run("gh", "pr", "view", strconv.Itoa(number), "--json", "title,body,author,additions,deletions,files")
	if err != nil {
		return "", fmt.Errorf("gh pr view: %w", err)
	}
	return out, nil
}

// LogAll returns all commits (sha + subject + author) for pattern analysis.
type LogEntry struct {
	SHA     string
	Author  string
	Subject string
	Date    string
}

func LogAll(limit int) ([]LogEntry, error) {
	args := []string{"log", "--format=%H\t%an\t%ad\t%s", "--date=short"}
	if limit > 0 {
		args = append(args, fmt.Sprintf("-n%d", limit))
	}
	out, err := run("git", args...)
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}
	var entries []LogEntry
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}
		entries = append(entries, LogEntry{
			SHA:     parts[0],
			Author:  parts[1],
			Date:    parts[2],
			Subject: parts[3],
		})
	}
	return entries, nil
}

// TopDir returns the root directory of the current git repo.
func TopDir() (string, error) {
	out, err := run("git", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("not inside a git repository")
	}
	return strings.TrimSpace(out), nil
}

func run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%s", strings.TrimSpace(string(ee.Stderr)))
		}
		return "", err
	}
	return string(out), nil
}
