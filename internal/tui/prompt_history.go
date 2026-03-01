package tui

// prompt_history.go — persistent input history for the prompt, matching opencode's history.tsx
// Stores the last 50 prompts in ~/.local/share/dcode/prompt-history.jsonl
// Up arrow = older, down arrow = newer (same as shells and opencode).

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
)

const maxHistoryEntries = 50

// PromptHistory manages a circular buffer of past prompt inputs.
type PromptHistory struct {
	entries []string // oldest first
	index   int      // current navigation position; 0 = "current" (not in history)
	file    string   // path to JSONL persistence file
}

// NewPromptHistory loads history from disk and returns a ready PromptHistory.
func NewPromptHistory() *PromptHistory {
	ph := &PromptHistory{}
	ph.file = historyFilePath()
	ph.load()
	ph.index = len(ph.entries) // points past the end = "current input"
	return ph
}

func historyFilePath() string {
	base, _ := os.UserHomeDir()
	return filepath.Join(base, ".local", "share", "dcode", "prompt-history.jsonl")
}

// load reads history entries from disk.
func (h *PromptHistory) load() {
	f, err := os.Open(h.file)
	if err != nil {
		return
	}
	defer f.Close()

	var entries []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		var s string
		if err := json.Unmarshal([]byte(line), &s); err == nil && s != "" {
			entries = append(entries, s)
		}
	}
	// Keep only the last maxHistoryEntries
	if len(entries) > maxHistoryEntries {
		entries = entries[len(entries)-maxHistoryEntries:]
	}
	h.entries = entries
}

// Append adds a new entry, persists it, and resets navigation to the end.
func (h *PromptHistory) Append(input string) {
	if input == "" {
		return
	}
	// Deduplicate: if same as last entry, skip
	if len(h.entries) > 0 && h.entries[len(h.entries)-1] == input {
		h.index = len(h.entries)
		return
	}
	h.entries = append(h.entries, input)
	if len(h.entries) > maxHistoryEntries {
		h.entries = h.entries[len(h.entries)-maxHistoryEntries:]
	}
	h.index = len(h.entries) // reset to "current" position
	h.persist()
}

// Up moves back in history (older) and returns the entry, or "" if at oldest.
func (h *PromptHistory) Up(currentInput string) string {
	if len(h.entries) == 0 {
		return currentInput
	}
	// If at the end and the user has typed something, don't consume it
	if h.index == len(h.entries) && currentInput != "" {
		// Just start navigating backwards
	}
	if h.index > 0 {
		h.index--
	}
	return h.entries[h.index]
}

// Down moves forward in history (newer) and returns the entry; "" means "current input".
func (h *PromptHistory) Down() string {
	if h.index < len(h.entries) {
		h.index++
	}
	if h.index == len(h.entries) {
		return "" // back to fresh input
	}
	return h.entries[h.index]
}

// Reset resets navigation to the end (fresh input position).
func (h *PromptHistory) Reset() {
	h.index = len(h.entries)
}

// persist writes all entries to the JSONL file.
func (h *PromptHistory) persist() {
	_ = os.MkdirAll(filepath.Dir(h.file), 0o755)
	f, err := os.Create(h.file)
	if err != nil {
		return
	}
	defer f.Close()
	for _, e := range h.entries {
		b, _ := json.Marshal(e)
		_, _ = f.Write(b)
		_, _ = f.Write([]byte("\n"))
	}
}
