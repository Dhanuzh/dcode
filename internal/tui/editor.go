package tui

// editor.go — External $EDITOR integration (ctrl+e)
//
// When the user presses ctrl+e, the current textarea content is written to a
// temp file, the user's $EDITOR is launched (via tea.ExecProcess), and when
// the editor exits the file content is read back into the textarea.

import (
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

// ExternalEditorDoneMsg is sent after the external editor process exits.
type ExternalEditorDoneMsg struct {
	Content string
	Err     error
}

// openExternalEditor suspends the TUI, opens $EDITOR on a temp file
// containing the current textarea text, then resumes and sends ExternalEditorDoneMsg.
func (m *Model) openExternalEditor() (tea.Model, tea.Cmd) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		// Fallback to common editors in order of preference
		for _, e := range []string{"nano", "vim", "vi"} {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
	}
	if editor == "" {
		m.setStatus("$EDITOR is not set")
		return m, nil
	}

	// Write current textarea content to a temp file
	content := m.textarea.Value()
	tmp, err := os.CreateTemp("", "dcode-edit-*.md")
	if err != nil {
		m.setStatus("Failed to create temp file: " + err.Error())
		return m, nil
	}
	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		m.setStatus("Failed to write temp file: " + err.Error())
		return m, nil
	}
	_ = tmp.Close()

	tmpPath := tmp.Name()

	// Build the editor command
	cmd := exec.Command(editor, tmpPath) //nolint:gosec
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		defer os.Remove(tmpPath)
		if err != nil {
			return ExternalEditorDoneMsg{Err: err}
		}
		data, readErr := os.ReadFile(tmpPath)
		if readErr != nil {
			return ExternalEditorDoneMsg{Err: readErr}
		}
		return ExternalEditorDoneMsg{Content: string(data)}
	})
}
