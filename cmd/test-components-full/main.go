package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yourusername/dcode/internal/tui/components"
)

// model represents the TUI application state
type model struct {
	currentView int
	dialog      *components.Dialog
	list        *components.List
	tree        *components.Tree
	table       *components.Table
	width       int
	height      int
}

func initialModel() model {
	// Create sample list
	listItems := []components.ListItem{
		{Title: "Syntax Highlighting", Description: "30+ languages supported", Icon: "🎨"},
		{Title: "Markdown Rendering", Description: "Beautiful terminal markdown", Icon: "📝"},
		{Title: "Diff Viewer", Description: "Git diff visualization", Icon: "🔍"},
		{Title: "Dialog Component", Description: "Modal dialogs and prompts", Icon: "💬"},
		{Title: "List Component", Description: "Selectable lists", Icon: "📋"},
		{Title: "Tree Component", Description: "File browser", Icon: "🌳"},
		{Title: "Table Component", Description: "Data tables", Icon: "📊"},
	}
	list := components.NewList(listItems, 10)
	list.Title = "DCode TUI Components"

	// Create sample tree
	cwd, _ := os.Getwd()
	tree, _ := components.NewTree(cwd, 15)

	// Create sample table
	columns := []components.TableColumn{
		{Title: "Component", Width: 20, Align: lipgloss.Left},
		{Title: "Lines", Width: 10, Align: lipgloss.Right},
		{Title: "Status", Width: 15, Align: lipgloss.Center},
	}
	table := components.NewTable(columns, 10)
	table.SetRows([]components.TableRow{
		{Cells: []string{"Syntax", "238", "✅ Complete"}},
		{Cells: []string{"Markdown", "213", "✅ Complete"}},
		{Cells: []string{"Diff", "220", "✅ Complete"}},
		{Cells: []string{"Dialog", "290", "✅ Complete"}},
		{Cells: []string{"List", "380", "✅ Complete"}},
		{Cells: []string{"Tree", "520", "✅ Complete"}},
		{Cells: []string{"Table", "410", "✅ Complete"}},
	})

	// Create sample dialog
	dialog := components.NewConfirmDialog(
		"Test Dialog",
		"This is a confirmation dialog. Press Enter to confirm, Esc to cancel.",
		func() tea.Msg { return "confirmed" },
		func() tea.Msg { return "cancelled" },
	)

	return model{
		currentView: 0,
		dialog:      dialog,
		list:        list,
		tree:        tree,
		table:       table,
		width:       80,
		height:      24,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Handle dialog first if visible
		if m.dialog.IsVisible() {
			m.dialog, cmd = m.dialog.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "tab":
			m.currentView = (m.currentView + 1) % 4

		case "d":
			m.dialog.Show()

		default:
			// Delegate to current view
			switch m.currentView {
			case 0:
				m.list, cmd = m.list.Update(msg)
			case 1:
				m.tree, cmd = m.tree.Update(msg)
			case 2:
				m.table, cmd = m.table.Update(msg)
			}
		}
	}

	return m, cmd
}

func (m model) View() string {
	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#CBA6F7")).
		Padding(1, 2)

	title := titleStyle.Render("DCode TUI Components Test - Press Tab to switch views, 'd' for dialog, 'q' to quit")

	// View labels
	viewLabels := []string{"List View", "Tree View", "Table View", "Syntax/Markdown/Diff View"}
	viewLabelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#89B4FA")).
		Padding(0, 2)

	currentViewLabel := viewLabelStyle.Render(fmt.Sprintf("Current: %s", viewLabels[m.currentView]))

	// Render current view
	var content string
	switch m.currentView {
	case 0:
		content = m.list.View()
	case 1:
		content = m.tree.View()
	case 2:
		content = m.table.View()
	case 3:
		content = renderStaticExamples()
	}

	// Container
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6C7086")).
		Padding(1, 2).
		Width(m.width - 4)

	contentView := containerStyle.Render(content)

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6C7086")).
		Padding(1, 2)

	var helpText string
	switch m.currentView {
	case 0:
		helpText = "↑/k: up | ↓/j: down | enter: select | tab: next view | d: dialog | q: quit"
	case 1:
		helpText = "↑/k: up | ↓/j: down | →/l: expand | ←/h: collapse | tab: next view | d: dialog | q: quit"
	case 2:
		helpText = "↑/k: up | ↓/j: down | enter: select | tab: next view | d: dialog | q: quit"
	case 3:
		helpText = "tab: next view | d: dialog | q: quit"
	}

	help := helpStyle.Render(helpText)

	// Compose final view
	mainView := lipgloss.JoinVertical(lipgloss.Left, title, currentViewLabel, contentView, help)

	// Overlay dialog if visible
	if m.dialog.IsVisible() {
		dialogOverlay := m.dialog.RenderOverlay(m.width, m.height)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, mainView+"\n"+dialogOverlay)
	}

	return mainView
}

func renderStaticExamples() string {
	highlighter := components.NewSyntaxHighlighter("monokai")

	// Go code example
	goCode := `package main

import "fmt"

func main() {
	message := "Hello, World!"
	fmt.Println(message)
}
`

	goExample := highlighter.HighlightCodeBlock(goCode, "go")

	// Markdown example
	renderer, _ := components.NewMarkdownRenderer(70, "dark")
	h1 := renderer.RenderHeading("Welcome to DCode", 1)
	h2 := renderer.RenderHeading("Features", 2)

	listItems := []string{
		"Syntax highlighting for 30+ languages",
		"Beautiful markdown rendering",
		"Git diff visualization",
		"Interactive TUI components",
	}
	list := renderer.RenderList(listItems)

	quote := renderer.RenderQuote("The best way to predict the future is to implement it.")

	// Diff example
	viewer := components.NewDiffViewer(70, "monokai")
	gitDiff := `diff --git a/main.go b/main.go
index 1234567..abcdefg 100644
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 package main

-func main() {}
+func main() {
+	fmt.Println("Hello, World!")
+}
`

	diffExample := viewer.RenderSimple(gitDiff)

	// Combine all examples
	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6C7086")).
		Render(string(lipgloss.NewStyle().Width(70).Render("─────────────────────────────────")))

	return lipgloss.JoinVertical(lipgloss.Left,
		"Syntax Highlighting Example:",
		goExample,
		separator,
		"Markdown Rendering Example:",
		h1,
		h2,
		list,
		quote,
		separator,
		"Diff Viewer Example:",
		diffExample,
	)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
