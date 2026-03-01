package tui

// header.go — renders the top bar of the session view, matching opencode's header.tsx
// Shows: title, session context info, cost, provider/model/agent badges, token usage.

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderHeader returns the full header bar string.
func (m *Model) renderHeader() string {
	if m.width == 0 {
		return ""
	}

	t := m.currentTheme

	titleSt := lipgloss.NewStyle().Bold(true).Foreground(t.Primary).Background(t.Background)

	// Left side: title + badges
	var left []string
	left = append(left, titleSt.Render(" DCode "))

	if m.providerInitializing {
		left = append(left, " ", m.spinner.View()+" "+lipgloss.NewStyle().Foreground(t.TextMuted).Render("Connecting to "+m.Provider+"..."))
	} else {
		provSt := lipgloss.NewStyle().Foreground(t.Background).Background(t.Success).Padding(0, 1)
		modSt := lipgloss.NewStyle().Foreground(t.Background).Background(t.Info).Padding(0, 1)
		agentSt := lipgloss.NewStyle().Foreground(t.Background).Background(t.Primary).Padding(0, 1).Bold(true)
		left = append(left,
			" ",
			provSt.Render(m.Provider),
			" ",
			modSt.Render(shortModel(m.Model_)),
			" ",
			agentSt.Render(m.Agent),
		)
	}

	// Message count
	msgSt := lipgloss.NewStyle().Foreground(t.TextMuted)
	left = append(left, " ", msgSt.Render(fmt.Sprintf("[%d msgs]", len(m.messages))))

	// Token usage
	if m.tokenTracker != nil {
		left = append(left, " ", m.RenderTokenUsage())
	}

	leftStr := lipgloss.JoinHorizontal(lipgloss.Center, left...)

	// Right side: status message
	if s := m.getStatus(); s != "" {
		dimSt := lipgloss.NewStyle().Foreground(t.TextMuted)
		leftStr += "  " + dimSt.Render(s)
	}

	// Separator
	sep := lipgloss.NewStyle().Foreground(t.Border).Render(strings.Repeat("─", m.width))
	return leftStr + "\n" + sep + "\n"
}
