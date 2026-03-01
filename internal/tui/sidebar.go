package tui

// sidebar.go — renders the right-side panel, matching opencode's sidebar layout.
// Shows: session title, context tokens, cost, MCP servers, LSP, todos.

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Dhanuzh/dcode/internal/tool"
)

const sidebarWidth = 42

// SidebarSection represents a collapsible sidebar section.
type SidebarSection struct {
	Title    string
	Expanded bool
}

// renderSidebar returns the full sidebar string at sidebarWidth columns.
func (m *Model) renderSidebar() string {
	t := m.currentTheme

	// Left border separator
	borderLine := lipgloss.NewStyle().
		Foreground(t.Border).
		Render(strings.Repeat("│", m.height))

	content := lipgloss.NewStyle().
		Width(sidebarWidth - 1).
		Height(m.height).
		Background(t.Background).
		PaddingTop(1).
		PaddingBottom(1).
		PaddingLeft(1).
		PaddingRight(1).
		Render(m.renderSidebarContent())

	return lipgloss.JoinHorizontal(lipgloss.Top, borderLine, content)
}

func (m *Model) renderSidebarContent() string {
	t := m.currentTheme
	dim := lipgloss.NewStyle().Foreground(t.TextMuted)
	faint := lipgloss.NewStyle().Foreground(t.TextDim)
	bold := lipgloss.NewStyle().Foreground(t.Text).Bold(true)
	success := lipgloss.NewStyle().Foreground(t.Success)
	errSt := lipgloss.NewStyle().Foreground(t.Error)
	warn := lipgloss.NewStyle().Foreground(t.Warning)
	sectionHead := lipgloss.NewStyle().Foreground(t.TextMuted).Bold(true)
	w := sidebarWidth - 3 // inner content width

	var sb strings.Builder

	// ── Session title ──────────────────────────────────────────────────
	sessionTitle := "New Session"
	if m.sessionID != "" {
		if sess, err := m.Store.Get(m.sessionID); err == nil && sess.Title != "" {
			sessionTitle = sess.Title
		}
	}
	if len(sessionTitle) > w {
		sessionTitle = sessionTitle[:w-3] + "..."
	}
	sb.WriteString(bold.Render(sessionTitle) + "\n\n")

	// ── Context / Cost ─────────────────────────────────────────────────
	sb.WriteString(sectionHead.Render("Context") + "\n")
	totalTokens, totalCost := 0, 0.0
	for _, msg := range m.messages {
		if msg.Role == "assistant" {
			totalTokens += msg.TokensIn + msg.TokensOut
			totalCost += msg.Cost
		}
	}
	sb.WriteString(dim.Render(fmt.Sprintf("%s tokens", formatTokens(totalTokens))) + "\n")
	if totalCost > 0 {
		sb.WriteString(dim.Render(fmt.Sprintf("$%.4f spent", totalCost)) + "\n")
	}
	sb.WriteString("\n")

	// ── MCP servers ────────────────────────────────────────────────────
	if m.mcpCount > 0 || m.mcpErrors {
		sb.WriteString(sectionHead.Render("MCP") + "\n")
		if m.mcpErrors {
			sb.WriteString(errSt.Render("● ") + dim.Render(fmt.Sprintf("%d server(s) — errors", m.mcpCount)) + "\n")
		} else {
			sb.WriteString(success.Render("● ") + dim.Render(fmt.Sprintf("%d connected", m.mcpCount)) + "\n")
		}
		sb.WriteString("\n")
	}

	// ── LSP ────────────────────────────────────────────────────────────
	sb.WriteString(sectionHead.Render("LSP") + "\n")
	if m.lspCount == 0 {
		sb.WriteString(faint.Render("No LSPs active") + "\n")
	} else {
		sb.WriteString(success.Render("● ") + dim.Render(fmt.Sprintf("%d active", m.lspCount)) + "\n")
	}
	sb.WriteString("\n")

	// ── Pending permissions ────────────────────────────────────────────
	if m.pendingPermissions > 0 {
		sb.WriteString(sectionHead.Render("Permissions") + "\n")
		sb.WriteString(warn.Render(fmt.Sprintf("△ %d pending", m.pendingPermissions)) + "\n\n")
	}

	// ── Todos ──────────────────────────────────────────────────────────
	todos := tool.GetSessionTodos(m.sessionID)
	if len(todos) > 0 {
		// Count by status
		var inProgress, pending, completed, cancelled int
		for _, td := range todos {
			switch td.Status {
			case "in_progress":
				inProgress++
			case "completed":
				completed++
			case "cancelled":
				cancelled++
			default:
				pending++
			}
		}
		sb.WriteString(sectionHead.Render("Todo") + faint.Render(fmt.Sprintf(" %d/%d done", completed, len(todos))) + "\n")

		_ = inProgress
		_ = cancelled
		_ = pending

		for _, td := range todos {
			bullet, clr, textClr := todoItemStyle(td.Status)
			bulletSt := lipgloss.NewStyle().Foreground(clr)
			textSt := lipgloss.NewStyle().Foreground(textClr)
			title := truncateSidebar(td.Title, w-3)
			line := bulletSt.Render(bullet) + " " + textSt.Render(title)
			sb.WriteString(line + "\n")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// todoItemStyle returns bullet, bullet color, and text color for a todo status.
func todoItemStyle(status string) (bullet string, bulletClr, textClr lipgloss.Color) {
	switch status {
	case "in_progress":
		return "▶", lipgloss.Color("#F9E2AF"), lipgloss.Color("#CDD6F4") // yellow bullet, normal text
	case "completed":
		return "✓", lipgloss.Color("#A6E3A1"), lipgloss.Color("#6C7086") // green check, dim text
	case "cancelled":
		return "✗", lipgloss.Color("#F38BA8"), lipgloss.Color("#6C7086") // red X, dim text
	default: // pending
		return "○", lipgloss.Color("#6C7086"), lipgloss.Color("#A6ADC8") // dim circle, muted text
	}
}

// filterActiveTodos returns todos that are not completed.
func filterActiveTodos(todos []tool.TodoItem) []tool.TodoItem {
	var out []tool.TodoItem
	for _, t := range todos {
		if t.Status != "completed" {
			out = append(out, t)
		}
	}
	return out
}

// todoStyle returns a bullet and color for a given todo status (used in older code).
func todoStyle(status string) (string, lipgloss.Color) {
	b, clr, _ := todoItemStyle(status)
	return b, clr
}

func truncateSidebar(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
