package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderViewportWithScrollbar renders the viewport content with a scrollbar on the right
func (m *Model) RenderViewportWithScrollbar() string {
	content := m.viewport.View()

	// If content is empty or viewport is too small, just return content
	if content == "" || m.viewport.Height < 3 {
		return content
	}

	// Calculate scrollbar properties
	totalLines := m.viewport.TotalLineCount()
	visibleLines := m.viewport.Height
	scrollPercentage := m.viewport.ScrollPercent()

	// If all content is visible, no scrollbar needed
	if totalLines <= visibleLines {
		return content
	}

	// Split content into lines
	lines := strings.Split(content, "\n")
	if len(lines) > m.viewport.Height {
		lines = lines[:m.viewport.Height]
	}

	// Calculate scrollbar position
	scrollbarHeight := m.viewport.Height
	scrollbarPos := int(scrollPercentage * float64(scrollbarHeight-1))

	// Scrollbar characters
	scrollbarStyle := lipgloss.NewStyle().Foreground(m.currentTheme.Border)
	scrollThumbStyle := lipgloss.NewStyle().Foreground(m.currentTheme.Primary)

	// maxWidth is the viewport width – lines from viewport.View() are already
	// constrained to this width, so we must NOT byte-slice them (that would
	// corrupt embedded ANSI escape sequences).
	maxWidth := m.viewport.Width

	var result strings.Builder
	for i, line := range lines {
		result.WriteString(line)

		// Pad the line to fill the viewport width using visual width so that
		// ANSI escape codes (which are invisible) don't count against the limit.
		currentWidth := lipgloss.Width(line)
		if currentWidth < maxWidth {
			result.WriteString(strings.Repeat(" ", maxWidth-currentWidth))
		}

		// Add scrollbar character
		result.WriteString(" ")
		if i == scrollbarPos {
			result.WriteString(scrollThumbStyle.Render("█"))
		} else if i == 0 {
			result.WriteString(scrollbarStyle.Render("▲"))
		} else if i == len(lines)-1 {
			result.WriteString(scrollbarStyle.Render("▼"))
		} else {
			result.WriteString(scrollbarStyle.Render("│"))
		}

		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// GetScrollIndicator returns a scroll position indicator for the status bar
func (m *Model) GetScrollIndicator() string {
	if m.viewport.TotalLineCount() <= m.viewport.Height {
		return "" // All content visible
	}

	scrollPercentage := m.viewport.ScrollPercent()

	dimStyle := lipgloss.NewStyle().Foreground(m.currentTheme.TextDim)
	indicatorStyle := lipgloss.NewStyle().Foreground(m.currentTheme.Primary)

	var indicator string
	switch {
	case scrollPercentage < 0.01:
		indicator = "⬆ Top"
	case scrollPercentage > 0.99:
		indicator = "⬇ Bottom"
	default:
		indicator = fmt.Sprintf("↕ %d%%", int(scrollPercentage*100))
	}

	return dimStyle.Render("[") + indicatorStyle.Render(indicator) + dimStyle.Render("]")
}
