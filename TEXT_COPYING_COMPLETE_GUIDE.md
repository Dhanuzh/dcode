# 📋 Text Copying - Complete Solution Guide

## ❗ Terminal Limitation Explained

When DCode runs with mouse support, it captures mouse events for scrolling and navigation. This is **standard behavior for all TUI applications** (htop, vim, tmux, etc.).

**The Trade-off:**
- ✅ Mouse wheel scrolling works
- ✅ Click to focus works  
- ✅ Visual scrollbar works
- ❌ Normal mouse text selection is blocked

This is a **fundamental terminal limitation**, not a bug.

---

## ✅ Two Solutions Implemented

### Solution 1: **Ctrl+Y - Quick Copy** ⭐ RECOMMENDED

**Best for:** Copying AI responses quickly

**How it works:**
1. Get an AI response
2. Press `Ctrl+Y`
3. Last assistant message → clipboard
4. Paste anywhere with `Cmd+V` or `Ctrl+V`

**Example:**
```
You: Write a function to sort an array