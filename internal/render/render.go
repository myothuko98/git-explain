package render

import (
	"fmt"
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

func isTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// ExplainResult renders the LLM explanation to stdout.
func ExplainResult(title, meta, providerName, explanation string) {
	if !isTTY() {
		fmt.Printf("=== %s ===\n%s\n\n%s\n", title, meta, explanation)
		return
	}

	header := titleStyle.Render("🔍 " + title)
	metaLine := labelStyle.Render(meta)
	body := blockStyle.Render(valueStyle.Render(strings.TrimSpace(explanation)))
	footer := providerStyle.Render("via " + providerName)

	fmt.Println()
	fmt.Println(header)
	fmt.Println(metaLine)
	fmt.Println(body)
	fmt.Println(footer)
	fmt.Println()
}

// Error prints an error message.
func Error(msg string) {
	if !isTTY() {
		fmt.Fprintln(os.Stderr, "error: "+msg)
		return
	}
	fmt.Fprintln(os.Stderr, errorStyle.Render("✗ "+msg))
}

// Header prints a section header.
func Header(msg string) {
	if !isTTY() {
		fmt.Println("=== " + msg + " ===")
		return
	}
	fmt.Println()
	fmt.Println(titleStyle.Render("▸ " + msg))
}

// Plain prints plain text, respecting TTY.
func Plain(msg string) {
	fmt.Println(msg)
}
