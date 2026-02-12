# DCode - AI-Powered Coding Assistant

DCode is an advanced AI coding assistant built with Go, inspired by OpenCode. It helps you write, edit, and understand code through natural language interactions in your terminal.

## Features

- **Multiple AI Providers**: Support for both Anthropic Claude and OpenAI GPT models
- **Powerful Tools**: File operations (read, write, edit), code search (glob, grep), and command execution (bash)
- **Intelligent Agent**: Autonomous tool usage with conversation loops for complex tasks
- **Simple Configuration**: YAML-based configuration with environment variable support
- **Clean CLI**: Beautiful terminal interface with colored output

## Installation

### Prerequisites

- Go 1.22 or higher
- An API key from either Anthropic or OpenAI

### Build from Source

```bash
# Clone the repository
cd dcode

# Build the binary
go build -o dcode ./cmd/dcode

# Optionally, install to your PATH
sudo mv dcode /usr/local/bin/
```

## Configuration

### Quick Setup with Login Command (Recommended)

The easiest way to get started:

```bash
dcode login
```

This will interactively:
1. Ask which AI provider you want to use
2. Prompt for your API key (securely, without showing it on screen)
3. Save it to `~/.config/dcode/credentials.json` with secure permissions

To remove stored credentials:
```bash
dcode logout
```

### API Keys

DCode supports three ways to provide API keys (in priority order):

1. **Stored Credentials** (recommended): Use `dcode login` to securely store keys
2. **Environment Variables**: Set `ANTHROPIC_API_KEY` or `OPENAI_API_KEY`
3. **Config File**: Add keys to `~/.config/dcode/dcode.yaml`

#### Option 1: Stored Credentials (Recommended)

```bash
dcode login
```

Your API keys are stored in `~/.config/dcode/credentials.json` with 0600 permissions (readable only by you).

#### Option 2: Environment Variables

Set your API key using environment variables:

```bash
# For Anthropic Claude
export ANTHROPIC_API_KEY=your_anthropic_key_here

# For OpenAI GPT
export OPENAI_API_KEY=your_openai_key_here
```

#### Option 3: Config File

### Config File

Create a configuration file at `~/.config/dcode/dcode.yaml`:

```yaml
# Default provider: anthropic or openai
provider: anthropic

# Model to use (optional, defaults based on provider)
model: claude-sonnet-4.5

# API keys (optional if set via environment variables)
anthropic_api_key: your_key_here
openai_api_key: your_key_here

# Generation parameters
max_tokens: 8192
temperature: 0.0

# Features
streaming: true
verbose: false
```

You can also create project-specific configs in your project root:
- `./dcode.yaml`
- `./.dcode/dcode.yaml`

## Usage

### First Time Setup

Configure your API keys:

```bash
dcode login
```

### Basic Usage

Run dcode in interactive mode:

```bash
dcode
```

### Authentication Commands

```bash
# Configure API keys (interactive prompt)
dcode login

# Remove stored credentials
dcode logout

# Show help
dcode --help
dcode login --help
```

### Command Line Options

```bash
# Use a specific provider
dcode --provider openai

# Use a specific model
dcode --model gpt-4-turbo

# Use a custom config file
dcode --config /path/to/config.yaml

# Enable verbose output
dcode --verbose
```

### Example Interactions

```
You: Create a new Go file that implements a simple HTTP server on port 8080

DCode: I'll create an HTTP server for you.
[Calling tool: write]
[Wrote: server.go]
Successfully wrote 245 bytes to server.go

I've created a simple HTTP server in server.go that listens on port 8080...

You: Add error handling for port binding

DCode: I'll add error handling to the server.
[Calling tool: read]
[Read: server.go]
[Calling tool: edit]
[Edited: server.go]
Successfully replaced 1 occurrence(s) in server.go

I've added proper error handling for port binding...
```

## Available Tools

DCode has access to the following tools:

### File Operations

- **read**: Read files with optional line offset/limit
  ```
  Read server.go from line 10 to 30
  ```

- **write**: Create or overwrite files
  ```
  Create a new file main.go with a hello world program
  ```

- **edit**: Perform exact string replacements
  ```
  Replace the port number with 9090 in server.go
  ```

### Code Search

- **glob**: Find files using glob patterns
  ```
  Find all Go test files
  ```

- **grep**: Search file contents with regex
  ```
  Search for all TODO comments in Go files
  ```

### Command Execution

- **bash**: Execute shell commands
  ```
  Run the tests
  Build the project
  Check git status
  ```

## Project Structure

```
dcode/
├── cmd/dcode/              # Main application entry point
│   └── main.go            # CLI implementation
├── internal/              # Private application code
│   ├── agent/            # Agent system prompt
│   ├── config/           # Configuration management
│   ├── provider/         # AI provider implementations
│   │   ├── provider.go   # Provider interface
│   │   ├── anthropic.go  # Anthropic Claude client
│   │   └── openai.go     # OpenAI GPT client
│   ├── session/          # Conversation session handling
│   │   └── session.go    # Session management
│   └── tool/             # Tool system
│       ├── tool.go       # Tool interface and registry
│       ├── read.go       # File reading tool
│       ├── write.go      # File writing tool
│       ├── edit.go       # File editing tool
│       ├── bash.go       # Command execution tool
│       ├── glob.go       # File pattern matching tool
│       └── grep.go       # Content search tool
├── go.mod                # Go module definition
└── README.md            # This file
```

## Architecture

DCode follows a modular architecture inspired by OpenCode:

1. **Provider Layer**: Abstract interface for AI providers (Anthropic, OpenAI)
2. **Tool System**: Pluggable tools that the AI can use to interact with the system
3. **Session Management**: Handles conversation state and message history
4. **Agent Loop**: Autonomous execution loop that calls tools and processes results

## Supported AI Models

### Anthropic Claude
- claude-sonnet-4.5 (default)
- claude-3-opus
- claude-3-sonnet
- claude-3-haiku

### OpenAI GPT
- gpt-4-turbo (default)
- gpt-4
- gpt-3.5-turbo

## Development

### Adding New Tools

To add a new tool, implement the `Tool` interface in `internal/tool/`:

```go
type CustomTool struct{}

func (t *CustomTool) Name() string {
    return "custom"
}

func (t *CustomTool) Description() string {
    return "Description of what the tool does"
}

func (t *CustomTool) InputSchema() map[string]interface{} {
    return tool.CreateSchema("object", "Parameters", map[string]interface{}{
        "param1": map[string]interface{}{
            "type": "string",
            "description": "Description",
        },
    }, []string{"param1"})
}

func (t *CustomTool) Execute(ctx context.Context, input map[string]interface{}) (*tool.Result, error) {
    // Implementation
    return &tool.Result{
        Title:  "Custom Tool Result",
        Output: "Tool output here",
    }, nil
}
```

Then register it in `main.go`:

```go
toolReg.Register(&CustomTool{})
```

### Adding New Providers

To add a new AI provider, implement the `Provider` interface in `internal/provider/`:

```go
type CustomProvider struct{}

func (p *CustomProvider) Name() string {
    return "custom"
}

func (p *CustomProvider) CreateMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error) {
    // Implementation
}

func (p *CustomProvider) StreamMessage(ctx context.Context, req *MessageRequest, callback func(*StreamChunk) error) error {
    // Implementation
}
```

## Differences from OpenCode

While inspired by OpenCode, DCode has some key differences:

- **Language**: Built with Go instead of TypeScript/Bun
- **Simplicity**: Focused on core features without web UI/desktop app
- **Provider Implementation**: Direct HTTP API calls instead of SDK dependencies
- **Configuration**: Simple YAML-based config vs complex layered config
- **Scope**: CLI-only (no server mode, MCP, or plugin system yet)

## Future Enhancements

Potential features for future versions:

- [ ] Server mode with REST API
- [ ] Streaming support for real-time responses
- [ ] Plugin system for custom tools
- [ ] Web UI
- [ ] Session persistence and history
- [ ] Multi-file editing support
- [ ] LSP integration
- [ ] More AI providers (Gemini, Mistral, etc.)
- [ ] Task and todo management
- [ ] Git integration tools
- [ ] Testing tools

## License

MIT License - feel free to use this project however you like!

## Contributing

Contributions are welcome! Please feel free to submit pull requests or open issues.

## Acknowledgments

- Inspired by [OpenCode](https://opencode.ai)
- Built with [Cobra](https://github.com/spf13/cobra) for CLI
- Uses [Viper](https://github.com/spf13/viper) for configuration
- Colored output by [color](https://github.com/fatih/color)

## Support

For issues, questions, or contributions, please open an issue on GitHub.

---

**Happy Coding with DCode!**
