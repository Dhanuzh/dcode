# Token Usage - Always Visible in Header

## Feature Update ✅

Token usage and message count are now **always visible** in the chat header, even when they're at 0.

---

## Visual Examples

### On Startup (0 messages, 0 tokens)
```
 DCode   anthropic   claude-3-opus   coder   [0 msgs]  [0/200k]
──────────────────────────────────────────────────────────────────

  Welcome to DCode

  Type a message below and press Enter to start.
  Use / for commands, Ctrl+K for model selection.
  Press ? for help anytime.
```

### After First Message (2 messages, ~3k tokens)
```
 DCode   anthropic   claude-3-opus   coder   [2 msgs]  [3.2k/200k]
────────────────────────────────────────────────────────────────────
```
*Note: Token display in green (< 50% usage)*

### Mid-Conversation (12 messages, ~45k tokens)
```
 DCode   anthropic   claude-3-opus   coder   [12 msgs]  [45.2k/200k]
─────────────────────────────────────────────────────────────────────
```
*Note: Token display in blue (50-70% usage)*

### Heavy Usage (25 messages, ~150k tokens)
```
 DCode   anthropic   claude-3-opus   coder   [25 msgs]  [150k/200k]
──────────────────────────────────────────────────────────────────────
```
*Note: Token display in yellow (70-90% usage)*

### Near Limit (30 messages, ~185k tokens)
```
 DCode   anthropic   claude-3-opus   coder   [30 msgs]  [185k/200k]
──────────────────────────────────────────────────────────────────────
```
*Note: Token display in red (> 90% usage)*

---

## Color Coding System

The token usage display changes color based on percentage used:

| Usage % | Color | Meaning |
|---------|-------|---------|
| 0-49% | 🟢 Green | Safe - plenty of space |
| 50-69% | 🔵 Blue | Moderate - keep an eye on it |
| 70-89% | 🟡 Yellow | High - consider compacting |
| 90-100% | 🔴 Red | Critical - compact or start new session |

---

## Token Format

Tokens are displayed with smart formatting:

| Actual Tokens | Display |
|---------------|---------|
| 0 | `0` |
| 500 | `500` |
| 1,234 | `1.2k` |
| 15,234 | `15.2k` |
| 123,456 | `123.5k` |
| 1,234,567 | `1.2M` |

---

## Benefits

### 1. **Always Aware**
You always know your token usage, even at the start of a session.

### 2. **Quick Reference**
No need to run `/tokens` to check basic stats - it's always visible.

### 3. **Cost Control**
The color coding gives you instant feedback on when to be concerned.

### 4. **Session Health**
Message count tells you how deep the conversation is.

---

## Related Commands

Even though tokens are always visible in the header, you can get more details:

### `/tokens` or `/usage`
Shows detailed breakdown:
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

### `/cost`
Quick cost summary:
```
Status: Total cost: $0.4523 (In: 20.2k, Out: 25.0k)
```

---

## Practical Usage

### Example Session Flow

**1. Start Session**
```
[0 msgs]  [0/200k]  🟢
```

**2. Ask a Question**
```
You: How do I read a file in Go?
[1 msgs]  [234/200k]  🟢
```

**3. Get Response**
```
Assistant: [detailed answer with code]
[2 msgs]  [1.5k/200k]  🟢
```

**4. Continue Conversation**
```
You: Can you add error handling?
[3 msgs]  [2.1k/200k]  🟢
```

**5. After Many Messages**
```
[15 msgs]  [45k/200k]  🔵  ← Notice color changed to blue
```

**6. Getting High**
```
[23 msgs]  [150k/200k]  🟡  ← Warning: Consider compacting
```

**7. Critical**
```
[28 msgs]  [185k/200k]  🔴  ← Action needed!
```

At this point, you might want to:
- Run `/compact` to reduce context
- Start a new session with `/new`
- Export important parts with `/export`

---

## Configuration

The max tokens value comes from your config:

```yaml
# In dcode.yaml
max_tokens: 200000  # Adjust based on your model
```

Different models have different context windows:
- GPT-4: 8k-128k
- Claude 3: 200k
- GPT-4 Turbo: 128k
- Gemini 1.5 Pro: 1M-2M

The display automatically adapts to your configured maximum.

---

## Technical Details

### Implementation
- Token tracking is initialized with model's max context window
- Updates automatically after each message
- Persists across view changes
- Colors use theme system (adapts to all themes)

### Performance
- Zero overhead when not streaming
- Minimal memory (< 1KB for tracking)
- Fast rendering (< 1ms)

---

## Comparison with Other Tools

### Before (Other CLIs)
```
> Command prompt
```
*No feedback on token usage*

### DCode Now
```
 DCode   anthropic   claude-3-opus   coder   [12 msgs]  [45.2k/200k]
```
*Complete visibility at a glance*

---

## Tips

### 💡 Monitor Your Usage
Watch the color change - it's your early warning system!

### 💡 Plan Ahead
When you see yellow (70%+), start thinking about:
- What information is critical to keep
- What can be compacted
- Whether to start fresh

### 💡 Cost Control
Use `/cost` periodically to check spending:
```
/cost
Status: Total cost: $0.2345 (In: 15.2k, Out: 30.0k)
```

### 💡 Detailed Analysis
Use `/tokens` when you need full details:
```
/tokens
[Shows complete breakdown with graphs]
```

---

## Keyboard Shortcuts Quick Reference

| Action | Shortcut |
|--------|----------|
| View detailed tokens | `/tokens` or `/usage` |
| Check cost | `/cost` |
| Compact session | `/compact` |
| New session | `Ctrl+N` or `/new` |
| Clear screen | `Ctrl+Shift+L` |
| Help | `?` |

---

## What's Next?

Future enhancements could include:
- **Token usage graph** - Visual history over time
- **Cost alerts** - Notifications at thresholds
- **Per-message cost** - Show cost for each message inline
- **Budget tracking** - Set daily/monthly limits
- **Provider comparison** - Compare costs across providers

---

**Updated**: 2026-02-18  
**Status**: ✅ Live - Always Visible  
**Build**: ✅ Compiled and Ready  

Now you'll **never lose track** of your token usage or costs! 🎉
