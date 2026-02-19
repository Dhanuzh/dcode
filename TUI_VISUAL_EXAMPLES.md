# TUI Visual Mockups & Examples

This document shows **before/after** examples and ASCII mockups of planned improvements.

---

## 📱 Current vs Improved Status Bar

### Before (Current)
```
 DCode   anthropic   claude-3-opus   coder  
───────────────────────────────────────────────────────────────────
```

### After (Improved)
```
 DCode   ● anthropic   claude-3-opus   coder   [12 msgs]  [45k/200k ○○○○○●●●●● tokens]  2h 15m
─────────────────────────────────────────────────────────────────────────────────────────────────────
   ^         ^             ^              ^         ^              ^                ^
   |         |             |              |         |              |                |
  Logo   Status LED    Provider        Agent    Msg Count    Token Usage      Session Age
        (●=connected)
```

**Colors**:
- `●` Green = connected, Yellow = connecting, Red = error
- Token bar: Green → Yellow → Red as usage increases
- Session age: Dim gray

---

## 💬 Enhanced Message Display

### Before (Current)
```
┃ User: How do I read a file in Go?
┃
┃ Assistant: Here's how to read a file:
┃ 
┃ ```go
┃ content, err := os.ReadFile("file.txt")
┃ ```
```

### After (Improved)
```
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  You                                              14:35 ┃
┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫
┃                                                           ┃
┃  How do I read a file in Go?                             ┃
┃                                                           ┃
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃  Assistant                                        14:35 ┃
┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫
┃                                                           ┃
┃  Here's how to read a file in Go:                       ┃
┃                                                           ┃
┃  ╭───────────────────────────────────────╮               ┃
┃  │ 📄 Go            [y] copy  [▸] expand │               ┃
┃  ├───────────────────────────────────────┤               ┃
┃  │ content, err := os.ReadFile("f.txt") │               ┃
┃  │ if err != nil {                       │               ┃
┃  │     log.Fatal(err)                    │               ┃
┃  │ }                                     │               ┃
┃  ╰───────────────────────────────────────╯               ┃
┃                                                           ┃
┃  💡 Tip: Add error handling for production code          ┃
┃                                                           ┃
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
```

**Features**:
- Timestamps in corner
- Code block header with language icon
- Action hints (copy, expand)
- Syntax highlighting (shown with colors in terminal)
- Collapsible sections
- Helpful tips/warnings

---

## 🔧 Tool Call Display

### Before (Current)
```
⚡ read: file.go
Result: [file content]
```

### After (Improved - Expanded)
```
╭────────────────────────────────────────────────────────────╮
│ ⚙️  Tool Call: read                            [14:35:23] │
├────────────────────────────────────────────────────────────┤
│ Parameters:                                                │
│   path: "internal/tui/tui.go"                             │
│   offset: 1                                                │
│   limit: 100                                               │
│                                                             │
│ Result: ✓ Success (1.2s)                                  │
│   Read 2,208 lines, 64.5 KB                               │
│                                                             │
│ [▾] View Content   [e] View Error Log   [r] Retry         │
╰────────────────────────────────────────────────────────────╯
```

### After (Improved - Collapsed)
```
╭────────────────────────────────────────────────────────────╮
│ ⚙️  read("internal/tui/tui.go") → ✓ 2,208 lines  [▸] Expand │
╰────────────────────────────────────────────────────────────╯
```

**Features**:
- Collapsible details
- Success/failure indicators
- Duration and size stats
- Quick actions
- Color coding (green=success, red=error, yellow=warning)

---

## 🎛️ Command Palette (Enhanced)

### Before (Current)
```
┌──────────────────────────────────────────┐
│  Select Model                            │
├──────────────────────────────────────────┤
│  > Model: GPT-4                          │
│    Provider: Connect Provider            │
│    Agent: Cycle Agent                    │
│    Session: New Session                  │
└──────────────────────────────────────────┘
```

### After (Improved)
```
╭──────────────────────────────────────────────────────────────╮
│  🔍 Command Palette                           [Ctrl+Shift+P] │
├──────────────────────────────────────────────────────────────┤
│  Search: mod█                                                │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  📊 Model                                                    │
│  ▸  Select Model                              [Ctrl+K]      │
│     Switch to GPT-4                                          │
│                                                               │
│  🔌 Recently Used                                            │
│     /model gpt-4                              5 min ago      │
│     /theme dracula                            1 hour ago     │
│     /new                                      2 hours ago    │
│                                                               │
│  💡 Try: Type command name or shortcut                       │
╰──────────────────────────────────────────────────────────────╯
```

**Features**:
- Fuzzy search with highlighting
- Categories with icons
- Keyboard shortcuts shown
- Recently used commands
- Command preview/description
- Helpful tips

---

## 📊 Progress Indicators

### File Operation Progress
```
╭────────────────────────────────────────────────╮
│  Writing to disk...                            │
│  ████████████████████████░░░░░░░░░░  60%      │
│  3 of 5 files • 1.2 MB / 2.0 MB • 12s left    │
╰────────────────────────────────────────────────╯
```

### Batch Operation Progress
```
╭────────────────────────────────────────────────╮
│  Processing files...                           │
│  ┌──────────────────────────────────────────┐ │
│  │ ✓ file1.go                    250 lines  │ │
│  │ ✓ file2.go                    180 lines  │ │
│  │ ⏳ file3.go                   processing  │ │
│  │ ⏸ file4.go                    waiting    │ │
│  │ ⏸ file5.go                    waiting    │ │
│  └──────────────────────────────────────────┘ │
│  Progress: 2/5 complete (40%)                 │
╰────────────────────────────────────────────────╯
```

### Streaming Response
```
╭────────────────────────────────────────────────╮
│  ● Generating response...                      │
│  ⚡ Current: Analyzing code structure          │
│  ┊┊┊┊┊┊┊┊┊┊████████░░░░░░░░░░░░░░░░░░        │
│  Tokens: 1,234 / 4,096                        │
╰────────────────────────────────────────────────╯
```

---

## 🎨 Theme Selector

### Before (Current)
```
┌──────────────────────────────────┐
│  Settings                        │
├──────────────────────────────────┤
│  > Theme: dark                   │
│    Streaming: true               │
│    Max Tokens: 4096              │
└──────────────────────────────────┘
```

### After (Improved)
```
╭────────────────────────────────────────────────────────────╮
│  🎨 Theme Selection                                        │
├────────────────────────────────────────────────────────────┤
│                                                             │
│  ▸ Catppuccin Mocha         ■ ■ ■ ■ ■    ● Current       │
│    Dracula                  ■ ■ ■ ■ ■                      │
│    Tokyo Night              ■ ■ ■ ■ ■                      │
│    Nord                     ■ ■ ■ ■ ■                      │
│    Gruvbox                  ■ ■ ■ ■ ■                      │
│    One Dark                 ■ ■ ■ ■ ■                      │
│                                                             │
│  ┌────────────────────────────────────────────────────┐   │
│  │  Preview:                                          │   │
│  │  ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓   │
│  │  ┃ This is how text will look               ┃   │
│  │  ┃ • Primary color                           ┃   │
│  │  ┃ • Secondary color                         ┃   │
│  │  ┃ • Success, Warning, Error colors          ┃   │
│  │  ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛   │
│  └────────────────────────────────────────────────────┘   │
│                                                             │
│  [Enter] Apply  [Esc] Cancel  [t] Toggle dark/light       │
╰────────────────────────────────────────────────────────────╯
```

**Features**:
- Live preview of theme colors
- Visual color swatches
- Current theme indicator
- Quick toggle dark/light mode
- Sample text rendering

---

## 🗂️ Session List (Enhanced)

### Before (Current)
```
┌──────────────────────────────────┐
│  Sessions                        │
├──────────────────────────────────┤
│  > Session 1                     │
│    Session 2                     │
│    Session 3                     │
└──────────────────────────────────┘
```

### After (Improved)
```
╭────────────────────────────────────────────────────────────────╮
│  📚 Sessions                                    [Ctrl+L]       │
├────────────────────────────────────────────────────────────────┤
│  🔍 Filter: █                                   [12 sessions]  │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Today                                                          │
│  ▸ 🔵 Fix login bug                             [12 msgs] 2h   │
│    🟢 Implement search feature                  [45 msgs] 4h   │
│                                                                 │
│  Yesterday                                                      │
│    🟡 Refactor database code                    [23 msgs]      │
│    🟢 Add unit tests                            [15 msgs]      │
│                                                                 │
│  This Week                                                      │
│    🟢 Setup CI/CD pipeline                      [67 msgs]      │
│    🔴 Debug memory leak                         [89 msgs]      │
│                                                                 │
│  ┌────────────────────────────────────────────────────────┐   │
│  │  Session: Fix login bug                                │   │
│  │  Created: 2h ago                                       │   │
│  │  Messages: 12 (4 user, 8 assistant)                   │   │
│  │  Tokens: 15,234 / 200,000 (7.6%)                      │   │
│  │  Model: Claude 3 Opus                                  │   │
│  │  Status: Active 🔵                                     │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                                 │
│  [Enter] Open  [d] Delete  [e] Export  [r] Rename  [/] Filter │
╰────────────────────────────────────────────────────────────────╯
```

**Features**:
- Grouped by time period
- Status indicators (🔵 active, 🟢 complete, 🟡 in progress, 🔴 error)
- Message/token counts
- Preview panel
- Quick filter
- Keyboard shortcuts for actions

---

## 💾 Confirmation Dialog

```
╭────────────────────────────────────────────────────────────╮
│  ⚠️  Delete Session?                                       │
├────────────────────────────────────────────────────────────┤
│                                                             │
│  You are about to delete:                                  │
│                                                             │
│  📄 "Fix login bug"                                        │
│     • 12 messages                                          │
│     • Created 2 hours ago                                  │
│     • Last modified 10 minutes ago                         │
│                                                             │
│  ⚠️  This action cannot be undone!                         │
│                                                             │
│  ┌────────────────┐  ┌────────────────┐                   │
│  │  🗑️  Delete    │  │  ✖️  Cancel    │                   │
│  └────────────────┘  └────────────────┘                   │
│       [Enter]              [Esc]                           │
╰────────────────────────────────────────────────────────────╯
```

**Features**:
- Clear warning icon
- Details about what will be deleted
- Visual emphasis on danger
- Clear button labels
- Keyboard shortcuts

---

## 🚨 Error Display

### Before (Simple)
```
Error: Failed to read file
```

### After (Expanded)
```
╭────────────────────────────────────────────────────────────╮
│  ❌ Error: Failed to read file                             │
├────────────────────────────────────────────────────────────┤
│                                                             │
│  File: internal/tui/nonexistent.go                        │
│  Error: no such file or directory                          │
│                                                             │
│  Stack Trace:                                              │
│  ┌──────────────────────────────────────────────────────┐ │
│  │ 1. tool.Read() at tool/read.go:45                   │ │
│  │ 2. agent.Execute() at agent/agent.go:123            │ │
│  │ 3. session.Run() at session/session.go:89           │ │
│  └──────────────────────────────────────────────────────┘ │
│                                                             │
│  Suggestions:                                              │
│  • Check if the file path is correct                      │
│  • Ensure the file exists in the working directory       │
│  • Try using an absolute path                            │
│                                                             │
│  [c] Copy Error  [r] Retry  [i] Report Issue  [Esc] Close │
╰────────────────────────────────────────────────────────────╯
```

### After (Collapsed)
```
╭────────────────────────────────────────────────────────────╮
│  ❌ Failed to read file: nonexistent.go    [e] View Details │
╰────────────────────────────────────────────────────────────╯
```

**Features**:
- Expandable/collapsible
- Stack trace
- Suggestions for resolution
- Quick actions
- Copy error for bug reports

---

## 📈 Token Usage Visualization

```
╭────────────────────────────────────────────────────────────╮
│  📊 Token Usage                                            │
├────────────────────────────────────────────────────────────┤
│                                                             │
│  Current Session: 15,234 / 200,000 tokens (7.6%)          │
│  ██░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░     │
│                                                             │
│  Breakdown:                                                │
│  • Input:  6,234 tokens (41%)  ████░░░░░░                 │
│  • Output: 9,000 tokens (59%)  ██████░░░░                 │
│                                                             │
│  Estimated Cost: $0.23                                     │
│                                                             │
│  History (last 10 messages):                               │
│  ┌──────────────────────────────────────────────────────┐ │
│  │ ▁▂▃▄▅▆█▇▆▅▄▃▂▁  ← Token usage per message           │ │
│  └──────────────────────────────────────────────────────┘ │
│                                                             │
│  💡 Tip: Use /compact to reduce token usage                │
╰────────────────────────────────────────────────────────────╯
```

---

## 🎛️ Split View (Planned)

```
╭─────────────────────────────────────┬──────────────────────────────────╮
│  Chat View                          │  Session List                    │
├─────────────────────────────────────┼──────────────────────────────────┤
│                                     │                                  │
│  ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓  │  ▸ Fix login bug        2h      │
│  ┃ You                         ┃  │    Implement search      4h      │
│  ┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫  │    Refactor database     1d      │
│  ┃ Hello!                       ┃  │    Add unit tests        1d      │
│  ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛  │                                  │
│                                     │  [Ctrl+L] Focus list             │
│  ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓  │                                  │
│  ┃ Assistant                    ┃  │  Preview:                        │
│  ┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫  │  ┌──────────────────────────┐   │
│  ┃ Hi! How can I help?          ┃  │  │ Fix login bug            │   │
│  ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛  │  │ Created: 2h ago          │   │
│                                     │  │ Messages: 12             │   │
│  [Ctrl+H] Split horizontal          │  │ Tokens: 15k              │   │
│  [Ctrl+W] Close pane                │  └──────────────────────────┘   │
╰─────────────────────────────────────┴──────────────────────────────────╯
```

**Features**:
- Side-by-side views
- Resizable panes
- Independent scrolling
- Focus indicators
- Keyboard navigation

---

## 🎯 Focus Indicators

### Input Focused
```
╭━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╮
┃  Message DCode... (Enter to send, / for commands)      ┃
┃  █                                                      ┃
╰━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━╯
    ▲
    └─ Thick purple border = focused
```

### Viewport Focused
```
┌─────────────────────────────────────────────────────────┐
│  (Messages scrollable here)                             │
│  ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓  │ ◀─ Highlighted
│  ┃ You                                              ┃  │    scrollbar
│  ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛  │
└─────────────────────────────────────────────────────────┘

╭─────────────────────────────────────────────────────────╮
│  Message DCode...                                       │
╰─────────────────────────────────────────────────────────╯
    ▲
    └─ Thin gray border = not focused
```

---

## 🎨 Color Coding Guide

### Status Colors
- 🔵 **Blue** - Active/Running
- 🟢 **Green** - Success/Complete
- 🟡 **Yellow** - Warning/In Progress
- 🔴 **Red** - Error/Failed
- ⚪ **Gray** - Inactive/Disabled

### Semantic Colors
- **Primary** (Purple) - Main actions, focus
- **Secondary** (Blue) - Secondary actions
- **Success** (Green) - Confirmations, success
- **Warning** (Yellow) - Warnings, cautions
- **Error** (Red) - Errors, destructive actions
- **Info** (Cyan) - Information, hints

### Text Colors
- **Normal** (White/Light Gray) - Main content
- **Muted** (Gray) - Secondary information
- **Dim** (Dark Gray) - Tertiary information
- **Bright** (Bright White) - Emphasis

---

## 📐 Layout Principles

### Spacing
- **Padding**: 1-2 spaces inside containers
- **Margin**: 1 line between sections
- **Alignment**: Left-aligned for text, centered for titles

### Typography
- **Headers**: Bold, Primary color
- **Body**: Normal weight, Normal color
- **Code**: Monospace font (automatic in terminal)
- **Emphasis**: Bold or Italic

### Borders
- **Rounded** (`╭─╮ │ ╰─╯`) - Main containers
- **Double** (`╔═╗ ║ ╚═╝`) - Emphasized containers
- **Light** (`┌─┐ │ └─┘`) - Secondary containers
- **Thick** (`┏━┓ ┃ ┗━┛`) - Focused elements

---

## 🎬 Animation Examples

### Loading Spinner States
```
Frame 1: ⠋  Frame 2: ⠙  Frame 3: ⠹  Frame 4: ⠸
Frame 5: ⠼  Frame 6: ⠴  Frame 7: ⠦  Frame 8: ⠧
Frame 9: ⠇  Frame 10: ⠏
```

### Progress Bar Animation
```
Step 1: ░░░░░░░░░░░░░░░░░░░░  0%
Step 2: ██░░░░░░░░░░░░░░░░░░  10%
Step 3: ████░░░░░░░░░░░░░░░░  20%
Step 4: ██████░░░░░░░░░░░░░░  30%
...
Step 10: ████████████████████  100%
```

### Fade In Effect (simulated with opacity)
```
Frame 1: ░░░░░░░░░  (10% visible)
Frame 2: ▒▒▒▒▒▒▒▒▒  (30% visible)
Frame 3: ▓▓▓▓▓▓▓▓▓  (60% visible)
Frame 4: █████████  (100% visible)
```

---

## 🔮 Future Concepts

### Session Timeline View
```
╭────────────────────────────────────────────────────────────╮
│  Timeline: Fix login bug                                   │
├────────────────────────────────────────────────────────────┤
│                                                             │
│  14:00 ─┬─ 👤 Started session                              │
│         │                                                   │
│  14:05 ─┼─ 💬 Asked about login flow                       │
│         │                                                   │
│  14:07 ─┼─ 🤖 Suggested authentication approach            │
│         │                                                   │
│  14:10 ─┼─ ⚙️  read(auth.go) → 234 lines                   │
│         │                                                   │
│  14:12 ─┼─ 💬 Requested bug fix                            │
│         │                                                   │
│  14:15 ─┼─ ⚙️  edit(auth.go) → Modified 5 lines            │
│         │                                                   │
│  14:20 ─┼─ ⚙️  bash(go test) → ✓ All tests passed          │
│         │                                                   │
│  14:25 ─┴─ ✓ Session completed                            │
│                                                             │
│  [t] Jump to time  [f] Filter events  [e] Export timeline │
╰────────────────────────────────────────────────────────────╯
```

### Cost Dashboard
```
╭────────────────────────────────────────────────────────────╮
│  💰 Cost Analysis                                          │
├────────────────────────────────────────────────────────────┤
│                                                             │
│  Today:      $2.34    ▲ +15% from yesterday               │
│  This Week:  $12.45   ▼ -5% from last week                │
│  This Month: $45.67   ▲ +20% from last month              │
│                                                             │
│  By Provider:                                              │
│  ┌────────────────────────────────────────────────────┐   │
│  │ OpenAI     ████████████░░░░░░░░  $25.00  55%      │   │
│  │ Anthropic  ██████░░░░░░░░░░░░░░  $15.00  33%      │   │
│  │ Google     ██░░░░░░░░░░░░░░░░░░  $ 5.67  12%      │   │
│  └────────────────────────────────────────────────────┘   │
│                                                             │
│  By Model:                                                 │
│  • GPT-4:        $18.50  (1.2M tokens)                    │
│  • Claude Opus:  $15.00  (800K tokens)                    │
│  • Gemini Pro:   $ 5.67  (1.5M tokens)                    │
│                                                             │
│  [d] Daily  [w] Weekly  [m] Monthly  [e] Export CSV       │
╰────────────────────────────────────────────────────────────╯
```

---

## 📝 Typography & Icons Reference

### Message Type Icons
- 👤 User message
- 🤖 Assistant message
- ⚙️  Tool call
- 💡 Tip/Hint
- ⚠️  Warning
- ❌ Error
- ✓ Success
- ⏳ Loading/Processing

### Action Icons
- 📄 File/Document
- 📁 Folder
- 🔍 Search
- ⚡ Quick action
- 🔧 Settings
- 💾 Save
- 🗑️  Delete
- ✏️  Edit
- 📋 Copy
- 📤 Export
- 📥 Import

### Status Icons
- ● Connected (filled circle)
- ○ Disconnected (empty circle)
- ◐ Connecting (half-filled)
- ▶ Play/Expand
- ⏸ Pause
- ■ Stop
- ↻ Refresh/Retry
- ✗ Close/Cancel

---

**Last Updated**: 2026-02-18  
**Purpose**: Visual reference for TUI improvements  
**Status**: Living document - will be updated as features are implemented
