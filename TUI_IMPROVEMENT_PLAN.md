# TUI Improvement Plan for DCode

## Overview
This document outlines comprehensive improvements for the DCode TUI (Terminal User Interface) system. The current implementation uses Bubble Tea, Lip Gloss, and custom components but has opportunities for enhancement in UX, performance, and functionality.

## Current State Analysis

### ✅ Strengths
- Well-structured component architecture
- Theme system with multiple built-in themes (Catppuccin, Dracula, Tokyo Night, Nord, Gruvbox)
- Syntax highlighting with Chroma
- Markdown rendering with Glamour
- Custom components: Table, List, Dialog, Tree, Diff Viewer
- Streaming support for LLM responses
- Session management
- Command palette

### ⚠️ Areas for Improvement
1. **Hardcoded colors** - Many colors are hardcoded instead of using theme system
2. **Limited responsiveness** - Some components don't adapt well to small terminals
3. **No progress indicators** - Missing visual feedback for long operations
4. **Limited animations** - No smooth transitions or loading states
5. **Keyboard shortcuts** - Could be more discoverable and consistent
6. **Component reusability** - Some styling is duplicated across files
7. **Accessibility** - No high contrast mode or screen reader considerations
8. **Performance** - Viewport rendering could be optimized for large message histories

---

## Phase 1: Core UX Improvements (High Priority)

### 1.1 Theme System Integration
**Goal**: Replace all hardcoded colors with theme-based styling

**Changes**:
- [ ] Create a unified `Styles` struct in `internal/tui/styles.go` that uses theme colors
- [ ] Refactor `tui.go` to use theme colors for all inline styles
- [ ] Update component styling to accept theme-aware style parameters
- [ ] Add method to refresh all styles when theme changes

**Files to modify**:
- `internal/tui/tui.go` (lines 44-126)
- `internal/tui/components/*.go`
- Create `internal/tui/styles.go`

**Benefits**:
- Consistent theming across all components
- Easier to add new themes
- No visual glitches when switching themes

### 1.2 Enhanced Status Bar
**Goal**: Provide better contextual information and visual feedback

**Features**:
- [ ] Show current file/context when using tools
- [ ] Display token usage counter (current/max)
- [ ] Add session metadata (message count, duration)
- [ ] Animated indicators for background tasks
- [ ] Color-coded connection status (green/yellow/red)

**Implementation**:
```go
type StatusBar struct {
    Provider       string
    Model          string
    Agent          string
    TokensUsed     int
    TokensMax      int
    MessageCount   int
    SessionAge     time.Duration
    ConnectionStatus ConnectionStatus
    ActiveTool     string
    Theme          *theme.Theme
}
```

### 1.3 Progress Indicators
**Goal**: Show visual feedback for long-running operations

**Components needed**:
- [ ] Spinner with operation name (already partially implemented)
- [ ] Progress bar for file operations
- [ ] Percentage display for batch operations
- [ ] Estimated time remaining

**New file**: `internal/tui/components/progress.go`

### 1.4 Message Rendering Improvements
**Goal**: Better message display with context and actions

**Features**:
- [ ] Collapsible tool call details
- [ ] Copy button for code blocks
- [ ] Line numbers for code blocks (toggle)
- [ ] Syntax highlighting preview in collapsed state
- [ ] Message timestamps (toggle)
- [ ] Message actions menu (copy, regenerate, edit)

### 1.5 Enhanced Viewport Navigation
**Goal**: Better navigation in long conversations

**Features**:
- [ ] Jump to top/bottom (Home/End keys)
- [ ] Search in conversation (Ctrl+F)
- [ ] Bookmarks/markers for important messages
- [ ] Minimap/scrollbar indicator
- [ ] Smooth scroll animation

---

## Phase 2: Advanced Features (Medium Priority)

### 2.1 Split Panes
**Goal**: Allow side-by-side views

**Features**:
- [ ] Split horizontal (Ctrl+Shift+H)
- [ ] Split vertical (Ctrl+Shift+V)
- [ ] View session list + chat simultaneously
- [ ] View help + chat simultaneously
- [ ] Adjustable pane sizes with mouse

**New file**: `internal/tui/components/splitview.go`

### 2.2 Enhanced Command Palette
**Goal**: Make command palette more powerful

**Features**:
- [ ] Recent commands history
- [ ] Fuzzy search with highlighting
- [ ] Command categories with icons
- [ ] Keyboard shortcuts display
- [ ] Preview pane for commands
- [ ] Custom command aliases

**Improvements to**: `internal/tui/components/commandpalette.go`

### 2.3 File Browser Integration
**Goal**: Built-in file navigation

**Features**:
- [ ] Tree view of project files
- [ ] Quick file preview
- [ ] File search/filter
- [ ] Git status indicators
- [ ] Fuzzy file finder (Ctrl+O)

**Use**: `internal/tui/components/tree.go` (already exists!)

### 2.4 Tabs for Multiple Sessions
**Goal**: Work with multiple sessions simultaneously

**Features**:
- [ ] Tab bar at top
- [ ] Cycle through tabs (Ctrl+Tab)
- [ ] Close tab (Ctrl+W)
- [ ] Tab indicators (unsaved, has errors)
- [ ] Drag to reorder

**Use**: `internal/tui/components/tabs.go` (already exists!)

### 2.5 Rich Context Menu
**Goal**: Right-click or shortcut context menus

**Features**:
- [ ] Message context menu (copy, regenerate, edit, delete)
- [ ] Code block menu (copy, run, save)
- [ ] Selection menu (explain, improve, test)
- [ ] Provider-specific actions

---

## Phase 3: Polish & Performance (Medium Priority)

### 3.1 Animations & Transitions
**Goal**: Smooth, delightful interactions

**Features**:
- [ ] Fade in/out for dialogs
- [ ] Slide transitions for view changes
- [ ] Smooth scrolling in viewport
- [ ] Loading skeleton screens
- [ ] Success/error toast animations

**New file**: `internal/tui/animations.go`

### 3.2 Responsive Design
**Goal**: Work well in all terminal sizes

**Features**:
- [ ] Minimum size detection with warning
- [ ] Adaptive layouts for small terminals
- [ ] Mobile-friendly key bindings (for Termux, etc.)
- [ ] Graceful degradation for limited color support

### 3.3 Performance Optimization
**Goal**: Handle large sessions efficiently

**Improvements**:
- [ ] Virtual scrolling for long message lists
- [ ] Lazy rendering for off-screen content
- [ ] Message pagination/chunking
- [ ] Cached rendered output
- [ ] Background compaction

### 3.4 Keyboard Shortcuts
**Goal**: Consistent, discoverable shortcuts

**Improvements**:
- [ ] Global shortcuts reference (F1 or ?)
- [ ] View-specific shortcuts displayed in status bar
- [ ] Customizable key bindings via config
- [ ] Vim mode option
- [ ] Emacs mode option

### 3.5 Accessibility
**Goal**: Usable by everyone

**Features**:
- [ ] High contrast theme
- [ ] Larger text option
- [ ] Screen reader friendly mode (plain text)
- [ ] Keyboard-only navigation
- [ ] Focus indicators
- [ ] ARIA-like attributes (when possible)

---

## Phase 4: Advanced Polish (Low Priority)

### 4.1 Interactive Tutorials
**Goal**: In-app learning experience

**Features**:
- [ ] First-run tutorial
- [ ] Interactive hints system
- [ ] Feature discovery tips
- [ ] Contextual help

### 4.2 Customization
**Goal**: Let users personalize their experience

**Features**:
- [ ] Custom theme editor
- [ ] Layout presets (compact, spacious, minimal)
- [ ] Custom status bar format
- [ ] Configurable prompt format
- [ ] Per-provider custom styling

### 4.3 Advanced Diff Viewer
**Goal**: Better change visualization

**Features**:
- [ ] Word-level diffs
- [ ] Character-level diffs for small changes
- [ ] Syntax-aware diffs
- [ ] 3-way merge view
- [ ] Interactive conflict resolution

### 4.4 Session Insights
**Goal**: Visualize session statistics

**Features**:
- [ ] Token usage graph
- [ ] Tool usage breakdown
- [ ] Response time analytics
- [ ] Cost estimator
- [ ] Export session report

### 4.5 Collaboration Features
**Goal**: Share and collaborate

**Features**:
- [ ] Export conversation as markdown
- [ ] Share snippet as image (terminal screenshot)
- [ ] QR code for sharing config
- [ ] Import/export sessions

---

## Phase 5: Component Library Enhancements

### 5.1 New Components Needed

#### Breadcrumb Navigation
```go
// internal/tui/components/breadcrumb.go
type Breadcrumb struct {
    Items []BreadcrumbItem
    Separator string
    Theme *theme.Theme
}
```

#### Toast Notifications
```go
// internal/tui/components/toast.go
type Toast struct {
    Message string
    Type ToastType // Success, Error, Warning, Info
    Duration time.Duration
    Progress float64
}
```

#### Dropdown Menu
```go
// internal/tui/components/dropdown.go
type Dropdown struct {
    Items []DropdownItem
    Selected int
    Width int
    MaxHeight int
    IsOpen bool
}
```

#### Badge Component
```go
// internal/tui/components/badge.go
type Badge struct {
    Text string
    Variant BadgeVariant // Primary, Secondary, Success, Warning, Error
    Size BadgeSize // Small, Medium, Large
}
```

#### Card Component
```go
// internal/tui/components/card.go
type Card struct {
    Title string
    Content string
    Footer string
    Bordered bool
    Elevated bool
}
```

### 5.2 Component Improvements

#### Table Component
- [ ] Column resizing
- [ ] Row selection with checkbox
- [ ] Inline editing
- [ ] Filtering per column
- [ ] Export to CSV

#### List Component
- [ ] Drag and drop reordering
- [ ] Grouped items with headers
- [ ] Expandable items
- [ ] Infinite scroll
- [ ] Search/filter

#### Dialog Component
- [ ] Custom content support
- [ ] Multiple dialog stacking
- [ ] Draggable dialogs
- [ ] Resizable dialogs
- [ ] Modal backdrop blur

#### Tree Component
- [ ] Lazy loading for large directories
- [ ] Search/filter in tree
- [ ] Multi-select nodes
- [ ] Drag and drop
- [ ] Context menu per node

---

## Implementation Strategy

### Sprint 1 (Week 1-2): Foundation
1. Theme system integration (1.1)
2. Enhanced status bar (1.2)
3. Progress indicators (1.3)

### Sprint 2 (Week 3-4): Core UX
4. Message rendering improvements (1.4)
5. Viewport navigation (1.5)
6. Keyboard shortcuts (3.4)

### Sprint 3 (Week 5-6): Advanced Features
7. Split panes (2.1)
8. Enhanced command palette (2.2)
9. File browser integration (2.3)

### Sprint 4 (Week 7-8): Polish
10. Animations & transitions (3.1)
11. Responsive design (3.2)
12. Performance optimization (3.3)

### Sprint 5 (Week 9-10): Refinement
13. Accessibility (3.5)
14. Component enhancements (5.2)
15. Testing and bug fixes

---

## Technical Considerations

### Dependencies
- ✅ `github.com/charmbracelet/bubbletea` - Already in use
- ✅ `github.com/charmbracelet/lipgloss` - Already in use
- ✅ `github.com/charmbracelet/bubbles` - Already in use (partially)
- ⚠️ `github.com/charmbracelet/harmonica` - Consider for animations
- ⚠️ `github.com/muesli/reflow` - Consider for text wrapping
- ⚠️ `github.com/atotto/clipboard` - Consider for clipboard support

### Testing Strategy
1. Unit tests for each component
2. Integration tests for view transitions
3. Visual regression tests (snapshot testing)
4. Performance benchmarks for large sessions
5. Manual testing on different terminals (iTerm2, Windows Terminal, Alacritty, etc.)

### Documentation Needs
1. Component usage guide
2. Theming guide
3. Keyboard shortcuts reference
4. Architecture documentation
5. Contribution guide for TUI components

---

## Success Metrics

### User Experience
- [ ] Reduced time to complete common tasks (measure with analytics)
- [ ] Fewer keyboard strokes for frequent operations
- [ ] Positive user feedback on smoothness and responsiveness

### Technical
- [ ] 100% theme coverage (no hardcoded colors)
- [ ] < 16ms render time for 60fps smooth scrolling
- [ ] < 100ms response time for all interactions
- [ ] Support for terminals as small as 80x24

### Quality
- [ ] 80%+ test coverage for TUI components
- [ ] Zero accessibility violations
- [ ] Works on all major terminal emulators

---

## Quick Wins (Can implement immediately)

1. **Add `?` key for help anywhere** - Easy keyboard shortcut
2. **Status message with auto-dismiss** - Better UX for transient messages
3. **Copy button hint on hover** - Show "Press Y to copy" on code blocks
4. **Session name editor** - Press F2 to rename current session
5. **Clear screen shortcut** - Ctrl+L to clear messages
6. **Jump to bottom on new message** - Auto-scroll when new message arrives
7. **Message count in status bar** - Show N/M messages
8. **Theme preview in settings** - Show sample when selecting theme
9. **Error details expansion** - Click or press 'e' to see full error
10. **Vim-style navigation** - j/k for scrolling (already partially done)

---

## Next Steps

1. **Review this plan** - Get feedback from team/users
2. **Prioritize features** - Based on user needs and effort
3. **Create issues** - Break down into implementable tasks
4. **Start Sprint 1** - Begin with foundation work
5. **Iterate** - Get user feedback and adjust plan

---

## Notes

- Keep backwards compatibility where possible
- Follow Bubble Tea best practices
- Maintain separation of concerns (model/update/view)
- Use composition over inheritance for components
- Document all public APIs
- Add examples for each new component

---

**Last Updated**: 2026-02-18  
**Version**: 1.0  
**Status**: Draft - Ready for Review
