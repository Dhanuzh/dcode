# Enhanced UI Features - Copy, Scrollbar & Real-time Token Display

## 🎉 New Features Implemented

### 1. **Mouse Support & Text Copying** 🖱️

**What's New:**
- Full mouse support enabled throughout the TUI
- Mouse wheel scrolling in messages
- Click to focus input or viewport
- **Text selection with your terminal's native selection** (Cmd+C/Ctrl+C to copy)

**How to Copy Text:**

1. **Using Mouse:**
   - Select text with your mouse (click and drag)
   - Copy with `Cmd+C` (Mac) or `Ctrl+C` (Linux/Windows)
   - Your terminal emulator handles the selection

2. **Using Keyboard** (terminal-dependent):
   - Some terminals support keyboard selection modes
   - Check your terminal's documentation

**Supported Actions:**
- 🖱️ **Mouse wheel** - Scroll messages up/down
- 🖱️ **Click** - Switch focus between input and viewport
- 🖱️ **Drag to select** - Select text to copy
- 🖱️ **Visual scrollbar** - See scroll position

---

### 2. **Visual Scrollbar** 📊

**What's New:**
A scrollbar appears on the right side of the chat viewport showing:
- Current scroll position
- Total content length indicator
- Top/bottom indicators

**Scrollbar Indicators:**
```
▲  ← Top of content
│  ← Middle sections
█  ← Current position (highlighted)
│  ← More content
▼  ← Bottom of content
```

**Footer Scroll Indicator:**
```
[SCROLL] [↕ 45%]  ← Shows scroll percentage
[SCROLL] [⬆ Top]  ← At top
[SCROLL] [⬇ Bottom] ← At bottom
```

---

### 3. **Enhanced Real-time Token Display** ⚡

**What's New:**
The header now shows **much more detailed** token information in real-time:

**New Format:**
```
[● tokens/max percentage% $cost]
```

**Examples:**

**Start (0 tokens):**
```
[● 0/200k]
```

**After first message (with cost):**
```
[● 3.2k/200k 1.6% $0.045]
  ↑    ↑      ↑     ↑
  │    │      │     └─ Real-time cost
  │    │      └─────── Percentage used
  │    └──────────────── Total/max tokens
  └───────────────────── Status indicator (color-coded)
```

**Mid-conversation:**
```
[● 45.2k/200k 22.6% $0.523]
```

**Heavy usage:**
```
[● 150k/200k 75.0% $1.234]
```

**Color-Coded Indicator:**
- 🟢 **Green ●** (0-49%) - Safe
- 🔵 **Blue ●** (50-69%) - Moderate
- 🟡 **Yellow ●** (70-89%) - High
- 🔴 **Red ●** (90-100%) - Critical

---

## 📊 Visual Examples

### Full Header View

**Low Usage:**
```
 DCode   anthropic   claude-3-opus   coder   [5 msgs]  [● 3.2k/200k 1.6% $0.045]
                                               ︿︿︿︿︿   ︿︿︿︿︿︿︿︿︿︿︿︿︿︿︿︿︿︿︿︿︿︿︿︿
                                                Count    Real-time detailed stats
```

**Moderate Usage:**
```
 DCode   anthropic   claude-3-opus   coder   [12 msgs]  [● 45.2k/200k 22.6% $0.523]
                                                         ↑ Blue indicator
```

**High Usage:**
```
 DCode   anthropic   claude-3-opus   coder   [23 msgs]  [● 150k/200k 75.0% $1.234]
                                                         ↑ Yellow warning
```

**Critical:**
```
 DCode   anthropic   claude-3-opus   coder   [28 msgs]  [● 185k/200k 92.5% $2.345]
                                                         ↑ Red alert!
```

### Chat View with Scrollbar

```
 DCode   anthropic   claude-3-opus   [12 msgs]  [● 45k/200k 22.6% $0.52]
──────────────────────────────────────────────────────────────────────────

┃ You: How do I read a file?                                          ▲
┃                                                                      │
┃ Assistant: Here's how...                                            │
┃ [code block]                                                         │
┃                                                                      │
┃ You: Can you add error handling?                                    █ ← Current position
┃                                                                      │
┃ Assistant: Sure, here's the updated version...                      │
┃ [code block with error handling]                                    │
┃                                                                      │
┃ You: Thanks!                                                         ▼

⚙️  Running write • file.go

──────────────────────────────────────────────────────────────────────────
Message DCode...
──────────────────────────────────────────────────────────────────────────
[SCROLL] [↕ 45%] Tab focus  Enter send  Ctrl+K model  / commands
         ︿︿︿︿︿︿
         Scroll position indicator
```

---

## 🖱️ Mouse Interactions

### Scrolling
- **Mouse Wheel Up** - Scroll messages up
- **Mouse Wheel Down** - Scroll messages down
- Smooth scrolling in viewport

### Focus Management
- **Click in viewport area** - Focus viewport (for scrolling)
- **Click in input area** - Focus input (for typing)
- Visual indicator shows current focus

### Text Selection
- **Click and drag** - Select text
- **Cmd+C/Ctrl+C** - Copy selected text
- Works with your terminal's native selection

---

## ⌨️ Keyboard Shortcuts (Updated)

| Key | Action |
|-----|--------|
| `Tab` | Toggle focus (input ↔ viewport) |
| `Up/Down` | Scroll when viewport focused |
| `PgUp/PgDn` | Page up/down when viewport focused |
| `Home/End` | Jump to top/bottom |
| `?` | Help (anywhere) |

---

## 💡 Pro Tips

### Copying Text
1. **Select text** with your mouse
2. **Right-click** → Copy (or Cmd/Ctrl+C)
3. **Paste** anywhere with Cmd/Ctrl+V

### Monitoring Costs
Watch the header for real-time updates:
- `$0.045` - Under 5 cents, very cheap
- `$0.523` - About 50 cents, moderate
- `$1.234` - Over a dollar, significant
- `$2.345` - Multiple dollars, heavy usage

### Using the Scrollbar
- **Visual feedback** - See where you are in the conversation
- **Quick jump** - Click to approximate position (terminal-dependent)
- **Percentage indicator** - Footer shows exact position

### Best Practices
1. **Keep an eye on the ●** - Color tells you everything
2. **Monitor percentage** - Compact at 70%+
3. **Watch the cost** - Real-time spending awareness
4. **Use scrollbar** - Quick visual reference for long chats

---

## 🎨 Terminal Compatibility

### Best Experience:
- **iTerm2** (Mac) - Full mouse support, smooth scrolling
- **Windows Terminal** - Great mouse support
- **Alacritty** - Fast, good mouse support
- **Kitty** - Excellent mouse and text selection

### Good Experience:
- **GNOME Terminal** - Basic mouse support
- **Konsole** - Good compatibility
- **Terminator** - Works well

### Limited:
- **tmux/screen** - Enable mouse mode (`set -g mouse on`)
- **SSH sessions** - Depends on local terminal

---

## 🔧 Troubleshooting

### Can't Copy Text?
**Solution:** Your terminal needs to support mouse text selection:
- iTerm2: Just select with mouse
- Windows Terminal: Shift+Click to select
- tmux: Hold Shift while selecting

### Scrollbar Not Showing?
**Solution:** Scrollbar only appears when content exceeds viewport height:
- Send more messages to see it
- Resize terminal to make viewport smaller

### Mouse Not Working?
**Solution:** Check terminal emulator settings:
- Enable "Mouse reporting" or "Application cursor keys"
- Some terminals require explicit mouse mode

### Cost Not Showing?
**Solution:** Cost shows after first message with token usage:
- Send at least one message
- Provider must return token counts
- Check `/tokens` for more details

---

## 📊 Comparison

### Before:
```
 DCode   anthropic   claude-3-opus   coder  
───────────────────────────────────────────

[messages - no scrollbar]
(no way to copy text easily)
(no real-time cost info)
```

### After:
```
 DCode   anthropic   claude-3-opus   [12 msgs]  [● 45k/200k 22.6% $0.52]
─────────────────────────────────────────────────────────────────────────

[messages]                                                            ▲
...                                                                   │
...                                                                   █
...                                                                   │
...                                                                   ▼

[SCROLL] [↕ 45%] Tab focus  Enter send  ...
```

**Improvements:**
- ✅ Click and copy any text
- ✅ Visual scrollbar for navigation
- ✅ Real-time cost and percentage
- ✅ Mouse wheel scrolling
- ✅ Scroll position indicator

---

## 🎯 Quick Reference

### Header Info:
```
[● tokens/max percentage% $cost]
 ↑    ↑       ↑          ↑
 │    │       │          └─ Cost in dollars
 │    │       └──────────── Usage percentage
 │    └──────────────────── Current/max tokens
 └───────────────────────── Color-coded status
```

### Scrollbar:
```
▲ - Top
│ - Content
█ - Current position (highlighted)
│ - More content
▼ - Bottom
```

### Footer:
```
[SCROLL] [↕ 45%] - Scroll position
[SCROLL] [⬆ Top] - At top
[SCROLL] [⬇ Bottom] - At bottom
```

---

## 🚀 What's Next?

Future enhancements could include:
- **Clickable scrollbar** - Jump to position
- **Copy button** on code blocks
- **Export selection** - Save selected text
- **Search in messages** - Find specific text
- **Minimap** - Thumbnail view of conversation

---

**Updated**: 2026-02-18  
**Status**: ✅ Live and Tested  
**Build**: ✅ Compiled Successfully  

Now you can **easily copy text**, **see detailed token usage in real-time**, and **navigate with a visual scrollbar**! 🎉
