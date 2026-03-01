package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ListItem represents an item in the list
type ListItem struct {
	Title       string
	Description string
	Icon        string
	Data        interface{}
	Selected    bool
	Disabled    bool
}

// List represents a selectable list component
type List struct {
	Items         []ListItem
	cursor        int
	offset        int
	Height        int
	Width         int
	MultiSelect   bool
	ShowIcons     bool
	ShowSelection bool
	Title         string
	EmptyMessage  string

	// Callbacks
	OnSelect func(item ListItem) tea.Msg
	OnChange func(item ListItem)
}

// ListKeyMap defines keybindings for lists
type ListKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Space  key.Binding
	Escape key.Binding
}

var DefaultListKeys = ListKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Space: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
}

// NewList creates a new list component
func NewList(items []ListItem, height int) *List {
	return &List{
		Items:         items,
		cursor:        0,
		offset:        0,
		Height:        height,
		Width:         60,
		MultiSelect:   false,
		ShowIcons:     true,
		ShowSelection: true,
		EmptyMessage:  "No items",
	}
}

// SetItems updates the list items
func (l *List) SetItems(items []ListItem) {
	l.Items = items
	if l.cursor >= len(items) {
		l.cursor = len(items) - 1
	}
	if l.cursor < 0 {
		l.cursor = 0
	}
	l.updateOffset()
}

// AddItem adds an item to the list
func (l *List) AddItem(item ListItem) {
	l.Items = append(l.Items, item)
}

// RemoveItem removes an item at index
func (l *List) RemoveItem(index int) {
	if index < 0 || index >= len(l.Items) {
		return
	}
	l.Items = append(l.Items[:index], l.Items[index+1:]...)
	if l.cursor >= len(l.Items) {
		l.cursor = len(l.Items) - 1
	}
	if l.cursor < 0 {
		l.cursor = 0
	}
	l.updateOffset()
}

// GetCursor returns the current cursor position
func (l *List) GetCursor() int {
	return l.cursor
}

// SetCursor sets the cursor position
func (l *List) SetCursor(pos int) {
	if pos < 0 {
		pos = 0
	}
	if pos >= len(l.Items) {
		pos = len(l.Items) - 1
	}
	l.cursor = pos
	l.updateOffset()
}

// GetSelectedItem returns the currently selected item
func (l *List) GetSelectedItem() *ListItem {
	if l.cursor < 0 || l.cursor >= len(l.Items) {
		return nil
	}
	return &l.Items[l.cursor]
}

// GetSelectedItems returns all selected items (for multi-select)
func (l *List) GetSelectedItems() []ListItem {
	var selected []ListItem
	for _, item := range l.Items {
		if item.Selected {
			selected = append(selected, item)
		}
	}
	return selected
}

// ClearSelection clears all selections
func (l *List) ClearSelection() {
	for i := range l.Items {
		l.Items[i].Selected = false
	}
}

// updateOffset adjusts the scroll offset to keep cursor in view
func (l *List) updateOffset() {
	if l.cursor < l.offset {
		l.offset = l.cursor
	}
	if l.cursor >= l.offset+l.Height {
		l.offset = l.cursor - l.Height + 1
	}
	if l.offset < 0 {
		l.offset = 0
	}
}

// Update handles list input
func (l *List) Update(msg tea.Msg) (*List, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultListKeys.Up):
			if l.cursor > 0 {
				l.cursor--
				l.updateOffset()
				if l.OnChange != nil && l.cursor < len(l.Items) {
					l.OnChange(l.Items[l.cursor])
				}
			}
		case key.Matches(msg, DefaultListKeys.Down):
			if l.cursor < len(l.Items)-1 {
				l.cursor++
				l.updateOffset()
				if l.OnChange != nil && l.cursor < len(l.Items) {
					l.OnChange(l.Items[l.cursor])
				}
			}
		case key.Matches(msg, DefaultListKeys.Enter):
			if l.cursor >= 0 && l.cursor < len(l.Items) {
				item := l.Items[l.cursor]
				if !item.Disabled {
					if l.OnSelect != nil {
						return l, func() tea.Msg {
							return l.OnSelect(item)
						}
					}
				}
			}
		case key.Matches(msg, DefaultListKeys.Space):
			if l.MultiSelect && l.cursor >= 0 && l.cursor < len(l.Items) {
				if !l.Items[l.cursor].Disabled {
					l.Items[l.cursor].Selected = !l.Items[l.cursor].Selected
				}
			}
		}
	}

	return l, nil
}

// View renders the list
func (l *List) View() string {
	if len(l.Items) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086")).
			Italic(true).
			Padding(1, 2)
		return emptyStyle.Render(l.EmptyMessage)
	}

	var items []string

	// Render visible items
	visibleEnd := l.offset + l.Height
	if visibleEnd > len(l.Items) {
		visibleEnd = len(l.Items)
	}

	for i := l.offset; i < visibleEnd; i++ {
		item := l.Items[i]
		items = append(items, l.renderItem(item, i == l.cursor))
	}

	// Add title if present
	if l.Title != "" {
		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#CBA6F7")).
			Padding(0, 1).
			Width(l.Width)
		items = append([]string{titleStyle.Render(l.Title)}, items...)
	}

	// Add scroll indicators
	if l.offset > 0 {
		scrollUpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086")).
			Align(lipgloss.Center).
			Width(l.Width)
		items = append([]string{scrollUpStyle.Render("↑ More items above")}, items...)
	}
	if visibleEnd < len(l.Items) {
		scrollDownStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086")).
			Align(lipgloss.Center).
			Width(l.Width)
		items = append(items, scrollDownStyle.Render("↓ More items below"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, items...)
}

// renderItem renders a single list item
func (l *List) renderItem(item ListItem, isCursor bool) string {
	// Build item content
	var parts []string

	// Selection indicator (for multi-select)
	if l.MultiSelect && l.ShowSelection {
		if item.Selected {
			parts = append(parts, lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A6E3A1")).
				Render("☑"))
		} else {
			parts = append(parts, lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6C7086")).
				Render("☐"))
		}
	}

	// Icon
	if l.ShowIcons && item.Icon != "" {
		parts = append(parts, item.Icon)
	}

	// Title
	titleStyle := lipgloss.NewStyle()
	if item.Disabled {
		titleStyle = titleStyle.Foreground(lipgloss.Color("#6C7086")).Italic(true)
	} else {
		titleStyle = titleStyle.Foreground(lipgloss.Color("#CDD6F4"))
	}
	parts = append(parts, titleStyle.Render(item.Title))

	// Description (on new line if present)
	var descriptionLine string
	if item.Description != "" {
		descStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086")).
			Italic(true)
		descriptionLine = "\n  " + descStyle.Render(item.Description)
	}

	content := strings.Join(parts, " ") + descriptionLine

	// Apply cursor style
	itemStyle := lipgloss.NewStyle().
		Width(l.Width-2).
		Padding(0, 1)

	if isCursor {
		itemStyle = itemStyle.
			Background(lipgloss.Color("#313244")).
			Border(lipgloss.Border{Left: "▐"}).
			BorderForeground(lipgloss.Color("#89B4FA"))
	}

	return itemStyle.Render(content)
}

// Filter filters items by predicate
func (l *List) Filter(predicate func(item ListItem) bool) {
	var filtered []ListItem
	for _, item := range l.Items {
		if predicate(item) {
			filtered = append(filtered, item)
		}
	}
	l.SetItems(filtered)
}

// Sort sorts items by comparator
func (l *List) Sort(less func(i, j ListItem) bool) {
	// Simple bubble sort (good enough for TUI lists)
	n := len(l.Items)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if less(l.Items[j+1], l.Items[j]) {
				l.Items[j], l.Items[j+1] = l.Items[j+1], l.Items[j]
			}
		}
	}
}
