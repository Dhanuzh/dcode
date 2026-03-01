package components

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PaneOrientation defines how panes are split
type PaneOrientation int

const (
	Horizontal PaneOrientation = iota // Left-Right split
	Vertical                          // Top-Bottom split
)

// Pane represents a single pane in a split layout
type Pane struct {
	ID        string
	Content   string
	Title     string
	Width     int
	Height    int
	Focused   bool
	MinWidth  int
	MinHeight int

	// Custom render function
	Render func(width, height int, focused bool) string
}

// SplitPane manages a split pane layout
type SplitPane struct {
	Orientation PaneOrientation
	Panes       []*Pane
	Sizes       []int // Size of each pane (width for horizontal, height for vertical)
	FocusedPane int
	Width       int
	Height      int
	Resizing    bool
	ResizeIndex int // Which divider is being resized

	// Styling
	DividerChar  string
	DividerColor lipgloss.Color
	FocusColor   lipgloss.Color
}

// SplitPaneKeyMap defines keybindings for split panes
type SplitPaneKeyMap struct {
	FocusNext   key.Binding
	FocusPrev   key.Binding
	ResizeLeft  key.Binding
	ResizeRight key.Binding
	ResizeUp    key.Binding
	ResizeDown  key.Binding
}

var DefaultSplitPaneKeys = SplitPaneKeyMap{
	FocusNext: key.NewBinding(
		key.WithKeys("ctrl+]", "ctrl+l"),
		key.WithHelp("ctrl+]", "next pane"),
	),
	FocusPrev: key.NewBinding(
		key.WithKeys("ctrl+[", "ctrl+h"),
		key.WithHelp("ctrl+[", "prev pane"),
	),
	ResizeLeft: key.NewBinding(
		key.WithKeys("ctrl+left"),
		key.WithHelp("ctrl+←", "resize left"),
	),
	ResizeRight: key.NewBinding(
		key.WithKeys("ctrl+right"),
		key.WithHelp("ctrl+→", "resize right"),
	),
	ResizeUp: key.NewBinding(
		key.WithKeys("ctrl+up"),
		key.WithHelp("ctrl+↑", "resize up"),
	),
	ResizeDown: key.NewBinding(
		key.WithKeys("ctrl+down"),
		key.WithHelp("ctrl+↓", "resize down"),
	),
}

// NewSplitPane creates a new split pane layout
func NewSplitPane(orientation PaneOrientation, width, height int) *SplitPane {
	return &SplitPane{
		Orientation:  orientation,
		Panes:        []*Pane{},
		Sizes:        []int{},
		FocusedPane:  0,
		Width:        width,
		Height:       height,
		DividerChar:  "│",
		DividerColor: lipgloss.Color("#6C7086"),
		FocusColor:   lipgloss.Color("#89B4FA"),
	}
}

// AddPane adds a pane to the layout
func (sp *SplitPane) AddPane(pane *Pane) {
	sp.Panes = append(sp.Panes, pane)

	// Calculate initial size
	if sp.Orientation == Horizontal {
		// Distribute width evenly
		paneWidth := sp.Width / len(sp.Panes)
		sp.Sizes = make([]int, len(sp.Panes))
		for i := range sp.Sizes {
			sp.Sizes[i] = paneWidth
		}
	} else {
		// Distribute height evenly
		paneHeight := sp.Height / len(sp.Panes)
		sp.Sizes = make([]int, len(sp.Panes))
		for i := range sp.Sizes {
			sp.Sizes[i] = paneHeight
		}
	}

	sp.updatePaneSizes()
}

// RemovePane removes a pane by index
func (sp *SplitPane) RemovePane(index int) {
	if index < 0 || index >= len(sp.Panes) {
		return
	}

	sp.Panes = append(sp.Panes[:index], sp.Panes[index+1:]...)
	sp.Sizes = append(sp.Sizes[:index], sp.Sizes[index+1:]...)

	// Adjust focused pane
	if sp.FocusedPane >= len(sp.Panes) {
		sp.FocusedPane = len(sp.Panes) - 1
	}
	if sp.FocusedPane < 0 {
		sp.FocusedPane = 0
	}

	sp.updatePaneSizes()
}

// FocusNext focuses the next pane
func (sp *SplitPane) FocusNext() {
	if len(sp.Panes) == 0 {
		return
	}
	sp.Panes[sp.FocusedPane].Focused = false
	sp.FocusedPane = (sp.FocusedPane + 1) % len(sp.Panes)
	sp.Panes[sp.FocusedPane].Focused = true
}

// FocusPrev focuses the previous pane
func (sp *SplitPane) FocusPrev() {
	if len(sp.Panes) == 0 {
		return
	}
	sp.Panes[sp.FocusedPane].Focused = false
	sp.FocusedPane--
	if sp.FocusedPane < 0 {
		sp.FocusedPane = len(sp.Panes) - 1
	}
	sp.Panes[sp.FocusedPane].Focused = true
}

// GetFocusedPane returns the currently focused pane
func (sp *SplitPane) GetFocusedPane() *Pane {
	if sp.FocusedPane < 0 || sp.FocusedPane >= len(sp.Panes) {
		return nil
	}
	return sp.Panes[sp.FocusedPane]
}

// Resize adjusts pane sizes
func (sp *SplitPane) Resize(delta int) {
	if len(sp.Panes) < 2 {
		return
	}

	// Resize focused pane and adjacent pane
	focusedIdx := sp.FocusedPane
	adjacentIdx := focusedIdx + 1
	if adjacentIdx >= len(sp.Panes) {
		adjacentIdx = focusedIdx - 1
		focusedIdx = adjacentIdx + 1
	}

	if adjacentIdx < 0 {
		return
	}

	// Apply resize with bounds checking
	newFocusedSize := sp.Sizes[focusedIdx] + delta
	newAdjacentSize := sp.Sizes[adjacentIdx] - delta

	// Check minimum sizes
	minSize := 10
	if sp.Orientation == Horizontal {
		if sp.Panes[focusedIdx].MinWidth > 0 {
			minSize = sp.Panes[focusedIdx].MinWidth
		}
	} else {
		if sp.Panes[focusedIdx].MinHeight > 0 {
			minSize = sp.Panes[focusedIdx].MinHeight
		}
	}

	if newFocusedSize >= minSize && newAdjacentSize >= minSize {
		sp.Sizes[focusedIdx] = newFocusedSize
		sp.Sizes[adjacentIdx] = newAdjacentSize
		sp.updatePaneSizes()
	}
}

// SetSize sets the total size of the split pane
func (sp *SplitPane) SetSize(width, height int) {
	sp.Width = width
	sp.Height = height
	sp.updatePaneSizes()
}

// updatePaneSizes updates individual pane dimensions
func (sp *SplitPane) updatePaneSizes() {
	if sp.Orientation == Horizontal {
		// Horizontal split - distribute width
		for i, pane := range sp.Panes {
			pane.Width = sp.Sizes[i]
			pane.Height = sp.Height
		}
	} else {
		// Vertical split - distribute height
		for i, pane := range sp.Panes {
			pane.Width = sp.Width
			pane.Height = sp.Sizes[i]
		}
	}
}

// Update handles split pane input
func (sp *SplitPane) Update(msg tea.Msg) (*SplitPane, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultSplitPaneKeys.FocusNext):
			sp.FocusNext()
		case key.Matches(msg, DefaultSplitPaneKeys.FocusPrev):
			sp.FocusPrev()
		case key.Matches(msg, DefaultSplitPaneKeys.ResizeRight):
			if sp.Orientation == Horizontal {
				sp.Resize(5)
			}
		case key.Matches(msg, DefaultSplitPaneKeys.ResizeLeft):
			if sp.Orientation == Horizontal {
				sp.Resize(-5)
			}
		case key.Matches(msg, DefaultSplitPaneKeys.ResizeDown):
			if sp.Orientation == Vertical {
				sp.Resize(2)
			}
		case key.Matches(msg, DefaultSplitPaneKeys.ResizeUp):
			if sp.Orientation == Vertical {
				sp.Resize(-2)
			}
		}
	case tea.MouseMsg:
		// Handle mouse resize (clicking and dragging dividers)
		if msg.Type == tea.MouseLeft {
			// Check if click is on a divider
			sp.handleDividerClick(msg.X, msg.Y)
		}
	}

	return sp, nil
}

// handleDividerClick handles clicking on pane dividers
func (sp *SplitPane) handleDividerClick(x, y int) {
	// Calculate divider positions and check if click is on one
	if sp.Orientation == Horizontal {
		currentX := 0
		for i := 0; i < len(sp.Panes)-1; i++ {
			currentX += sp.Sizes[i]
			// Divider is at currentX, allow 2-char click zone
			if x >= currentX-1 && x <= currentX+1 {
				sp.ResizeIndex = i
				sp.Resizing = true
				return
			}
		}
	} else {
		currentY := 0
		for i := 0; i < len(sp.Panes)-1; i++ {
			currentY += sp.Sizes[i]
			// Divider is at currentY, allow 2-line click zone
			if y >= currentY-1 && y <= currentY+1 {
				sp.ResizeIndex = i
				sp.Resizing = true
				return
			}
		}
	}
	sp.Resizing = false
}

// View renders the split pane layout
func (sp *SplitPane) View() string {
	if len(sp.Panes) == 0 {
		return ""
	}

	var paneViews []string

	// Render each pane
	for _, pane := range sp.Panes {
		var content string

		if pane.Render != nil {
			// Use custom render function
			content = pane.Render(pane.Width, pane.Height, pane.Focused)
		} else {
			// Default rendering
			content = pane.Content
		}

		// Apply focus styling
		borderColor := sp.DividerColor
		if pane.Focused {
			borderColor = sp.FocusColor
		}

		// Create pane with border
		paneStyle := lipgloss.NewStyle().
			Width(pane.Width).
			Height(pane.Height).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor)

		// Add title if present
		if pane.Title != "" {
			titleStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(borderColor)
			content = titleStyle.Render(pane.Title) + "\n" + content
		}

		paneViews = append(paneViews, paneStyle.Render(content))
	}

	// Join panes with dividers
	if sp.Orientation == Horizontal {
		// Horizontal split - join left to right
		return lipgloss.JoinHorizontal(lipgloss.Top, paneViews...)
	} else {
		// Vertical split - join top to bottom
		return lipgloss.JoinVertical(lipgloss.Left, paneViews...)
	}
}

// CreateTwoPane creates a simple two-pane layout
func CreateTwoPane(orientation PaneOrientation, width, height int, leftContent, rightContent string) *SplitPane {
	sp := NewSplitPane(orientation, width, height)

	pane1 := &Pane{
		ID:      "pane1",
		Content: leftContent,
		Title:   "",
		Focused: true,
	}

	pane2 := &Pane{
		ID:      "pane2",
		Content: rightContent,
		Title:   "",
		Focused: false,
	}

	sp.AddPane(pane1)
	sp.AddPane(pane2)

	return sp
}

// CreateThreePane creates a three-pane layout
func CreateThreePane(orientation PaneOrientation, width, height int) *SplitPane {
	sp := NewSplitPane(orientation, width, height)

	for i := 0; i < 3; i++ {
		pane := &Pane{
			ID:      lipgloss.NewStyle().Render(string(rune('A' + i))),
			Content: "",
			Focused: i == 0,
		}
		sp.AddPane(pane)
	}

	return sp
}
