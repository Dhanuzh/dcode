# dcode

**dcode** is an AI coding agent that runs locally on your computer. It connects to multiple AI providers — OpenAI, Anthropic, GitHub Copilot, Groq, Mistral, and more — and helps you write, debug, and refactor code directly from your terminal.

## Install

```sh
npm install -g dcode
```

```sh
# or with bun
bun install -g dcode
```

## Quickstart

```sh
dcode
```

On first run, dcode will prompt you to sign in. You can choose from:

1. **Sign in with ChatGPT** — uses your OpenAI/ChatGPT account
2. **Sign in with Device Code** — for headless / remote machines
3. **Provide your own API key** — OpenAI usage-based billing
4. **Sign in with GitHub Copilot** — uses your Copilot subscription (OAuth)
5. **Sign in with Anthropic** — uses your Anthropic API key

## Supported Providers

Set `model_provider` in `~/.dcode/config.toml`:

| Provider | `model_provider` value | Env var |
|---|---|---|
| OpenAI | `openai` | `OPENAI_API_KEY` |
| Anthropic | `anthropic` | `ANTHROPIC_API_KEY` |
| GitHub Copilot | `github-copilot` | `GITHUB_COPILOT_TOKEN` |
| Groq | `groq` | `GROQ_API_KEY` |
| Mistral | `mistral` | `MISTRAL_API_KEY` |
| OpenRouter | `openrouter` | `OPENROUTER_API_KEY` |
| DeepSeek | `deepseek` | `DEEPSEEK_API_KEY` |
| Together AI | `together` | `TOGETHER_API_KEY` |
| Perplexity | `perplexity` | `PERPLEXITY_API_KEY` |
| xAI (Grok) | `xai` | `XAI_API_KEY` |
| Cohere | `cohere` | `COHERE_API_KEY` |
| Google Gemini | `google` | `GEMINI_API_KEY` |
| Ollama (local) | `ollama` | — |

**Example `~/.dcode/config.toml`:**

```toml
model = "claude-opus-4-5"
model_provider = "anthropic"
```

## Usage

```sh
# Start interactive TUI
dcode

# Run a one-shot prompt
dcode "explain this codebase"

# Ask about a specific file
dcode "refactor src/main.rs to use async"

# Check login status
dcode login --status

# Log in with GitHub Copilot
dcode login

# Log out
dcode logout
```

## Config

Config file: `~/.dcode/config.toml`

```toml
model = "gpt-4o"
model_provider = "openai"

# Use a local Ollama model
# model_provider = "ollama"
# model = "llama3"
```

Override config dir with `DCODE_HOME` env var.

## Building from source

Requires Rust (stable toolchain):

```sh
cd dcode-rs
cargo build --release --bin dcode
# binary at: target/release/dcode
```

## License

Apache-2.0
