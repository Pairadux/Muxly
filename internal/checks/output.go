package checks

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	hintStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	headerStyle  = lipgloss.NewStyle().Bold(true)
)

const (
	symbolOK      = "✓"
	symbolWarning = "!"
	symbolError   = "✗"
)

func FormatResult(r CheckResult) string {
	var symbol, message string

	switch r.Status {
	case StatusOK:
		symbol = successStyle.Render(symbolOK)
	case StatusWarning:
		symbol = warningStyle.Render(symbolWarning)
	case StatusError:
		symbol = errorStyle.Render(symbolError)
	}

	message = r.Message
	if r.Detail != "" {
		message = fmt.Sprintf("%s %s", r.Message, hintStyle.Render(r.Detail))
	}

	result := fmt.Sprintf("  %s %s", symbol, message)

	if r.Hint != "" {
		result += fmt.Sprintf("\n    └─ %s", hintStyle.Render(r.Hint))
	}

	return result
}

func FormatSection(title string, results []CheckResult, quiet bool) string {
	var sb strings.Builder

	if quiet {
		var filtered []CheckResult
		for _, r := range results {
			if r.Status != StatusOK {
				filtered = append(filtered, r)
			}
		}
		if len(filtered) == 0 {
			return ""
		}
		results = filtered
	}

	sb.WriteString(headerStyle.Render(title))
	sb.WriteString("\n")

	for _, r := range results {
		sb.WriteString(FormatResult(r))
		sb.WriteString("\n")
	}

	return sb.String()
}

func FormatSummary(results []CheckResult) string {
	errors, warnings := CountByStatus(results)

	if errors == 0 && warnings == 0 {
		return successStyle.Render("All checks passed")
	}

	var parts []string
	if errors > 0 {
		parts = append(parts, errorStyle.Render(fmt.Sprintf("%d error(s)", errors)))
	}
	if warnings > 0 {
		parts = append(parts, warningStyle.Render(fmt.Sprintf("%d warning(s)", warnings)))
	}

	return fmt.Sprintf("Found %s", strings.Join(parts, " and "))
}
