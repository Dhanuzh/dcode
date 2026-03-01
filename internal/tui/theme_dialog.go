package tui

// theme_dialog.go — theme picker dialog, matching opencode's dialog-theme-list.tsx.
//
// /theme (no args) or theme.change command opens ViewTheme.
// Up/down navigates; the live theme preview updates on movement.
// Enter confirms; Esc/q cancels and restores the previous theme.

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// themeDialogState holds all runtime state for the theme picker.
type themeDialogState struct {
	themes        []string // sorted theme names
	selected      int      // current cursor position
	filter        string   // incremental search
	previousTheme string   // theme to restore on cancel
}

// openThemeDialog opens the theme picker, saving the current theme for potential restore.
func (m *Model) openThemeDialog() (tea.Model, tea.Cmd) {
	names := m.themeRegistry.List()
	sort.Slice(names, func(i, j int) bool {
		return strings.ToLower(names[i]) < strings.ToLower(names[j])
	})

	current := m.themeRegistry.Current().Name
	sel := 0
	for i, n := range names {
		if n == current {
			sel = i
			break
		}
	}

	m.themeDialog = &themeDialogState{
		themes:        names,
		selected:      sel,
		previousTheme: current,
	}
	m.view = ViewTheme
	return m, nil
}

// filteredThemes returns themes that match the current filter.
func (s *themeDialogState) filteredThemes() []string {
	if s.filter == "" {
		return s.themes
	}
	f := strings.ToLower(s.filter)
	var out []string
	for _, n := range s.themes {
		if strings.Contains(strings.ToLower(n), f) {
			out = append(out, n)
		}
	}
	return out
}

// updateThemeDialog handles key input for the theme picker.
func (m Model) updateThemeDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.themeDialog == nil {
		m.view = ViewChat
		return m, nil
	}
	s := m.themeDialog
	items := s.filteredThemes()

	switch msg.String() {
	case "esc", "q":
		// Cancel — restore original theme
		_ = m.themeRegistry.SetCurrent(s.previousTheme)
		m.currentTheme = m.themeRegistry.Current()
		m.applyThemeToComponents()
		m.themeDialog = nil
		m.view = ViewChat
		m.focusTextarea()
		return m, nil

	case "enter":
		// Confirm
		if len(items) > 0 && s.selected < len(items) {
			chosen := items[s.selected]
			_ = m.themeRegistry.SetCurrent(chosen)
			m.currentTheme = m.themeRegistry.Current()
			m.applyThemeToComponents()
			m.Config.Theme = chosen
		}
		m.themeDialog = nil
		m.view = ViewChat
		m.focusTextarea()
		return m, nil

	case "up", "k", "ctrl+p":
		if s.selected > 0 {
			s.selected--
		} else if len(items) > 0 {
			s.selected = len(items) - 1
		}
		m.livePreviewTheme(items)
		return m, nil

	case "down", "j", "ctrl+n":
		if s.selected < len(items)-1 {
			s.selected++
		} else {
			s.selected = 0
		}
		m.livePreviewTheme(items)
		return m, nil

	case "backspace":
		if len(s.filter) > 0 {
			s.filter = s.filter[:len(s.filter)-1]
			s.selected = 0
			m.livePreviewTheme(s.filteredThemes())
		}
		return m, nil

	default:
		// Single printable char → add to filter
		if len(msg.String()) == 1 {
			s.filter += msg.String()
			s.selected = 0
			m.livePreviewTheme(s.filteredThemes())
		}
		return m, nil
	}
}

// livePreviewTheme sets the theme to the currently highlighted item without confirming.
func (m *Model) livePreviewTheme(items []string) {
	if m.themeDialog == nil || len(items) == 0 {
		return
	}
	idx := m.themeDialog.selected
	if idx >= len(items) {
		idx = len(items) - 1
	}
	_ = m.themeRegistry.SetCurrent(items[idx])
	m.currentTheme = m.themeRegistry.Current()
	m.applyThemeToComponents()
}

// applyThemeToComponents refreshes syntax/markdown/diff components after a theme change.
// The actual component constructors live in tui.go (components package is imported there).
// This helper is defined in tui.go to avoid import issues; here it's a forward reference.

// renderThemeDialog renders the theme picker overlay content.
func (m *Model) renderThemeDialog() string {
	if m.themeDialog == nil {
		return ""
	}
	s := m.themeDialog
	t := m.currentTheme
	items := s.filteredThemes()

	w := clampWidth(m.width, 52)

	dim := lipgloss.NewStyle().Foreground(t.TextMuted)
	title := lipgloss.NewStyle().Bold(true).Foreground(t.Primary).Padding(0, 1)
	selSt := lipgloss.NewStyle().
		Foreground(t.Background).
		Background(t.Primary).
		Bold(true).
		Width(w - 4)
	normalSt := lipgloss.NewStyle().
		Foreground(t.Text).
		Width(w - 4)
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderHighlight).
		Padding(0, 1).
		Width(w)

	var b strings.Builder
	b.WriteString(title.Render("Themes") + "\n")

	// Filter input
	filterLine := dim.Render("Filter: ")
	if s.filter != "" {
		filterLine += lipgloss.NewStyle().Foreground(t.Accent).Render(s.filter)
	} else {
		filterLine += lipgloss.NewStyle().Foreground(t.TextMuted).Render("type to search...")
	}
	b.WriteString(filterLine + "\n")
	b.WriteString(dim.Render(strings.Repeat("─", w-4)) + "\n")

	if len(items) == 0 {
		b.WriteString(dim.Render("  No themes match") + "\n")
	} else {
		// Show max 16 items with scrolling window
		maxShow := 16
		start := 0
		if s.selected >= maxShow {
			start = s.selected - maxShow + 1
		}
		if start+maxShow > len(items) {
			start = len(items) - maxShow
			if start < 0 {
				start = 0
			}
		}
		end := start + maxShow
		if end > len(items) {
			end = len(items)
		}
		visible := items[start:end]

		for i, name := range visible {
			globalIdx := start + i
			// Mark selected vs normal
			prefix := "  "
			if globalIdx == s.selected {
				b.WriteString(selSt.Render(prefix+name) + "\n")
			} else {
				b.WriteString(normalSt.Render(prefix+name) + "\n")
			}
		}

		// Scroll indicator
		if len(items) > maxShow {
			b.WriteString(dim.Render(strings.Repeat("─", w-4)) + "\n")
			b.WriteString(dim.Render(
				lipgloss.NewStyle().Foreground(t.TextMuted).Render(
					strings.Repeat(" ", w/2-8)+
						fmt.Sprintf("%d/%d", s.selected+1, len(items)),
				),
			) + "\n")
		}
	}

	b.WriteString(dim.Render(strings.Repeat("─", w-4)) + "\n")
	b.WriteString(dim.Render("  ↑/↓ navigate  Enter confirm  Esc cancel"))

	return border.Render(b.String())
}
