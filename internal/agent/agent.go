package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Dhanuzh/dcode/internal/config"
)

// promptContextCache holds the cached result of buildPromptWithContext so that
// git subprocesses and directory walks only run once per TTL, not on every
// agentic step.
var (
	promptContextCache     string
	promptContextCacheKey  string // workdir + date, used as cache key
	promptContextCacheTime time.Time
	promptContextMu        sync.Mutex
	promptContextTTL       = 30 * time.Second // refresh at most every 30 s
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
	date := currentDate()
	cacheKey := workdir + "|" + date

	promptContextMu.Lock()
	if promptContextCacheKey == cacheKey && !promptContextCacheTime.IsZero() && time.Since(promptContextCacheTime) < promptContextTTL {
		cached := promptContextCache
		promptContextMu.Unlock()
		return basePrompt + cached
	}
	promptContextMu.Unlock()

	// Git branch only (skip status to save tokens)
	gitBranch := ""
	isGitRepo := false
	if cmd, err := exec.Command("git", "branch", "--show-current").Output(); err == nil {
		gitBranch = strings.TrimSpace(string(cmd))
		isGitRepo = true
	}

	// Load custom instructions (project dir only)
	customInstructions := loadCustomInstructions(workdir)

	context := fmt.Sprintf(`

<env>
  Working directory: %s
  Is directory a git repo: %v
  Platform: %s
  Today's date: %s
</env>`, workdir, isGitRepo, runtime.GOOS, date)

	if gitBranch != "" {
		context += fmt.Sprintf("\n- Git Branch: %s", gitBranch)
	}

	if customInstructions != "" {
		context += "\n\n" + customInstructions
	}

	promptContextMu.Lock()
	promptContextCache = context
	promptContextCacheKey = cacheKey
	promptContextCacheTime = time.Now()
	promptContextMu.Unlock()

	return basePrompt + context
}

func currentDate() string {
	return time.Now().Format("Mon Jan 02 2006")
}

// loadCustomInstructions loads instruction files from the project directory only.
// It does NOT walk up the directory tree to avoid injecting large files from
// ancestor/home directories into every request (which inflates token usage).
func loadCustomInstructions(workdir string) string {
	var instructions []string

	// Only look in the immediate project directory and its .dcode subdirectory.
	candidates := []string{
		filepath.Join(workdir, ".dcode", "instructions.md"),
		filepath.Join(workdir, "AGENTS.md"),
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := strings.TrimSpace(string(data))
		if content == "" {
			continue
		}
		// Cap each instruction file at 4 KB to prevent runaway token usage.
		const maxInstructionBytes = 4096
		if len(content) > maxInstructionBytes {
			content = content[:maxInstructionBytes] + "\n... (truncated)"
		}
		instructions = append(instructions, content)
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
const CoderPrompt = `You are DCode, an AI coding agent in a CLI.

Rules: be concise; read before editing; use glob/grep/codesearch to find things; use bash for git/builds/tests; break tasks into steps with the todo tool; follow project conventions; run parallel tool calls when independent.`

// PlannerPrompt is the system prompt for the planner agent
const PlannerPrompt = `You are DCode in Plan mode — read-only analysis. You CANNOT modify files.

Analyze code, explore the codebase, suggest specific changes with snippets. Use tools for context.`

// ExplorerPrompt is the system prompt for the explorer subagent
const ExplorerPrompt = `You are a file search specialist. Use Glob/Grep/Read to find things. Return absolute paths. Do not modify files.`

// GeneralPrompt is the system prompt for the general subagent
const GeneralPrompt = `You are a research agent for complex multi-step tasks. Break questions into sub-questions, research thoroughly, run parallel tool calls when independent.`

// CompactionPrompt is the system prompt for the compaction agent
const CompactionPrompt = `Summarize this conversation concisely. Include: what was done, what is in progress, files modified, what to do next, key decisions and constraints.`

// TitlePrompt is the system prompt for the title generation agent
const TitlePrompt = `Output ONLY a session title, ≤50 chars, no punctuation, same language as the user. No explanations. Always output something.`

// SummaryPrompt is the system prompt for the summary generation agent
const SummaryPrompt = `Summarize what was done in 2-3 sentences, like a PR description. First person, describe changes not process. Preserve any final question or request to the user.`
