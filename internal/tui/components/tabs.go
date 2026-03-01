package components

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Tab represents a single tab
type Tab struct {
	ID      string
	Title   string
	Content string
	Data    interface{} // Custom data attached to tab
	Dirty   bool        // Unsaved changes indicator
}

// TabBar manages multiple tabs
type TabBar struct {
	Tabs        []*Tab
	ActiveTab   int
	Width       int
	MaxTabs     int
	ShowNumbers bool // Show tab numbers (1-9)
	Closeable   bool // Allow closing tabs

	// Styling
	ActiveColor   lipgloss.Color
	InactiveColor lipgloss.Color
	BorderColor   lipgloss.Color

	// Callbacks
	OnTabChange func(tab *Tab) tea.Msg
	OnTabClose  func(tab *Tab) tea.Msg
}

// TabBarKeyMap defines keybindings for tabs
type TabBarKeyMap struct {
	TabNext  key.Binding
	TabPrev  key.Binding
	Tab1     key.Binding
	Tab2     key.Binding
	Tab3     key.Binding
	Tab4     key.Binding
	Tab5     key.Binding
	Tab6     key.Binding
	Tab7     key.Binding
	Tab8     key.Binding
	Tab9     key.Binding
	TabClose key.Binding
	TabNew   key.Binding
}

var DefaultTabBarKeys = TabBarKeyMap{
	TabNext: key.NewBinding(
		key.WithKeys("ctrl+tab", "ctrl+n"),
		key.WithHelp("ctrl+tab", "next tab"),
	),
	TabPrev: key.NewBinding(
		key.WithKeys("ctrl+shift+tab", "ctrl+p"),
		key.WithHelp("ctrl+shift+tab", "prev tab"),
	),
	Tab1: key.NewBinding(
		key.WithKeys("alt+1"),
		key.WithHelp("alt+1", "tab 1"),
	),
	Tab2: key.NewBinding(
		key.WithKeys("alt+2"),
		key.WithHelp("alt+2", "tab 2"),
	),
	Tab3: key.NewBinding(
		key.WithKeys("alt+3"),
		key.WithHelp("alt+3", "tab 3"),
	),
	Tab4: key.NewBinding(
		key.WithKeys("alt+4"),
		key.WithHelp("alt+4", "tab 4"),
	),
	Tab5: key.NewBinding(
		key.WithKeys("alt+5"),
		key.WithHelp("alt+5", "tab 5"),
	),
	Tab6: key.NewBinding(
		key.WithKeys("alt+6"),
		key.WithHelp("alt+6", "tab 6"),
	),
	Tab7: key.NewBinding(
		key.WithKeys("alt+7"),
		key.WithHelp("alt+7", "tab 7"),
	),
	Tab8: key.NewBinding(
		key.WithKeys("alt+8"),
		key.WithHelp("alt+8", "tab 8"),
	),
	Tab9: key.NewBinding(
		key.WithKeys("alt+9"),
		key.WithHelp("alt+9", "tab 9"),
	),
	TabClose: key.NewBinding(
		key.WithKeys("ctrl+w"),
		key.WithHelp("ctrl+w", "close tab"),
	),
	TabNew: key.NewBinding(
		key.WithKeys("ctrl+t"),
		key.WithHelp("ctrl+t", "new tab"),
	),
}

// NewTabBar creates a new tab bar
func NewTabBar(width int) *TabBar {
	return &TabBar{
		Tabs:          []*Tab{},
		ActiveTab:     0,
		Width:         width,
		MaxTabs:       9,
		ShowNumbers:   true,
		Closeable:     true,
		ActiveColor:   lipgloss.Color("#89B4FA"),
		InactiveColor: lipgloss.Color("#6C7086"),
		BorderColor:   lipgloss.Color("#313244"),
	}
}

// AddTab adds a new tab
func (tb *TabBar) AddTab(tab *Tab) {
	if len(tb.Tabs) >= tb.MaxTabs {
		return
	}
	tb.Tabs = append(tb.Tabs, tab)
}

// RemoveTab removes a tab by index
func (tb *TabBar) RemoveTab(index int) {
	if index < 0 || index >= len(tb.Tabs) {
		return
	}

	tab := tb.Tabs[index]

	// Call onClose callback if set
	if tb.OnTabClose != nil {
		tb.OnTabClose(tab)
	}

	tb.Tabs = append(tb.Tabs[:index], tb.Tabs[index+1:]...)

	// Adjust active tab
	if tb.ActiveTab >= len(tb.Tabs) {
		tb.ActiveTab = len(tb.Tabs) - 1
	}
	if tb.ActiveTab < 0 {
		tb.ActiveTab = 0
	}
}

// CloseCurrentTab closes the currently active tab
func (tb *TabBar) CloseCurrentTab() {
	if len(tb.Tabs) > 0 && tb.Closeable {
		tb.RemoveTab(tb.ActiveTab)
	}
}

// NextTab switches to the next tab
func (tb *TabBar) NextTab() {
	if len(tb.Tabs) == 0 {
		return
	}
	tb.ActiveTab = (tb.ActiveTab + 1) % len(tb.Tabs)

	if tb.OnTabChange != nil && tb.ActiveTab < len(tb.Tabs) {
		tb.OnTabChange(tb.Tabs[tb.ActiveTab])
	}
}

// PrevTab switches to the previous tab
func (tb *TabBar) PrevTab() {
	if len(tb.Tabs) == 0 {
		return
	}
	tb.ActiveTab--
	if tb.ActiveTab < 0 {
		tb.ActiveTab = len(tb.Tabs) - 1
	}

	if tb.OnTabChange != nil && tb.ActiveTab < len(tb.Tabs) {
		tb.OnTabChange(tb.Tabs[tb.ActiveTab])
	}
}

// SwitchToTab switches to a specific tab by index
func (tb *TabBar) SwitchToTab(index int) {
	if index < 0 || index >= len(tb.Tabs) {
		return
	}
	tb.ActiveTab = index

	if tb.OnTabChange != nil {
		tb.OnTabChange(tb.Tabs[tb.ActiveTab])
	}
}

// GetActiveTab returns the currently active tab
func (tb *TabBar) GetActiveTab() *Tab {
	if tb.ActiveTab < 0 || tb.ActiveTab >= len(tb.Tabs) {
		return nil
	}
	return tb.Tabs[tb.ActiveTab]
}

// GetTab returns a tab by ID
func (tb *TabBar) GetTab(id string) *Tab {
	for _, tab := range tb.Tabs {
		if tab.ID == id {
			return tab
		}
	}
	return nil
}

// SetWidth sets the tab bar width
func (tb *TabBar) SetWidth(width int) {
	tb.Width = width
}

// Update handles tab bar input
func (tb *TabBar) Update(msg tea.Msg) (*TabBar, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultTabBarKeys.TabNext):
			tb.NextTab()
		case key.Matches(msg, DefaultTabBarKeys.TabPrev):
			tb.PrevTab()
		case key.Matches(msg, DefaultTabBarKeys.Tab1):
			tb.SwitchToTab(0)
		case key.Matches(msg, DefaultTabBarKeys.Tab2):
			tb.SwitchToTab(1)
		case key.Matches(msg, DefaultTabBarKeys.Tab3):
			tb.SwitchToTab(2)
		case key.Matches(msg, DefaultTabBarKeys.Tab4):
			tb.SwitchToTab(3)
		case key.Matches(msg, DefaultTabBarKeys.Tab5):
			tb.SwitchToTab(4)
		case key.Matches(msg, DefaultTabBarKeys.Tab6):
			tb.SwitchToTab(5)
		case key.Matches(msg, DefaultTabBarKeys.Tab7):
			tb.SwitchToTab(6)
		case key.Matches(msg, DefaultTabBarKeys.Tab8):
			tb.SwitchToTab(7)
		case key.Matches(msg, DefaultTabBarKeys.Tab9):
			tb.SwitchToTab(8)
		case key.Matches(msg, DefaultTabBarKeys.TabClose):
			tb.CloseCurrentTab()
		}
	case tea.MouseMsg:
		// Handle mouse clicks on tabs
		if msg.Type == tea.MouseLeft {
			tb.handleTabClick(msg.X)
		}
	}

	return tb, nil
}

// handleTabClick handles clicking on tabs
func (tb *TabBar) handleTabClick(x int) {
	// Calculate which tab was clicked based on X position
	currentX := 0
	for i, tab := range tb.Tabs {
		tabWidth := tb.calculateTabWidth(tab)
		if x >= currentX && x < currentX+tabWidth {
			tb.SwitchToTab(i)
			return
		}
		currentX += tabWidth
	}
}

// calculateTabWidth calculates the width of a tab
func (tb *TabBar) calculateTabWidth(tab *Tab) int {
	// Base width from title
	width := len(tab.Title) + 4 // padding

	if tb.ShowNumbers {
		width += 3 // " 1:"
	}

	if tab.Dirty {
		width += 2 // " *"
	}

	if tb.Closeable {
		width += 2 // " x"
	}

	return width
}

// View renders the tab bar
func (tb *TabBar) View() string {
	if len(tb.Tabs) == 0 {
		return ""
	}

	var tabs []string

	for i, tab := range tb.Tabs {
		isActive := i == tb.ActiveTab

		// Build tab content
		var content string

		// Number (if enabled)
		if tb.ShowNumbers && i < 9 {
			content += fmt.Sprintf("%d:", i+1)
		}

		// Title
		content += " " + tab.Title

		// Dirty indicator
		if tab.Dirty {
			content += " *"
		}

		// Close button (if closeable)
		if tb.Closeable {
			content += " ×"
		}

		// Apply styling
		var tabStyle lipgloss.Style

		if isActive {
			tabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#1E1E2E")).
				Background(tb.ActiveColor).
				Bold(true).
				Padding(0, 1)
		} else {
			tabStyle = lipgloss.NewStyle().
				Foreground(tb.InactiveColor).
				Background(tb.BorderColor).
				Padding(0, 1)
		}

		tabs = append(tabs, tabStyle.Render(content))
	}

	// Join all tabs
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	// Add container with bottom border
	containerStyle := lipgloss.NewStyle().
		Width(tb.Width).
		Border(lipgloss.Border{Bottom: "─"}).
		BorderForeground(tb.BorderColor)

	return containerStyle.Render(tabBar)
}

// ViewContent renders the active tab's content
func (tb *TabBar) ViewContent() string {
	tab := tb.GetActiveTab()
	if tab == nil {
		return ""
	}
	return tab.Content
}

// RenameTab renames a tab
func (tb *TabBar) RenameTab(index int, newTitle string) {
	if index >= 0 && index < len(tb.Tabs) {
		tb.Tabs[index].Title = newTitle
	}
}

// MarkDirty marks a tab as having unsaved changes
func (tb *TabBar) MarkDirty(index int, dirty bool) {
	if index >= 0 && index < len(tb.Tabs) {
		tb.Tabs[index].Dirty = dirty
	}
}
