package render

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212"))

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255"))

	blockStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1)

	providerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
)

// IsTTY reports whether stdout is a terminal.
func IsTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// ExplainResult renders the full LLM explanation to stdout (non-streaming).
func ExplainResult(title, meta, providerName, explanation string) {
	if !IsTTY() {
		fmt.Printf("=== %s ===\n%s\n\n%s\n", title, meta, explanation)
		return
	}
	fmt.Println()
	fmt.Println(titleStyle.Render("🔍 " + title))
	if meta != "" {
		fmt.Println(labelStyle.Render(meta))
	}
	fmt.Println(blockStyle.Render(valueStyle.Render(strings.TrimSpace(explanation))))
	fmt.Println(providerStyle.Render("via " + providerName))
	fmt.Println()
}

// ExplainJSON outputs the result as JSON.
func ExplainJSON(title, meta, providerName, explanation string) {
	out := map[string]string{
		"title":       title,
		"meta":        meta,
		"provider":    providerName,
		"explanation": strings.TrimSpace(explanation),
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

// StreamHeader prints the title/meta before streaming begins.
func StreamHeader(title, meta string) {
	if !IsTTY() {
		fmt.Printf("=== %s ===\n%s\n\n", title, meta)
		return
	}
	fmt.Println()
	fmt.Println(titleStyle.Render("🔍 " + title))
	if meta != "" {
		fmt.Println(labelStyle.Render(meta))
	}
	fmt.Println()
}

// StreamFooter prints the provider attribution after streaming ends.
func StreamFooter(providerName string) {
	if !IsTTY() {
		fmt.Println()
		return
	}
	fmt.Println()
	fmt.Println(providerStyle.Render("via " + providerName))
	fmt.Println()
}

// StreamWriter returns the writer to use for streaming (stdout).
func StreamWriter() io.Writer { return os.Stdout }

// Error prints an error message.
func Error(msg string) {
	if !IsTTY() {
		fmt.Fprintln(os.Stderr, "error: "+msg)
		return
	}
	fmt.Fprintln(os.Stderr, errorStyle.Render("✗ "+msg))
}

// Header prints a section header.
func Header(msg string) {
	if !IsTTY() {
		fmt.Println("=== " + msg + " ===")
		return
	}
	fmt.Println()
	fmt.Println(titleStyle.Render("▸ " + msg))
}

// Plain prints plain text.
func Plain(msg string) {
	fmt.Println(msg)
}
