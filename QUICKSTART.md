# Quick Start Guide for DCode

This guide will help you get DCode up and running in 5 minutes.

## Step 1: Prerequisites

Make sure you have:
- Go 1.22 or higher installed
- An API key from either Anthropic or OpenAI

## Step 2: Build DCode

```bash
cd dcode
make build
```

Or manually:
```bash
go build -o dcode ./cmd/dcode
```

## Step 3: Set Up Your API Key

### Option A: Use the Login Command (Recommended)

```bash
./dcode login
```

This will:
- Prompt you to choose a provider (Anthropic or OpenAI)
- Securely ask for your API key (hidden input)
- Store it encrypted in `~/.config/dcode/credentials.json`

Example:
```
$ ./dcode login
DCode Login - Configure your API keys

Which AI provider would you like to use? (anthropic/openai/both): anthropic

Enter your Anthropic API key (from https://console.anthropic.com/)
Anthropic API Key: ********
✓ Anthropic API key saved

✓ Credentials saved to: ~/.config/dcode/credentials.json

You can now run 'dcode' to start using the assistant!
```

### Option B: Environment Variable

For Anthropic Claude:
```bash
export ANTHROPIC_API_KEY=sk-ant-your-key-here
```

For OpenAI GPT:
```bash
export OPENAI_API_KEY=sk-your-key-here
```

Add this to your `~/.bashrc` or `~/.zshrc` to make it permanent.

### Option C: Configuration File

Create `~/.config/dcode/dcode.yaml`:

```bash
mkdir -p ~/.config/dcode
cat > ~/.config/dcode/dcode.yaml << 'EOF'
provider: anthropic
anthropic_api_key: sk-ant-your-key-here
EOF
```

## Step 4: Run DCode

```bash
./dcode
```

You should see:
```
╔════════════════════════════════════════════╗
║          Welcome to DCode v1.0             ║
╚════════════════════════════════════════════╝

Provider: anthropic
Model: claude-sonnet-4.5
Working Directory: /your/current/directory

Type your message and press Enter. Type 'exit' to quit.

You: 
```

## Step 5: Try It Out!

### Example 1: Create a New File

```
You: Create a hello.go file that prints "Hello, DCode!"

DCode: I'll create that file for you.
[Calling tool: write]
[Wrote: hello.go]
Successfully wrote 123 bytes to hello.go

I've created hello.go with a simple program that prints "Hello, DCode!"
```

### Example 2: Search for Code

```
You: Find all Go files in this project

DCode: [Calling tool: glob]
Found 15 file(s) matching pattern '**/*.go':

cmd/dcode/main.go
internal/agent/agent.go
internal/config/config.go
...
```

### Example 3: Execute Commands

```
You: Run go version to check my Go installation

DCode: [Calling tool: bash]
[$ go version]
go version go1.22.0 linux/amd64
```

### Example 4: Edit Files

```
You: Change the greeting in hello.go to "Hello, World!"

DCode: [Calling tool: read]
[Read: hello.go]
[Calling tool: edit]
[Edited: hello.go]
Successfully replaced 1 occurrence(s) in hello.go

I've updated the greeting to "Hello, World!"
```

## Common Commands

### Login and Logout

```bash
# Configure your API keys (recommended)
./dcode login

# Remove stored credentials
./dcode logout

# Check stored credentials location
ls -la ~/.config/dcode/credentials.json
```

### Using Different Providers

```bash
# Use OpenAI instead of Anthropic
./dcode --provider openai

# Use a specific model
./dcode --model gpt-4-turbo
```

### Getting Help

```bash
./dcode --help
```

### Installing Globally

```bash
make install
# Or manually:
sudo mv dcode /usr/local/bin/
```

Then you can use `dcode` from anywhere!

## Tips for Best Results

1. **Be Specific**: Clear instructions get better results
   - Good: "Create a HTTP server in Go that serves on port 8080"
   - Less Good: "Make a server"

2. **Let DCode Use Tools**: It will automatically read files before editing them

3. **Chain Requests**: You can ask for multiple things
   - "Create main.go, then create tests for it, then run the tests"

4. **Check Git Status**: Ask DCode to check git status before making changes

5. **Review Changes**: Always review code that DCode writes

## Troubleshooting

### "Error: No API key found"

Make sure your API key is set either:
- In environment variable (`ANTHROPIC_API_KEY` or `OPENAI_API_KEY`)
- In config file (`~/.config/dcode/dcode.yaml`)

### "API error (401)"

Your API key is invalid. Double-check it's copied correctly.

### "API error (429)"

You've hit rate limits. Wait a moment and try again.

### Build Errors

Make sure you have all dependencies:
```bash
make deps
```

## Next Steps

- Read the full [README.md](README.md) for more details
- Check out the example config: `dcode.yaml.example`
- Explore the project structure in `internal/`
- Add custom tools by editing `internal/tool/`

## Getting Help

- Open an issue on GitHub
- Check the README for detailed documentation
- Review the example configuration file

---

Happy coding with DCode!
