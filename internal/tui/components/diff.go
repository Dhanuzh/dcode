package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// DiffViewer renders git diffs with syntax highlighting
type DiffViewer struct {
	width             int
	syntaxHighlighter *SyntaxHighlighter
}

// NewDiffViewer creates a new diff viewer
func NewDiffViewer(width int, theme string) *DiffViewer {
	return &DiffViewer{
		width:             width,
		syntaxHighlighter: NewSyntaxHighlighter(theme),
	}
}

// Render renders a git diff with colors
func (dv *DiffViewer) Render(diff string) string {
	// Use chroma's diff lexer for syntax highlighting
	highlighted := dv.syntaxHighlighter.HighlightDiff(diff)

	// Add border
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6C7086")).
		Padding(1, 2).
		Width(dv.width - 4)

	return style.Render(highlighted)
}

// RenderSimple renders a diff with simple color coding (no syntax highlighter)
func (dv *DiffViewer) RenderSimple(diff string) string {
	lines := strings.Split(diff, "\n")
	var b strings.Builder

	addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1"))    // Green
	removedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8"))  // Red
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#89B4FA"))   // Blue
	hunkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CBA6F7"))     // Purple
	contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CDD6F4"))  // Normal

	for _, line := range lines {
		if line == "" {
			b.WriteString("\n")
			continue
		}

		switch {
		case strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- "):
			b.WriteString(headerStyle.Render(line))
		case strings.HasPrefix(line, "@@"):
			b.WriteString(hunkStyle.Render(line))
		case strings.HasPrefix(line, "+"):
			b.WriteString(addedStyle.Render(line))
		case strings.HasPrefix(line, "-"):
			b.WriteString(removedStyle.Render(line))
		case strings.HasPrefix(line, "diff --git"):
			b.WriteString(headerStyle.Bold(true).Render(line))
		case strings.HasPrefix(line, "index "):
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086")).Render(line))
		default:
			b.WriteString(contextStyle.Render(line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// RenderSideBySide renders a side-by-side diff (simplified)
func (dv *DiffViewer) RenderSideBySide(oldContent, newContent string) string {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	var b strings.Builder
	maxLines := len(oldLines)
	if len(newLines) > maxLines {
		maxLines = len(newLines)
	}

	halfWidth := (dv.width - 6) / 2

	// Header
	leftHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#F38BA8")).
		Width(halfWidth).
		Align(lipgloss.Center).
		Render("BEFORE")

	rightHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#A6E3A1")).
		Width(halfWidth).
		Align(lipgloss.Center).
		Render("AFTER")

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
		leftHeader,
		lipgloss.NewStyle().Render(" │ "),
		rightHeader,
	))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", dv.width))
	b.WriteString("\n")

	// Content
	leftStyle := lipgloss.NewStyle().Width(halfWidth)
	rightStyle := lipgloss.NewStyle().Width(halfWidth)

	for i := 0; i < maxLines; i++ {
		left := ""
		if i < len(oldLines) {
			left = oldLines[i]
		}

		right := ""
		if i < len(newLines) {
			right = newLines[i]
		}

		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
			leftStyle.Render(left),
			lipgloss.NewStyle().Render(" │ "),
			rightStyle.Render(right),
		))
		b.WriteString("\n")
	}

	return b.String()
}

// RenderInline renders a compact inline diff
func (dv *DiffViewer) RenderInline(diff string) string {
	lines := strings.Split(diff, "\n")

	addedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#2E3440")).
		Foreground(lipgloss.Color("#A6E3A1"))

	removedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#2E3440")).
		Foreground(lipgloss.Color("#F38BA8"))

	var result []string
	for _, line := range lines {
		if strings.HasPrefix(line, "+") {
			result = append(result, addedStyle.Render(line))
		} else if strings.HasPrefix(line, "-") {
			result = append(result, removedStyle.Render(line))
		}
	}

	return strings.Join(result, "\n")
}

// SetWidth updates the viewer width
func (dv *DiffViewer) SetWidth(width int) {
	dv.width = width
}

// DiffStats represents statistics about a diff
type DiffStats struct {
	FilesChanged int
	Insertions   int
	Deletions    int
}

// CalculateStats calculates statistics from a diff
func CalculateStats(diff string) DiffStats {
	lines := strings.Split(diff, "\n")
	stats := DiffStats{}
	files := make(map[string]bool)

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				files[parts[2]] = true
			}
		} else if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			stats.Insertions++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			stats.Deletions++
		}
	}

	stats.FilesChanged = len(files)
	return stats
}

// RenderStats renders diff statistics
func (dv *DiffViewer) RenderStats(stats DiffStats) string {
	var b strings.Builder

	fileStyle := lipgloss.NewStyle().Bold(true)
	addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1"))
	delStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8"))

	b.WriteString(fileStyle.Render("Files changed: "))
	b.WriteString(lipgloss.NewStyle().Render(string(rune(stats.FilesChanged + '0'))))
	b.WriteString("  ")

	b.WriteString(addStyle.Render("+"))
	b.WriteString(addStyle.Render(string(rune(stats.Insertions + '0'))))
	b.WriteString("  ")

	b.WriteString(delStyle.Render("-"))
	b.WriteString(delStyle.Render(string(rune(stats.Deletions + '0'))))

	return b.String()
}
