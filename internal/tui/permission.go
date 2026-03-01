package tui

// permission.go — Interactive permission prompt overlay, matching opencode's permission.tsx
//
// When a tool needs user approval, the permission engine calls back into the TUI
// via a channel.  The TUI renders a full-width panel at the bottom of the chat
// (above the footer) with three choices: Allow once / Allow always / Reject.
// Left/right arrows (or h/l) move between choices; Enter confirms; Esc rejects.

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Dhanuzh/dcode/internal/permission"
)

// ─── Messages ────────────────────────────────────────────────────────────────

// PermissionRequestMsg is sent when a tool requires interactive permission.
// The TUI displays the prompt and later sends PermissionResponseMsg.
type PermissionRequestMsg struct {
	Req     *permission.Request
	ReplyCh chan bool // true = allow, false = deny
}

// ─── PermissionState embedded in Model ───────────────────────────────────────

// PermissionPromptState holds the current active permission prompt.
type PermissionPromptState struct {
	Active   bool
	Req      *permission.Request
	ReplyCh  chan bool
	Selected int    // 0=once, 1=always, 2=reject
	Stage    string // "prompt" | "always" | "reject"
}

// choices returns the label list for the current stage.
func (p *PermissionPromptState) choices() []string {
	if p.Stage == "always" {
		return []string{"Confirm", "Cancel"}
	}
	return []string{"Allow once", "Allow always", "Reject"}
}

// ─── Update handler ──────────────────────────────────────────────────────────

// handlePermissionMsg wires PermissionRequestMsg into the model.
func (m *Model) handlePermissionMsg(msg PermissionRequestMsg) (tea.Model, tea.Cmd) {
	m.permission = PermissionPromptState{
		Active:   true,
		Req:      msg.Req,
		ReplyCh:  msg.ReplyCh,
		Selected: 0,
		Stage:    "prompt",
	}
	m.pendingPermissions++
	return m, nil
}

// updatePermission processes keyboard input while a permission prompt is shown.
// Returns (handled bool, cmd).
func (m *Model) updatePermission(key string) (bool, tea.Cmd) {
	if !m.permission.Active {
		return false, nil
	}

	choices := m.permission.choices()

	switch key {
	case "left", "h":
		m.permission.Selected = (m.permission.Selected - 1 + len(choices)) % len(choices)
		return true, nil

	case "right", "l":
		m.permission.Selected = (m.permission.Selected + 1) % len(choices)
		return true, nil

	case "enter":
		return true, m.confirmPermission()

	case "esc":
		// Esc = reject / cancel
		m.permission.Selected = len(choices) - 1
		return true, m.confirmPermission()
	}
	return false, nil
}

func (m *Model) confirmPermission() tea.Cmd {
	p := &m.permission
	choices := p.choices()
	choice := choices[p.Selected]

	switch p.Stage {
	case "prompt":
		switch choice {
		case "Allow once":
			m.resolvePermission(true)
		case "Allow always":
			// Move to confirmation stage
			p.Stage = "always"
			p.Selected = 0
			return nil
		case "Reject":
			m.resolvePermission(false)
		}
	case "always":
		switch choice {
		case "Confirm":
			m.resolvePermission(true) // same as once for now; "always" rules TBD
		case "Cancel":
			p.Stage = "prompt"
			p.Selected = 0
			return nil
		}
	}
	return nil
}

func (m *Model) resolvePermission(allow bool) {
	if m.permission.ReplyCh != nil {
		// Non-blocking send; engine may have already timed out
		select {
		case m.permission.ReplyCh <- allow:
		default:
		}
	}
	if m.pendingPermissions > 0 {
		m.pendingPermissions--
	}
	m.permission = PermissionPromptState{}
}

// ─── Renderer ────────────────────────────────────────────────────────────────

// renderPermissionPrompt returns the permission panel string (empty if not active).
func (m *Model) renderPermissionPrompt() string {
	if !m.permission.Active || m.permission.Req == nil {
		return ""
	}

	t := m.currentTheme
	p := &m.permission
	w := m.viewport.Width + 2
	if w < 40 {
		w = 40
	}

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.Border{Left: "┃"}).
		BorderForeground(t.Warning).
		PaddingLeft(2)

	// Title line
	icon := lipgloss.NewStyle().Foreground(t.Warning).Render("△")
	title := "Permission required"
	if p.Stage == "always" {
		title = "Always allow"
	}
	titleLine := icon + " " + lipgloss.NewStyle().Foreground(t.Text).Bold(true).Render(title)

	// Description
	desc := formatPermissionDesc(p.Req)
	descLine := lipgloss.NewStyle().Foreground(t.TextMuted).Render("  " + desc)

	body := borderStyle.Render(lipgloss.JoinVertical(lipgloss.Left, titleLine, descLine))

	// Choice bar
	choices := p.choices()
	var choiceBtns []string
	for i, c := range choices {
		if i == p.Selected {
			btn := lipgloss.NewStyle().
				Foreground(t.Background).
				Background(t.Warning).
				Bold(true).
				Padding(0, 1).
				Render(c)
			choiceBtns = append(choiceBtns, btn)
		} else {
			btn := lipgloss.NewStyle().
				Foreground(t.TextMuted).
				Background(t.Border).
				Padding(0, 1).
				Render(c)
			choiceBtns = append(choiceBtns, btn)
		}
	}
	choiceBar := "  " + strings.Join(choiceBtns, "  ")

	hint := lipgloss.NewStyle().Foreground(t.TextDim).Render(
		"  ←→ select   enter confirm   esc reject",
	)

	footer := lipgloss.NewStyle().
		Background(t.Border).
		Width(w).
		Padding(0, 1).
		Render(choiceBar + "   " + hint)

	return lipgloss.JoinVertical(lipgloss.Left,
		dimStyle.Render(strings.Repeat("─", w)),
		body,
		footer,
	)
}

// formatPermissionDesc builds the single-line description for a permission request.
func formatPermissionDesc(req *permission.Request) string {
	icon := permissionIcon(req)
	switch req.Action {
	case permission.ActionBash:
		cmd := req.Path
		if d := req.Description; d != "" {
			return fmt.Sprintf("%s %s: %s", icon, d, cmd)
		}
		return fmt.Sprintf("%s $ %s", icon, cmd)
	case permission.ActionEdit:
		return fmt.Sprintf("%s Edit %s", icon, req.Path)
	case permission.ActionWrite:
		return fmt.Sprintf("%s Write %s", icon, req.Path)
	case permission.ActionRead:
		return fmt.Sprintf("%s Read %s", icon, req.Path)
	case permission.ActionDelete:
		return fmt.Sprintf("%s Delete %s", icon, req.Path)
	case permission.ActionNetwork:
		return fmt.Sprintf("%s WebFetch %s", icon, req.Path)
	case permission.ActionExternalDir:
		return fmt.Sprintf("%s Access external directory %s", icon, req.Path)
	default:
		if req.Description != "" {
			return fmt.Sprintf("%s %s", icon, req.Description)
		}
		return fmt.Sprintf("%s %s: %s", icon, req.Action, req.Path)
	}
}

func permissionIcon(req *permission.Request) string {
	switch req.Action {
	case permission.ActionBash:
		return "$"
	case permission.ActionEdit, permission.ActionWrite:
		return "←"
	case permission.ActionRead:
		return "→"
	case permission.ActionDelete:
		return "✕"
	case permission.ActionNetwork:
		return "%"
	case permission.ActionExternalDir:
		return "←"
	default:
		return "⚙"
	}
}
