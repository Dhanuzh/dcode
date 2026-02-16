package permission

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gobwas/glob"
)

// RuleSet manages allow/deny rules with glob and regex patterns
type RuleSet struct {
	allowedPathGlobs  []glob.Glob
	deniedPathGlobs   []glob.Glob
	allowedCmdRegexes []*regexp.Regexp
	deniedCmdRegexes  []*regexp.Regexp
}

// NewRuleSet creates a new rule set from configuration
func NewRuleSet(cfg *Config) (*RuleSet, error) {
	rs := &RuleSet{
		allowedPathGlobs:  make([]glob.Glob, 0, len(cfg.AllowedPaths)),
		deniedPathGlobs:   make([]glob.Glob, 0, len(cfg.DeniedPaths)),
		allowedCmdRegexes: make([]*regexp.Regexp, 0, len(cfg.AllowedCommands)),
		deniedCmdRegexes:  make([]*regexp.Regexp, 0, len(cfg.DeniedCommands)),
	}

	// Compile path globs
	for _, pattern := range cfg.AllowedPaths {
		g, err := glob.Compile(pattern, '/')
		if err != nil {
			return nil, err
		}
		rs.allowedPathGlobs = append(rs.allowedPathGlobs, g)
	}

	for _, pattern := range cfg.DeniedPaths {
		g, err := glob.Compile(pattern, '/')
		if err != nil {
			return nil, err
		}
		rs.deniedPathGlobs = append(rs.deniedPathGlobs, g)
	}

	// Compile command regexes
	for _, pattern := range cfg.AllowedCommands {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		rs.allowedCmdRegexes = append(rs.allowedCmdRegexes, re)
	}

	for _, pattern := range cfg.DeniedCommands {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		rs.deniedCmdRegexes = append(rs.deniedCmdRegexes, re)
	}

	return rs, nil
}

// IsPathAllowed checks if a path matches allowed patterns
func (rs *RuleSet) IsPathAllowed(path string) bool {
	// Normalize path
	normalized := filepath.Clean(path)

	for _, g := range rs.allowedPathGlobs {
		if g.Match(normalized) {
			return true
		}
	}
	return false
}

// IsPathDenied checks if a path matches denied patterns
func (rs *RuleSet) IsPathDenied(path string) bool {
	normalized := filepath.Clean(path)

	for _, g := range rs.deniedPathGlobs {
		if g.Match(normalized) {
			return true
		}
	}
	return false
}

// IsCommandAllowed checks if a command matches allowed patterns
func (rs *RuleSet) IsCommandAllowed(cmd string) bool {
	for _, re := range rs.allowedCmdRegexes {
		if re.MatchString(cmd) {
			return true
		}
	}
	return false
}

// IsCommandDenied checks if a command matches denied patterns
func (rs *RuleSet) IsCommandDenied(cmd string) bool {
	for _, re := range rs.deniedCmdRegexes {
		if re.MatchString(cmd) {
			return true
		}
	}
	return false
}

// IsExternalPath checks if a path is outside the project directory
func IsExternalPath(path, projectDir string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return false
	}

	rel, err := filepath.Rel(absProjectDir, absPath)
	if err != nil {
		return true
	}

	// If the path starts with "..", it's outside the project
	return strings.HasPrefix(rel, "..")
}

// IsSafeCommand checks if a bash command is generally safe
// This is used as a heuristic for auto-approval in lenient modes
func IsSafeCommand(cmd string) bool {
	// List of safe read-only commands
	safeCommands := []string{
		"ls", "cat", "echo", "pwd", "which", "whereis",
		"git status", "git log", "git diff", "git branch",
		"env", "printenv", "uname", "whoami", "date",
		"grep", "find", "head", "tail", "wc",
	}

	// Unsafe patterns
	unsafePatterns := []string{
		"rm ", "rm -", "> ", ">>", "|", "curl", "wget",
		"chmod", "chown", "sudo", "su ", "exec",
		"eval", "source", ". ", "kill", "pkill",
		"mv ", "cp ", "dd ", "mkfs", "format",
	}

	cmdLower := strings.ToLower(strings.TrimSpace(cmd))

	// Check if it's a known safe command
	for _, safe := range safeCommands {
		if strings.HasPrefix(cmdLower, safe) {
			return true
		}
	}

	// Check for unsafe patterns
	for _, unsafe := range unsafePatterns {
		if strings.Contains(cmdLower, unsafe) {
			return false
		}
	}

	// Default to unsafe for unknown commands
	return false
}
