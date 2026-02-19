# DCode TUI Architecture

## Current Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                          Main TUI Model                          │
│  ┌────────────┬──────────────┬────────────┬─────────────────┐  │
│  │  Viewport  │   Textarea   │  Spinner   │  Theme Registry │  │
│  └────────────┴──────────────┴────────────┴─────────────────┘  │
│                                                                   │
│  State:                                                           │
│  • Current view (chat/sessions/help/settings/etc.)               │
│  • Messages history                                               │
│  • Streaming state                                                │
│  • Dialog state                                                   │
│  • Session metadata                                               │
│                                                                   │
│  Dependencies:                                                    │
│  • Session Store                                                  │
│  • Prompt Engine                                                  │
│  • Config                                                         │
│  • Agent/Provider/Model info                                      │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                         Component Library                         │
│  ┌─────────────┬──────────────┬─────────────┬──────────────┐   │
│  │   Table     │     List     │   Dialog    │     Tree     │   │
│  └─────────────┴──────────────┴─────────────┴──────────────┘   │
│  ┌─────────────┬──────────────┬─────────────┬──────────────┐   │
│  │  Markdown   │    Syntax    │    Diff     │     Tabs     │   │
│  │  Renderer   │ Highlighter  │   Viewer    │              │   │
│  └─────────────┴──────────────┴─────────────┴──────────────┘   │
│  ┌─────────────┬──────────────┬─────────────────────────────┐  │
│  │   Command   │     Pane     │         (More planned)      │  │
│  │   Palette   │              │                             │  │
│  └─────────────┴──────────────┴─────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                          Theme System                             │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  Built-in Themes:                                         │   │
│  │  • Catppuccin Mocha (default)                            │   │
│  │  • Dracula                                                │   │
│  │  • Tokyo Night                                            │   │
│  │  • Nord                                                   │   │
│  │  • Gruvbox                                                │   │
│  │  • One Dark                                               │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                   │
│  Color Categories:                                                │
│  • Semantic (primary, secondary, accent, success, etc.)          │
│  • Text (normal, muted, dim, bright)                             │
│  • UI (background, surface, border, highlight)                   │
│  • Roles (user, assistant, system, tool)                         │
└─────────────────────────────────────────────────────────────────┘
```

## View Hierarchy

```
                    ┌──────────────┐
                    │   Main View  │
                    └──────┬───────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
    ┌────▼────┐      ┌─────▼─────┐    ┌─────▼─────┐
    │  Chat   │      │ Sessions  │    │   Help    │
    │  View   │      │   View    │    │   View    │
    └─────────┘      └───────────┘    └───────────┘
         │
    ┌────┴────────────────────────────────┐
    │                                     │
┌───▼──────┐                      ┌──────▼──────┐
│ Dialogs: │                      │  Overlays:  │
│ • Model  │                      │ • Command   │
│ • Provider│                     │   Palette   │
│ • Agent   │                      │ • Settings  │
│ • Settings│                      └─────────────┘
└──────────┘
```

## Message Rendering Flow

```
┌─────────────────────────────────────────────────────────────┐
│                     Message List                             │
│  ┌───────────────────────────────────────────────────────┐ │
│  │  User Message                                         │ │
│  │  ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓  │ │
│  │  ┃ Message text here                              ┃  │ │
│  │  ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛  │ │
│  └───────────────────────────────────────────────────────┘ │
│  ┌───────────────────────────────────────────────────────┐ │
│  │  Assistant Message                                    │ │
│  │  ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓  │ │
│  │  ┃ Response with markdown                         ┃  │ │
│  │  ┃                                                 ┃  │ │
│  │  ┃ ┌─────────────────────────────────────────┐   ┃  │ │
│  │  ┃ │ ```go                                   │   ┃  │ │
│  │  ┃ │ func example() {                        │   ┃  │ │
│  │  ┃ │     // Syntax highlighted code          │   ┃  │ │
│  │  ┃ │ }                                       │   ┃  │ │
│  │  ┃ │ ```                                     │   ┃  │ │
│  │  ┃ └─────────────────────────────────────────┘   ┃  │ │
│  │  ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛  │ │
│  └───────────────────────────────────────────────────────┘ │
│  ┌───────────────────────────────────────────────────────┐ │
│  │  Tool Call                                            │ │
│  │  ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓  │ │
│  │  ┃ ⚡ read(path="file.go")                       ┃  │ │
│  │  ┃   Result: [content preview]                    ┃  │ │
│  │  ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛  │ │
│  └───────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘

Processing Pipeline:
  Raw Text → Markdown Parser → Syntax Highlighter → Themed Renderer → Display
```

## Component State Management

```
┌──────────────────────────────────────────────────────────────┐
│                     Component Lifecycle                       │
│                                                                │
│  1. Init()                                                     │
│     • Set initial state                                        │
│     • Return initial commands                                  │
│     • Subscribe to events                                      │
│                                                                │
│  2. Update(msg)                                                │
│     • Process messages                                         │
│     • Update internal state                                    │
│     • Return new commands                                      │
│     • Chain commands if needed                                 │
│                                                                │
│  3. View()                                                     │
│     • Render current state                                     │
│     • Apply theme styles                                       │
│     • Return string output                                     │
│                                                                │
└──────────────────────────────────────────────────────────────┘

Message Flow:
  tea.Msg → Update() → State Change → View() → Terminal Output
            ↓
      Commands → Future Messages
```

## Planned Architecture Improvements

```
┌─────────────────────────────────────────────────────────────────┐
│                    Enhanced TUI Model                            │
│  ┌────────────┬──────────────┬────────────┬─────────────────┐  │
│  │  Split     │   Tab Bar    │   Status   │   Unified       │  │
│  │  View Mgr  │   Manager    │   Manager  │   Styles        │  │
│  └────────────┴──────────────┴────────────┴─────────────────┘  │
│  ┌────────────┬──────────────┬────────────┬─────────────────┐  │
│  │  Progress  │   Animation  │   Toast    │   Context       │  │
│  │  Tracker   │   Engine     │   Manager  │   Menu          │  │
│  └────────────┴──────────────┴────────────┴─────────────────┘  │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                   Extended Component Library                      │
│  New Components:                                                  │
│  • Breadcrumb   • Toast        • Dropdown    • Badge             │
│  • Card         • Progress Bar • Split View  • File Browser      │
│  • Minimap      • Context Menu • Status Bar  • Toolbar           │
│                                                                   │
│  Enhanced Existing:                                               │
│  • Table (resizing, filtering, sorting)                          │
│  • List (grouping, infinite scroll, drag & drop)                │
│  • Dialog (stacking, dragging, custom content)                   │
│  • Tree (lazy loading, search, multi-select)                    │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                      Style System v2                              │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  Unified Style Manager                                    │   │
│  │  • Theme-aware styling                                    │   │
│  │  • Component style variants                               │   │
│  │  • Responsive sizing                                      │   │
│  │  • Animation support                                      │   │
│  │  • Dark/Light mode detection                              │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## Data Flow Diagram

```
┌──────────────┐
│     User     │
│    Input     │
└──────┬───────┘
       │
       ▼
┌──────────────┐      ┌─────────────┐
│   Keyboard   │──────▶│  Update()   │
│    Events    │      │   Handler   │
└──────────────┘      └──────┬──────┘
                             │
       ┌─────────────────────┼─────────────────────┐
       │                     │                     │
       ▼                     ▼                     ▼
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│   Model     │      │  Component  │      │   Command   │
│   State     │      │   State     │      │   Dispatch  │
└──────┬──────┘      └──────┬──────┘      └──────┬──────┘
       │                     │                     │
       └─────────────────────┼─────────────────────┘
                             │
                             ▼
                      ┌─────────────┐
                      │   View()    │
                      │  Renderer   │
                      └──────┬──────┘
                             │
                             ▼
                      ┌─────────────┐      ┌─────────────┐
                      │   Theme     │──────▶│  Terminal   │
                      │   Styling   │      │   Output    │
                      └─────────────┘      └─────────────┘
```

## File Structure

```
internal/tui/
├── tui.go                    # Main TUI model
├── styles.go                 # Unified style definitions (NEW)
├── animations.go             # Animation helpers (NEW)
├── keybindings.go           # Centralized key mappings (NEW)
│
├── components/
│   ├── table.go             # Data tables
│   ├── list.go              # Selectable lists
│   ├── dialog.go            # Modal dialogs
│   ├── tree.go              # File/folder trees
│   ├── markdown.go          # Markdown rendering
│   ├── syntax.go            # Code syntax highlighting
│   ├── diff.go              # Diff viewer
│   ├── tabs.go              # Tab management
│   ├── pane.go              # Split panes
│   ├── commandpalette.go    # Command palette
│   │
│   ├── breadcrumb.go        # Breadcrumb nav (NEW)
│   ├── toast.go             # Toast notifications (NEW)
│   ├── dropdown.go          # Dropdown menus (NEW)
│   ├── badge.go             # Badge component (NEW)
│   ├── card.go              # Card layout (NEW)
│   ├── progress.go          # Progress bars (NEW)
│   ├── statusbar.go         # Enhanced status bar (NEW)
│   ├── contextmenu.go       # Context menus (NEW)
│   ├── filebrowser.go       # File browser (NEW)
│   └── minimap.go           # Minimap/scrollbar (NEW)
│
└── views/
    ├── chat.go              # Chat view (REFACTORED)
    ├── sessions.go          # Session list view (REFACTORED)
    ├── help.go              # Help view (REFACTORED)
    ├── settings.go          # Settings view (REFACTORED)
    └── file.go              # File browser view (NEW)
```

## Component Interaction Patterns

### Pattern 1: Dialog with Callback
```
┌─────────────┐
│   Main      │
│   View      │
└──────┬──────┘
       │ Show Dialog
       ▼
┌─────────────┐
│   Dialog    │◀──── User Input
│  Component  │
└──────┬──────┘
       │ Callback/Message
       ▼
┌─────────────┐
│   Main      │
│   View      │───────▶ Process Result
└─────────────┘
```

### Pattern 2: Streaming Updates
```
┌─────────────┐
│   Session   │
│   Manager   │
└──────┬──────┘
       │ Stream Events
       ▼
┌─────────────┐
│  Message    │◀──── Incremental Updates
│  Handler    │
└──────┬──────┘
       │ Update View
       ▼
┌─────────────┐
│  Viewport   │
│  Renderer   │───────▶ Display
└─────────────┘
```

### Pattern 3: Theme Changes
```
┌─────────────┐
│   User      │
│   Action    │
└──────┬──────┘
       │ Change Theme
       ▼
┌─────────────┐
│   Theme     │
│  Registry   │
└──────┬──────┘
       │ New Theme
       ▼
┌─────────────┐
│   Style     │
│   Manager   │───────▶ Rebuild Styles
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  All        │
│  Components │───────▶ Re-render
└─────────────┘
```

## Performance Optimization Strategy

```
┌─────────────────────────────────────────────────────────────┐
│                  Rendering Optimization                      │
│                                                               │
│  1. Virtual Scrolling                                         │
│     Only render visible messages                              │
│     • Window: visible height + buffer                         │
│     • Lazy load off-screen content                            │
│                                                               │
│  2. Caching                                                   │
│     Cache rendered output                                     │
│     • Cache key: content hash + theme + width                 │
│     • Invalidate on change                                    │
│                                                               │
│  3. Debouncing                                                │
│     Batch updates during streaming                            │
│     • Collect changes                                         │
│     • Render at 60fps max                                     │
│                                                               │
│  4. Differential Updates                                      │
│     Only update changed sections                              │
│     • Track dirty regions                                     │
│     • Partial re-renders                                      │
│                                                               │
└─────────────────────────────────────────────────────────────┘

Target Performance:
• Initial render: < 100ms
• Update cycle: < 16ms (60fps)
• Theme switch: < 200ms
• Large session (1000+ messages): < 50ms scroll
```

## State Management Strategy

```
┌──────────────────────────────────────────────────────────────┐
│                     State Categories                          │
│                                                                │
│  1. UI State (ephemeral)                                       │
│     • Current view                                             │
│     • Dialog visibility                                        │
│     • Cursor positions                                         │
│     • Scroll offsets                                           │
│                                                                │
│  2. Session State (persistent)                                 │
│     • Message history                                          │
│     • Session metadata                                         │
│     • Settings/preferences                                     │
│     • Tool state                                               │
│                                                                │
│  3. Application State (global)                                 │
│     • Current provider/model                                   │
│     • Authentication status                                    │
│     • Active sessions                                          │
│     • Recent files/commands                                    │
│                                                                │
└──────────────────────────────────────────────────────────────┘

State Update Pattern:
  msg → validate → update model → save (if needed) → re-render
```

---

## Implementation Notes

### Key Design Principles

1. **Separation of Concerns**: Keep model, update, and view logic separate
2. **Composability**: Build complex UIs from simple, reusable components
3. **Theme-First**: All styling goes through theme system
4. **Performance**: Optimize for responsiveness and smooth animations
5. **Accessibility**: Keyboard-first, with clear focus indicators
6. **Testability**: Each component is independently testable

### Migration Path

1. **Phase 1**: Refactor existing code to use unified style system
2. **Phase 2**: Extract views into separate files
3. **Phase 3**: Add new components one at a time
4. **Phase 4**: Integrate new features into main TUI
5. **Phase 5**: Optimize and polish

### Testing Strategy

```
Unit Tests
  ├── Component rendering
  ├── State transitions
  ├── Event handling
  └── Style application

Integration Tests
  ├── View switching
  ├── Dialog interactions
  ├── Theme changes
  └── Streaming updates

Visual Tests
  ├── Snapshot testing
  ├── Layout verification
  └── Theme consistency

Performance Tests
  ├── Render benchmarks
  ├── Memory profiling
  └── Scroll performance
```

---

**Last Updated**: 2026-02-18  
**Version**: 1.0  
**Status**: Living Document
