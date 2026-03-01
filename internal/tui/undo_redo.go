package tui

// undo_redo.go — Multi-step undo / redo for AI-applied file changes.
//
// Each entry in the stack is a git tree-hash produced by Snapshot.Track()
// right before the AI touched any files. Ctrl+Z walks back through the
// stack (undo); Ctrl+Y (when not streaming) or a new /redo command walks
// forward (redo).
//
// The stack is kept in memory for the lifetime of the TUI session.
// It is reset when a new session is created.

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Dhanuzh/dcode/internal/config"
	"github.com/Dhanuzh/dcode/internal/session"
)

// ─── Stack ───────────────────────────────────────────────────────────────────

const maxUndoSteps = 50

// undoEntry is one entry in the undo/redo stack.
type undoEntry struct {
	hash        string // git tree-hash of the working tree BEFORE the change
	description string // short human label (tool name + file, etc.)
}

// UndoRedoStack is a simple undo/redo stack.
// The cursor points at the entry that would be restored by the NEXT undo.
//
//	entries[0..cursor-1]  ← below the current state (available to redo)
//	entries[cursor..]     ← available to undo
//
// When a new entry is pushed, everything above the cursor is discarded
// (redo history is invalidated by new changes, just like most editors).
type UndoRedoStack struct {
	entries []undoEntry
	cursor  int // index of "current" tip; new pushes go here
}

// NewUndoRedoStack returns an empty stack.
func NewUndoRedoStack() *UndoRedoStack {
	return &UndoRedoStack{}
}

// Reset clears the stack (call on new session).
func (s *UndoRedoStack) Reset() {
	s.entries = nil
	s.cursor = 0
}

// Push records a new state BEFORE a batch of AI changes.
// Any redo history above the current cursor is discarded.
func (s *UndoRedoStack) Push(hash, description string) {
	// Trim redo history
	s.entries = s.entries[:s.cursor]
	s.entries = append(s.entries, undoEntry{hash: hash, description: description})
	// Cap size
	if len(s.entries) > maxUndoSteps {
		s.entries = s.entries[len(s.entries)-maxUndoSteps:]
	}
	s.cursor = len(s.entries)
}

// CanUndo reports whether there is a step to undo.
func (s *UndoRedoStack) CanUndo() bool { return s.cursor > 0 }

// CanRedo reports whether there is a step to redo (i.e. entries past cursor).
func (s *UndoRedoStack) CanRedo() bool { return s.cursor < len(s.entries) }

// UndoPeek returns the entry that would be restored on the next undo without
// advancing the cursor.
func (s *UndoRedoStack) UndoPeek() undoEntry {
	if s.cursor == 0 {
		return undoEntry{}
	}
	return s.entries[s.cursor-1]
}

// RedoPeek returns the entry that would be applied on the next redo.
func (s *UndoRedoStack) RedoPeek() undoEntry {
	if s.cursor >= len(s.entries) {
		return undoEntry{}
	}
	return s.entries[s.cursor]
}

// StepUndo moves the cursor back and returns the entry to restore.
func (s *UndoRedoStack) StepUndo() (undoEntry, bool) {
	if !s.CanUndo() {
		return undoEntry{}, false
	}
	s.cursor--
	return s.entries[s.cursor], true
}

// StepRedo moves the cursor forward and returns the entry to apply.
func (s *UndoRedoStack) StepRedo() (undoEntry, bool) {
	if !s.CanRedo() {
		return undoEntry{}, false
	}
	entry := s.entries[s.cursor]
	s.cursor++
	return entry, true
}

// UndoStepsAvailable returns the number of undo steps available.
func (s *UndoRedoStack) UndoStepsAvailable() int { return s.cursor }

// RedoStepsAvailable returns the number of redo steps available.
func (s *UndoRedoStack) RedoStepsAvailable() int { return len(s.entries) - s.cursor }

// ─── Tea Messages ─────────────────────────────────────────────────────────────

// UndoDoneMsg is sent when an undo operation completes.
type UndoDoneMsg struct {
	Description string
	Err         error
}

// RedoDoneMsg is sent when a redo operation completes.
type RedoDoneMsg struct {
	Description string
	Err         error
}

// SnapshotCapturedMsg is sent after a snapshot has been captured for a step.
type SnapshotCapturedMsg struct {
	Hash        string
	Description string
}

// ─── TUI integration ──────────────────────────────────────────────────────────

// captureSnapshot takes a git-tree snapshot of the current working directory
// and pushes it onto the undo stack. Called by the engine stream watcher
// right before each tool step that modifies files.
func (m *Model) captureSnapshotCmd(description string) tea.Cmd {
	return func() tea.Msg {
		snap := session.NewSnapshot(config.GetConfigDir(), config.GetProjectDir())
		hash, err := snap.Track()
		if err != nil || hash == "" {
			// No snapshot support (e.g. git not installed) — silently ignore
			return nil
		}
		return SnapshotCapturedMsg{Hash: hash, Description: description}
	}
}

// handleSnapshotCaptured processes a SnapshotCapturedMsg.
func (m *Model) handleSnapshotCaptured(msg SnapshotCapturedMsg) {
	if m.undoStack == nil {
		m.undoStack = NewUndoRedoStack()
	}
	m.undoStack.Push(msg.Hash, msg.Description)
}

// undoLastChange restores the working tree to the previous snapshot (Ctrl+Z).
func (m Model) undoLastChange() (tea.Model, tea.Cmd) {
	if m.isStreaming {
		m.setStatus("Cannot undo while generating")
		return m, nil
	}
	if m.sessionID == "" {
		m.setStatus("No active session")
		return m, nil
	}
	if m.undoStack == nil || !m.undoStack.CanUndo() {
		m.setStatus("Nothing to undo")
		return m, nil
	}

	entry, _ := m.undoStack.StepUndo()
	hash := entry.hash
	desc := entry.description

	return m, func() tea.Msg {
		snap := session.NewSnapshot(config.GetConfigDir(), config.GetProjectDir())
		if err := snap.Restore(hash); err != nil {
			return UndoDoneMsg{Err: fmt.Errorf("undo failed: %w", err)}
		}
		return UndoDoneMsg{Description: desc}
	}
}

// redoLastChange re-applies the next redo snapshot (Ctrl+Shift+Z or /redo).
func (m Model) redoLastChange() (tea.Model, tea.Cmd) {
	if m.isStreaming {
		m.setStatus("Cannot redo while generating")
		return m, nil
	}
	if m.sessionID == "" {
		m.setStatus("No active session")
		return m, nil
	}
	if m.undoStack == nil || !m.undoStack.CanRedo() {
		m.setStatus("Nothing to redo")
		return m, nil
	}

	entry, _ := m.undoStack.StepRedo()
	hash := entry.hash
	desc := entry.description

	return m, func() tea.Msg {
		snap := session.NewSnapshot(config.GetConfigDir(), config.GetProjectDir())
		if err := snap.Restore(hash); err != nil {
			return RedoDoneMsg{Err: fmt.Errorf("redo failed: %w", err)}
		}
		return RedoDoneMsg{Description: desc}
	}
}

// handleUndoDoneMsg processes the result of an undo operation.
func (m Model) handleUndoDoneMsg(msg UndoDoneMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		return m, m.showToast("Undo failed: "+msg.Err.Error(), ToastError, 6*time.Second)
	}
	// Reload session messages
	if m.sessionID != "" {
		if sess, err := m.Store.Get(m.sessionID); err == nil {
			m.messages = sess.Messages
		}
	}
	m.updateViewport()
	label := msg.Description
	if label == "" {
		label = "last step"
	}
	steps := 0
	if m.undoStack != nil {
		steps = m.undoStack.UndoStepsAvailable()
	}
	m.setStatus(fmt.Sprintf("Undone: %s  (%d more undo steps)", label, steps))
	return m, nil
}

// handleRedoDoneMsg processes the result of a redo operation.
func (m Model) handleRedoDoneMsg(msg RedoDoneMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		return m, m.showToast("Redo failed: "+msg.Err.Error(), ToastError, 6*time.Second)
	}
	if m.sessionID != "" {
		if sess, err := m.Store.Get(m.sessionID); err == nil {
			m.messages = sess.Messages
		}
	}
	m.updateViewport()
	label := msg.Description
	if label == "" {
		label = "last step"
	}
	steps := 0
	if m.undoStack != nil {
		steps = m.undoStack.RedoStepsAvailable()
	}
	m.setStatus(fmt.Sprintf("Redone: %s  (%d more redo steps)", label, steps))
	return m, nil
}
