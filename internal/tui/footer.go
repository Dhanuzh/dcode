package tui

// footer.go — renders the bottom bar of the session view, matching opencode's footer.tsx.
// Shows: working directory, vim mode, keybinds hint, LSP/MCP status counts.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderFooter returns the footer bar string.
// It replicates opencode's footer: left=cwd, right=lsp/mcp counts + keybind hints.
func (m *Model) renderFooter() string {
	t := m.currentTheme
	dim := lipgloss.NewStyle().Foreground(t.TextMuted)
	kb := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)

	// ── Left side: working directory ──────────────────────────────────
	cwd, _ := os.Getwd()
	dir := cwd
	if home, err := os.UserHomeDir(); err == nil {
		if rel, err := filepath.Rel(home, cwd); err == nil && !strings.HasPrefix(rel, "..") {
			dir = "~/" + rel
		}
	}
	// Truncate to last 40 chars if long
	if len(dir) > 40 {
		dir = "…" + dir[len(dir)-39:]
	}
	leftStr := dim.Render(dir)

	// ── Right side: LSP / MCP indicator counts ─────────────────────────
	// (LSP and MCP counts are populated via m.lspCount / m.mcpCount fields)
	var statusParts []string
	if m.mcpCount > 0 {
		mcpClr := t.Success
		if m.mcpErrors {
			mcpClr = t.Error
		}
		statusParts = append(statusParts, lipgloss.NewStyle().Foreground(mcpClr).Render("⊙")+" "+
			dim.Render(fmt.Sprintf("%d MCP", m.mcpCount)))
	}
	if m.lspCount > 0 {
		lspClr := t.Success
		statusParts = append(statusParts, lipgloss.NewStyle().Foreground(lspClr).Render("•")+" "+
			dim.Render(fmt.Sprintf("%d LSP", m.lspCount)))
	}
	if m.pendingPermissions > 0 {
		statusParts = append(statusParts, lipgloss.NewStyle().Foreground(t.Warning).Render(
			fmt.Sprintf("△ %d Permission%s", m.pendingPermissions, pluralS(m.pendingPermissions))))
	}
	if !m.mouseEnabled {
		statusParts = append(statusParts, lipgloss.NewStyle().Foreground(t.Warning).Bold(true).Render("✗ MOUSE OFF")+
			dim.Render(" (Ctrl+M to re-enable)"))
	}

	// ── Keybind hints — depends on input focus & vim mode ─────────────
	sep := dim.Render(strings.Repeat("─", m.width))
	var hints string
	focusBadge := m.RenderVimMode()
	scrollInd := m.GetScrollIndicator()
	if scrollInd != "" {
		focusBadge += " " + scrollInd
	}

	switch {
	case !m.focusInput:
		hints = lipgloss.JoinHorizontal(lipgloss.Center,
			dim.Render("[SCROLL]"), " ",
			dim.Render("j/k"), " scroll  ",
			dim.Render("gg/G"), " top/bot  ",
			dim.Render("Ctrl+d/u"), " half-page  ",
			dim.Render("i/Tab"), " input",
		)
	case m.vimState.Mode == VimModeNormal:
		hints = lipgloss.JoinHorizontal(lipgloss.Center,
			focusBadge, " ",
			dim.Render("i/a"), " insert  ",
			dim.Render("h/j/k/l"), " move  ",
			dim.Render("dd/yy"), " del/yank  ",
			dim.Render("Tab"), " scroll",
		)
	default: // INSERT mode
		hints = lipgloss.JoinHorizontal(lipgloss.Center,
			focusBadge, " ",
			kb.Render("Esc"), " normal  ",
			kb.Render("Enter"), " send  ",
			kb.Render("Alt+Enter"), " newline  ",
			kb.Render("Ctrl+K"), " model  ",
			kb.Render("/"), " commands",
		)
	}

	// Combine left+right status with hints
	rightStr := strings.Join(statusParts, "  ")
	var line1 string
	if rightStr != "" {
		pad := m.width - lipgloss.Width(leftStr) - lipgloss.Width(rightStr) - 2
		if pad < 1 {
			pad = 1
		}
		line1 = leftStr + strings.Repeat(" ", pad) + rightStr
	} else {
		line1 = leftStr
	}

	return sep + "\n" + line1 + "\n" + dim.Render(hints)
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
