# dcode

[![CI](https://github.com/Dhanuzh/dcode/actions/workflows/ci.yml/badge.svg)](https://github.com/Dhanuzh/dcode/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/go-1.24.2-blue)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/Dhanuzh/dcode)](https://github.com/Dhanuzh/dcode/releases)

**dcode** is an AI-powered terminal coding agent built in Go with a full Bubbletea TUI. It supports 20+ AI providers, autonomous tool use, multi-step undo/redo, live thinking display, and a rich @ file-picker — all without leaving your terminal.

---

## Features

- **20+ AI providers** — Anthropic, OpenAI, Google Gemini, GitHub Copilot, Azure OpenAI, AWS Bedrock, Groq, OpenRouter, xAI, DeepSeek, Mistral, DeepInfra, Cerebras, Together AI, Cohere, Perplexity, GitLab, Cloudflare, Replicate, and any OpenAI-compatible endpoint
- **Autonomous tool use** — reads, writes, edits files; searches with glob/grep; executes shell commands; browses the web; calls MCP servers
- **Rich TUI** — Bubbletea interface with Glamour-rendered markdown, Chroma syntax highlighting, scrollable history, and a light-green blinking cursor
- **Thinking display** — live streaming of extended-thinking / reasoning with opencode-style spinner and topic extraction
- **Multi-step undo/redo** — `Ctrl+Z` / `Ctrl+Shift+Z` backed by git snapshots; also `/undo` and `/redo` slash commands
- **Copy code blocks** — `Ctrl+Y` + digit copies the Nth code block from the last response; `/copy N` also works
- **@ file picker** — type `@` to attach any file (text, images, or directories); text files are injected as fenced code blocks, images as inline data
- **Session persistence** — all conversations saved as JSON with import/export and share
- **MCP support** — connects to local and remote Model Context Protocol servers
- **Git worktrees** — `dcode worktree create <name>` for isolated parallel sessions on separate branches
- **Non-interactive mode** — `dcode run "prompt"` for scripting and CI/CD pipelines
- **Shell completions** — bash, zsh, fish, and PowerShell via `dcode completion`

---

## Installation

### From a release (recommended)

Download the binary for your platform from the [Releases](https://github.com/Dhanuzh/dcode/releases) page and put it on your `PATH`:

```bash
# Linux (x86_64)
curl -L https://github.com/Dhanuzh/dcode/releases/latest/download/dcode_linux_x86_64.tar.gz | tar xz
sudo mv dcode /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/Dhanuzh/dcode/releases/latest/download/dcode_darwin_arm64.tar.gz | tar xz
sudo mv dcode /usr/local/bin/
```

### Via `go install`

```bash
go install github.com/Dhanuzh/dcode/cmd/dcode@latest
```

### Build from source

```bash
git clone https://github.com/Dhanuzh/dcode.git
cd dcode
make build        # builds ./dcode with version/commit ldflags
make install      # copies to /usr/local/bin (requires sudo)
```

---

## Quick Start

```bash
# First run — dcode guides you through authentication
dcode

# Or authenticate explicitly
dcode auth login

# Non-interactive one-shot prompt
dcode run "explain the main function in cmd/dcode/main.go"

# Use a specific provider and model
dcode --provider openai --model gpt-4o
```

---

## Authentication

dcode supports three credential sources, in priority order:

1. **Stored credentials** (recommended) — `dcode auth login` saves keys to `~/.config/dcode/credentials.json` with `0600` permissions
2. **Environment variables** — e.g. `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GOOGLE_API_KEY`
3. **Config file** — `~/.config/dcode/dcode.json` or project-local `.dcode/dcode.json`

Provider-specific OAuth flows:

```bash
dcode auth anthropic   # Anthropic OAuth PKCE flow (no API key needed)
dcode auth copilot     # GitHub Copilot device-code OAuth
dcode auth openai      # API key prompt
```

List configured providers:

```bash
dcode auth list
```

---

## Configuration

Global config lives at `~/.config/dcode/dcode.json`. Override per-project with `.dcode/dcode.json` in your repo root.

```bash
# View current resolved config
dcode config show

# Change settings
dcode config set provider anthropic
dcode config set model claude-sonnet-4-5
dcode config set theme dracula
```

| Key | Default | Description |
|---|---|---|
| `provider` | `anthropic` | Default AI provider |
| `model` | *(provider default)* | Model ID |
| `default_agent` | `coder` | Agent to use (`coder`, `planner`, `explorer`) |
| `max_tokens` | `8192` | Max output tokens |
| `streaming` | `true` | Stream responses |
| `theme` | `dark` | TUI colour theme |
| `snapshot` | `true` | Enable git snapshots for undo |

---

## TUI Key Bindings

| Key | Action |
|---|---|
| `Enter` | Send message |
| `Ctrl+C` | Cancel running request |
| `Ctrl+Z` | Undo last AI action |
| `Ctrl+Shift+Z` | Redo |
| `Ctrl+Y` + digit | Copy Nth code block |
| `@` | Open file picker (attach file or image) |
| `Ctrl+L` | Clear screen |
| `Ctrl+M` | Toggle mouse mode |
| `Tab` | Cycle through sessions / autocomplete |
| `↑` / `↓` | Scroll history |
| `/help` | List slash commands |
| `/undo`, `/redo` | Undo / redo |
| `/copy N` | Copy Nth code block |
| `/clear` | Clear conversation |
| `/model` | Switch model interactively |
| `/agent` | Switch agent |

---

## Available Commands

```
dcode                    # Start the TUI (default)
dcode run <prompt>       # Non-interactive single prompt
dcode auth               # Manage credentials
dcode config             # View/edit configuration
dcode models             # List all models by provider
dcode agent list         # List available agents
dcode session list       # List saved sessions
dcode session delete <id>
dcode export <session-id>
dcode import <file>
dcode share create <session-id>
dcode mcp list           # List MCP servers
dcode mcp add            # Add an MCP server
dcode worktree create <name>
dcode stats              # Usage statistics
dcode debug config       # Dump resolved config JSON
dcode version            # Show version and commit
dcode upgrade            # Self-update via go install
dcode completion bash    # Shell completion script
```

---

## Project Structure

```
dcode/
├── cmd/dcode/              # CLI entry point (Cobra commands)
├── internal/
│   ├── agent/              # Agent definitions and system prompts
│   ├── config/             # Config loading, credentials, provider registry
│   ├── earlyinit/          # Lipgloss dark-background init (WSL2 fix)
│   ├── provider/           # AI provider implementations (20+)
│   ├── server/             # HTTP API server (headless mode)
│   ├── session/            # Session store, prompt engine, compaction
│   ├── share/              # Session sharing client
│   ├── tool/               # Tool registry and implementations
│   ├── tui/                # Bubbletea TUI model and components
│   └── worktree/           # Git worktree management
├── .github/workflows/      # CI + release (GoReleaser)
├── .goreleaser.yml         # Cross-platform release config
├── Makefile                # Build targets with ldflags
├── go.mod
└── LICENSE
```

---

## Development

### Prerequisites

- Go 1.24.2+
- [golangci-lint](https://golangci-lint.run) (optional, for `make lint`)
- [goreleaser](https://goreleaser.com) (optional, for `make snapshot`)

### Common tasks

```bash
make build      # build with version/commit injected
make test       # go test -race ./...
make cover      # test + HTML coverage report
make fmt        # gofmt -w .
make lint       # golangci-lint run
make cross      # cross-compile Linux/macOS/Windows
make snapshot   # local goreleaser snapshot
```

### Adding a new AI provider

Implement the `Provider` interface in `internal/provider/`:

```go
type Provider interface {
    Name() string
    Models() []ModelInfo
    CreateMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error)
    StreamMessage(ctx context.Context, req *MessageRequest, cb func(*StreamChunk) error) error
}
```

Register it in `internal/provider/provider.go` → `CreateProvider`.

### Adding a new tool

Implement `tool.Tool` in `internal/tool/` and register it in `tool.GetRegistry()`.

---

## License

MIT — see [LICENSE](LICENSE).

---

## Acknowledgments

- Inspired by [opencode](https://github.com/sst/opencode)
- TUI powered by [Bubbletea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles), [Lipgloss](https://github.com/charmbracelet/lipgloss), [Glamour](https://github.com/charmbracelet/glamour)
- Syntax highlighting by [Chroma](https://github.com/alecthomas/chroma)
- CLI by [Cobra](https://github.com/spf13/cobra)
