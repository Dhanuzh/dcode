package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Dhanuzh/dcode/internal/config"
)

// AgentMode defines the agent's operational mode
type AgentMode string

const (
	ModePrimary  AgentMode = "primary"
	ModeSubagent AgentMode = "subagent"
	ModeAll      AgentMode = "all" // can be used as either primary or subagent
)

// PermissionAction defines what happens when a tool is invoked
type PermissionAction string

const (
	PermAllow PermissionAction = "allow"
	PermDeny  PermissionAction = "deny"
	PermAsk   PermissionAction = "ask"
)

// PermissionRule represents a single permission rule with glob pattern support
type PermissionRule struct {
	Permission string           `json:"permission"`
	Pattern    string           `json:"pattern"`
	Action     PermissionAction `json:"action"`
}

// Agent represents a configured AI agent with specific capabilities
type Agent struct {
	Name        string
	Description string
	Mode        AgentMode
	Prompt      string
	Model       string // "providerID/modelID" format
	Variant     string
	Steps       int
	Temperature float64
	TopP        float64
	Color       string
	Hidden      bool     // Hidden agents are not shown in agent picker
	Native      bool     // Built-in agent
	Tools       []string // Tool names this agent can use; empty = all
	Permission  []PermissionRule
	Options     map[string]interface{}
}

// EditTools is the set of tools that are gated by the "edit" permission
var EditTools = []string{"edit", "write", "patch", "multiedit", "apply_patch"}

// WildcardMatch performs simple glob-style matching
// Supports * (match any sequence) and ? (match single char)
func WildcardMatch(pattern, value string) bool {
	if pattern == "*" {
		return true
	}
	return wildcardMatchImpl(pattern, value)
}

func wildcardMatchImpl(pattern, value string) bool {
	for len(pattern) > 0 {
		switch pattern[0] {
		case '*':
			// Skip consecutive *
			for len(pattern) > 0 && pattern[0] == '*' {
				pattern = pattern[1:]
			}
			if len(pattern) == 0 {
				return true
			}
			for i := 0; i <= len(value); i++ {
				if wildcardMatchImpl(pattern, value[i:]) {
					return true
				}
			}
			return false
		case '?':
			if len(value) == 0 {
				return false
			}
			pattern = pattern[1:]
			value = value[1:]
		default:
			if len(value) == 0 || pattern[0] != value[0] {
				return false
			}
			pattern = pattern[1:]
			value = value[1:]
		}
	}
	return len(value) == 0
}

// EvaluatePermission evaluates permission rules for a given tool and pattern
// Rules are evaluated in order (last matching rule wins)
func EvaluatePermission(permission, pattern string, rulesets ...[]PermissionRule) PermissionRule {
	var merged []PermissionRule
	for _, rs := range rulesets {
		merged = append(merged, rs...)
	}

	// Find last matching rule
	result := PermissionRule{
		Permission: permission,
		Pattern:    "*",
		Action:     PermAsk, // default
	}

	for _, rule := range merged {
		if WildcardMatch(rule.Permission, permission) && WildcardMatch(rule.Pattern, pattern) {
			result = rule
		}
	}
	return result
}

// DisabledTools returns the set of tools that are fully denied by the ruleset
func DisabledTools(tools []string, ruleset []PermissionRule) map[string]bool {
	result := make(map[string]bool)
	for _, t := range tools {
		permission := t
		for _, editTool := range EditTools {
			if t == editTool {
				permission = "edit"
				break
			}
		}
		// Find last matching rule with pattern "*"
		for i := len(ruleset) - 1; i >= 0; i-- {
			rule := ruleset[i]
			if !WildcardMatch(rule.Permission, permission) {
				continue
			}
			if rule.Pattern == "*" && rule.Action == PermDeny {
				result[t] = true
			}
			break
		}
	}
	return result
}

// PermissionFromConfig converts a config-style permission map to rules
// Supports both simple ("allow") and nested ({"*": "allow", "*.env": "ask"}) forms
func PermissionFromConfig(perm map[string]interface{}) []PermissionRule {
	var rules []PermissionRule
	for key, value := range perm {
		switch v := value.(type) {
		case string:
			rules = append(rules, PermissionRule{
				Permission: key,
				Pattern:    "*",
				Action:     PermissionAction(v),
			})
		case map[string]interface{}:
			for pattern, action := range v {
				if actionStr, ok := action.(string); ok {
					// Expand ~ to home directory
					expandedPattern := expandPath(pattern)
					rules = append(rules, PermissionRule{
						Permission: key,
						Pattern:    expandedPattern,
						Action:     PermissionAction(actionStr),
					})
				}
			}
		case map[string]string:
			for pattern, action := range v {
				expandedPattern := expandPath(pattern)
				rules = append(rules, PermissionRule{
					Permission: key,
					Pattern:    expandedPattern,
					Action:     PermissionAction(action),
				})
			}
		}
	}
	return rules
}

func expandPath(pattern string) string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return pattern
	}
	if strings.HasPrefix(pattern, "~/") {
		return home + pattern[1:]
	}
	if pattern == "~" {
		return home
	}
	if strings.HasPrefix(pattern, "$HOME/") {
		return home + pattern[5:]
	}
	if pattern == "$HOME" {
		return home
	}
	return pattern
}

// MergePermissions merges multiple rulesets (last wins semantics)
func MergePermissions(rulesets ...[]PermissionRule) []PermissionRule {
	var merged []PermissionRule
	for _, rs := range rulesets {
		merged = append(merged, rs...)
	}
	return merged
}

// defaultPermissions returns the base permission ruleset for all agents
func defaultPermissions() []PermissionRule {
	return []PermissionRule{
		{Permission: "*", Pattern: "*", Action: PermAllow},
		{Permission: "doom_loop", Pattern: "*", Action: PermAsk},
		{Permission: "external_directory", Pattern: "*", Action: PermAsk},
		{Permission: "question", Pattern: "*", Action: PermDeny},
		{Permission: "plan_enter", Pattern: "*", Action: PermDeny},
		{Permission: "plan_exit", Pattern: "*", Action: PermDeny},
		// Protect .env files
		{Permission: "read", Pattern: "*", Action: PermAllow},
		{Permission: "read", Pattern: "*.env", Action: PermAsk},
		{Permission: "read", Pattern: "*.env.*", Action: PermAsk},
		{Permission: "read", Pattern: "*.env.example", Action: PermAllow},
	}
}

// BuiltinAgents returns all built-in agent definitions
func BuiltinAgents() map[string]*Agent {
	defaults := defaultPermissions()

	return map[string]*Agent{
		// build - the default primary agent (equivalent to opencode's "build")
		"coder": {
			Name:        "coder",
			Description: "The default agent. Executes tools based on configured permissions.",
			Mode:        ModePrimary,
			Steps:       50,
			Temperature: 0.0,
			Native:      true,
			Hidden:      false,
			Options:     map[string]interface{}{},
			Permission: MergePermissions(
				defaults,
				[]PermissionRule{
					{Permission: "question", Pattern: "*", Action: PermAllow},
					{Permission: "plan_enter", Pattern: "*", Action: PermAllow},
				},
			),
		},
		// plan - plan mode agent (disallows edit tools)
		"planner": {
			Name:        "planner",
			Description: "Plan mode. Disallows all edit tools.",
			Mode:        ModePrimary,
			Steps:       30,
			Temperature: 0.0,
			Native:      true,
			Hidden:      false,
			Options:     map[string]interface{}{},
			Permission: MergePermissions(
				defaults,
				[]PermissionRule{
					{Permission: "question", Pattern: "*", Action: PermAllow},
					{Permission: "plan_exit", Pattern: "*", Action: PermAllow},
					{Permission: "edit", Pattern: "*", Action: PermDeny},
					{Permission: "edit", Pattern: filepath.Join(".dcode", "plans", "*.md"), Action: PermAllow},
				},
			),
		},
		// general - subagent for complex multi-step tasks
		"general": {
			Name:        "general",
			Description: "General-purpose agent for researching complex questions and executing multi-step tasks. Use this agent to execute multiple units of work in parallel.",
			Mode:        ModeSubagent,
			Steps:       50,
			Temperature: 0.0,
			Native:      true,
			Hidden:      false,
			Options:     map[string]interface{}{},
			Permission: MergePermissions(
				defaults,
				[]PermissionRule{
					{Permission: "todoread", Pattern: "*", Action: PermDeny},
					{Permission: "todowrite", Pattern: "*", Action: PermDeny},
				},
			),
		},
		// explore - fast codebase exploration subagent
		"explorer": {
			Name:        "explorer",
			Description: `Fast agent specialized for exploring codebases. Use this when you need to quickly find files by patterns (eg. "src/components/**/*.tsx"), search code for keywords (eg. "API endpoints"), or answer questions about the codebase (eg. "how do API endpoints work?"). When calling this agent, specify the desired thoroughness level: "quick" for basic searches, "medium" for moderate exploration, or "very thorough" for comprehensive analysis across multiple locations and naming conventions.`,
			Mode:        ModeSubagent,
			Steps:       20,
			Temperature: 0.0,
			Prompt:      ExplorerPrompt,
			Native:      true,
			Hidden:      false,
			Options:     map[string]interface{}{},
			Permission: MergePermissions(
				defaults,
				[]PermissionRule{
					{Permission: "*", Pattern: "*", Action: PermDeny},
					{Permission: "grep", Pattern: "*", Action: PermAllow},
					{Permission: "glob", Pattern: "*", Action: PermAllow},
					{Permission: "list", Pattern: "*", Action: PermAllow},
					{Permission: "ls", Pattern: "*", Action: PermAllow},
					{Permission: "bash", Pattern: "*", Action: PermAllow},
					{Permission: "webfetch", Pattern: "*", Action: PermAllow},
					{Permission: "websearch", Pattern: "*", Action: PermAllow},
					{Permission: "codesearch", Pattern: "*", Action: PermAllow},
					{Permission: "read", Pattern: "*", Action: PermAllow},
				},
			),
		},
		// compaction - hidden agent for context compaction
		"compaction": {
			Name:    "compaction",
			Mode:    ModePrimary,
			Native:  true,
			Hidden:  true,
			Prompt:  CompactionPrompt,
			Options: map[string]interface{}{},
			Permission: MergePermissions(
				defaults,
				[]PermissionRule{
					{Permission: "*", Pattern: "*", Action: PermDeny},
				},
			),
		},
		// title - hidden agent for generating session titles
		"title": {
			Name:        "title",
			Mode:        ModePrimary,
			Native:      true,
			Hidden:      true,
			Temperature: 0.5,
			Prompt:      TitlePrompt,
			Options:     map[string]interface{}{},
			Permission: MergePermissions(
				defaults,
				[]PermissionRule{
					{Permission: "*", Pattern: "*", Action: PermDeny},
				},
			),
		},
		// summary - hidden agent for generating session summaries
		"summary": {
			Name:    "summary",
			Mode:    ModePrimary,
			Native:  true,
			Hidden:  true,
			Prompt:  SummaryPrompt,
			Options: map[string]interface{}{},
			Permission: MergePermissions(
				defaults,
				[]PermissionRule{
					{Permission: "*", Pattern: "*", Action: PermDeny},
				},
			),
		},
	}
}

// GetAgent returns an agent by name, checking custom configs first, then builtins
func GetAgent(name string, cfg *config.Config) *Agent {
	agents := BuiltinAgents()

	// Apply custom agent configs from config
	if cfg != nil && cfg.Agents != nil {
		for key, agentCfg := range cfg.Agents {
			if key == name {
				existing, ok := agents[key]
				if !ok {
					// New custom agent
					existing = &Agent{
						Name:       key,
						Mode:       ModeAll,
						Permission: defaultPermissions(),
						Options:    map[string]interface{}{},
						Native:     false,
					}
					agents[key] = existing
				}

				// Apply overrides
				if agentCfg.Model != "" {
					existing.Model = agentCfg.Model
				}
				if agentCfg.Prompt != "" {
					existing.Prompt = agentCfg.Prompt
				}
				if agentCfg.Description != "" {
					existing.Description = agentCfg.Description
				}
				if agentCfg.Temperature != 0 {
					existing.Temperature = agentCfg.Temperature
				}
				if agentCfg.Mode != "" {
					existing.Mode = AgentMode(agentCfg.Mode)
				}
				if agentCfg.Steps > 0 {
					existing.Steps = agentCfg.Steps
				}
				if len(agentCfg.Tools) > 0 {
					existing.Tools = agentCfg.Tools
				}
				if len(agentCfg.Permission) > 0 {
					// Convert simple string map to permission rules
					perm := make(map[string]interface{})
					for k, v := range agentCfg.Permission {
						perm[k] = v
					}
					existing.Permission = MergePermissions(
						existing.Permission,
						PermissionFromConfig(perm),
					)
				}
				break
			}
		}
	}

	if a, ok := agents[name]; ok {
		return a
	}

	// Default to coder
	return agents["coder"]
}

// ListAgents returns all visible agents (non-hidden), sorted with default first
func ListAgents(cfg *config.Config) []*Agent {
	agents := BuiltinAgents()

	// Apply custom agents
	if cfg != nil && cfg.Agents != nil {
		for key, agentCfg := range cfg.Agents {
			existing, ok := agents[key]
			if !ok {
				existing = &Agent{
					Name:       key,
					Mode:       ModeAll,
					Permission: defaultPermissions(),
					Options:    map[string]interface{}{},
					Native:     false,
				}
				agents[key] = existing
			}
			if agentCfg.Model != "" {
				existing.Model = agentCfg.Model
			}
			if agentCfg.Description != "" {
				existing.Description = agentCfg.Description
			}
			if agentCfg.Mode != "" {
				existing.Mode = AgentMode(agentCfg.Mode)
			}
			if agentCfg.Steps > 0 {
				existing.Steps = agentCfg.Steps
			}
		}
	}

	var result []*Agent
	defaultName := "coder"
	if cfg != nil && cfg.DefaultAgent != "" {
		defaultName = cfg.DefaultAgent
	}

	// Add default first
	if a, ok := agents[defaultName]; ok && !a.Hidden {
		result = append(result, a)
	}

	// Add rest (non-hidden, non-default)
	for name, a := range agents {
		if a.Hidden || name == defaultName {
			continue
		}
		result = append(result, a)
	}

	return result
}

// DefaultAgent returns the name of the default agent
func DefaultAgent(cfg *config.Config) string {
	if cfg != nil && cfg.DefaultAgent != "" {
		agents := BuiltinAgents()
		if a, ok := agents[cfg.DefaultAgent]; ok {
			if a.Mode == ModeSubagent {
				return "coder" // subagents can't be default
			}
			if a.Hidden {
				return "coder" // hidden agents can't be default
			}
			return cfg.DefaultAgent
		}
	}
	return "coder"
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
	case "general":
		return buildPromptWithContext(GeneralPrompt)
	case "compaction":
		return CompactionPrompt // No context needed
	case "title":
		return TitlePrompt // No context needed
	case "summary":
		return SummaryPrompt // No context needed
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
	isGitRepo := false
	if cmd, err := exec.Command("git", "branch", "--show-current").Output(); err == nil {
		gitBranch = strings.TrimSpace(string(cmd))
		isGitRepo = true
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

<env>
  Working directory: %s
  Is directory a git repo: %v
  Platform: %s
  Today's date: %s
</env>`, workdir, isGitRepo, platform, currentDate())

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

func currentDate() string {
	// Return formatted date like "Mon Feb 11 2026"
	return fmt.Sprintf("%s", strings.Fields(fmt.Sprintf("%v", func() string {
		cmd := exec.Command("date", "+%a %b %d %Y")
		out, err := cmd.Output()
		if err != nil {
			return "unknown"
		}
		return strings.TrimSpace(string(out))
	}()))[0:])
}

// loadCustomInstructions walks up the directory tree loading instruction files
// Follows the same pattern as opencode: AGENTS.md, CLAUDE.md, .dcode/instructions.md
func loadCustomInstructions(workdir string) string {
	var instructions []string

	// Walk up directory tree looking for instruction files
	dir := workdir
	for {
		candidates := []string{
			filepath.Join(dir, ".dcode", "instructions.md"),
			filepath.Join(dir, ".dcode", "AGENTS.md"),
			filepath.Join(dir, "AGENTS.md"),
			filepath.Join(dir, "CLAUDE.md"),
			filepath.Join(dir, ".github", "AGENTS.md"),
		}

		for _, path := range candidates {
			if data, err := os.ReadFile(path); err == nil {
				content := strings.TrimSpace(string(data))
				if content != "" {
					header := fmt.Sprintf("Instructions from: %s", path)
					instructions = append(instructions, header+"\n"+content)
				}
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return strings.Join(instructions, "\n\n")
}

// LoadAgentFromMarkdown loads an agent definition from a markdown file
// Format: YAML frontmatter between --- delimiters, followed by prompt content
func LoadAgentFromMarkdown(path string) (*Agent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent file: %w", err)
	}

	content := string(data)
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

	agent := &Agent{
		Name:       name,
		Mode:       ModeAll,
		Native:     false,
		Options:    map[string]interface{}{},
		Permission: defaultPermissions(),
	}

	// Parse frontmatter if present
	if strings.HasPrefix(content, "---") {
		parts := strings.SplitN(content[3:], "---", 2)
		if len(parts) == 2 {
			frontmatter := strings.TrimSpace(parts[0])
			agent.Prompt = strings.TrimSpace(parts[1])

			// Simple YAML-like parsing of frontmatter
			for _, line := range strings.Split(frontmatter, "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				kv := strings.SplitN(line, ":", 2)
				if len(kv) != 2 {
					continue
				}
				key := strings.TrimSpace(kv[0])
				value := strings.TrimSpace(kv[1])

				switch key {
				case "description":
					agent.Description = value
				case "mode":
					agent.Mode = AgentMode(value)
				case "model":
					agent.Model = value
				case "hidden":
					agent.Hidden = value == "true"
				case "temperature":
					var t float64
					fmt.Sscanf(value, "%f", &t)
					agent.Temperature = t
				case "color":
					agent.Color = value
				case "variant":
					agent.Variant = value
				case "steps":
					var s int
					fmt.Sscanf(value, "%d", &s)
					if s > 0 {
						agent.Steps = s
					}
				}
			}
		} else {
			agent.Prompt = content
		}
	} else {
		agent.Prompt = content
	}

	return agent, nil
}

// LoadCustomAgents loads agent definitions from .dcode/agents/*.md files
func LoadCustomAgents(workdir string) map[string]*Agent {
	result := make(map[string]*Agent)
	agentDir := filepath.Join(workdir, ".dcode", "agents")

	entries, err := os.ReadDir(agentDir)
	if err != nil {
		return result
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".md" && ext != ".txt" {
			continue
		}

		path := filepath.Join(agentDir, entry.Name())
		agent, err := LoadAgentFromMarkdown(path)
		if err != nil {
			continue
		}

		result[agent.Name] = agent
	}

	return result
}

// CoderPrompt is the default system prompt for the coder agent
const CoderPrompt = `You are DCode, an advanced AI coding agent operating as an interactive CLI tool.

You help developers with software engineering tasks using tools for reading, writing, searching code, and executing commands.

## Core Rules

1. **Be concise** — your output displays in a terminal.
2. **Always read before editing** — use exact string matching with enough context for unique matches. Preserve indentation.
3. **Search effectively** — use glob for file patterns, grep for content search, codesearch for symbols, ls for directory structure.
4. **Execute carefully** — use bash for git, builds, tests. Quote paths with spaces. Chain with &&.
5. **Solve problems systematically** — break complex tasks into steps, use the todo tool to track progress, verify with tests.
6. **Handle errors gracefully** — if a tool fails, explain why and try alternatives. Never give up without trying multiple approaches.
7. **Write quality code** — follow project conventions, write idiomatic code, consider edge cases.
8. **Use parallel execution** — use batch or task tools when operations are independent.`

// PlannerPrompt is the system prompt for the planner agent
const PlannerPrompt = `You are DCode in Plan mode — a read-only analysis and exploration agent.

You can analyze code, explore the codebase, and provide recommendations, but you CANNOT modify files.

Guidelines:
- Provide thorough analysis of code, architecture, and potential issues.
- Use tools to gather comprehensive context before answering.
- Suggest specific changes with code snippets the user can apply.
- Consider system design and long-term maintainability.`

// ExplorerPrompt is the system prompt for the explorer subagent
const ExplorerPrompt = `You are a file search specialist that excels at navigating and exploring codebases.

Guidelines:
- Use Glob for file pattern matching, Grep for content search with regex, Read for specific files.
- Adapt search approach based on the thoroughness level specified by the caller.
- Return file paths as absolute paths. Do not modify files or system state.
- Complete the search request efficiently and report findings clearly.`

// GeneralPrompt is the system prompt for the general subagent
const GeneralPrompt = `You are a research agent for complex multi-step tasks. You can explore code, run commands, edit files, and gather information.

Guidelines:
- Break complex questions into sub-questions.
- Research thoroughly before providing answers and provide evidence.
- Execute independent units of work in parallel when possible.`

// CompactionPrompt is the system prompt for the compaction agent
const CompactionPrompt = `You are a helpful AI assistant tasked with summarizing conversations.

When asked to summarize, provide a detailed but concise summary of the conversation.
Focus on information that would be helpful for continuing the conversation, including:
- What was done
- What is currently being worked on
- Which files are being modified
- What needs to be done next
- Key user requests, constraints, or preferences that should persist
- Important technical decisions and why they were made

Your summary should be comprehensive enough to provide context but concise enough to be quickly understood.`

// TitlePrompt is the system prompt for the title generation agent
const TitlePrompt = `You are a title generator. You output ONLY a thread title. Nothing else.

Generate a brief title that would help the user find this conversation later.

Rules:
- you MUST use the same language as the user message you are summarizing
- Title must be grammatically correct and read naturally - no word salad
- Never include tool names in the title (e.g. "read tool", "bash tool", "edit tool")
- Focus on the main topic or question the user needs to retrieve
- Vary your phrasing - avoid repetitive patterns like always starting with "Analyzing"
- When a file is mentioned, focus on WHAT the user wants to do WITH the file, not just that they shared it
- Keep exact: technical terms, numbers, filenames, HTTP codes
- Remove: the, this, my, a, an
- Never assume tech stack
- Never use tools
- NEVER respond to questions, just generate a title for the conversation
- The title should NEVER include "summarizing" or "generating" when generating a title
- DO NOT SAY YOU CANNOT GENERATE A TITLE OR COMPLAIN ABOUT THE INPUT
- Always output something meaningful, even if the input is minimal
- A single line, 50 characters or less
- No explanations
- If the user message is short or conversational (e.g. "hello", "lol", "what's up", "hey"):
  create a title that reflects the user's tone or intent (such as Greeting, Quick check-in, Light chat, Intro message, etc.)`

// SummaryPrompt is the system prompt for the summary generation agent
const SummaryPrompt = `Summarize what was done in this conversation. Write like a pull request description.

Rules:
- 2-3 sentences max
- Describe the changes made, not the process
- Do not mention running tests, builds, or other validation steps
- Do not explain what the user asked for
- Write in first person (I added..., I fixed...)
- Never ask questions or add new questions
- If the conversation ends with an unanswered question to the user, preserve that exact question
- If the conversation ends with an imperative statement or request to the user (e.g. "Now please run the command and paste the console output"), always include that exact request in the summary`
