# 🎯 Quick Reference Card

## Token Usage & Loading States - New Features

---

## 📊 Header Display (Always Visible)

```
 DCode   provider   model   agent   [msgs]  [tokens/max]
```

**Example**:
```
 DCode   anthropic   claude-3-opus   coder   [12 msgs]  [45.2k/200k]
                                              ︿︿︿︿︿︿   ︿︿︿︿︿︿︿︿︿︿︿
                                              Message    Token Usage
                                               Count     (Color Coded)
```

---

## 🎨 Color Coding

| Color | Usage | Action |
|-------|-------|--------|
| 🟢 Green | 0-49% | Keep going! |
| 🔵 Blue | 50-69% | Monitor usage |
| 🟡 Yellow | 70-89% | Consider compacting |
| 🔴 Red | 90-100% | Compact or new session! |

---

## ⌨️ New Shortcuts

| Key | Action |
|-----|--------|
| `?` | Help (anywhere) |
| `Ctrl+Shift+L` | Clear screen |
| `Tab` | Toggle focus |

---

## 💬 New Commands

| Command | What It Does |
|---------|--------------|
| `/tokens` | Detailed token breakdown |
| `/usage` | Same as /tokens |
| `/cost` | Quick cost summary |

---

## 🔄 Loading States

You'll now see:
- 🔌 `Connecting to anthropic`
- 💭 `Thinking`
- ⚙️  `Running read • file.go (1.2s)`
- 🤖 `Generating response`

---

## 📈 Token Details View

Type `/tokens` to see:

```
📊 Token Usage

Total: 45,234 / 200,000 tokens
Input:  20,234
Output: 25,000

████████████░░░░░░░░░░░░░░ 22.6%

Total Cost: $0.4523

Last Message:
  In:  1,234 tokens
  Out: 2,000 tokens
  Cost: $0.0456
```

---

## 💰 Quick Cost Check

Type `/cost` to see:
```
Total cost: $0.4523 (In: 20.2k, Out: 25.0k)
```

---

## 🎯 Quick Tips

1. **Watch the colors** - They're your early warning system
2. **Check costs often** - Use `/cost` to stay informed
3. **Compact when yellow** - Save context with `/compact`
4. **Clear when needed** - `Ctrl+Shift+L` for fresh start
5. **Help anytime** - Just press `?`

---

## 📱 Typical Session Flow

```
Start:    [0 msgs]  [0/200k]     🟢
First:    [2 msgs]  [3.2k/200k]  🟢
Active:   [12 msgs] [45k/200k]   🔵
Heavy:    [23 msgs] [150k/200k]  🟡  ← Consider /compact
Critical: [28 msgs] [185k/200k]  🔴  ← Action needed!
```

---

## 🆘 When Tokens Are High

**Yellow (70-89%)**:
- Run `/compact` to reduce context
- Review conversation - what's essential?
- Consider wrapping up soon

**Red (90-100%)**:
- `/compact` immediately
- Or `/new` to start fresh
- `/export` to save important parts

---

## 📝 Quick Command Reference

```bash
# Token Management
/tokens     # Full details
/cost       # Quick cost
/compact    # Reduce context

# Session Management
/new        # New session
/clear      # Clear screen
Ctrl+Shift+L # Clear + reset tokens

# Navigation
?           # Help (anywhere)
Tab         # Toggle focus
Esc         # Back/Cancel
```

---

**Print this card and keep it handy!** 📌
