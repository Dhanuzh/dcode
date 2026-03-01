package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DialogType represents the type of dialog
type DialogType int

const (
	DialogInfo DialogType = iota
	DialogWarning
	DialogError
	DialogConfirm
	DialogInput
	DialogOAuthCode // OAuth authorization code input dialog
)

// DialogButton represents a button in the dialog
type DialogButton struct {
	Label   string
	Primary bool
	Action  func() tea.Msg
}

// Dialog represents a modal dialog component
type Dialog struct {
	Type    DialogType
	Title   string
	Message string
	Buttons []DialogButton
	Width   int
	Height  int

	// Input dialog fields
	Input         string
	Placeholder   string
	cursorPos     int
	OnInputSubmit func(string) tea.Msg

	// OAuth code dialog fields
	OAuthURL         string // The authorization URL to display
	OAuthInstruction string // Instruction text (e.g., "Paste the authorization code here:")
	OAuthError       string // Error message (e.g., "Invalid code")

	// State
	selectedButton int
	visible        bool
}

// DialogKeyMap defines keybindings for dialogs
type DialogKeyMap struct {
	Left   key.Binding
	Right  key.Binding
	Enter  key.Binding
	Escape key.Binding
}

var DefaultDialogKeys = DialogKeyMap{
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "previous button"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "next button"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
}

// NewDialog creates a new dialog
func NewDialog(dialogType DialogType, title, message string, buttons []DialogButton) *Dialog {
	return &Dialog{
		Type:           dialogType,
		Title:          title,
		Message:        message,
		Buttons:        buttons,
		Width:          60,
		Height:         12,
		selectedButton: 0,
		visible:        false,
	}
}

// NewConfirmDialog creates a confirmation dialog
func NewConfirmDialog(title, message string, onConfirm, onCancel func() tea.Msg) *Dialog {
	return NewDialog(DialogConfirm, title, message, []DialogButton{
		{Label: "Confirm", Primary: true, Action: onConfirm},
		{Label: "Cancel", Primary: false, Action: onCancel},
	})
}

// NewInputDialog creates an input dialog
func NewInputDialog(title, placeholder string, onSubmit func(string) tea.Msg) *Dialog {
	return &Dialog{
		Type:          DialogInput,
		Title:         title,
		Placeholder:   placeholder,
		Width:         60,
		Height:        10,
		OnInputSubmit: onSubmit,
		Buttons: []DialogButton{
			{Label: "Submit", Primary: true, Action: nil},
			{Label: "Cancel", Primary: false, Action: nil},
		},
		visible: false,
	}
}

// NewOAuthCodeDialog creates an OAuth authorization code input dialog
// This displays an OAuth URL and prompts for the authorization code
func NewOAuthCodeDialog(title, url, instruction string, onSubmit func(string) tea.Msg) *Dialog {
	if instruction == "" {
		instruction = "Paste the authorization code here:"
	}
	return &Dialog{
		Type:             DialogOAuthCode,
		Title:            title,
		OAuthURL:         url,
		OAuthInstruction: instruction,
		Placeholder:      "Authorization code",
		Width:            70,
		Height:           14,
		OnInputSubmit:    onSubmit,
		visible:          false,
	}
}

// SetOAuthError sets an error message on the OAuth dialog
func (d *Dialog) SetOAuthError(err string) {
	d.OAuthError = err
}

// ClearOAuthError clears the error message
func (d *Dialog) ClearOAuthError() {
	d.OAuthError = ""
}

// Show makes the dialog visible
func (d *Dialog) Show() {
	d.visible = true
}

// Hide hides the dialog
func (d *Dialog) Hide() {
	d.visible = false
}

// IsVisible returns whether the dialog is visible
func (d *Dialog) IsVisible() bool {
	return d.visible
}

// Update handles dialog input
func (d *Dialog) Update(msg tea.Msg) (*Dialog, tea.Cmd) {
	if !d.visible {
		return d, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultDialogKeys.Left):
			if d.selectedButton > 0 {
				d.selectedButton--
			}
		case key.Matches(msg, DefaultDialogKeys.Right):
			if d.selectedButton < len(d.Buttons)-1 {
				d.selectedButton++
			}
		case key.Matches(msg, DefaultDialogKeys.Enter):
			if (d.Type == DialogInput || d.Type == DialogOAuthCode) && d.OnInputSubmit != nil {
				// Submit input
				input := d.Input
				d.Hide()
				return d, func() tea.Msg {
					return d.OnInputSubmit(input)
				}
			} else if len(d.Buttons) > 0 && d.selectedButton < len(d.Buttons) && d.Buttons[d.selectedButton].Action != nil {
				d.Hide()
				return d, func() tea.Msg {
					return d.Buttons[d.selectedButton].Action()
				}
			}
			d.Hide()
		case key.Matches(msg, DefaultDialogKeys.Escape):
			d.Hide()
		default:
			// Handle input for DialogInput and DialogOAuthCode types
			if d.Type == DialogInput || d.Type == DialogOAuthCode {
				switch msg.String() {
				case "backspace":
					if len(d.Input) > 0 && d.cursorPos > 0 {
						d.Input = d.Input[:d.cursorPos-1] + d.Input[d.cursorPos:]
						d.cursorPos--
					}
				case "delete":
					if d.cursorPos < len(d.Input) {
						d.Input = d.Input[:d.cursorPos] + d.Input[d.cursorPos+1:]
					}
				case "ctrl+v":
					// Allow paste (handled by terminal, but we accept multi-char input)
				default:
					if msg.Type == tea.KeyRunes {
						runes := msg.Runes
						for _, r := range runes {
							d.Input = d.Input[:d.cursorPos] + string(r) + d.Input[d.cursorPos:]
							d.cursorPos++
						}
					}
				}
			}
		}
	}

	return d, nil
}

// View renders the dialog
func (d *Dialog) View() string {
	if !d.visible {
		return ""
	}

	// OAuth code dialog has special rendering
	if d.Type == DialogOAuthCode {
		return d.renderOAuthCodeDialog()
	}

	// Determine colors based on dialog type
	var titleColor, borderColor lipgloss.Color
	switch d.Type {
	case DialogError:
		titleColor = lipgloss.Color("#F38BA8")
		borderColor = lipgloss.Color("#F38BA8")
	case DialogWarning:
		titleColor = lipgloss.Color("#F9E2AF")
		borderColor = lipgloss.Color("#F9E2AF")
	case DialogConfirm:
		titleColor = lipgloss.Color("#89B4FA")
		borderColor = lipgloss.Color("#89B4FA")
	default:
		titleColor = lipgloss.Color("#CBA6F7")
		borderColor = lipgloss.Color("#6C7086")
	}

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(titleColor).
		Padding(0, 1)

	// Message
	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CDD6F4")).
		Padding(1, 2).
		Width(d.Width - 4)

	// Input field (for DialogInput)
	var inputView string
	if d.Type == DialogInput {
		inputText := d.Input
		if inputText == "" {
			inputText = d.Placeholder
		}
		inputStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CDD6F4")).
			Background(lipgloss.Color("#313244")).
			Padding(0, 1).
			Width(d.Width - 6)
		inputView = inputStyle.Render(inputText)
	}

	// Buttons
	var buttonViews []string
	for i, btn := range d.Buttons {
		btnStyle := lipgloss.NewStyle().Padding(0, 2)
		if i == d.selectedButton {
			if btn.Primary {
				btnStyle = btnStyle.
					Foreground(lipgloss.Color("#1E1E2E")).
					Background(lipgloss.Color("#89B4FA")).
					Bold(true)
			} else {
				btnStyle = btnStyle.
					Foreground(lipgloss.Color("#1E1E2E")).
					Background(lipgloss.Color("#6C7086"))
			}
		} else {
			btnStyle = btnStyle.
				Foreground(lipgloss.Color("#CDD6F4")).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#6C7086"))
		}
		buttonViews = append(buttonViews, btnStyle.Render(btn.Label))
	}

	buttonsRow := lipgloss.JoinHorizontal(lipgloss.Left, buttonViews...)
	buttonsStyle := lipgloss.NewStyle().
		Padding(1, 0).
		Width(d.Width - 4).
		Align(lipgloss.Center)

	// Compose dialog content
	var content string
	if d.Type == DialogInput {
		content = lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render(d.Title),
			messageStyle.Render(d.Message),
			lipgloss.NewStyle().Padding(0, 2).Render(inputView),
			buttonsStyle.Render(buttonsRow),
		)
	} else {
		content = lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render(d.Title),
			messageStyle.Render(d.Message),
			buttonsStyle.Render(buttonsRow),
		)
	}

	// Dialog container with border
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 0).
		Width(d.Width)

	return dialogStyle.Render(content)
}

// renderOAuthCodeDialog renders the OAuth authorization code input dialog
// matching the opencode style with title, esc, URL, input, and enter submit
func (d *Dialog) renderOAuthCodeDialog() string {
	// Colors matching the screenshot
	textColor := lipgloss.Color("#CDD6F4")    // Light text
	textMuted := lipgloss.Color("#6C7086")    // Muted/gray text
	primaryColor := lipgloss.Color("#FAB387") // Orange/amber for URL (Catppuccin Peach)
	errorColor := lipgloss.Color("#F38BA8")   // Red for errors
	borderColor := lipgloss.Color("#6C7086")  // Border color

	var b strings.Builder

	// Header row: Title on left, "esc" on right
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(textColor)

	escStyle := lipgloss.NewStyle().
		Foreground(textMuted)

	headerWidth := d.Width - 4
	titleText := titleStyle.Render(d.Title)
	escText := escStyle.Render("esc")

	// Calculate padding to right-align "esc"
	titleLen := lipgloss.Width(titleText)
	escLen := lipgloss.Width(escText)
	padding := headerWidth - titleLen - escLen
	if padding < 1 {
		padding = 1
	}

	b.WriteString(titleText + strings.Repeat(" ", padding) + escText + "\n\n")

	// Instruction text
	instructionStyle := lipgloss.NewStyle().
		Foreground(textMuted)

	b.WriteString(instructionStyle.Render(d.OAuthInstruction) + "\n\n")

	// OAuth URL in orange/primary color (wrapped if needed)
	urlStyle := lipgloss.NewStyle().
		Foreground(primaryColor)

	// Wrap URL to fit dialog width
	wrappedURL := wrapText(d.OAuthURL, d.Width-6)
	b.WriteString(urlStyle.Render(wrappedURL) + "\n\n")

	// Error message (if any)
	if d.OAuthError != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(errorColor)
		b.WriteString(errorStyle.Render(d.OAuthError) + "\n\n")
	}

	// Input field with placeholder
	inputText := d.Input
	inputFgColor := textColor
	if inputText == "" {
		inputText = d.Placeholder
		inputFgColor = textMuted
	}

	// Add cursor indicator
	if d.Input != "" {
		inputText = d.Input + "▊"
	}

	inputStyle := lipgloss.NewStyle().
		Foreground(inputFgColor).
		Background(lipgloss.Color("#313244")).
		Padding(0, 1).
		Width(d.Width - 6)

	b.WriteString(inputStyle.Render(inputText) + "\n\n")

	// Footer: "enter submit"
	enterStyle := lipgloss.NewStyle().
		Foreground(textColor)

	submitStyle := lipgloss.NewStyle().
		Foreground(textMuted)

	b.WriteString(enterStyle.Render("enter") + " " + submitStyle.Render("submit"))

	// Dialog container with border
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 2).
		Width(d.Width)

	return dialogStyle.Render(b.String())
}

// wrapText wraps text to fit within maxWidth
func wrapText(text string, maxWidth int) string {
	if len(text) <= maxWidth {
		return text
	}

	var result strings.Builder
	for len(text) > maxWidth {
		result.WriteString(text[:maxWidth])
		result.WriteString("\n")
		text = text[maxWidth:]
	}
	if len(text) > 0 {
		result.WriteString(text)
	}
	return result.String()
}

// RenderOverlay renders the dialog as an overlay (with background dim)
func (d *Dialog) RenderOverlay(width, height int) string {
	if !d.visible {
		return ""
	}

	dialogView := d.View()

	// Center the dialog
	dialogHeight := strings.Count(dialogView, "\n") + 1
	dialogWidth := d.Width

	verticalPadding := (height - dialogHeight) / 2
	horizontalPadding := (width - dialogWidth) / 2

	if verticalPadding < 0 {
		verticalPadding = 0
	}
	if horizontalPadding < 0 {
		horizontalPadding = 0
	}

	// Create centered dialog
	centered := lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		dialogView,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#1E1E2E")),
	)

	return centered
}
