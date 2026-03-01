package components

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TreeNode represents a node in the tree
type TreeNode struct {
	Name     string
	Path     string
	IsDir    bool
	IsOpen   bool
	Children []*TreeNode
	Parent   *TreeNode
	Level    int
	Size     int64
	ModTime  int64
}

// Tree represents a file tree component
type Tree struct {
	Root         *TreeNode
	cursor       int
	offset       int
	Height       int
	Width        int
	ShowHidden   bool
	ShowIcons    bool
	flatNodes    []*TreeNode // Flattened view for rendering
	currentIndex int

	// Callbacks
	OnSelect func(node *TreeNode) tea.Msg
}

// TreeKeyMap defines keybindings for tree
type TreeKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Left   key.Binding
	Right  key.Binding
	Escape key.Binding
}

var DefaultTreeKeys = TreeKeyMap{
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
		key.WithHelp("enter", "select/toggle"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "collapse"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "expand"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
}

// NewTree creates a new tree component
func NewTree(rootPath string, height int) (*Tree, error) {
	root, err := buildTree(rootPath, 0, nil, false)
	if err != nil {
		return nil, err
	}

	tree := &Tree{
		Root:       root,
		cursor:     0,
		offset:     0,
		Height:     height,
		Width:      60,
		ShowHidden: false,
		ShowIcons:  true,
	}

	tree.rebuildFlatView()
	return tree, nil
}

// buildTree recursively builds a tree from a directory
func buildTree(path string, level int, parent *TreeNode, showHidden bool) (*TreeNode, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	node := &TreeNode{
		Name:    filepath.Base(path),
		Path:    path,
		IsDir:   info.IsDir(),
		IsOpen:  false,
		Level:   level,
		Parent:  parent,
		Size:    info.Size(),
		ModTime: info.ModTime().Unix(),
	}

	// Load children for directories
	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return node, nil // Return node even if we can't read dir
		}

		for _, entry := range entries {
			// Skip hidden files unless ShowHidden is true
			if !showHidden && strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			childPath := filepath.Join(path, entry.Name())
			childInfo, err := entry.Info()
			if err != nil {
				continue
			}

			child := &TreeNode{
				Name:    entry.Name(),
				Path:    childPath,
				IsDir:   entry.IsDir(),
				IsOpen:  false,
				Level:   level + 1,
				Parent:  node,
				Size:    childInfo.Size(),
				ModTime: childInfo.ModTime().Unix(),
			}

			node.Children = append(node.Children, child)
		}

		// Sort children: directories first, then alphabetically
		sort.Slice(node.Children, func(i, j int) bool {
			if node.Children[i].IsDir != node.Children[j].IsDir {
				return node.Children[i].IsDir
			}
			return strings.ToLower(node.Children[i].Name) < strings.ToLower(node.Children[j].Name)
		})
	}

	return node, nil
}

// rebuildFlatView rebuilds the flattened view of visible nodes
func (t *Tree) rebuildFlatView() {
	t.flatNodes = make([]*TreeNode, 0)
	t.flattenNode(t.Root)
}

// flattenNode recursively flattens nodes for rendering
func (t *Tree) flattenNode(node *TreeNode) {
	if node == nil {
		return
	}

	t.flatNodes = append(t.flatNodes, node)

	if node.IsDir && node.IsOpen {
		for _, child := range node.Children {
			t.flattenNode(child)
		}
	}
}

// GetCurrentNode returns the currently selected node
func (t *Tree) GetCurrentNode() *TreeNode {
	if t.cursor < 0 || t.cursor >= len(t.flatNodes) {
		return nil
	}
	return t.flatNodes[t.cursor]
}

// SetCursor sets the cursor position
func (t *Tree) SetCursor(pos int) {
	if pos < 0 {
		pos = 0
	}
	if pos >= len(t.flatNodes) {
		pos = len(t.flatNodes) - 1
	}
	t.cursor = pos
	t.updateOffset()
}

// updateOffset adjusts the scroll offset to keep cursor in view
func (t *Tree) updateOffset() {
	if t.cursor < t.offset {
		t.offset = t.cursor
	}
	if t.cursor >= t.offset+t.Height {
		t.offset = t.cursor - t.Height + 1
	}
	if t.offset < 0 {
		t.offset = 0
	}
}

// ToggleNode toggles a directory node open/closed
func (t *Tree) ToggleNode(node *TreeNode) {
	if !node.IsDir {
		return
	}

	node.IsOpen = !node.IsOpen

	// Load children on first open if not loaded
	if node.IsOpen && len(node.Children) == 0 {
		entries, err := os.ReadDir(node.Path)
		if err == nil {
			for _, entry := range entries {
				if !t.ShowHidden && strings.HasPrefix(entry.Name(), ".") {
					continue
				}

				childPath := filepath.Join(node.Path, entry.Name())
				childInfo, err := entry.Info()
				if err != nil {
					continue
				}

				child := &TreeNode{
					Name:    entry.Name(),
					Path:    childPath,
					IsDir:   entry.IsDir(),
					IsOpen:  false,
					Level:   node.Level + 1,
					Parent:  node,
					Size:    childInfo.Size(),
					ModTime: childInfo.ModTime().Unix(),
				}

				node.Children = append(node.Children, child)
			}

			// Sort children
			sort.Slice(node.Children, func(i, j int) bool {
				if node.Children[i].IsDir != node.Children[j].IsDir {
					return node.Children[i].IsDir
				}
				return strings.ToLower(node.Children[i].Name) < strings.ToLower(node.Children[j].Name)
			})
		}
	}

	t.rebuildFlatView()
}

// Update handles tree input
func (t *Tree) Update(msg tea.Msg) (*Tree, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultTreeKeys.Up):
			if t.cursor > 0 {
				t.cursor--
				t.updateOffset()
			}
		case key.Matches(msg, DefaultTreeKeys.Down):
			if t.cursor < len(t.flatNodes)-1 {
				t.cursor++
				t.updateOffset()
			}
		case key.Matches(msg, DefaultTreeKeys.Enter):
			node := t.GetCurrentNode()
			if node != nil {
				if node.IsDir {
					t.ToggleNode(node)
				} else if t.OnSelect != nil {
					return t, func() tea.Msg {
						return t.OnSelect(node)
					}
				}
			}
		case key.Matches(msg, DefaultTreeKeys.Right):
			node := t.GetCurrentNode()
			if node != nil && node.IsDir && !node.IsOpen {
				t.ToggleNode(node)
			}
		case key.Matches(msg, DefaultTreeKeys.Left):
			node := t.GetCurrentNode()
			if node != nil {
				if node.IsDir && node.IsOpen {
					t.ToggleNode(node)
				} else if node.Parent != nil {
					// Move to parent
					for i, n := range t.flatNodes {
						if n == node.Parent {
							t.cursor = i
							t.updateOffset()
							break
						}
					}
				}
			}
		}
	}

	return t, nil
}

// View renders the tree
func (t *Tree) View() string {
	if len(t.flatNodes) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086")).
			Italic(true).
			Padding(1, 2)
		return emptyStyle.Render("Empty directory")
	}

	var items []string

	// Render visible nodes
	visibleEnd := t.offset + t.Height
	if visibleEnd > len(t.flatNodes) {
		visibleEnd = len(t.flatNodes)
	}

	for i := t.offset; i < visibleEnd; i++ {
		node := t.flatNodes[i]
		items = append(items, t.renderNode(node, i == t.cursor))
	}

	// Add scroll indicators
	if t.offset > 0 {
		scrollUpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086")).
			Align(lipgloss.Center).
			Width(t.Width)
		items = append([]string{scrollUpStyle.Render("↑")}, items...)
	}
	if visibleEnd < len(t.flatNodes) {
		scrollDownStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086")).
			Align(lipgloss.Center).
			Width(t.Width)
		items = append(items, scrollDownStyle.Render("↓"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, items...)
}

// renderNode renders a single tree node
func (t *Tree) renderNode(node *TreeNode, isCursor bool) string {
	// Indentation
	indent := strings.Repeat("  ", node.Level)

	// Tree branch characters
	var branch string
	if node.IsDir {
		if node.IsOpen {
			branch = "▼ "
		} else {
			branch = "▶ "
		}
	} else {
		branch = "  "
	}

	// Icon
	icon := ""
	if t.ShowIcons {
		if node.IsDir {
			if node.IsOpen {
				icon = "📂"
			} else {
				icon = "📁"
			}
		} else {
			icon = getFileIcon(node.Name)
		}
	}

	// Name style
	nameStyle := lipgloss.NewStyle()
	if node.IsDir {
		nameStyle = nameStyle.Foreground(lipgloss.Color("#89B4FA")).Bold(true)
	} else {
		nameStyle = nameStyle.Foreground(lipgloss.Color("#CDD6F4"))
	}

	content := indent + branch + icon + " " + nameStyle.Render(node.Name)

	// Apply cursor style
	itemStyle := lipgloss.NewStyle().
		Width(t.Width-2).
		Padding(0, 1)

	if isCursor {
		itemStyle = itemStyle.
			Background(lipgloss.Color("#313244")).
			Border(lipgloss.Border{Left: "▐"}).
			BorderForeground(lipgloss.Color("#89B4FA"))
	}

	return itemStyle.Render(content)
}

// getFileIcon returns an icon for a file based on extension
func getFileIcon(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	iconMap := map[string]string{
		".go":         "🔵",
		".js":         "📜",
		".ts":         "📘",
		".jsx":        "⚛️",
		".tsx":        "⚛️",
		".py":         "🐍",
		".rb":         "💎",
		".rs":         "🦀",
		".java":       "☕",
		".c":          "⚙️",
		".cpp":        "⚙️",
		".h":          "⚙️",
		".md":         "📝",
		".txt":        "📄",
		".json":       "📋",
		".yaml":       "📋",
		".yml":        "📋",
		".xml":        "📋",
		".html":       "🌐",
		".css":        "🎨",
		".scss":       "🎨",
		".sql":        "🗄️",
		".sh":         "🔧",
		".bash":       "🔧",
		".zsh":        "🔧",
		".fish":       "🔧",
		".dockerfile": "🐳",
		".gitignore":  "📁",
		".env":        "🔐",
	}

	if icon, ok := iconMap[ext]; ok {
		return icon
	}

	// Default file icon
	return "📄"
}

// Refresh refreshes the tree from the filesystem
func (t *Tree) Refresh() error {
	root, err := buildTree(t.Root.Path, 0, nil, t.ShowHidden)
	if err != nil {
		return err
	}
	t.Root = root
	t.rebuildFlatView()
	return nil
}

// ExpandAll recursively expands all directories
func (t *Tree) ExpandAll() {
	t.expandNode(t.Root)
	t.rebuildFlatView()
}

// expandNode recursively expands a node and its children
func (t *Tree) expandNode(node *TreeNode) {
	if !node.IsDir {
		return
	}

	node.IsOpen = true
	for _, child := range node.Children {
		t.expandNode(child)
	}
}

// CollapseAll collapses all directories
func (t *Tree) CollapseAll() {
	t.collapseNode(t.Root)
	t.rebuildFlatView()
}

// collapseNode recursively collapses a node and its children
func (t *Tree) collapseNode(node *TreeNode) {
	if !node.IsDir {
		return
	}

	node.IsOpen = false
	for _, child := range node.Children {
		t.collapseNode(child)
	}
}
