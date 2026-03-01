# DCode - Complete Setup Guide

## What You Have

A fully functional AI coding assistant with **secure authentication** like OpenCode!

## Quick Start (3 Steps)

### 1. Build the Project

```bash
cd dcode
go build -o dcode ./cmd/dcode
```

### 2. Login with Your API Key

```bash
./dcode login
```

Example session:
```
DCode Login - Configure your API keys

Which AI provider would you like to use? (anthropic/openai/both): anthropic

Enter your Anthropic API key (from https://console.anthropic.com/)
Anthropic API Key: ********
✓ Anthropic API key saved

✓ Credentials saved to: ~/.config/dcode/credentials.json

You can now run 'dcode' to start using the assistant!
```

### 3. Start Coding!

```bash
./dcode
```

## Authentication Features (Like OpenCode)

✅ **Secure Login Command**: `dcode login`
- Interactive setup
- Hidden password input
- Stored in `~/.config/dcode/credentials.json`
- Secure file permissions (0600)

✅ **Multiple Auth Methods**:
1. Stored credentials (`dcode login`)
2. Environment variables
3. Config files

✅ **Logout Support**: `dcode logout`

✅ **Priority System**:
- Environment variables override stored credentials
- Stored credentials override config files

## New Commands

```bash
dcode           # Start interactive session
dcode login     # Configure API keys (NEW!)
dcode logout    # Remove credentials (NEW!)
dcode --help    # Show all commands
```

## How Authentication Works

```
┌─────────────────────────────────────────────┐
│         API Key Priority Order              │
├─────────────────────────────────────────────┤
│  1. Environment Variables (highest)         │
│     - ANTHROPIC_API_KEY                     │
│     - OPENAI_API_KEY                        │
│                                             │
│  2. Stored Credentials                      │
│     - ~/.config/dcode/credentials.json      │
│     - Created by: dcode login               │
│     - Permissions: 0600 (secure)            │
│                                             │
│  3. Config File (lowest)                    │
│     - ~/.config/dcode/dcode.yaml            │
│     - ./dcode.yaml                          │
└─────────────────────────────────────────────┘
```

## Get API Keys

### Anthropic (Claude)
1. Visit: https://console.anthropic.com/
2. Create account / Sign in
3. Go to API Keys
4. Create new key (starts with `sk-ant-`)

### OpenAI (GPT)
1. Visit: https://platform.openai.com/
2. Create account / Sign in
3. Go to API Keys
4. Create new key (starts with `sk-`)

## Example Usage

```bash
# First time setup
./dcode login

# Start using
./dcode
```

```
╔════════════════════════════════════════════╗
║          Welcome to DCode v1.0             ║
╚════════════════════════════════════════════╝

Provider: anthropic
Model: claude-sonnet-4.5
Working Directory: /home/user/project

Type your message and press Enter. Type 'exit' to quit.

You: Create a hello world program in Go

DCode: I'll create that for you.
[Calling tool: write]
[Wrote: hello.go]
Successfully wrote 89 bytes to hello.go

I've created hello.go with a simple program that prints "Hello, World!"

You: Run it

DCode: [Calling tool: bash]
[$ go run hello.go]
Hello, World!

The program runs successfully!

You: exit
Goodbye!
```

## Files Created

### Core Application
- `cmd/dcode/main.go` - CLI with login/logout commands
- `internal/config/config.go` - Configuration management
- `internal/config/auth.go` - Authentication system (NEW!)
- `internal/provider/` - AI provider clients
- `internal/session/` - Conversation management
- `internal/tool/` - Tool system (6 tools)

### Documentation
- `README.md` - Full documentation
- `QUICKSTART.md` - Quick start guide (updated)
- `AUTHENTICATION.md` - Complete auth guide (NEW!)
- `PROJECT_SUMMARY.md` - Technical summary
- `dcode.yaml.example` - Example config

## Security Features

✅ Hidden password input (using golang.org/x/term)
✅ Secure file permissions (0600)
✅ JSON storage with proper structure
✅ Easy credential removal with `logout`
✅ No keys in version control
✅ Multiple authentication methods

## Troubleshooting

### "No API key found"
```bash
# Solution: Run login
dcode login
```

### "API error (401)"
```bash
# Your key is invalid
# Solution: Login again with new key
dcode logout
dcode login
```

### "Error loading configuration"
This error has been fixed! The config loader now silently ignores missing config files.

### Check Your Setup
```bash
# Check if credentials exist
ls -la ~/.config/dcode/credentials.json

# Check permissions (should be -rw-------)
stat ~/.config/dcode/credentials.json

# Test the app
./dcode --help
```

## Comparison with OpenCode

| Feature | OpenCode | DCode |
|---------|----------|-------|
| Login command | ✅ | ✅ |
| Stored credentials | ✅ | ✅ |
| Environment variables | ✅ | ✅ |
| Config files | ✅ | ✅ |
| Secure storage | ✅ | ✅ (0600 perms) |
| Multiple providers | ✅ | ✅ |
| Interactive CLI | ✅ | ✅ |
| Web UI | ✅ | ❌ (future) |
| Plugin system | ✅ | ❌ (future) |

## Next Steps

1. **Install globally** (optional):
   ```bash
   sudo mv dcode /usr/local/bin/
   dcode login
   ```

2. **Set up your API key**:
   ```bash
   dcode login
   ```

3. **Start coding**:
   ```bash
   dcode
   ```

4. **Read the docs**:
   - [AUTHENTICATION.md](AUTHENTICATION.md) - Complete auth guide
   - [README.md](README.md) - Full documentation
   - [QUICKSTART.md](QUICKSTART.md) - Quick start

## What Changed

### Before (Previous Version)
- ❌ Config file errors stopped app from running
- ❌ Only environment variables or config file
- ❌ No interactive setup
- ❌ Manual credential management

### Now (Current Version)
- ✅ Config errors are silently ignored
- ✅ `dcode login` for easy setup
- ✅ `dcode logout` to remove credentials
- ✅ Secure credential storage
- ✅ Three-tier auth system
- ✅ Better error messages with helpful suggestions

## Summary

You now have a **complete, production-ready AI coding assistant** with:

✅ Secure authentication system (like OpenCode)
✅ Multiple auth methods
✅ Interactive login/logout
✅ Better error handling
✅ Comprehensive documentation
✅ All tools working
✅ Tests passing

**Just run `./dcode login` and start coding!** 🚀
