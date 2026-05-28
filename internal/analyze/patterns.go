package analyze

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	gitpkg "github.com/myothuko98/git-explain/internal/git"
)

// AuthorPattern holds stats for a single author.
type AuthorPattern struct {
	Author        string
	TopKeywords   []string
	TotalCommits  int
	FixRatio      float64
	RefactorRatio float64
}

// TeamPatterns analyzes commit log entries and returns per-author stats.
func TeamPatterns(entries []gitpkg.LogEntry, filterAuthor string) []AuthorPattern {
	type stats struct {
		keywords                map[string]int
		total, fixes, refactors int
	}
	authorMap := map[string]*stats{}
	for _, e := range entries {
		if filterAuthor != "" && !strings.EqualFold(e.Author, filterAuthor) {
			continue
		}
		s, ok := authorMap[e.Author]
		if !ok {
			s = &stats{keywords: map[string]int{}}
			authorMap[e.Author] = s
		}
		s.total++
		lower := strings.ToLower(e.Subject)
		if containsAny(lower, "fix", "bug", "hotfix", "patch", "correct") {
			s.fixes++
		}
		if containsAny(lower, "refactor", "clean", "rename", "tidy") {
			s.refactors++
		}
		for _, word := range tokenize(lower) {
			s.keywords[word]++
		}
	}

	var result []AuthorPattern
	for author, s := range authorMap {
		if s.total == 0 {
			continue
		}
		result = append(result, AuthorPattern{
			Author:        author,
			TotalCommits:  s.total,
			FixRatio:      float64(s.fixes) / float64(s.total),
			RefactorRatio: float64(s.refactors) / float64(s.total),
			TopKeywords:   topN(s.keywords, 5),
		})
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].TotalCommits != result[j].TotalCommits {
			return result[i].TotalCommits > result[j].TotalCommits
		}
		return result[i].Author < result[j].Author // stable alphabetical tiebreaker
	})
	return result
}

// FormatPattern returns a human-readable summary of an AuthorPattern.
func FormatPattern(p AuthorPattern) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s (%d commits)\n", p.Author, p.TotalCommits)
	fmt.Fprintf(&sb, "  Fix ratio:      %.0f%%\n", p.FixRatio*100)
	fmt.Fprintf(&sb, "  Refactor ratio: %.0f%%\n", p.RefactorRatio*100)
	if len(p.TopKeywords) > 0 {
		fmt.Fprintf(&sb, "  Top keywords:   %s\n", strings.Join(p.TopKeywords, ", "))
	}
	return sb.String()
}

// ── file hotspot detection ────────────────────────────────────────────────────

// FileHotspot represents a file that appears frequently in commit messages.
type FileHotspot struct {
	Path    string
	Changes int
}

// filePathRe matches common source and config file names within commit messages.
var filePathRe = regexp.MustCompile(`\b[\w./-]+\.(?:go|ts|tsx|js|jsx|py|rs|java|rb|php|swift|kt|cs|sql|proto|yaml|yml|json|toml|tf|sh|md)\b`)

// HotspotFiles returns the top n most frequently mentioned files across all commit subjects.
func HotspotFiles(entries []gitpkg.LogEntry, n int) []FileHotspot {
	freq := make(map[string]int)
	for _, e := range entries {
		seen := make(map[string]bool)
		for _, m := range filePathRe.FindAllString(e.Subject, -1) {
			if !seen[m] {
				seen[m] = true
				freq[m]++
			}
		}
	}
	type kv struct {
		k string
		v int
	}
	var pairs []kv
	for k, v := range freq {
		pairs = append(pairs, kv{k, v})
	}
	sort.SliceStable(pairs, func(i, j int) bool {
		if pairs[i].v != pairs[j].v {
			return pairs[i].v > pairs[j].v
		}
		return pairs[i].k < pairs[j].k
	})
	var out []FileHotspot
	for i, p := range pairs {
		if i >= n {
			break
		}
		out = append(out, FileHotspot{Path: p.k, Changes: p.v})
	}
	return out
}

// FormatHotspots returns a human-readable summary of file hotspots.
func FormatHotspots(hotspots []FileHotspot) string {
	if len(hotspots) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("File hotspots (most frequently referenced in commits):\n")
	for i, h := range hotspots {
		fmt.Fprintf(&sb, "  %d. %s (%d commits)\n", i+1, h.Path, h.Changes)
	}
	return sb.String()
}

func containsAny(s string, keywords ...string) bool {
	for _, k := range keywords {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}

var stopWords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true,
	"to": true, "in": true, "of": true, "for": true, "by": true,
	"is": true, "it": true, "be": true, "as": true, "at": true,
	"on": true, "if": true, "no": true, "up": true, "do": true,
}

func tokenize(s string) []string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return !('a' <= r && r <= 'z' || '0' <= r && r <= '9') //nolint:staticcheck
	})
	var out []string
	for _, w := range words {
		if len(w) > 2 && !stopWords[w] {
			out = append(out, w)
		}
	}
	return out
}

func topN(freq map[string]int, n int) []string {
	type kv struct {
		k string
		v int
	}
	var pairs []kv
	for k, v := range freq {
		pairs = append(pairs, kv{k, v})
	}
	sort.SliceStable(pairs, func(i, j int) bool {
		if pairs[i].v != pairs[j].v {
			return pairs[i].v > pairs[j].v
		}
		return pairs[i].k < pairs[j].k // alphabetical tiebreaker for determinism
	})
	var out []string
	for i, p := range pairs {
		if i >= n {
			break
		}
		out = append(out, p.k)
	}
	return out
}
