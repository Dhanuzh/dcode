package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

// CommandItem represents a command in the palette
type CommandItem struct {
	ID          string
	Title       string
	Description string
	Category    string
	Keybind     string
	Icon        string
	Action      func() tea.Msg
}

// CommandPalette represents a command palette with fuzzy search
type CommandPalette struct {
	Commands       []CommandItem
	FilteredCmds   []CommandItem
	Input          textinput.Model
	Selected       int
	Width          int
	Height         int
	Visible        bool
	RecentCommands []string
	MaxRecent      int

	// Styling
	TitleColor    lipgloss.Color
	CategoryColor lipgloss.Color
	KeybindColor  lipgloss.Color
	SelectedBg    lipgloss.Color
	BorderColor   lipgloss.Color

	// Callbacks
	OnSelect func(item CommandItem) tea.Msg
	OnCancel func() tea.Msg
}

// CommandPaletteKeyMap defines keybindings
type CommandPaletteKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Cancel key.Binding
}

var DefaultCommandPaletteKeys = CommandPaletteKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "ctrl+k"),
		key.WithHelp("↑/ctrl+k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "ctrl+j"),
		key.WithHelp("↓/ctrl+j", "down"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
}

// NewCommandPalette creates a new command palette
func NewCommandPalette(width, height int) *CommandPalette {
	ti := textinput.New()
	ti.Placeholder = "Search commands..."
	ti.Focus()
	ti.Width = width - 4

	return &CommandPalette{
		Commands:       []CommandItem{},
		FilteredCmds:   []CommandItem{},
		Input:          ti,
		Selected:       0,
		Width:          width,
		Height:         height,
		Visible:        false,
		RecentCommands: []string{},
		MaxRecent:      10,
		TitleColor:     lipgloss.Color("#CBA6F7"),
		CategoryColor:  lipgloss.Color("#6C7086"),
		KeybindColor:   lipgloss.Color("#89B4FA"),
		SelectedBg:     lipgloss.Color("#313244"),
		BorderColor:    lipgloss.Color("#6C7086"),
	}
}

// AddCommand adds a command to the palette
func (cp *CommandPalette) AddCommand(cmd CommandItem) {
	cp.Commands = append(cp.Commands, cmd)
	cp.FilteredCmds = append(cp.FilteredCmds, cmd)
}

// AddCommands adds multiple commands
func (cp *CommandPalette) AddCommands(cmds []CommandItem) {
	cp.Commands = append(cp.Commands, cmds...)
	cp.FilteredCmds = append(cp.FilteredCmds, cmds...)
}

// Show shows the command palette
func (cp *CommandPalette) Show() {
	cp.Visible = true
	cp.Input.Focus()
	cp.Input.SetValue("")
	cp.FilteredCmds = cp.Commands
	cp.Selected = 0
}

// Hide hides the command palette
func (cp *CommandPalette) Hide() {
	cp.Visible = false
	cp.Input.Blur()
}

// IsVisible returns whether the palette is visible
func (cp *CommandPalette) IsVisible() bool {
	return cp.Visible
}

// GetSelected returns the currently selected command
func (cp *CommandPalette) GetSelected() *CommandItem {
	if cp.Selected < 0 || cp.Selected >= len(cp.FilteredCmds) {
		return nil
	}
	return &cp.FilteredCmds[cp.Selected]
}

// AddToRecent adds a command to recent commands
func (cp *CommandPalette) AddToRecent(id string) {
	// Remove if already in list
	for i, recent := range cp.RecentCommands {
		if recent == id {
			cp.RecentCommands = append(cp.RecentCommands[:i], cp.RecentCommands[i+1:]...)
			break
		}
	}

	// Add to front
	cp.RecentCommands = append([]string{id}, cp.RecentCommands...)

	// Trim to max
	if len(cp.RecentCommands) > cp.MaxRecent {
		cp.RecentCommands = cp.RecentCommands[:cp.MaxRecent]
	}
}

// filterCommands filters commands using fuzzy search
func (cp *CommandPalette) filterCommands(query string) {
	if query == "" {
		cp.FilteredCmds = cp.Commands
		cp.Selected = 0
		return
	}

	// Build searchable strings
	searchStrs := make([]string, len(cp.Commands))
	for i, cmd := range cp.Commands {
		searchStrs[i] = cmd.Title + " " + cmd.Description + " " + cmd.Category
	}

	// Fuzzy search
	matches := fuzzy.Find(query, searchStrs)

	// Build filtered list
	cp.FilteredCmds = make([]CommandItem, 0, len(matches))
	for _, match := range matches {
		cp.FilteredCmds = append(cp.FilteredCmds, cp.Commands[match.Index])
	}

	// Reset selection
	cp.Selected = 0
}

// Update handles command palette input
func (cp *CommandPalette) Update(msg tea.Msg) (*CommandPalette, tea.Cmd) {
	if !cp.Visible {
		return cp, nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultCommandPaletteKeys.Up):
			if cp.Selected > 0 {
				cp.Selected--
			}
		case key.Matches(msg, DefaultCommandPaletteKeys.Down):
			if cp.Selected < len(cp.FilteredCmds)-1 {
				cp.Selected++
			}
		case key.Matches(msg, DefaultCommandPaletteKeys.Select):
			selected := cp.GetSelected()
			if selected != nil {
				cp.AddToRecent(selected.ID)
				cp.Hide()
				if cp.OnSelect != nil {
					return cp, func() tea.Msg {
						return cp.OnSelect(*selected)
					}
				}
				if selected.Action != nil {
					return cp, func() tea.Msg {
						return selected.Action()
					}
				}
			}
		case key.Matches(msg, DefaultCommandPaletteKeys.Cancel):
			cp.Hide()
			if cp.OnCancel != nil {
				return cp, func() tea.Msg {
					return cp.OnCancel()
				}
			}
		default:
			// Update input
			cp.Input, cmd = cp.Input.Update(msg)
			// Filter commands based on input
			cp.filterCommands(cp.Input.Value())
			return cp, cmd
		}
	}

	return cp, cmd
}

// View renders the command palette
func (cp *CommandPalette) View() string {
	if !cp.Visible {
		return ""
	}

	var sections []string

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(cp.TitleColor).
		Padding(0, 1)
	sections = append(sections, titleStyle.Render("Command Palette"))

	// Input
	inputStyle := lipgloss.NewStyle().
		Padding(1, 2)
	sections = append(sections, inputStyle.Render(cp.Input.View()))

	// Results count
	countStyle := lipgloss.NewStyle().
		Foreground(cp.CategoryColor).
		Padding(0, 2)
	resultText := ""
	if cp.Input.Value() != "" {
		resultText = lipgloss.NewStyle().Render(
			countStyle.Render(lipgloss.NewStyle().Render(
				lipgloss.NewStyle().Render(
					lipgloss.NewStyle().Render(
						lipgloss.NewStyle().Render(
							lipgloss.NewStyle().Render(
								lipgloss.NewStyle().Render(
									resultText))))))))
	}

	// Commands list
	maxItems := cp.Height - 8
	if maxItems < 1 {
		maxItems = 1
	}

	visibleStart := cp.Selected - maxItems/2
	if visibleStart < 0 {
		visibleStart = 0
	}
	visibleEnd := visibleStart + maxItems
	if visibleEnd > len(cp.FilteredCmds) {
		visibleEnd = len(cp.FilteredCmds)
		visibleStart = visibleEnd - maxItems
		if visibleStart < 0 {
			visibleStart = 0
		}
	}

	for i := visibleStart; i < visibleEnd; i++ {
		cmd := cp.FilteredCmds[i]
		sections = append(sections, cp.renderCommand(cmd, i == cp.Selected))
	}

	// Empty state
	if len(cp.FilteredCmds) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(cp.CategoryColor).
			Italic(true).
			Padding(2, 2)
		sections = append(sections, emptyStyle.Render("No commands found"))
	}

	// Help
	helpStyle := lipgloss.NewStyle().
		Foreground(cp.CategoryColor).
		Padding(1, 2)
	sections = append(sections, helpStyle.Render("↑↓: navigate | enter: select | esc: cancel"))

	// Container
	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cp.BorderColor).
		Width(cp.Width-4).
		Padding(1, 0)

	return containerStyle.Render(content)
}

// renderCommand renders a single command item
func (cp *CommandPalette) renderCommand(cmd CommandItem, selected bool) string {
	var parts []string

	// Icon
	if cmd.Icon != "" {
		parts = append(parts, cmd.Icon)
	}

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true)
	if selected {
		titleStyle = titleStyle.Foreground(cp.TitleColor)
	}
	parts = append(parts, titleStyle.Render(cmd.Title))

	// Description
	if cmd.Description != "" {
		descStyle := lipgloss.NewStyle().
			Foreground(cp.CategoryColor).
			Italic(true)
		parts = append(parts, descStyle.Render(cmd.Description))
	}

	line := strings.Join(parts, " ")

	// Category and keybind on the right
	var rightParts []string
	if cmd.Category != "" {
		categoryStyle := lipgloss.NewStyle().
			Foreground(cp.CategoryColor)
		rightParts = append(rightParts, categoryStyle.Render("["+cmd.Category+"]"))
	}
	if cmd.Keybind != "" {
		keybindStyle := lipgloss.NewStyle().
			Foreground(cp.KeybindColor).
			Bold(true)
		rightParts = append(rightParts, keybindStyle.Render(cmd.Keybind))
	}

	if len(rightParts) > 0 {
		line += " " + strings.Join(rightParts, " ")
	}

	// Apply selection background
	itemStyle := lipgloss.NewStyle().
		Width(cp.Width-8).
		Padding(0, 2)

	if selected {
		itemStyle = itemStyle.Background(cp.SelectedBg)
	}

	return itemStyle.Render(line)
}

// RenderOverlay renders the command palette as a centered overlay
func (cp *CommandPalette) RenderOverlay(width, height int) string {
	if !cp.Visible {
		return ""
	}

	paletteView := cp.View()

	// Center the palette
	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		paletteView,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#1E1E2E")),
	)
}
