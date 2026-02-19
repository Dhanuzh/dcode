# How to Copy Text from DCode TUI

## The Problem
Terminal apps with mouse support capture mouse events, preventing normal text selection.
This is a limitation of all TUI apps (vim, htop, tmux, etc.), not a bug.

## ✅ Solution 1: Ctrl+Y (Quick Copy) - RECOMMENDED

**Press Ctrl+Y to copy the last assistant message**

Example:
- AI responds with code
- Press Ctrl+Y
- Status shows: "Copied 1,234 characters to clipboard"
- Paste anywhere (Cmd+V or Ctrl+V)

## ✅ Solution 2: Shift+Mouse (Terminal Selection)

**Hold Shift while selecting text with mouse**

This bypasses the app and uses terminal's native selection.

Works in:
- iTerm2 (Mac) - Shift+Click and drag
- Windows Terminal - Shift+Click and drag
- Alacritty - Shift+Click and drag
- Kitty - Shift+Click and drag
- GNOME Terminal - Shift+Click and drag

## Quick Reference

| Method | How | Best For |
|--------|-----|----------|
| Ctrl+Y | Press shortcut | Quick copy of AI response |
| Shift+Mouse | Hold Shift, select | Copy any text, any amount |

## Testing It

1. Run dcode
2. Get an AI response
3. Try both methods:
   - Press Ctrl+Y → Should copy to clipboard
   - Shift+Click and drag → Should select text

## Troubleshooting

**Ctrl+Y not working?**
- Make sure an assistant message exists
- Check status bar for error message

**Shift+Mouse not working?**
- Try your terminal's documentation
- Some terminals use different modifiers
- Try: Shift+Click, Option+Click, Alt+Click

**Still can't copy?**
- Use /tokens command to see full output
- Take a screenshot
- Use terminal's buffer search (Cmd+F in iTerm2)

