// Package earlyinit must be imported before github.com/charmbracelet/bubbletea
// in cmd/dcode/main.go. Its init function pre-sets lipgloss's dark-background
// flag so that bubbletea's own init (tea_init.go) finds the value already
// cached and skips the OSC 11 terminal colour query entirely.
//
// Background: bubbletea v1 calls lipgloss.HasDarkBackground() in its package
// init as a workaround to avoid the query blocking later. On WSL2 the
// cursor-position response (\e[1;1R) arrives before the OSC 11 response, so
// termenv concludes "OSC not supported", exits early, and leaves the OSC reply
// (\e]11;rgb:0000/0000/0000\a) sitting in the PTY buffer.  BubbleTea then
// reads those bytes as keyboard input and they appear as garbage text in the
// textarea.  Calling SetHasDarkBackground here prevents the query from ever
// being sent.
package earlyinit

import "github.com/charmbracelet/lipgloss"

func init() {
	lipgloss.SetHasDarkBackground(true)
}
