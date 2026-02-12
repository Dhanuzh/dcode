# DCode Authentication Guide

## Overview

DCode supports multiple ways to authenticate with AI providers. This guide explains all available options.

## Quick Start (Recommended)

The easiest way to get started is with the `login` command:

```bash
dcode login
```

This interactive command will:
1. Ask which provider you want to use (Anthropic, OpenAI, or both)
2. Securely prompt for your API key (input is hidden)
3. Store it in `~/.config/dcode/credentials.json` with secure permissions

## Authentication Methods

DCode checks for API keys in the following order:

1. **Environment Variables** (highest priority)
2. **Stored Credentials** (via `dcode login`)
3. **Config File** (lowest priority)

### Method 1: Stored Credentials (Recommended)

**Pros:**
- Secure storage with file permissions (0600)
- Easy to set up with `dcode login`
- Works across all terminal sessions
- Easy to revoke with `dcode logout`

**Setup:**
```bash
dcode login
```

**Example Session:**
```
$ dcode login
DCode Login - Configure your API keys

Which AI provider would you like to use? (anthropic/openai/both): anthropic

Enter your Anthropic API key (from https://console.anthropic.com/)
Anthropic API Key: ********
✓ Anthropic API key saved

✓ Credentials saved to: /home/user/.config/dcode/credentials.json

You can now run 'dcode' to start using the assistant!
```

**Location:**
- Linux/Mac: `~/.config/dcode/credentials.json`
- Permissions: `0600` (read/write by owner only)

**Logout:**
```bash
dcode logout
```

### Method 2: Environment Variables

**Pros:**
- Highest priority (overrides other methods)
- Good for CI/CD pipelines
- No files on disk
- Per-session or permanent

**Setup:**

For Anthropic Claude:
```bash
export ANTHROPIC_API_KEY=sk-ant-your-key-here
```

For OpenAI GPT:
```bash
export OPENAI_API_KEY=sk-your-key-here
```

**Make it Permanent:**

Add to your shell config file (`~/.bashrc`, `~/.zshrc`, etc.):

```bash
# Add to ~/.bashrc or ~/.zshrc
export ANTHROPIC_API_KEY=sk-ant-your-key-here
export OPENAI_API_KEY=sk-your-key-here
```

Then reload:
```bash
source ~/.bashrc  # or ~/.zshrc
```

**Verify:**
```bash
echo $ANTHROPIC_API_KEY
```

### Method 3: Config File

**Pros:**
- Simple YAML format
- Can include other settings
- Per-project or global configs
- Version controllable (without sensitive data)

**Setup:**

Create `~/.config/dcode/dcode.yaml`:

```yaml
provider: anthropic
anthropic_api_key: sk-ant-your-key-here
openai_api_key: sk-your-openai-key-here

# Other settings
model: claude-sonnet-4.5
max_tokens: 8192
temperature: 0.0
streaming: true
verbose: false
```

**Security Warning:** 
- Never commit API keys to version control
- Use `.gitignore` to exclude config files
- Consider using environment variables or stored credentials instead

**Project-Specific Config:**

You can also create a config in your project root:

```bash
# Option 1: In project root
cat > dcode.yaml << EOF
provider: openai
model: gpt-4-turbo
EOF

# Option 2: In .dcode directory
mkdir -p .dcode
cat > .dcode/dcode.yaml << EOF
provider: anthropic
model: claude-3-opus
EOF
```

**Note:** Project configs inherit from global config but don't typically store API keys.

## Getting API Keys

### Anthropic Claude

1. Go to https://console.anthropic.com/
2. Sign up or log in
3. Navigate to API Keys
4. Create a new API key
5. Copy the key (starts with `sk-ant-`)

**Models Available:**
- `claude-sonnet-4.5` (recommended)
- `claude-3-opus` (most capable)
- `claude-3-sonnet` (balanced)
- `claude-3-haiku` (fast)

### OpenAI GPT

1. Go to https://platform.openai.com/
2. Sign up or log in
3. Navigate to API Keys
4. Create a new API key
5. Copy the key (starts with `sk-`)

**Models Available:**
- `gpt-4-turbo` (recommended)
- `gpt-4` (standard)
- `gpt-3.5-turbo` (fast)

## Troubleshooting

### "No API key found"

If you see:
```
Error: No API key found for provider 'anthropic'
```

**Solutions:**
1. Run `dcode login` to store credentials
2. Set environment variable: `export ANTHROPIC_API_KEY=your_key`
3. Create config file with your API key

### "API error (401): Unauthorized"

Your API key is invalid or expired.

**Solutions:**
1. Verify your API key is correct
2. Check if the key has been revoked in the provider's console
3. Generate a new key and update with `dcode login`

### "API error (429): Rate limit exceeded"

You've hit the provider's rate limits.

**Solutions:**
1. Wait a few minutes before trying again
2. Check your API usage in the provider's dashboard
3. Consider upgrading your API plan

### Multiple API Keys

If you have keys for both providers:

```bash
# Use dcode login with "both"
dcode login
# Choose: both

# Or set both environment variables
export ANTHROPIC_API_KEY=sk-ant-xxx
export OPENAI_API_KEY=sk-xxx

# Then switch providers as needed
dcode --provider anthropic
dcode --provider openai
```

### Checking Current Authentication

```bash
# Check if credentials file exists
ls -la ~/.config/dcode/credentials.json

# Check environment variables
echo $ANTHROPIC_API_KEY
echo $OPENAI_API_KEY

# Check config file
cat ~/.config/dcode/dcode.yaml
```

### Removing Credentials

```bash
# Remove stored credentials
dcode logout

# Or manually
rm ~/.config/dcode/credentials.json

# Unset environment variables
unset ANTHROPIC_API_KEY
unset OPENAI_API_KEY

# Remove from shell config
# Edit ~/.bashrc or ~/.zshrc and remove export lines
```

## Security Best Practices

1. **Never commit API keys to Git:**
   ```bash
   # Add to .gitignore
   echo "dcode.yaml" >> .gitignore
   echo ".dcode/" >> .gitignore
   ```

2. **Use stored credentials for personal machines:**
   ```bash
   dcode login
   ```

3. **Use environment variables for servers/CI:**
   ```bash
   export ANTHROPIC_API_KEY=sk-ant-xxx
   ```

4. **Rotate keys regularly:**
   - Generate new keys every few months
   - Revoke old keys in provider console
   - Update with `dcode login`

5. **Check file permissions:**
   ```bash
   ls -la ~/.config/dcode/credentials.json
   # Should show: -rw------- (600)
   ```

6. **Use project configs without keys:**
   ```yaml
   # dcode.yaml (safe to commit)
   provider: anthropic
   model: claude-sonnet-4.5
   max_tokens: 8192
   # NO API KEYS HERE!
   ```

## Advanced Configuration

### Multiple Profiles

You can use different configs for different projects:

```bash
# Project A - uses Anthropic
cd project-a
cat > .dcode/dcode.yaml << EOF
provider: anthropic
model: claude-sonnet-4.5
EOF

# Project B - uses OpenAI
cd ../project-b
cat > .dcode/dcode.yaml << EOF
provider: openai
model: gpt-4-turbo
EOF
```

API keys are still managed globally via `dcode login` or environment variables.

### Override Provider Per Command

```bash
# Use Anthropic (default)
dcode

# Force OpenAI for this session
dcode --provider openai

# Force specific model
dcode --provider anthropic --model claude-3-opus
```

## Summary

**Recommended Setup:**
```bash
# One-time setup
dcode login

# Start using
dcode
```

**Priority Order:**
1. Environment variables (checked first)
2. Stored credentials (via `dcode login`)
3. Config file (checked last)

**Common Commands:**
```bash
dcode login          # Configure API keys
dcode logout         # Remove stored keys
dcode --help         # Show help
dcode --provider X   # Use specific provider
```

For more help, see:
- [README.md](README.md) - Full documentation
- [QUICKSTART.md](QUICKSTART.md) - Getting started guide
