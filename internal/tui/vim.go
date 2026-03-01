package tui

// ─── Vim Mode for DCode TUI ──────────────────────────────────────────────────────
//
// Provides full vim keybinding support for:
//   • The chat viewport (scroll navigation): j/k/gg/G/Ctrl-d/Ctrl-u/Ctrl-f/Ctrl-b/yy
//   • The textarea input (normal + insert mode): all core vim motions and operators
//
// Modes:
//   VimModeInsert  – default typing; Esc → Normal
//   VimModeNormal  – navigation / operator mode; i/a/A/I/o/O → Insert

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/lipgloss"
)

// ─── Types ───────────────────────────────────────────────────────────────────────

// VimMode is the current editing mode for the input textarea.
type VimMode int

const (
	VimModeInsert VimMode = iota // Default – normal typing
	VimModeNormal                // Navigation / command mode
)

func (v VimMode) String() string {
	if v == VimModeNormal {
		return "NORMAL"
	}
	return "INSERT"
}

// VimState holds all state for vim key handling.
type VimState struct {
	Mode        VimMode
	PendingOp   string // Pending operator waiting for a motion: "d","y","c"
	LastKey     string // Previous key for double-key detection (gg, dd, yy, cc)
	CountStr    string // Numeric count being accumulated ("12" before "j" = 12j)
	ReplaceNext bool   // True after "r" – next key replaces the char under cursor
	YankBuffer  string // Local yank/paste buffer
}

// count returns the numeric prefix, defaulting to 1 and capping at 999.
func (vs *VimState) count() int {
	if vs.CountStr == "" {
		return 1
	}
	n := 0
	for _, c := range vs.CountStr {
		if c < '0' || c > '9' {
			return 1
		}
		n = n*10 + int(c-'0')
	}
	if n <= 0 {
		n = 1
	}
	if n > 999 {
		n = 999
	}
	return n
}

// reset clears transient state after a command finishes.
func (vs *VimState) reset() {
	vs.PendingOp = ""
	vs.LastKey = ""
	vs.CountStr = ""
	vs.ReplaceNext = false
}

// ─── Status bar rendering ─────────────────────────────────────────────────────────

// RenderVimMode returns a styled mode badge for the footer (e.g. "NORMAL" or "INSERT").
func (m *Model) RenderVimMode() string {
	badge := lipgloss.NewStyle().Bold(true).Padding(0, 1)
	switch m.vimState.Mode {
	case VimModeNormal:
		s := badge.Background(lipgloss.Color("#89B4FA")).Foreground(lipgloss.Color("#1E1E2E")).Render("NORMAL")
		if m.vimState.CountStr != "" {
			s += dimStyle.Render(" " + m.vimState.CountStr)
		}
		if m.vimState.PendingOp != "" {
			s += dimStyle.Render(" " + m.vimState.PendingOp + "…")
		}
		if m.vimState.ReplaceNext {
			s += dimStyle.Render(" r…")
		}
		return s
	default:
		return badge.Background(lipgloss.Color("#A6E3A1")).Foreground(lipgloss.Color("#1E1E2E")).Render("INSERT")
	}
}

// ─── Editor state helpers ─────────────────────────────────────────────────────────

// editorState captures the current textarea content, line, col, and all lines.
type editorState struct {
	value    string
	lines    []string
	line     int // current logical line (0-based)
	col      int // character offset within current line (0-based)
	lineText string
}

// getEditorState snapshots the current textarea state.
func (m *Model) getEditorState() editorState {
	value := m.textarea.Value()
	lines := strings.Split(value, "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}
	line := m.textarea.Line()
	if line >= len(lines) {
		line = len(lines) - 1
	}
	col := m.textarea.LineInfo().CharOffset
	lineText := lines[line]
	return editorState{value: value, lines: lines, line: line, col: col, lineText: lineText}
}

// repositionCursor places the textarea cursor at (targetLine, targetCol) after SetValue.
// SetValue resets cursor to the end, so we navigate back.
func (m *Model) repositionCursor(targetLine, targetCol int) {
	total := m.textarea.LineCount()
	// Cursor is currently on the last line; move up to target line
	for i := total - 1; i > targetLine; i-- {
		m.textarea.CursorUp()
	}
	// Set column within target line
	lines := strings.Split(m.textarea.Value(), "\n")
	if targetLine < len(lines) {
		cap := len([]rune(lines[targetLine]))
		if targetCol > cap {
			targetCol = cap
		}
	}
	m.textarea.SetCursor(targetCol)
}

// ─── Word boundary helpers ─────────────────────────────────────────────────────────

// wordForwardPos returns the column after moving n words forward from col in lineText.
func wordForwardPos(lineText string, col, n int) int {
	runes := []rune(lineText)
	pos := col
	for w := 0; w < n; w++ {
		if pos >= len(runes) {
			break
		}
		// Skip current word characters
		for pos < len(runes) && !unicode.IsSpace(runes[pos]) {
			pos++
		}
		// Skip whitespace
		for pos < len(runes) && unicode.IsSpace(runes[pos]) {
			pos++
		}
	}
	return pos
}

// wordBackwardPos returns the column after moving n words backward from col in lineText.
func wordBackwardPos(lineText string, col, n int) int {
	runes := []rune(lineText)
	pos := col
	for w := 0; w < n; w++ {
		if pos <= 0 {
			break
		}
		pos-- // step back
		// Skip trailing whitespace going backward
		for pos > 0 && unicode.IsSpace(runes[pos]) {
			pos--
		}
		// Skip word chars going backward
		for pos > 0 && !unicode.IsSpace(runes[pos-1]) {
			pos--
		}
	}
	return pos
}

// wordEndPos returns the column at the end of the n-th word forward from col.
func wordEndPos(lineText string, col, n int) int {
	runes := []rune(lineText)
	pos := col
	for w := 0; w < n; w++ {
		if pos >= len(runes)-1 {
			break
		}
		pos++
		// Skip whitespace
		for pos < len(runes) && unicode.IsSpace(runes[pos]) {
			pos++
		}
		// Move to end of word
		for pos < len(runes)-1 && !unicode.IsSpace(runes[pos+1]) {
			pos++
		}
	}
	return pos
}

// firstNonSpaceCol returns the column of the first non-whitespace char in the line.
func firstNonSpaceCol(lineText string) int {
	for i, r := range lineText {
		if !unicode.IsSpace(r) {
			return i
		}
	}
	return 0
}

// ─── Viewport vim navigation ──────────────────────────────────────────────────────

// handleVimViewport handles vim scroll keys when the viewport (chat history) is focused.
// Returns true if the key was consumed.
func (m *Model) handleVimViewport(key string) bool {
	vs := &m.vimState
	count := vs.count()

	// Accumulate numeric count (1-9, or 0 if already counting)
	if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
		vs.CountStr += key
		return true
	}
	if key == "0" && vs.CountStr != "" {
		vs.CountStr += "0"
		return true
	}

	consumed := true
	switch key {

	// ── Vertical scroll ──────────────────────────────────────────────
	case "j", "down":
		m.viewport.LineDown(count)
		vs.reset()
	case "k", "up":
		m.viewport.LineUp(count)
		vs.reset()

	// ── Half-page / full-page scroll ─────────────────────────────────
	case "ctrl+d":
		for i := 0; i < count; i++ {
			m.viewport.HalfViewDown()
		}
		vs.reset()
	case "ctrl+u":
		for i := 0; i < count; i++ {
			m.viewport.HalfViewUp()
		}
		vs.reset()
	case "ctrl+f", "pgdown":
		for i := 0; i < count; i++ {
			m.viewport.ViewDown()
		}
		vs.reset()
	case "ctrl+b", "pgup":
		for i := 0; i < count; i++ {
			m.viewport.ViewUp()
		}
		vs.reset()

	// ── Jump to top / bottom ──────────────────────────────────────────
	case "G":
		m.viewport.GotoBottom()
		vs.reset()
	case "g":
		if vs.LastKey == "g" {
			m.viewport.GotoTop()
			vs.reset()
		} else {
			vs.LastKey = "g"
		}

	// ── Horizontal (minimal – viewport rarely needs it) ───────────────
	case "h", "left", "l", "right":
		vs.reset()

	// ── Yank visible content to clipboard ────────────────────────────
	case "y":
		if vs.LastKey == "y" {
			content := m.viewport.View()
			if err := clipboard.WriteAll(content); err == nil {
				m.setStatus("Yanked viewport to clipboard")
			}
			vs.reset()
		} else {
			vs.LastKey = "y"
		}

	// ── Enter input ───────────────────────────────────────────────────
	case "i":
		m.focusTextarea()
		vs.reset()

	default:
		vs.reset()
		consumed = false
	}

	return consumed
}

// ─── Textarea vim normal mode ─────────────────────────────────────────────────────

// handleVimNormal processes a key press while the textarea is in vim normal mode.
// Returns (consumed, switchToInsert).
func (m *Model) handleVimNormal(key string) (consumed bool, switchToInsert bool) {
	vs := &m.vimState
	count := vs.count()

	// ── r: replace-next mode ─────────────────────────────────────────
	if vs.ReplaceNext {
		vs.ReplaceNext = false
		if len(key) == 1 {
			st := m.getEditorState()
			runes := []rune(st.lineText)
			if st.col < len(runes) {
				runes[st.col] = rune(key[0])
				st.lines[st.line] = string(runes)
				m.textarea.SetValue(strings.Join(st.lines, "\n"))
				m.repositionCursor(st.line, st.col)
			}
		}
		vs.reset()
		return true, false
	}

	// ── Pending operator + motion ────────────────────────────────────
	if vs.PendingOp != "" {
		op := vs.PendingOp
		vs.reset()
		return m.handleVimOperator(op, key, count)
	}

	// ── Numeric count accumulation ───────────────────────────────────
	if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
		vs.CountStr += key
		vs.LastKey = ""
		return true, false
	}
	if key == "0" && vs.CountStr != "" {
		vs.CountStr += "0"
		vs.LastKey = ""
		return true, false
	}

	switch key {

	// ════════════════════════════════════════════════════════════════
	// Mode switches
	// ════════════════════════════════════════════════════════════════

	case "i": // Insert before cursor
		vs.reset()
		return true, true

	case "a": // Append after cursor
		st := m.getEditorState()
		if st.col < len([]rune(st.lineText)) {
			m.textarea.SetCursor(st.col + 1)
		}
		vs.reset()
		return true, true

	case "A": // Append at end of line
		m.textarea.CursorEnd()
		vs.reset()
		return true, true

	case "I": // Insert at first non-whitespace
		st := m.getEditorState()
		m.textarea.SetCursor(firstNonSpaceCol(st.lineText))
		vs.reset()
		return true, true

	case "o": // Open new line below, insert
		m.textarea.CursorEnd()
		m.textarea.InsertString("\n")
		m.textarea.CursorDown()
		m.textarea.CursorStart()
		vs.reset()
		return true, true

	case "O": // Open new line above, insert
		st := m.getEditorState()
		m.textarea.CursorStart()
		m.textarea.InsertString("\n")
		// After InsertString("\n"), cursor should be on the new line; go back up
		m.repositionCursor(st.line, 0)
		vs.reset()
		return true, true

	case "s": // Substitute char (delete + insert)
		st := m.getEditorState()
		runes := []rune(st.lineText)
		if st.col < len(runes) {
			runes = append(runes[:st.col], runes[st.col+1:]...)
			st.lines[st.line] = string(runes)
			m.textarea.SetValue(strings.Join(st.lines, "\n"))
			m.repositionCursor(st.line, st.col)
		}
		vs.reset()
		return true, true

	// ════════════════════════════════════════════════════════════════
	// Basic movement: h j k l
	// ════════════════════════════════════════════════════════════════

	case "h", "left":
		st := m.getEditorState()
		newCol := st.col - count
		if newCol < 0 {
			newCol = 0
		}
		m.textarea.SetCursor(newCol)
		vs.reset()

	case "l", "right":
		st := m.getEditorState()
		max := len([]rune(st.lineText))
		if max > 0 {
			max-- // normal mode: cursor doesn't go past last char
		}
		newCol := st.col + count
		if newCol > max {
			newCol = max
		}
		if newCol < 0 {
			newCol = 0
		}
		m.textarea.SetCursor(newCol)
		vs.reset()

	case "j", "down":
		for i := 0; i < count; i++ {
			m.textarea.CursorDown()
		}
		vs.reset()

	case "k", "up":
		for i := 0; i < count; i++ {
			m.textarea.CursorUp()
		}
		vs.reset()

	// ════════════════════════════════════════════════════════════════
	// Line navigation: 0  ^  $
	// ════════════════════════════════════════════════════════════════

	case "0": // Beginning of line
		m.textarea.CursorStart()
		vs.reset()

	case "$", "end": // End of line
		m.textarea.CursorEnd()
		vs.reset()

	case "^": // First non-whitespace
		st := m.getEditorState()
		m.textarea.SetCursor(firstNonSpaceCol(st.lineText))
		vs.reset()

	// ════════════════════════════════════════════════════════════════
	// Word movement: w  b  e  W  B  E
	// ════════════════════════════════════════════════════════════════

	case "w", "W": // Word forward
		st := m.getEditorState()
		newCol := wordForwardPos(st.lineText, st.col, count)
		if newCol <= len([]rune(st.lineText)) {
			m.textarea.SetCursor(newCol)
		} else if st.line < len(st.lines)-1 {
			// Crossed line boundary – go to next line start
			m.textarea.CursorDown()
			m.textarea.CursorStart()
		}
		vs.reset()

	case "b", "B": // Word backward
		st := m.getEditorState()
		newCol := wordBackwardPos(st.lineText, st.col, count)
		m.textarea.SetCursor(newCol)
		vs.reset()

	case "e", "E": // Word end
		st := m.getEditorState()
		newCol := wordEndPos(st.lineText, st.col, count)
		if newCol < len([]rune(st.lineText)) {
			m.textarea.SetCursor(newCol)
		}
		vs.reset()

	// ════════════════════════════════════════════════════════════════
	// Document navigation: g g  G
	// ════════════════════════════════════════════════════════════════

	case "g":
		if vs.LastKey == "g" {
			// gg → first line, first non-whitespace
			total := m.textarea.LineCount()
			for i := 0; i < total; i++ {
				m.textarea.CursorUp()
			}
			m.textarea.CursorStart()
			st := m.getEditorState()
			m.textarea.SetCursor(firstNonSpaceCol(st.lineText))
			vs.reset()
		} else {
			vs.LastKey = "g"
		}
		return true, false

	case "G": // Last line, end
		total := m.textarea.LineCount()
		for i := 0; i < total; i++ {
			m.textarea.CursorDown()
		}
		m.textarea.CursorEnd()
		vs.reset()

	// ════════════════════════════════════════════════════════════════
	// Delete operators: x  X  D  d(d/motion)
	// ════════════════════════════════════════════════════════════════

	case "x": // Delete char under cursor
		st := m.getEditorState()
		runes := []rune(st.lineText)
		for i := 0; i < count && st.col < len(runes); i++ {
			runes = append(runes[:st.col], runes[st.col+1:]...)
		}
		st.lines[st.line] = string(runes)
		m.textarea.SetValue(strings.Join(st.lines, "\n"))
		col := st.col
		if col >= len([]rune(string(runes))) {
			col = len([]rune(string(runes))) - 1
		}
		if col < 0 {
			col = 0
		}
		m.repositionCursor(st.line, col)
		vs.reset()

	case "X": // Delete char before cursor
		st := m.getEditorState()
		runes := []rune(st.lineText)
		newCol := st.col
		for i := 0; i < count && newCol > 0; i++ {
			newCol--
			runes = append(runes[:newCol], runes[newCol+1:]...)
		}
		st.lines[st.line] = string(runes)
		m.textarea.SetValue(strings.Join(st.lines, "\n"))
		m.repositionCursor(st.line, newCol)
		vs.reset()

	case "D": // Delete to end of line
		st := m.getEditorState()
		runes := []rune(st.lineText)
		if st.col < len(runes) {
			st.lines[st.line] = string(runes[:st.col])
			m.textarea.SetValue(strings.Join(st.lines, "\n"))
			m.repositionCursor(st.line, st.col)
		}
		vs.reset()

	case "d":
		if vs.LastKey == "d" {
			// dd → delete current line
			m.vimDeleteLine()
			vs.reset()
		} else {
			vs.LastKey = "d"
			vs.PendingOp = "d"
			return true, false
		}

	// ════════════════════════════════════════════════════════════════
	// Change operators: C  c(c/motion)
	// ════════════════════════════════════════════════════════════════

	case "C": // Change to end of line
		st := m.getEditorState()
		runes := []rune(st.lineText)
		if st.col < len(runes) {
			st.lines[st.line] = string(runes[:st.col])
			m.textarea.SetValue(strings.Join(st.lines, "\n"))
			m.repositionCursor(st.line, st.col)
		}
		vs.reset()
		return true, true

	case "c":
		if vs.LastKey == "c" {
			// cc → change line (delete content, stay in line, insert)
			st := m.getEditorState()
			st.lines[st.line] = ""
			m.textarea.SetValue(strings.Join(st.lines, "\n"))
			m.repositionCursor(st.line, 0)
			vs.reset()
			return true, true
		}
		vs.LastKey = "c"
		vs.PendingOp = "c"
		return true, false

	// ════════════════════════════════════════════════════════════════
	// Yank operators: y(y/motion)
	// ════════════════════════════════════════════════════════════════

	case "y":
		if vs.LastKey == "y" {
			// yy → yank current line
			st := m.getEditorState()
			vs.YankBuffer = st.lineText + "\n"
			_ = clipboard.WriteAll(st.lineText)
			m.setStatus(fmt.Sprintf("Yanked: %s", vimTruncate(st.lineText, 40)))
			vs.reset()
		} else {
			vs.LastKey = "y"
			vs.PendingOp = "y"
			return true, false
		}

	// ════════════════════════════════════════════════════════════════
	// Paste: p  P
	// ════════════════════════════════════════════════════════════════

	case "p": // Paste after cursor / below line
		m.vimPaste(true, vs.YankBuffer)
		vs.reset()

	case "P": // Paste before cursor / above line
		m.vimPaste(false, vs.YankBuffer)
		vs.reset()

	// ════════════════════════════════════════════════════════════════
	// Replace: r
	// ════════════════════════════════════════════════════════════════

	case "r": // Replace single char – wait for next key
		vs.ReplaceNext = true
		vs.CountStr = ""
		vs.LastKey = ""
		return true, false

	// ════════════════════════════════════════════════════════════════
	// Join: J
	// ════════════════════════════════════════════════════════════════

	case "J": // Join next line onto current
		st := m.getEditorState()
		if st.line < len(st.lines)-1 {
			nextLine := strings.TrimLeft(st.lines[st.line+1], " \t")
			joined := strings.TrimRight(st.lineText, " ") + " " + nextLine
			st.lines = append(st.lines[:st.line], st.lines[st.line+1:]...)
			st.lines[st.line] = joined
			endCol := len([]rune(strings.TrimRight(st.lineText, " ")))
			m.textarea.SetValue(strings.Join(st.lines, "\n"))
			m.repositionCursor(st.line, endCol)
		}
		vs.reset()

	// ════════════════════════════════════════════════════════════════
	// Undo / Redo (best-effort via textarea internals)
	// ════════════════════════════════════════════════════════════════

	case "u": // Undo – not supported natively; inform user
		m.setStatus("Undo not available (textarea limitation)")
		vs.reset()

	// ════════════════════════════════════════════════════════════════
	// Clipboard yank all  /  clear
	// ════════════════════════════════════════════════════════════════

	case "Y": // Yank entire textarea content to clipboard
		content := m.textarea.Value()
		if err := clipboard.WriteAll(content); err == nil {
			m.setStatus(fmt.Sprintf("Yanked all (%d chars) to clipboard", len(content)))
		}
		vs.reset()

	default:
		// Unknown key in normal mode – consume without passing to textarea
		vs.reset()
	}

	return true, false
}

// ─── Operator + motion ────────────────────────────────────────────────────────────

// handleVimOperator applies a pending operator (d/y/c) with the given motion key.
func (m *Model) handleVimOperator(op, motion string, count int) (consumed bool, switchToInsert bool) {
	vs := &m.vimState
	st := m.getEditorState()
	runes := []rune(st.lineText)

	yank := func(text string) {
		vs.YankBuffer = text
		_ = clipboard.WriteAll(text)
		m.setStatus(fmt.Sprintf("Yanked: %s", vimTruncate(text, 40)))
	}

	deleteRange := func(fromCol, toCol int) {
		if fromCol < 0 {
			fromCol = 0
		}
		if toCol > len(runes) {
			toCol = len(runes)
		}
		if fromCol >= toCol {
			return
		}
		deleted := string(runes[fromCol:toCol])
		if op == "y" {
			yank(deleted)
			return
		}
		st.lines[st.line] = string(append(runes[:fromCol], runes[toCol:]...))
		m.textarea.SetValue(strings.Join(st.lines, "\n"))
		m.repositionCursor(st.line, fromCol)
	}

	switch motion {
	// operator + same key = line operation (dd, yy, cc)
	case "d":
		if op == "d" {
			for i := 0; i < count; i++ {
				m.vimDeleteLine()
			}
		}
	case "y":
		if op == "y" {
			vs.YankBuffer = st.lineText + "\n"
			_ = clipboard.WriteAll(st.lineText)
			m.setStatus(fmt.Sprintf("Yanked: %s", vimTruncate(st.lineText, 40)))
		}
	case "c":
		if op == "c" {
			st.lines[st.line] = ""
			m.textarea.SetValue(strings.Join(st.lines, "\n"))
			m.repositionCursor(st.line, 0)
			switchToInsert = true
		}

	// word-forward motion
	case "w", "W":
		newCol := wordForwardPos(st.lineText, st.col, count)
		deleteRange(st.col, newCol)
		switchToInsert = op == "c"

	// word-backward motion
	case "b", "B":
		newCol := wordBackwardPos(st.lineText, st.col, count)
		deleteRange(newCol, st.col)
		if op == "y" { /* already handled in deleteRange */
		}
		if op != "y" {
			// reposition to newCol (deleteRange already moved cursor there)
		}
		switchToInsert = op == "c"

	// word-end motion
	case "e", "E":
		newCol := wordEndPos(st.lineText, st.col, count) + 1
		deleteRange(st.col, newCol)
		switchToInsert = op == "c"

	// to end of line
	case "$":
		deleteRange(st.col, len(runes))
		switchToInsert = op == "c"

	// to beginning of line
	case "0":
		deleteRange(0, st.col)
		m.textarea.CursorStart()
		switchToInsert = op == "c"

	// to first non-whitespace
	case "^":
		fnCol := firstNonSpaceCol(st.lineText)
		if op != "y" {
			deleteRange(fnCol, st.col)
		} else {
			deleteRange(st.col, fnCol)
		}
		switchToInsert = op == "c"

	default:
		// Unknown motion – cancel
	}

	vs.reset()
	return true, switchToInsert
}

// ─── Helpers ──────────────────────────────────────────────────────────────────────

// vimDeleteLine deletes the current logical line, adjusting the cursor.
func (m *Model) vimDeleteLine() {
	st := m.getEditorState()
	if len(st.lines) == 1 {
		// Only line – just clear it
		m.textarea.SetValue("")
		return
	}
	// Yank it before deleting
	m.vimState.YankBuffer = st.lineText + "\n"

	newLines := make([]string, 0, len(st.lines)-1)
	newLines = append(newLines, st.lines[:st.line]...)
	newLines = append(newLines, st.lines[st.line+1:]...)
	m.textarea.SetValue(strings.Join(newLines, "\n"))

	targetLine := st.line
	if targetLine >= len(newLines) {
		targetLine = len(newLines) - 1
	}
	m.repositionCursor(targetLine, 0)
}

// vimPaste pastes the given text (or system clipboard if text is empty).
// after=true → paste after cursor/below line; after=false → paste before/above.
func (m *Model) vimPaste(after bool, yankBuf string) {
	text := yankBuf
	if text == "" {
		if cb, err := clipboard.ReadAll(); err == nil {
			text = cb
		}
	}
	if text == "" {
		return
	}

	isLinePaste := strings.HasSuffix(text, "\n")
	if isLinePaste {
		content := strings.TrimSuffix(text, "\n")
		st := m.getEditorState()
		insertAt := st.line
		if after {
			insertAt++
		}
		newLines := make([]string, 0, len(st.lines)+1)
		newLines = append(newLines, st.lines[:insertAt]...)
		newLines = append(newLines, content)
		newLines = append(newLines, st.lines[insertAt:]...)
		m.textarea.SetValue(strings.Join(newLines, "\n"))
		m.repositionCursor(insertAt, 0)
	} else {
		st := m.getEditorState()
		runes := []rune(st.lineText)
		insertPos := st.col
		if after && insertPos < len(runes) {
			insertPos++
		}
		newLine := string(runes[:insertPos]) + text + string(runes[insertPos:])
		st.lines[st.line] = newLine
		m.textarea.SetValue(strings.Join(st.lines, "\n"))
		m.repositionCursor(st.line, insertPos+len([]rune(text)))
	}
}

// vimTruncate truncates s to at most max runes, appending "…" if needed.
func vimTruncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}
