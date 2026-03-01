package tui

// autocomplete.go — slash-command and @-file autocomplete popup above the input.
//
// Trigger rules:
//   "/" at position 0 of the input → show slash-command list
//   "@" anywhere (preceded by whitespace or start) → show file picker
//
// Navigation:
//   Up / Ctrl+P   → previous item
//   Down / Ctrl+N → next item
//   Enter / Tab   → select item
//   Esc           → dismiss

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Dhanuzh/dcode/internal/session"
)

// AutocompleteMode identifies what the popup is completing.
type AutocompleteMode int

const (
	AutocompleteOff   AutocompleteMode = iota
	AutocompleteSlash                  // "/" commands
	AutocompleteAt                     // "@" file picker
)

// AutocompleteItem is a single choice in the popup.
type AutocompleteItem struct {
	Display     string
	Value       string // "@file:<path>" or "@dir:<path>" or slash value
	Description string
	IsDir       bool
}

// AutocompleteState holds all state for the popup.
type AutocompleteState struct {
	Mode     AutocompleteMode
	Items    []AutocompleteItem
	Selected int
	Filter   string // text typed after the trigger
	// For @ file picker: the directory currently being listed
	AtDir string
}

// ── Slash autocomplete ────────────────────────────────────────────────────────

// buildSlashItems returns all slash commands as autocomplete items.
func buildSlashItems() []AutocompleteItem {
	cmds := allCommands()
	items := make([]AutocompleteItem, 0, len(cmds))
	for _, c := range cmds {
		if c.Slash == "" {
			continue
		}
		desc := c.Category
		if c.Keybind != "" {
			desc += "  [" + c.Keybind + "]"
		}
		items = append(items, AutocompleteItem{
			Display:     c.Slash,
			Value:       c.Slash + " ",
			Description: desc,
		})
	}
	return items
}

// openSlashAutocomplete opens the slash-command popup including custom commands.
func (m *Model) openSlashAutocomplete() {
	items := buildSlashItems()

	// Append custom commands from .dcode/commands/
	if m.Config != nil && m.Config.Commands != nil {
		for name, cmd := range m.Config.Commands {
			slash := "/" + name
			desc := cmd.Description
			if desc == "" {
				desc = "Custom command"
			}
			items = append(items, AutocompleteItem{
				Display:     slash,
				Value:       slash + " ",
				Description: desc,
			})
		}
	}

	m.autocomplete = AutocompleteState{
		Mode:   AutocompleteSlash,
		Items:  items,
		Filter: "",
	}
}

// ── @ file picker ─────────────────────────────────────────────────────────────

// imageExtensions is the set of file extensions treated as images.
var imageExtensions = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
}

// fileIcon returns a short icon + language hint for a file extension.
func fileIcon(name string) (icon, lang string) {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	// Images
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return "󰋩 ", "image"
	// Go
	case ".go":
		return "󰟓 ", "go"
	// Python
	case ".py":
		return " ", "python"
	// JS/TS
	case ".js":
		return " ", "javascript"
	case ".ts":
		return " ", "typescript"
	case ".jsx":
		return " ", "jsx"
	case ".tsx":
		return " ", "tsx"
	// Rust
	case ".rs":
		return " ", "rust"
	// C/C++
	case ".c", ".h":
		return " ", "c"
	case ".cpp", ".cc", ".cxx", ".hpp":
		return " ", "cpp"
	// Java
	case ".java":
		return " ", "java"
	// Shell
	case ".sh", ".bash", ".zsh":
		return " ", "bash"
	// Markdown
	case ".md", ".mdx":
		return " ", "markdown"
	// JSON/YAML/TOML
	case ".json":
		return " ", "json"
	case ".yaml", ".yml":
		return " ", "yaml"
	case ".toml":
		return " ", "toml"
	// HTML/CSS
	case ".html", ".htm":
		return " ", "html"
	case ".css", ".scss", ".sass":
		return " ", "css"
	// SQL
	case ".sql":
		return " ", "sql"
	// Dockerfile
	case ".dockerfile":
		return "󰡨 ", "dockerfile"
	// Config / text
	case ".env", ".ini", ".conf", ".cfg":
		return " ", "ini"
	case ".txt":
		return "󰈙 ", "text"
	// XML
	case ".xml":
		return "󰗀 ", "xml"
	default:
		return "󰈔 ", ""
	}
}

// buildAtFileItems scans dir for files and subdirectories, returning them as
// autocomplete items. Directories come first, then files, both sorted A-Z.
// Hidden files (dot-prefixed) are skipped.
// A ".." entry is prepended when dir is not ".".
func buildAtFileItems(dir string) []AutocompleteItem {
	if dir == "" {
		dir = "."
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var dirs, files []AutocompleteItem

	// Parent directory entry
	if dir != "." {
		dirs = append(dirs, AutocompleteItem{
			Display:     "󰁍  ..",
			Value:       "@dir:" + filepath.Dir(dir),
			Description: "parent directory",
			IsDir:       true,
		})
	}

	for _, e := range entries {
		name := e.Name()
		// Skip hidden files
		if strings.HasPrefix(name, ".") {
			continue
		}

		fullPath := filepath.Join(dir, name)
		if dir == "." {
			fullPath = name
		}

		if e.IsDir() {
			dirs = append(dirs, AutocompleteItem{
				Display:     "󰉋  " + name + "/",
				Value:       "@dir:" + fullPath,
				Description: "directory",
				IsDir:       true,
			})
		} else {
			icon, lang := fileIcon(name)
			desc := lang
			if desc == "" {
				desc = "file"
			}
			// Get file size for description
			if info, err := e.Info(); err == nil {
				size := info.Size()
				switch {
				case size < 1024:
					desc += fmt.Sprintf("  %dB", size)
				case size < 1024*1024:
					desc += fmt.Sprintf("  %.1fKB", float64(size)/1024)
				default:
					desc += fmt.Sprintf("  %.1fMB", float64(size)/(1024*1024))
				}
			}
			files = append(files, AutocompleteItem{
				Display:     icon + name,
				Value:       "@file:" + fullPath,
				Description: desc,
				IsDir:       false,
			})
		}
	}

	// Sort dirs and files separately A-Z
	sort.Slice(dirs[func() int {
		if dir != "." {
			return 1
		}
		return 0
	}():], func(i, j int) bool {
		off := func() int {
			if dir != "." {
				return 1
			}
			return 0
		}()
		return dirs[i+off].Display < dirs[j+off].Display
	})
	sort.Slice(files, func(i, j int) bool { return files[i].Display < files[j].Display })

	return append(dirs, files...)
}

// ── Shared helpers ─────────────────────────────────────────────────────────────

// filteredItems returns items filtered by the current filter string.
func (ac *AutocompleteState) filteredItems() []AutocompleteItem {
	if ac.Filter == "" {
		return ac.Items
	}
	f := strings.ToLower(ac.Filter)
	var out []AutocompleteItem
	for _, item := range ac.Items {
		// Strip icon prefix for matching
		dispPlain := strings.TrimSpace(item.Display)
		if strings.Contains(strings.ToLower(dispPlain), f) ||
			strings.Contains(strings.ToLower(item.Description), f) {
			out = append(out, item)
		}
	}
	return out
}

// closeAutocomplete dismisses the popup.
func (m *Model) closeAutocomplete() {
	m.autocomplete = AutocompleteState{}
}

// autocompleteVisible reports whether the popup is shown.
func (m *Model) autocompleteVisible() bool {
	return m.autocomplete.Mode != AutocompleteOff
}

// ── Render ────────────────────────────────────────────────────────────────────

// renderAutocomplete renders the popup as a string to be overlaid just above the input.
func (m *Model) renderAutocomplete() string {
	if !m.autocompleteVisible() {
		return ""
	}

	t := m.currentTheme
	dim := lipgloss.NewStyle().Foreground(t.TextMuted)
	normalSt := lipgloss.NewStyle().Foreground(t.Text)
	selSt := lipgloss.NewStyle().
		Foreground(t.Background).
		Background(t.Primary).
		Bold(true)
	descSt := lipgloss.NewStyle().Foreground(t.TextMuted)
	dirSt := lipgloss.NewStyle().Foreground(lipgloss.Color("#89B4FA")) // blue for dirs

	items := m.autocomplete.filteredItems()
	if len(items) == 0 {
		if m.autocomplete.Mode == AutocompleteAt {
			items = []AutocompleteItem{{Display: "No files found", Description: "try a different filter"}}
		} else {
			items = []AutocompleteItem{{Display: "No matching commands", Description: ""}}
		}
	}

	// Clamp selection
	if m.autocomplete.Selected >= len(items) {
		m.autocomplete.Selected = len(items) - 1
	}
	if m.autocomplete.Selected < 0 {
		m.autocomplete.Selected = 0
	}

	// Show at most 12 items with scrolling window
	maxShow := 12
	start := 0
	if m.autocomplete.Selected >= maxShow {
		start = m.autocomplete.Selected - maxShow + 1
	}
	if start+maxShow > len(items) {
		start = len(items) - maxShow
		if start < 0 {
			start = 0
		}
	}
	visible := items[start:]
	if len(visible) > maxShow {
		visible = visible[:maxShow]
	}

	// Header for @ mode showing current directory
	var header string
	if m.autocomplete.Mode == AutocompleteAt {
		atDir := m.autocomplete.AtDir
		if atDir == "" || atDir == "." {
			atDir = "./"
		}
		header = dim.Render("  @ " + atDir)
		if m.autocomplete.Filter != "" {
			header += dim.Render("  filter: ") + normalSt.Render(m.autocomplete.Filter)
		}
		header += "\n"
	}

	// Find max display width for alignment
	maxW := 0
	for _, it := range visible {
		w := lipgloss.Width(it.Display)
		if w > maxW {
			maxW = w
		}
	}
	if maxW > 40 {
		maxW = 40
	}

	var rows []string
	for i, it := range visible {
		globalIdx := start + i
		// Pad display to align descriptions
		displayW := lipgloss.Width(it.Display)
		padding := strings.Repeat(" ", maxW-displayW)

		var row string
		if globalIdx == m.autocomplete.Selected {
			row = selSt.Render("  "+it.Display+padding+"  ") + " " + descSt.Render(it.Description)
		} else {
			if it.IsDir {
				row = dirSt.Render("  "+it.Display+padding+"  ") + " " + dim.Render(it.Description)
			} else {
				row = normalSt.Render("  "+it.Display+padding+"  ") + " " + dim.Render(it.Description)
			}
		}
		rows = append(rows, row)
	}

	// Scroll indicators
	if start > 0 {
		rows = append([]string{dim.Render("  ↑ " + fmt.Sprintf("%d more", start))}, rows...)
	}
	if start+len(visible) < len(items) {
		rows = append(rows, dim.Render("  ↓ "+fmt.Sprintf("%d more", len(items)-start-len(visible))))
	}

	body := header + strings.Join(rows, "\n")

	boxSt := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Background(t.Background)

	return boxSt.Render(body)
}

// ── Key handling ──────────────────────────────────────────────────────────────

// handleAutocompleteKey handles keys when the autocomplete popup is visible.
// Returns true if the key was consumed.
func (m *Model) handleAutocompleteKey(key string) bool {
	if !m.autocompleteVisible() {
		return false
	}

	items := m.autocomplete.filteredItems()

	switch key {
	case "up", "ctrl+p":
		if m.autocomplete.Selected > 0 {
			m.autocomplete.Selected--
		} else if len(items) > 0 {
			m.autocomplete.Selected = len(items) - 1
		}
		return true

	case "down", "ctrl+n":
		if m.autocomplete.Selected < len(items)-1 {
			m.autocomplete.Selected++
		} else {
			m.autocomplete.Selected = 0
		}
		return true

	case "enter", "tab":
		if len(items) == 0 || m.autocomplete.Selected >= len(items) {
			m.closeAutocomplete()
			return true
		}
		sel := items[m.autocomplete.Selected]
		m.closeAutocomplete()

		switch {
		case strings.HasPrefix(sel.Value, "@dir:"):
			// Navigate into directory — reopen popup in that dir
			newDir := strings.TrimPrefix(sel.Value, "@dir:")
			m.autocomplete = AutocompleteState{
				Mode:   AutocompleteAt,
				Items:  buildAtFileItems(newDir),
				Filter: "",
				AtDir:  newDir,
			}
			// Keep the @<filter> in textarea but reset filter
			v := m.textarea.Value()
			if idx, _ := atTriggerPos(v); idx >= 0 {
				m.textarea.SetValue(v[:idx+1]) // keep the @ char
				m.textarea.CursorEnd()
			}

		case strings.HasPrefix(sel.Value, "@file:"):
			filePath := strings.TrimPrefix(sel.Value, "@file:")
			ext := strings.ToLower(filepath.Ext(filePath))

			// Strip the @<filter> from textarea
			v := m.textarea.Value()
			var prefix string
			if idx, _ := atTriggerPos(v); idx >= 0 {
				prefix = v[:idx]
			}

			if _, isImage := imageExtensions[ext]; isImage {
				// Image: load as base64 attachment
				if att, err := loadImageAttachment(filePath); err == nil {
					m.pendingImages = append(m.pendingImages, *att)
					m.textarea.SetValue(prefix)
					m.textarea.CursorEnd()
					m.setStatus(fmt.Sprintf("Attached image: %s", filepath.Base(filePath)))
				} else {
					m.setStatus("Failed to load image: " + err.Error())
				}
			} else {
				// Text / code file: read content and inject as fenced code block
				content, err := os.ReadFile(filePath)
				if err != nil {
					m.setStatus("Failed to read file: " + err.Error())
					return true
				}
				_, lang := fileIcon(filepath.Base(filePath))
				// Build the inline file reference
				ref := fmt.Sprintf("@%s\n```%s\n%s\n```\n", filePath, lang, strings.TrimRight(string(content), "\n"))
				m.textarea.SetValue(prefix + ref)
				m.textarea.CursorEnd()
				m.setStatus(fmt.Sprintf("Attached file: %s (%d lines)", filepath.Base(filePath), strings.Count(string(content), "\n")+1))
			}

		default:
			// Slash command: insert value into textarea
			m.textarea.SetValue(sel.Value)
			m.textarea.CursorEnd()
		}
		return true

	case "esc":
		m.closeAutocomplete()
		return true

	case "backspace":
		if len(m.autocomplete.Filter) > 0 {
			// Trim last rune from filter
			runes := []rune(m.autocomplete.Filter)
			m.autocomplete.Filter = string(runes[:len(runes)-1])
			m.autocomplete.Selected = 0
			// Also update items if dir changed
			m.autocomplete.Items = buildAtFileItems(m.autocomplete.AtDir)
		} else {
			// Backspace on the trigger char → close
			m.closeAutocomplete()
		}
		return true
	}

	// Single printable character — append to filter
	if len([]rune(key)) == 1 {
		m.autocomplete.Filter += key
		m.autocomplete.Selected = 0
		return true
	}

	return false
}

// ── Trigger detection ─────────────────────────────────────────────────────────

// atTriggerPos returns the byte offset of the last "@" trigger in the input
// that is preceded by whitespace or the start of the string, along with the
// filter text typed after it. Returns -1 if no trigger is active.
func atTriggerPos(v string) (triggerIdx int, filter string) {
	for i := len(v) - 1; i >= 0; i-- {
		ch := v[i]
		if ch == '@' {
			if i == 0 || v[i-1] == ' ' || v[i-1] == '\t' || v[i-1] == '\n' {
				return i, v[i+1:]
			}
			return -1, ""
		}
		if ch == ' ' || ch == '\t' || ch == '\n' {
			return -1, ""
		}
	}
	return -1, ""
}

// maybeOpenAutocomplete checks the current textarea value and opens/updates
// the popup if trigger conditions are met. Called after every keystroke.
func (m *Model) maybeOpenAutocomplete() {
	v := m.textarea.Value()

	// "/" at position 0 → slash commands
	if v == "/" {
		if !m.autocompleteVisible() {
			m.openSlashAutocomplete()
		}
		return
	}
	if strings.HasPrefix(v, "/") && !strings.Contains(v, " ") {
		if m.autocomplete.Mode == AutocompleteSlash {
			m.autocomplete.Filter = v[1:]
			m.autocomplete.Selected = 0
		} else if !m.autocompleteVisible() {
			m.openSlashAutocomplete()
			m.autocomplete.Filter = v[1:]
		}
		return
	}

	// "@" trigger → file picker
	if idx, filter := atTriggerPos(v); idx >= 0 {
		if m.autocomplete.Mode == AutocompleteAt {
			// Already open — just update filter
			m.autocomplete.Filter = filter
			m.autocomplete.Selected = 0
		} else {
			// Open fresh at CWD
			dir := "."
			m.autocomplete = AutocompleteState{
				Mode:   AutocompleteAt,
				Items:  buildAtFileItems(dir),
				Filter: filter,
				AtDir:  dir,
			}
		}
		return
	}

	// Close if we left the trigger zone
	if m.autocomplete.Mode == AutocompleteSlash && !strings.HasPrefix(v, "/") {
		m.closeAutocomplete()
	}
	if m.autocomplete.Mode == AutocompleteAt {
		if _, _ = atTriggerPos(v); !strings.Contains(v, "@") {
			m.closeAutocomplete()
		}
	}
}

// ── File loaders ──────────────────────────────────────────────────────────────

// loadImageAttachment reads an image file from disk, base64-encodes it, and
// returns a session.ImageAttachment ready to be staged on the model.
func loadImageAttachment(path string) (*session.ImageAttachment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	mediaType, ok := imageExtensions[ext]
	if !ok {
		mediaType = "image/png"
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	return &session.ImageAttachment{
		MediaType: mediaType,
		Data:      encoded,
		FileName:  filepath.Base(path),
	}, nil
}
