# TUI Quick Start Improvements

## 🚀 Quick Wins (Can Implement Today)

These are small, high-impact improvements that can be done quickly.

---

## 1. Add Help Shortcut Anywhere

**What**: Press `?` to see help from any view

**Why**: Makes the app more discoverable

**Implementation**:
```go
// In tui.go Update() method
case tea.KeyMsg:
    if msg.String() == "?" {
        m.view = ViewHelp
        return m, nil
    }
```

**Effort**: 5 minutes  
**Impact**: High

---

## 2. Auto-scroll to Bottom on New Message

**What**: When a new message arrives, automatically scroll to show it

**Why**: Users shouldn't have to manually scroll to see responses

**Implementation**:
```go
// In handleStreamMsg() or handleDoneMsg()
func (m Model) handleDoneMsg() (tea.Model, tea.Cmd) {
    m.isStreaming = false
    m.streamingText.Reset()
    if m.sessionID != "" {
        if sess, err := m.Store.Get(m.sessionID); err == nil {
            m.messages = sess.Messages
        }
    }
    m.updateViewport()
    m.viewport.GotoBottom() // ADD THIS LINE
    return m, nil
}
```

**Effort**: 2 minutes  
**Impact**: High

---

## 3. Message Count in Status Bar

**What**: Show "Message 5/12" in status bar

**Why**: Helps users know where they are in conversation

**Implementation**:
```go
// In renderChat() header section
statusParts := []string{
    fmt.Sprintf("Messages: %d", len(m.messages)),
}
if m.sessionID != "" {
    statusParts = append(statusParts, fmt.Sprintf("Session: %s", shortID(m.sessionID)))
}
```

**Effort**: 10 minutes  
**Impact**: Medium

---

## 4. Confirmation for Destructive Actions

**What**: Ask "Are you sure?" before deleting sessions

**Why**: Prevent accidental data loss

**Implementation**:
```go
// In updateSessionList when 'd' is pressed
case "d":
    // Instead of immediate delete, show confirmation
    m.showConfirmDialog(
        "Delete Session",
        "Are you sure you want to delete this session?",
        func() { m.Store.Delete(sessionID) },
    )
```

**Effort**: 30 minutes (need to add confirmation dialog)  
**Impact**: High (prevents data loss)

---

## 5. Copy Code Block Hint

**What**: Show "Press 'y' to copy" when hovering over code blocks

**Why**: Improves discoverability of copy feature

**Implementation**:
```go
// In renderMessage() for code blocks
codeBlockStyle := lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(theme.Border)

header := dimStyle.Render("Press 'y' to copy") + " " + 
         lipgloss.NewStyle().Foreground(theme.Info).Render(language)
```

**Effort**: 15 minutes  
**Impact**: Medium

---

## 6. Session Age Display

**What**: Show how old the session is (e.g., "2h ago")

**Why**: Helps identify recent vs old sessions

**Implementation**:
```go
// In renderSessionListView()
func formatAge(created time.Time) string {
    dur := time.Since(created)
    if dur < time.Hour {
        return fmt.Sprintf("%dm ago", int(dur.Minutes()))
    }
    if dur < 24*time.Hour {
        return fmt.Sprintf("%dh ago", int(dur.Hours()))
    }
    return created.Format("Jan 2")
}
```

**Effort**: 15 minutes  
**Impact**: Low-Medium

---

## 7. Token Usage Display

**What**: Show estimated tokens used in current session

**Why**: Helps users track API costs

**Implementation**:
```go
// Add to Model struct
type Model struct {
    // ... existing fields
    tokensUsed int
    tokensMax  int
}

// In status bar
if m.tokensUsed > 0 {
    tokenStyle := lipgloss.NewStyle().Foreground(yellow)
    if m.tokensUsed > m.tokensMax*0.8 {
        tokenStyle = tokenStyle.Foreground(red)
    }
    status += " " + tokenStyle.Render(fmt.Sprintf("%dk/%dk tokens", m.tokensUsed/1000, m.tokensMax/1000))
}
```

**Effort**: 1 hour (need to calculate tokens)  
**Impact**: High (for cost-conscious users)

---

## 8. Clear Screen Shortcut

**What**: Press Ctrl+L to clear current view

**Why**: Standard terminal shortcut

**Implementation**:
```go
// In updateChat()
case "ctrl+l":
    m.messages = []session.Message{}
    m.updateViewport()
    m.setStatus("Screen cleared")
    return m, nil
```

**Effort**: 2 minutes  
**Impact**: Medium

---

## 9. Error Details Expansion

**What**: Press 'e' or Enter on error to see full details

**Why**: Errors are often truncated, need to see full message

**Implementation**:
```go
// Store full error in message metadata
type Message struct {
    // ... existing fields
    ErrorDetails string
    Expanded     bool
}

// In message rendering
if msg.HasError && !msg.Expanded {
    footer := dimStyle.Render("Press 'e' for details")
}
```

**Effort**: 45 minutes  
**Impact**: Medium

---

## 10. Theme Preview

**What**: Show color preview when selecting theme in settings

**Why**: Visual feedback before committing

**Implementation**:
```go
// In settings dialog
func renderThemePreview(theme *theme.Theme) string {
    return lipgloss.JoinHorizontal(
        lipgloss.Center,
        lipgloss.NewStyle().Foreground(theme.Primary).Render("■ "),
        lipgloss.NewStyle().Foreground(theme.Secondary).Render("■ "),
        lipgloss.NewStyle().Foreground(theme.Success).Render("■ "),
        lipgloss.NewStyle().Foreground(theme.Warning).Render("■ "),
        lipgloss.NewStyle().Foreground(theme.Error).Render("■ "),
    )
}
```

**Effort**: 20 minutes  
**Impact**: Low-Medium

---

## 🔧 Easy UX Improvements

### A. Better Focus Indicators

**Current**: Just text "[INPUT]" or "[SCROLL]"  
**Better**: Visual border color change

```go
// In renderChat() for textarea
border := lipgloss.RoundedBorder()
if m.focusInput {
    border = lipgloss.ThickBorder()
}
textareaStyle := lipgloss.NewStyle().
    Border(border).
    BorderForeground(m.currentTheme.Primary)
```

**Effort**: 15 minutes  
**Impact**: Medium

---

### B. Loading States

**Current**: Just spinner  
**Better**: Descriptive loading message

```go
// In streaming indicator
if m.isStreaming {
    status := "Generating response"
    if m.currentTool != "" {
        status = fmt.Sprintf("Running %s...", m.currentTool)
    }
    ind := m.spinner.View() + " " + dimStyle.Render(status)
}
```

**Effort**: 5 minutes  
**Impact**: Medium

---

### C. Empty State Messages

**Current**: Just shows empty viewport  
**Better**: Helpful getting started message

```go
// Already implemented in renderChat() but could be enhanced
welcome := lipgloss.NewStyle().
    Foreground(purple).
    Bold(true).
    Render("  Welcome to DCode ✨")

tips := []string{
    "💡 Press / for commands",
    "💡 Press Ctrl+K to change model",
    "💡 Press ? for help",
}
```

**Effort**: 10 minutes  
**Impact**: High (for new users)

---

### D. Connection Status Indicator

**Current**: Just shows provider name  
**Better**: Visual indicator of connection state

```go
// Add to status bar
func connectionIndicator(connected bool, connecting bool) string {
    if connecting {
        return lipgloss.NewStyle().Foreground(yellow).Render("○")
    }
    if connected {
        return lipgloss.NewStyle().Foreground(green).Render("●")
    }
    return lipgloss.NewStyle().Foreground(red).Render("●")
}
```

**Effort**: 20 minutes  
**Impact**: Medium

---

### E. Recent Commands History

**Current**: No command history  
**Better**: Press up/down in command palette for recent commands

```go
// Add to Model
type Model struct {
    // ...
    commandHistory []string
    historyIndex   int
}

// In command palette
case "up":
    if m.historyIndex > 0 {
        m.historyIndex--
        m.textarea.SetValue(m.commandHistory[m.historyIndex])
    }
```

**Effort**: 30 minutes  
**Impact**: High

---

## 🎨 Visual Polish

### F. Message Timestamps

**What**: Show when each message was sent  
**How**: Small timestamp in corner of each message

```go
timeStyle := lipgloss.NewStyle().
    Foreground(overlay).
    Align(lipgloss.Right)

timestamp := msg.Timestamp.Format("15:04")
```

**Effort**: 15 minutes  
**Impact**: Low

---

### G. Smooth Scrolling

**What**: Animate scroll instead of jumping  
**How**: Use small incremental updates

```go
// In viewport navigation
case "pgdown":
    for i := 0; i < 10; i++ {
        m.viewport.LineDown(1)
        // Small delay between steps for animation effect
    }
```

**Effort**: 1 hour (needs proper implementation)  
**Impact**: Medium (polish)

---

### H. Syntax Highlighting Preview

**What**: Show language icon next to code blocks  
**How**: Add file type icons

```go
var langIcons = map[string]string{
    "go":         "🔵",
    "python":     "🐍",
    "javascript": "💛",
    "rust":       "🦀",
    "typescript": "💙",
}

icon := langIcons[language]
```

**Effort**: 10 minutes  
**Impact**: Low (visual polish)

---

## 📊 Priority Matrix

```
High Impact, Low Effort (DO FIRST):
├── Auto-scroll to bottom (#2)
├── Help shortcut (#1)
├── Clear screen (#8)
├── Better loading states (B)
└── Empty state messages (C)

High Impact, Medium Effort (DO NEXT):
├── Token usage display (#7)
├── Confirmation dialogs (#4)
├── Recent command history (E)
└── Better focus indicators (A)

Medium Impact, Low Effort (QUICK WINS):
├── Message count (#3)
├── Copy hint (#5)
├── Connection indicator (D)
└── Theme preview (#10)

Low Impact (LATER):
├── Session age (#6)
├── Message timestamps (F)
└── Syntax icons (H)
```

---

## 🚦 Implementation Order (Recommended)

### Day 1: Core UX (2-3 hours)
1. ✅ Auto-scroll to bottom
2. ✅ Help shortcut anywhere
3. ✅ Better loading states
4. ✅ Empty state improvements
5. ✅ Clear screen shortcut

### Day 2: Visual Feedback (2-3 hours)
6. ✅ Message count in status
7. ✅ Connection status indicator
8. ✅ Better focus indicators
9. ✅ Copy hint for code blocks

### Day 3: Safety & History (3-4 hours)
10. ✅ Confirmation dialogs
11. ✅ Recent command history
12. ✅ Error details expansion

### Day 4: Polish (2 hours)
13. ✅ Token usage display
14. ✅ Theme preview
15. ✅ Session age display
16. ✅ Message timestamps

---

## 🧪 Testing Checklist

For each improvement, test:
- [ ] Works in small terminal (80x24)
- [ ] Works in large terminal (200x50)
- [ ] Works with all themes
- [ ] Keyboard shortcuts work
- [ ] Mouse interaction works (if applicable)
- [ ] Doesn't break existing functionality
- [ ] Performance is acceptable

---

## 📝 Code Quality Checklist

- [ ] Uses theme system (no hardcoded colors)
- [ ] Follows existing code style
- [ ] Has descriptive variable names
- [ ] Includes comments for complex logic
- [ ] Updates documentation if needed
- [ ] No magic numbers (use constants)
- [ ] Error handling is proper

---

## 🐛 Common Pitfalls to Avoid

1. **Hardcoded Colors**: Always use `m.currentTheme.*`
2. **Screen Size Assumptions**: Test on small terminals
3. **Blocking Operations**: Use goroutines for slow tasks
4. **Memory Leaks**: Clean up goroutines and channels
5. **Z-Index Issues**: Dialogs should overlay properly
6. **Race Conditions**: Be careful with concurrent state updates
7. **Terminal Compatibility**: Test on different terminals

---

## 💡 Tips

- **Use lipgloss Width/Height**: Measure rendered content, not string length
- **Cache Rendered Output**: For expensive renders (syntax highlighting)
- **Debounce Rapid Updates**: During streaming
- **Profile Performance**: Use pprof for slow renders
- **Test Theme Switching**: Ensure all styles update
- **Keep State Minimal**: Only store what's needed
- **Use Bubble Tea Examples**: Learn from official examples

---

## 🔗 Useful Resources

- [Bubble Tea Docs](https://github.com/charmbracelet/bubbletea)
- [Lip Gloss Style Guide](https://github.com/charmbracelet/lipgloss)
- [Bubbles Components](https://github.com/charmbracelet/bubbles)
- [TUI Design Patterns](https://github.com/charmbracelet/bubbletea/tree/master/examples)

---

**Last Updated**: 2026-02-18  
**Ready to implement!** Pick any item and start coding! 🚀
