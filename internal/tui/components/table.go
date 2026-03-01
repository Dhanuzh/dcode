package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TableColumn represents a column in the table
type TableColumn struct {
	Title string
	Width int
	Align lipgloss.Position // lipgloss.Left, lipgloss.Center, lipgloss.Right
}

// TableRow represents a row in the table
type TableRow struct {
	Cells []string
	Data  interface{}
}

// Table represents a data table component
type Table struct {
	Columns     []TableColumn
	Rows        []TableRow
	cursor      int
	offset      int
	Height      int
	Width       int
	ShowHeaders bool
	ShowBorders bool
	Selectable  bool
	Sortable    bool
	sortColumn  int
	sortAsc     bool

	// Callbacks
	OnSelect func(row TableRow) tea.Msg
}

// TableKeyMap defines keybindings for tables
type TableKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Left   key.Binding
	Right  key.Binding
	Enter  key.Binding
	Sort   key.Binding
	Escape key.Binding
}

var DefaultTableKeys = TableKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "scroll left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "scroll right"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Sort: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "sort"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
}

// NewTable creates a new table component
func NewTable(columns []TableColumn, height int) *Table {
	return &Table{
		Columns:     columns,
		Rows:        []TableRow{},
		cursor:      0,
		offset:      0,
		Height:      height,
		Width:       80,
		ShowHeaders: true,
		ShowBorders: true,
		Selectable:  true,
		Sortable:    true,
		sortColumn:  -1,
		sortAsc:     true,
	}
}

// SetRows updates the table rows
func (t *Table) SetRows(rows []TableRow) {
	t.Rows = rows
	if t.cursor >= len(rows) {
		t.cursor = len(rows) - 1
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
	t.updateOffset()
}

// AddRow adds a row to the table
func (t *Table) AddRow(row TableRow) {
	t.Rows = append(t.Rows, row)
}

// RemoveRow removes a row at index
func (t *Table) RemoveRow(index int) {
	if index < 0 || index >= len(t.Rows) {
		return
	}
	t.Rows = append(t.Rows[:index], t.Rows[index+1:]...)
	if t.cursor >= len(t.Rows) {
		t.cursor = len(t.Rows) - 1
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
	t.updateOffset()
}

// GetCursor returns the current cursor position
func (t *Table) GetCursor() int {
	return t.cursor
}

// SetCursor sets the cursor position
func (t *Table) SetCursor(pos int) {
	if pos < 0 {
		pos = 0
	}
	if pos >= len(t.Rows) {
		pos = len(t.Rows) - 1
	}
	t.cursor = pos
	t.updateOffset()
}

// GetSelectedRow returns the currently selected row
func (t *Table) GetSelectedRow() *TableRow {
	if t.cursor < 0 || t.cursor >= len(t.Rows) {
		return nil
	}
	return &t.Rows[t.cursor]
}

// updateOffset adjusts the scroll offset to keep cursor in view
func (t *Table) updateOffset() {
	if t.cursor < t.offset {
		t.offset = t.cursor
	}
	headerOffset := 0
	if t.ShowHeaders {
		headerOffset = 1
	}
	if t.cursor >= t.offset+t.Height-headerOffset {
		t.offset = t.cursor - t.Height + headerOffset + 1
	}
	if t.offset < 0 {
		t.offset = 0
	}
}

// SortByColumn sorts the table by a column index
func (t *Table) SortByColumn(column int) {
	if !t.Sortable || column < 0 || column >= len(t.Columns) {
		return
	}

	// Toggle sort direction if same column
	if t.sortColumn == column {
		t.sortAsc = !t.sortAsc
	} else {
		t.sortColumn = column
		t.sortAsc = true
	}

	// Simple bubble sort (good enough for TUI tables)
	n := len(t.Rows)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			compare := strings.Compare(
				strings.ToLower(t.Rows[j].Cells[column]),
				strings.ToLower(t.Rows[j+1].Cells[column]),
			)

			shouldSwap := false
			if t.sortAsc {
				shouldSwap = compare > 0
			} else {
				shouldSwap = compare < 0
			}

			if shouldSwap {
				t.Rows[j], t.Rows[j+1] = t.Rows[j+1], t.Rows[j]
			}
		}
	}
}

// Update handles table input
func (t *Table) Update(msg tea.Msg) (*Table, tea.Cmd) {
	if !t.Selectable {
		return t, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultTableKeys.Up):
			if t.cursor > 0 {
				t.cursor--
				t.updateOffset()
			}
		case key.Matches(msg, DefaultTableKeys.Down):
			if t.cursor < len(t.Rows)-1 {
				t.cursor++
				t.updateOffset()
			}
		case key.Matches(msg, DefaultTableKeys.Enter):
			if t.cursor >= 0 && t.cursor < len(t.Rows) {
				row := t.Rows[t.cursor]
				if t.OnSelect != nil {
					return t, func() tea.Msg {
						return t.OnSelect(row)
					}
				}
			}
		}
	}

	return t, nil
}

// View renders the table
func (t *Table) View() string {
	var rows []string

	// Calculate column widths if not set
	for i := range t.Columns {
		if t.Columns[i].Width == 0 {
			t.Columns[i].Width = t.calculateColumnWidth(i)
		}
	}

	// Render header
	if t.ShowHeaders {
		rows = append(rows, t.renderHeader())
		if t.ShowBorders {
			rows = append(rows, t.renderSeparator())
		}
	}

	// Render rows
	if len(t.Rows) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086")).
			Italic(true).
			Padding(1, 2).
			Align(lipgloss.Center).
			Width(t.Width)
		rows = append(rows, emptyStyle.Render("No data"))
	} else {
		visibleEnd := t.offset + t.Height
		if t.ShowHeaders {
			visibleEnd = t.offset + t.Height - 1
		}
		if visibleEnd > len(t.Rows) {
			visibleEnd = len(t.Rows)
		}

		for i := t.offset; i < visibleEnd; i++ {
			rows = append(rows, t.renderRow(t.Rows[i], i == t.cursor))
		}
	}

	// Add scroll indicators
	if t.offset > 0 {
		scrollUpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086")).
			Align(lipgloss.Center).
			Width(t.Width)
		rows = append([]string{scrollUpStyle.Render("↑ More rows above")}, rows...)
	}
	visibleEnd := t.offset + t.Height
	if t.ShowHeaders {
		visibleEnd--
	}
	if visibleEnd < len(t.Rows) {
		scrollDownStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086")).
			Align(lipgloss.Center).
			Width(t.Width)
		rows = append(rows, scrollDownStyle.Render("↓ More rows below"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// renderHeader renders the table header
func (t *Table) renderHeader() string {
	var cells []string

	for i, col := range t.Columns {
		cellStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#CBA6F7")).
			Width(col.Width).
			Align(col.Align).
			Padding(0, 1)

		title := col.Title

		// Add sort indicator
		if t.sortColumn == i {
			if t.sortAsc {
				title += " ↑"
			} else {
				title += " ↓"
			}
		}

		cells = append(cells, cellStyle.Render(title))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, cells...)
}

// renderSeparator renders a separator line
func (t *Table) renderSeparator() string {
	var parts []string

	for _, col := range t.Columns {
		sep := strings.Repeat("─", col.Width+2)
		parts = append(parts, sep)
	}

	separatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6C7086"))

	return separatorStyle.Render(strings.Join(parts, ""))
}

// renderRow renders a table row
func (t *Table) renderRow(row TableRow, isCursor bool) string {
	var cells []string

	for i, col := range t.Columns {
		cellStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CDD6F4")).
			Width(col.Width).
			Align(col.Align).
			Padding(0, 1)

		// Highlight cursor row
		if isCursor && t.Selectable {
			cellStyle = cellStyle.
				Background(lipgloss.Color("#313244")).
				Bold(true)
		}

		cellValue := ""
		if i < len(row.Cells) {
			cellValue = row.Cells[i]
		}

		// Truncate if too long
		if len(cellValue) > col.Width {
			cellValue = cellValue[:col.Width-3] + "..."
		}

		cells = append(cells, cellStyle.Render(cellValue))
	}

	rowView := lipgloss.JoinHorizontal(lipgloss.Top, cells...)

	// Add left border for cursor
	if isCursor && t.Selectable {
		rowStyle := lipgloss.NewStyle().
			Border(lipgloss.Border{Left: "▐"}).
			BorderForeground(lipgloss.Color("#89B4FA"))
		return rowStyle.Render(rowView)
	}

	return rowView
}

// calculateColumnWidth calculates optimal width for a column
func (t *Table) calculateColumnWidth(columnIndex int) int {
	if columnIndex < 0 || columnIndex >= len(t.Columns) {
		return 10
	}

	// Start with title width
	maxWidth := len(t.Columns[columnIndex].Title) + 4

	// Check all rows
	for _, row := range t.Rows {
		if columnIndex < len(row.Cells) {
			cellWidth := len(row.Cells[columnIndex]) + 2
			if cellWidth > maxWidth {
				maxWidth = cellWidth
			}
		}
	}

	// Cap at reasonable max
	if maxWidth > 40 {
		maxWidth = 40
	}

	return maxWidth
}

// Filter filters rows by predicate
func (t *Table) Filter(predicate func(row TableRow) bool) {
	var filtered []TableRow
	for _, row := range t.Rows {
		if predicate(row) {
			filtered = append(filtered, row)
		}
	}
	t.SetRows(filtered)
}
