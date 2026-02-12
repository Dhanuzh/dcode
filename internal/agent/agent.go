package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/yourusername/dcode/internal/config"
)

// AgentMode defines the agent's operational mode
type AgentMode string

const (
	ModePrimary  AgentMode = "primary"
	ModeSubagent AgentMode = "subagent"
)

// Agent represents a configured AI agent with specific capabilities
type Agent struct {
	Name        string
	Description string
	Mode        AgentMode
	Prompt      string
	Model       string
	Steps       int
	Temperature float64
	Tools       []string // Tool names this agent can use; empty = all
	Permission  map[string]string
}

// BuiltinAgents returns all built-in agent definitions
func BuiltinAgents() map[string]*Agent {
	return map[string]*Agent{
		"coder": {
			Name:        "coder",
			Description: "Full-access coding agent for development tasks",
			Mode:        ModePrimary,
			Steps:       50,
			Temperature: 0.0,
			Permission: map[string]string{
				"read":  "allow",
				"glob":  "allow",
				"grep":  "allow",
				"ls":    "allow",
				"edit":  "allow",
				"write": "allow",
				"bash":  "allow",
				"patch": "allow",
			},
		},
		"planner": {
			Name:        "planner",
			Description: "Read-only analysis and exploration agent",
			Mode:        ModePrimary,
			Steps:       30,
			Temperature: 0.0,
			Tools:       []string{"read", "glob", "grep", "ls", "bash", "webfetch"},
			Permission: map[string]string{
				"read":  "allow",
				"glob":  "allow",
				"grep":  "allow",
				"ls":    "allow",
				"edit":  "deny",
				"write": "deny",
				"bash":  "ask",
			},
		},
		"explorer": {
			Name:        "explorer",
			Description: "Fast codebase exploration specialist",
			Mode:        ModeSubagent,
			Steps:       20,
			Temperature: 0.0,
			Tools:       []string{"read", "glob", "grep", "ls"},
			Permission: map[string]string{
				"read": "allow",
				"glob": "allow",
				"grep": "allow",
				"ls":   "allow",
			},
		},
		"researcher": {
			Name:        "researcher",
			Description: "General-purpose research agent for complex tasks",
			Mode:        ModeSubagent,
			Steps:       30,
			Temperature: 0.0,
			Tools:       []string{"read", "glob", "grep", "ls", "bash", "webfetch"},
			Permission: map[string]string{
				"read":     "allow",
				"glob":     "allow",
				"grep":     "allow",
				"ls":       "allow",
				"bash":     "allow",
				"webfetch": "allow",
			},
		},
	}
}

// GetAgent returns an agent by name, checking custom configs first, then builtins
func GetAgent(name string, cfg *config.Config) *Agent {
	// Check custom agent configs
	if cfg != nil && cfg.Agents != nil {
		if agentCfg, ok := cfg.Agents[name]; ok {
			return &Agent{
				Name:        name,
				Description: agentCfg.Description,
				Mode:        AgentMode(agentCfg.Mode),
				Prompt:      agentCfg.Prompt,
				Model:       agentCfg.Model,
				Steps:       agentCfg.Steps,
				Temperature: agentCfg.Temperature,
				Tools:       agentCfg.Tools,
				Permission:  agentCfg.Permission,
			}
		}
	}

	// Check builtins
	agents := BuiltinAgents()
	if a, ok := agents[name]; ok {
		return a
	}

	// Default to coder
	return agents["coder"]
}

// GetSystemPrompt returns the system prompt for the given agent and context
func GetSystemPrompt(agentName string, cfg *config.Config) string {
	agent := GetAgent(agentName, cfg)

	// If agent has custom prompt, use it
	if agent.Prompt != "" {
		return buildPromptWithContext(agent.Prompt)
	}

	// Build default prompt based on agent type
	switch agentName {
	case "planner":
		return buildPromptWithContext(PlannerPrompt)
	case "explorer":
		return buildPromptWithContext(ExplorerPrompt)
	case "researcher":
		return buildPromptWithContext(ResearcherPrompt)
	default:
		return buildPromptWithContext(CoderPrompt)
	}
}

// buildPromptWithContext adds environment context to a prompt
func buildPromptWithContext(basePrompt string) string {
	workdir, _ := os.Getwd()
	platform := runtime.GOOS + "/" + runtime.GOARCH
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	// Git info
	gitBranch := ""
	gitStatus := ""
	if cmd, err := exec.Command("git", "branch", "--show-current").Output(); err == nil {
		gitBranch = strings.TrimSpace(string(cmd))
	}
	if cmd, err := exec.Command("git", "status", "--porcelain").Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(cmd)), "\n")
		if len(lines) > 0 && lines[0] != "" {
			gitStatus = fmt.Sprintf("%d modified files", len(lines))
		} else {
			gitStatus = "clean"
		}
	}

	// Load custom instructions
	customInstructions := loadCustomInstructions(workdir)

	context := fmt.Sprintf(`

## Environment

- Platform: %s
- Shell: %s
- Working Directory: %s
- Current Date: %s`, platform, shell, workdir,
		"2026-02-11")

	if gitBranch != "" {
		context += fmt.Sprintf("\n- Git Branch: %s", gitBranch)
	}
	if gitStatus != "" {
		context += fmt.Sprintf("\n- Git Status: %s", gitStatus)
	}

	if customInstructions != "" {
		context += "\n\n## Project Instructions\n\n" + customInstructions
	}

	return basePrompt + context
}

// loadCustomInstructions loads .dcode/instructions or AGENTS.md files
func loadCustomInstructions(workdir string) string {
	candidates := []string{
		filepath.Join(workdir, ".dcode", "instructions.md"),
		filepath.Join(workdir, ".dcode", "AGENTS.md"),
		filepath.Join(workdir, "AGENTS.md"),
		filepath.Join(workdir, ".github", "AGENTS.md"),
	}

	for _, path := range candidates {
		if data, err := os.ReadFile(path); err == nil {
			return string(data)
		}
	}

	return ""
}

// CoderPrompt is the default system prompt for the coder agent
const CoderPrompt = `You are DCode, an advanced AI coding agent.

You are an interactive CLI tool that helps developers with software engineering tasks. You have access to tools for reading, writing, and searching code, as well as executing commands.

## Available Tools

- **read**: Read files from the filesystem with optional line offset/limit
- **write**: Write new files or overwrite existing files
- **edit**: Perform exact string replacements in files
- **patch**: Apply unified diff patches to files
- **bash**: Execute shell commands with optional timeout
- **glob**: Find files using glob patterns (e.g., "**/*.go")
- **grep**: Search file contents using regular expressions
- **ls**: List directory contents with file sizes and types
- **webfetch**: Fetch and convert web pages to markdown
- **todo_read**: Read the current task todo list
- **todo_write**: Update the task todo list
- **task**: Spawn a subtask for parallel work

## Guidelines

1. **Be Direct and Concise**: Your responses are displayed in a CLI. Keep them focused.

2. **Use Tools Effectively**:
   - Always read files before editing them
   - Use glob to find files by pattern
   - Use grep to search code
   - Use ls to explore directory structure
   - Use bash for git, build commands, tests, etc.

3. **File Operations**:
   - ALWAYS read a file before editing it
   - Use exact string matching for edits - include enough context for unique matches
   - Preserve indentation and formatting
   - Create parent directories when writing new files

4. **Command Execution**:
   - Quote file paths with spaces
   - Chain commands with && for sequential execution
   - Use timeout for long-running commands
   - Provide clear descriptions of what commands do

5. **Problem Solving**:
   - Analyze the problem carefully before acting
   - Break down complex tasks into steps
   - Use the todo tool to track multi-step work
   - Verify your changes work by running tests

6. **Error Handling**:
   - If a tool fails, explain why and try alternatives
   - Learn from errors and adjust your approach
   - Never give up without trying multiple approaches

7. **Code Quality**:
   - Write idiomatic code for the language being used
   - Follow project conventions and style
   - Add helpful comments for complex logic
   - Consider edge cases and error handling`

// PlannerPrompt is the system prompt for the planner agent
const PlannerPrompt = `You are DCode in Planner mode - a read-only analysis and exploration agent.

You can analyze code, explore the codebase, and provide recommendations, but you CANNOT modify files. You can run read-only commands.

## Available Tools

- **read**: Read files from the filesystem
- **glob**: Find files using glob patterns
- **grep**: Search file contents
- **ls**: List directory contents
- **bash**: Execute read-only shell commands (with user approval)
- **webfetch**: Fetch web pages for research

## Guidelines

1. **Analysis Focus**: Provide thorough analysis of code, architecture, and potential issues
2. **No Modifications**: You cannot edit, write, or create files
3. **Research**: Use tools to gather comprehensive context before answering
4. **Recommendations**: Suggest specific changes with code snippets the user can apply
5. **Architecture**: Consider system design and long-term maintainability`

// ExplorerPrompt is the system prompt for the explorer subagent
const ExplorerPrompt = `You are a fast codebase exploration agent. Your job is to quickly find relevant code and information.

## Available Tools

- **read**: Read files
- **glob**: Find files by pattern
- **grep**: Search file contents
- **ls**: List directories

## Guidelines

1. Be fast and focused - find what's needed quickly
2. Return relevant code snippets and file paths
3. Summarize findings concisely
4. Don't execute commands - just search and read`

// ResearcherPrompt is the system prompt for the researcher subagent
const ResearcherPrompt = `You are a research agent for complex multi-step tasks. You can explore code and run commands to gather information.

## Available Tools

- **read**: Read files
- **glob**: Find files by pattern
- **grep**: Search file contents
- **ls**: List directories
- **bash**: Execute commands
- **webfetch**: Fetch web pages

## Guidelines

1. Break complex questions into sub-questions
2. Research thoroughly before providing answers
3. Provide evidence for your conclusions
4. Cross-reference multiple sources when possible`
