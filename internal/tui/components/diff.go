package components

import (
	"fmt"
	"path/filepath"
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

	addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1"))   // Green
	removedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8")) // Red
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#89B4FA"))  // Blue
	hunkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CBA6F7"))    // Purple
	contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CDD6F4")) // Normal

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

// ─── LCS-based Edit Diff ────────────────────────────────────────────────────────

// EditDiffOp represents the type of diff operation
type EditDiffOp int

const (
	DiffEqual  EditDiffOp = iota // Line is unchanged
	DiffDelete                   // Line was removed
	DiffInsert                   // Line was added
)

// EditDiffLine represents a single line in the diff output
type EditDiffLine struct {
	Op      EditDiffOp
	OldLine int    // 1-based line number in old content (0 if insert)
	NewLine int    // 1-based line number in new content (0 if delete)
	Text    string // Line content
}

// ComputeLineDiff computes a line-level diff using the LCS algorithm
func ComputeLineDiff(oldContent, newContent string) []EditDiffLine {
	oldLines := splitLines(oldContent)
	newLines := splitLines(newContent)

	m := len(oldLines)
	n := len(newLines)

	// Build LCS table
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if oldLines[i-1] == newLines[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	// Backtrack to produce diff
	var result []EditDiffLine
	i, j := m, n
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
			result = append(result, EditDiffLine{
				Op:      DiffEqual,
				OldLine: i,
				NewLine: j,
				Text:    oldLines[i-1],
			})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			result = append(result, EditDiffLine{
				Op:      DiffInsert,
				NewLine: j,
				Text:    newLines[j-1],
			})
			j--
		} else {
			result = append(result, EditDiffLine{
				Op:      DiffDelete,
				OldLine: i,
				Text:    oldLines[i-1],
			})
			i--
		}
	}

	// Reverse (backtracking gives us reversed order)
	for left, right := 0, len(result)-1; left < right; left, right = left+1, right-1 {
		result[left], result[right] = result[right], result[left]
	}

	return result
}

// RenderEditDiff renders a side-by-side diff view for edit/write tool output
func (dv *DiffViewer) RenderEditDiff(oldContent, newContent, filePath string, maxHeight int) string {
	if maxHeight <= 0 {
		maxHeight = 30
	}

	diffLines := ComputeLineDiff(oldContent, newContent)

	// Count changes
	added, removed := 0, 0
	for _, dl := range diffLines {
		switch dl.Op {
		case DiffInsert:
			added++
		case DiffDelete:
			removed++
		}
	}

	// Narrow terminal fallback: inline format
	if dv.width < 80 {
		return dv.renderEditDiffInline(diffLines, filePath, added, removed, maxHeight)
	}

	return dv.renderEditDiffSideBySide(diffLines, filePath, added, removed, maxHeight)
}

func (dv *DiffViewer) renderEditDiffSideBySide(diffLines []EditDiffLine, filePath string, added, removed, maxHeight int) string {
	var b strings.Builder

	// Styles
	removedBg := lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8")).Background(lipgloss.Color("#3B1D26"))
	addedBg := lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1")).Background(lipgloss.Color("#1D3B26"))
	contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086"))
	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086"))
	headerLabelRed := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F38BA8"))
	headerLabelGreen := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A6E3A1"))
	fileStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#89B4FA"))
	statStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086"))

	// Available width for each side
	halfWidth := (dv.width - 7) / 2 // 7 = " │ " separator (3) + line numbers (4)
	if halfWidth < 10 {
		halfWidth = 10
	}
	lineNumW := 4

	// File header
	displayPath := filePath
	if len(displayPath) > dv.width-20 {
		displayPath = "..." + displayPath[len(displayPath)-(dv.width-23):]
	}
	b.WriteString(fileStyle.Render(filepath.Base(displayPath)))
	b.WriteString(" ")
	b.WriteString(statStyle.Render(fmt.Sprintf("-%d +%d", removed, added)))
	b.WriteString("\n")

	// Column headers
	leftHeader := headerLabelRed.Width(lineNumW + halfWidth).Align(lipgloss.Center).Render("REMOVED")
	rightHeader := headerLabelGreen.Width(lineNumW + halfWidth).Align(lipgloss.Center).Render("ADDED")
	b.WriteString(leftHeader)
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086")).Render(" │ "))
	b.WriteString(rightHeader)
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086")).Render(strings.Repeat("─", dv.width)))
	b.WriteString("\n")

	// Build paired rows: group deletes and inserts together
	type row struct {
		leftNum   string
		leftText  string
		leftOp    EditDiffOp
		rightNum  string
		rightText string
		rightOp   EditDiffOp
	}

	var rows []row
	i := 0
	for i < len(diffLines) {
		dl := diffLines[i]
		switch dl.Op {
		case DiffEqual:
			rows = append(rows, row{
				leftNum:   fmt.Sprintf("%d", dl.OldLine),
				leftText:  dl.Text,
				leftOp:    DiffEqual,
				rightNum:  fmt.Sprintf("%d", dl.NewLine),
				rightText: dl.Text,
				rightOp:   DiffEqual,
			})
			i++
		case DiffDelete:
			// Collect consecutive deletes
			var deletes []EditDiffLine
			for i < len(diffLines) && diffLines[i].Op == DiffDelete {
				deletes = append(deletes, diffLines[i])
				i++
			}
			// Collect consecutive inserts that follow
			var inserts []EditDiffLine
			for i < len(diffLines) && diffLines[i].Op == DiffInsert {
				inserts = append(inserts, diffLines[i])
				i++
			}
			// Pair them up
			maxPair := len(deletes)
			if len(inserts) > maxPair {
				maxPair = len(inserts)
			}
			for k := 0; k < maxPair; k++ {
				r := row{}
				if k < len(deletes) {
					r.leftNum = fmt.Sprintf("%d", deletes[k].OldLine)
					r.leftText = deletes[k].Text
					r.leftOp = DiffDelete
				}
				if k < len(inserts) {
					r.rightNum = fmt.Sprintf("%d", inserts[k].NewLine)
					r.rightText = inserts[k].Text
					r.rightOp = DiffInsert
				}
				rows = append(rows, r)
			}
		case DiffInsert:
			// Orphan insert (no preceding delete)
			rows = append(rows, row{
				rightNum:  fmt.Sprintf("%d", dl.NewLine),
				rightText: dl.Text,
				rightOp:   DiffInsert,
			})
			i++
		}
	}

	// Render rows with truncation
	rendered := 0
	for _, r := range rows {
		if rendered >= maxHeight {
			remaining := len(rows) - rendered
			b.WriteString(statStyle.Render(fmt.Sprintf("  ... %d more lines", remaining)))
			b.WriteString("\n")
			break
		}

		// Left side
		leftNum := lineNumStyle.Width(lineNumW).Align(lipgloss.Right).Render(r.leftNum)
		leftContent := truncateLine(r.leftText, halfWidth-1)
		switch r.leftOp {
		case DiffDelete:
			leftContent = removedBg.Width(halfWidth - 1).Render(leftContent)
		case DiffEqual:
			leftContent = contextStyle.Width(halfWidth - 1).Render(leftContent)
		default:
			leftContent = lipgloss.NewStyle().Width(halfWidth - 1).Render("")
		}

		// Right side
		rightNum := lineNumStyle.Width(lineNumW).Align(lipgloss.Right).Render(r.rightNum)
		rightContent := truncateLine(r.rightText, halfWidth-1)
		switch r.rightOp {
		case DiffInsert:
			rightContent = addedBg.Width(halfWidth - 1).Render(rightContent)
		case DiffEqual:
			rightContent = contextStyle.Width(halfWidth - 1).Render(rightContent)
		default:
			rightContent = lipgloss.NewStyle().Width(halfWidth - 1).Render("")
		}

		sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086")).Render(" │ ")
		b.WriteString(leftNum + " " + leftContent + sep + rightNum + " " + rightContent + "\n")
		rendered++
	}

	// Wrap in a border
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6C7086")).
		Padding(0, 1).
		Width(dv.width - 2)

	return borderStyle.Render(b.String())
}

func (dv *DiffViewer) renderEditDiffInline(diffLines []EditDiffLine, filePath string, added, removed, maxHeight int) string {
	var b strings.Builder

	removedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8"))
	addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1"))
	contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086"))
	fileStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#89B4FA"))
	statStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086"))

	b.WriteString(fileStyle.Render(filepath.Base(filePath)))
	b.WriteString(" ")
	b.WriteString(statStyle.Render(fmt.Sprintf("-%d +%d", removed, added)))
	b.WriteString("\n")

	rendered := 0
	for _, dl := range diffLines {
		if rendered >= maxHeight {
			remaining := len(diffLines) - rendered
			b.WriteString(statStyle.Render(fmt.Sprintf("... %d more lines", remaining)))
			b.WriteString("\n")
			break
		}
		switch dl.Op {
		case DiffDelete:
			b.WriteString(removedStyle.Render("- " + dl.Text))
		case DiffInsert:
			b.WriteString(addedStyle.Render("+ " + dl.Text))
		case DiffEqual:
			b.WriteString(contextStyle.Render("  " + dl.Text))
		}
		b.WriteString("\n")
		rendered++
	}

	return b.String()
}

// splitLines splits content into lines, handling empty content
func splitLines(content string) []string {
	if content == "" {
		return nil
	}
	lines := strings.Split(content, "\n")
	// Remove trailing empty line from trailing newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// truncateLine truncates a line to fit within maxWidth
func truncateLine(line string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if len(line) <= maxWidth {
		return line
	}
	if maxWidth <= 3 {
		return line[:maxWidth]
	}
	return line[:maxWidth-3] + "..."
}
