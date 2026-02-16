# TUI Components Library

Production-ready Bubble Tea components for building advanced terminal user interfaces.

## 📦 Components

| Component | Purpose | Lines | Key Features |
|-----------|---------|-------|--------------|
| **Syntax** | Code highlighting | 243 | 30+ languages, 7 themes |
| **Markdown** | Markdown rendering | 211 | Glamour integration, code blocks |
| **Diff** | Git diff visualization | 220 | Color-coded, side-by-side |
| **Dialog** | Modal dialogs | 290 | 5 types, input support |
| **List** | Selectable lists | 380 | Multi-select, icons |
| **Tree** | File browser | 520 | Lazy loading, expand/collapse |
| **Table** | Data tables | 410 | Sortable, auto-sizing |

## 🚀 Quick Start

```go
import "github.com/yourusername/dcode/internal/tui/components"

// Syntax highlighting
highlighter := components.NewSyntaxHighlighter("monokai")
code := highlighter.HighlightCodeBlock(goCode, "go")

// Markdown rendering
renderer, _ := components.NewMarkdownRenderer(80, "dark")
markdown := renderer.RenderWithHighlighting(text)

// Diff viewing
viewer := components.NewDiffViewer(100, "monokai")
diff := viewer.RenderSimple(gitDiff)

// Dialog
dialog := components.NewConfirmDialog("Delete?", "Are you sure?", onYes, onNo)
dialog.Show()

// List
items := []components.ListItem{
    {Title: "Item 1", Icon: "📋"},
}
list := components.NewList(items, 10)
list, cmd = list.Update(msg)
view := list.View()

// Tree
tree, _ := components.NewTree("/path/to/dir", 15)
tree, cmd = tree.Update(msg)
view := tree.View()

// Table
columns := []components.TableColumn{
    {Title: "Name", Width: 20, Align: lipgloss.Left},
}
table := components.NewTable(columns, 10)
table.SetRows(rows)
table, cmd = table.Update(msg)
view := table.View()
```

## 🧪 Testing

Run the interactive test program:

```bash
go build -o test-components-full ./cmd/test-components-full
./test-components-full
```

## 📚 Documentation

See [PHASE2_COMPONENTS.md](../../../PHASE2_COMPONENTS.md) for comprehensive documentation, usage examples, and integration guide.

## 🎨 Theming

All components support theming:
- `"monokai"` / `"dark"` - Dark theme (default)
- `"dracula"` - Dracula
- `"nord"` - Nord
- `"solarized-dark"` / `"solarized-light"` - Solarized
- `"github"` / `"light"` - Light theme

## 🔑 Key Bindings

### Dialog
- `←/→` or `h/l` - Navigate buttons
- `Enter` - Confirm
- `Esc` - Cancel

### List
- `↑/↓` or `k/j` - Navigate
- `Enter` - Select
- `Space` - Toggle (multi-select)

### Tree
- `↑/↓` or `k/j` - Navigate
- `→/l` - Expand directory
- `←/h` - Collapse directory
- `Enter` - Select file/toggle directory

### Table
- `↑/↓` or `k/j` - Navigate rows
- `←/→` or `h/l` - Scroll columns
- `Enter` - Select row
- `s` - Sort by column

## 💡 Patterns

### Component Update Loop
```go
component, cmd := component.Update(msg)
view := component.View()
```

### Callbacks
```go
list.OnSelect = func(item ListItem) tea.Msg {
    return MyMsg{item}
}
```

### Composition
```go
// Combine components
leftPane := list.View()
rightPane := tree.View()
splitView := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

// Overlay dialog
if dialog.IsVisible() {
    view = dialog.RenderOverlay(width, height)
}
```

## ✅ Status

- [x] All components compile
- [x] Interactive test program
- [x] Keyboard navigation
- [x] Theme support
- [x] Documentation
- [x] Production-ready

**Ready for integration into main TUI!**
