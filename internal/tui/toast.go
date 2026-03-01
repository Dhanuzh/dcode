package tui

// toast.go — non-blocking toast notifications, matching opencode's toast.tsx
// Toasts appear in the top-right corner and auto-dismiss after a configurable duration.

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ToastKind controls the colour of the toast.
type ToastKind int

const (
	ToastInfo ToastKind = iota
	ToastSuccess
	ToastWarning
	ToastError
)

// Toast is a single notification entry.
type Toast struct {
	Message string
	Kind    ToastKind
	Expiry  time.Time
}

// ToastDismissMsg is fired by the timer to remove an expired toast.
type ToastDismissMsg struct{}

// toastTickCmd schedules the next dismiss check after d.
func toastTickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return ToastDismissMsg{}
	})
}

// showToast adds a new toast and returns a Cmd to auto-dismiss it.
func (m *Model) showToast(msg string, kind ToastKind, dur time.Duration) tea.Cmd {
	if dur == 0 {
		dur = 4 * time.Second
	}
	m.toasts = append(m.toasts, Toast{
		Message: msg,
		Kind:    kind,
		Expiry:  time.Now().Add(dur),
	})
	return toastTickCmd(dur + 100*time.Millisecond)
}

// pruneToasts removes all expired toasts.
func (m *Model) pruneToasts() {
	now := time.Now()
	kept := m.toasts[:0]
	for _, t := range m.toasts {
		if now.Before(t.Expiry) {
			kept = append(kept, t)
		}
	}
	m.toasts = kept
}

// renderToasts returns the toast stack as a string to be placed in the top-right corner.
func (m *Model) renderToasts() string {
	m.pruneToasts()
	if len(m.toasts) == 0 {
		return ""
	}

	t := m.currentTheme
	var lines []string
	for _, toast := range m.toasts {
		var bg lipgloss.Color
		switch toast.Kind {
		case ToastSuccess:
			bg = t.Success
		case ToastWarning:
			bg = t.Warning
		case ToastError:
			bg = t.Error
		default:
			bg = t.Primary
		}
		style := lipgloss.NewStyle().
			Foreground(t.Background).
			Background(bg).
			Padding(0, 2).
			Bold(true)
		lines = append(lines, style.Render(toast.Message))
	}
	return strings.Join(lines, "\n")
}

// injectToastsIntoView overlays toasts in the top-right corner of a rendered screen.
func (m *Model) injectToastsIntoView(screen string) string {
	toast := m.renderToasts()
	if toast == "" {
		return screen
	}

	// Place the toast block in the top-right
	toastLines := strings.Split(toast, "\n")
	screenLines := strings.Split(screen, "\n")

	maxToastW := 0
	for _, l := range toastLines {
		if w := lipgloss.Width(l); w > maxToastW {
			maxToastW = w
		}
	}

	startX := m.width - maxToastW - 2
	if startX < 0 {
		startX = 0
	}

	result := make([]string, len(screenLines))
	for i, line := range screenLines {
		ti := i
		if ti < len(toastLines) {
			// Pad the screen line to startX then append toast
			lineW := lipgloss.Width(line)
			pad := startX - lineW
			if pad > 0 {
				line += strings.Repeat(" ", pad)
			}
			line += toastLines[ti]
		}
		result[i] = line
	}
	return strings.Join(result, "\n")
}
