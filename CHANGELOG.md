# Changelog

All notable changes to dcode are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
dcode uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [2.0.0] — 2026-03-01

### Added

**Core**
- 20+ AI provider support: Anthropic, OpenAI, Google Gemini, GitHub Copilot, Azure OpenAI, AWS Bedrock, Groq, OpenRouter, xAI, DeepSeek, Mistral, DeepInfra, Cerebras, Together AI, Cohere, Perplexity, GitLab, Cloudflare, Replicate, and any OpenAI-compatible endpoint
- Anthropic OAuth PKCE flow — authenticate without an API key
- GitHub Copilot device-code OAuth flow
- MCP (Model Context Protocol) support for local and remote servers
- Git worktree management (`dcode worktree create/remove/list/prune`)
- Session sharing (`dcode share create <id>`)
- Non-interactive mode (`dcode run "prompt"`) for scripting and CI/CD
- Shell completion scripts for bash, zsh, fish, and PowerShell
- `dcode stats` — session usage and token statistics
- `dcode debug` subcommands: `config`, `paths`, `env`
- `dcode upgrade` — self-update via `go install`
- `dcode uninstall` — remove all dcode data

**TUI**
- Full Bubbletea TUI with alt-screen, Glamour markdown rendering, and Chroma syntax highlighting
- Multi-step undo/redo backed by git snapshots (`Ctrl+Z` / `Ctrl+Shift+Z`, `/undo`, `/redo`)
- Live thinking/reasoning display with opencode-style spinner and topic extraction
- Copy code blocks with `Ctrl+Y` + digit or `/copy N`
- `@` file picker — attach text files (injected as fenced code blocks) or images; supports directory navigation and all file types
- Light-green blinking block cursor
- Custom Glamour prose theme with coloured headings, bullets, and inline code
- Proper fenced code block borders with Chroma line-by-line rendering (bypasses lipgloss width constraints)
- Mouse mode toggle (`Ctrl+M`) — off by default so terminal text selection works

**Providers**
- Google Gemini: inline image data (`inlineData`) support
- OpenAI-compatible: multi-content / image part support
- All providers: correct image+text message ordering in `RunWithAttachments`

**Build / CI**
- GoReleaser config with Linux/macOS/Windows amd64+arm64 binaries
- Makefile with `ldflags` version/commit injection, cross-compile target, coverage target
- CI workflow updated to Go 1.24.2 with correct module cache path and race-detector tests
- `SaveConfig` file permissions hardened from `0644` → `0600`

### Changed
- `.gitignore` — added `dcode_bin`, `.dcode/dcode.json`, `*.jsonc`, `dist/`, `node_modules/`
- `CODEOWNERS` — updated from placeholder to `@Dhanuzh`
- Removed committed `dcode_bin` binary from repository

---

## [1.0.0] — Initial release

- Basic TUI with Anthropic and OpenAI support
- File read/write/edit/glob/grep/bash tools
- Session persistence
- YAML-based configuration
