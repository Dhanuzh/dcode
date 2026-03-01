package permission

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
)

// Engine implements the permission checking logic
type Engine struct {
	mu       sync.RWMutex
	config   *Config
	ruleSet  *RuleSet
	mode     Mode
	cache    map[string]*Response // Cache recent decisions
	cacheMax int
}

// NewEngine creates a new permission engine
func NewEngine(cfg *Config) (*Engine, error) {
	if cfg == nil {
		cfg = DefaultConfig(".")
	}

	ruleSet, err := NewRuleSet(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create ruleset: %w", err)
	}

	return &Engine{
		config:   cfg,
		ruleSet:  ruleSet,
		mode:     cfg.DefaultMode,
		cache:    make(map[string]*Response),
		cacheMax: 1000, // Limit cache size
	}, nil
}

// AskPermission prompts the user for approval
func (e *Engine) AskPermission(ctx context.Context, req *Request) (bool, error) {
	if e.config.PromptFunc == nil {
		return false, ErrNoPromptFunc
	}
	return e.config.PromptFunc(ctx, req)
}
func (e *Engine) Check(ctx context.Context, req *Request) (*Response, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s:%s", req.Action, req.Path, req.Description)
	if cached, ok := e.cache[cacheKey]; ok {
		return cached, nil
	}

	// Determine which mode to use for this action
	mode := e.getModeForAction(req.Action)

	// Auto mode - allow everything (with some safety checks)
	if mode == ModeAuto {
		// Still deny obviously dangerous operations
		if req.Action == ActionBash && !IsSafeCommand(req.Path) {
			// Check if explicitly denied
			if e.ruleSet.IsCommandDenied(req.Path) {
				resp := &Response{
					Allowed: false,
					Mode:    mode,
					Reason:  "Command matches denied pattern",
				}
				e.cacheResponse(cacheKey, resp)
				return resp, nil
			}
		}

		resp := &Response{
			Allowed: true,
			Mode:    mode,
			Reason:  "Auto-approved in auto mode",
		}
		e.cacheResponse(cacheKey, resp)
		return resp, nil
	}

	// Deny mode - reject everything
	if mode == ModeDeny {
		resp := &Response{
			Allowed: false,
			Mode:    mode,
			Reason:  "Denied by deny mode",
		}
		e.cacheResponse(cacheKey, resp)
		return resp, nil
	}

	// Prompt mode - check rules then prompt if needed
	if mode == ModePrompt {
		// Check denied patterns first
		if allowed, reason := e.checkDeniedRules(req); !allowed {
			resp := &Response{
				Allowed: false,
				Mode:    mode,
				Reason:  reason,
			}
			e.cacheResponse(cacheKey, resp)
			return resp, nil
		}

		// Check allowed patterns
		if allowed, reason := e.checkAllowedRules(req); allowed {
			resp := &Response{
				Allowed: true,
				Mode:    mode,
				Reason:  reason,
			}
			e.cacheResponse(cacheKey, resp)
			return resp, nil
		}

		// Need to prompt user
		if e.config.PromptFunc == nil {
			return nil, ErrNoPromptFunc
		}

		// Release lock before prompting (may take time)
		e.mu.RUnlock()
		allowed, err := e.config.PromptFunc(ctx, req)
		e.mu.RLock()

		if err != nil {
			return nil, err
		}

		resp := &Response{
			Allowed: allowed,
			Mode:    mode,
			Reason:  "User decision",
		}
		e.cacheResponse(cacheKey, resp)
		return resp, nil
	}

	// Unknown mode
	return &Response{
		Allowed: false,
		Mode:    mode,
		Reason:  "Unknown permission mode",
	}, nil
}

// SetMode changes the global permission mode
func (e *Engine) SetMode(mode Mode) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.mode = mode
	e.cache = make(map[string]*Response) // Clear cache on mode change
}

// GetMode returns the current permission mode
func (e *Engine) GetMode() Mode {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.mode
}

// SetActionMode sets the mode for a specific action
func (e *Engine) SetActionMode(action Action, mode Mode) {
	e.mu.Lock()
	defer e.mu.Unlock()

	switch action {
	case ActionBash:
		e.config.BashMode = mode
	case ActionEdit:
		e.config.EditMode = mode
	case ActionWrite:
		e.config.WriteMode = mode
	}

	e.cache = make(map[string]*Response) // Clear cache
}

// getModeForAction returns the appropriate mode for an action
func (e *Engine) getModeForAction(action Action) Mode {
	switch action {
	case ActionBash:
		if e.config.BashMode != "" {
			return e.config.BashMode
		}
	case ActionEdit:
		if e.config.EditMode != "" {
			return e.config.EditMode
		}
	case ActionWrite:
		if e.config.WriteMode != "" {
			return e.config.WriteMode
		}
	}
	return e.mode
}

// checkDeniedRules checks if a request matches any deny rules
func (e *Engine) checkDeniedRules(req *Request) (bool, string) {
	switch req.Action {
	case ActionBash:
		if e.ruleSet.IsCommandDenied(req.Path) {
			return false, "Command matches denied pattern"
		}

	case ActionRead, ActionWrite, ActionEdit, ActionDelete:
		if e.ruleSet.IsPathDenied(req.Path) {
			return false, "Path matches denied pattern"
		}

		// Check external directory access
		if !e.config.AllowExternalDir && IsExternalPath(req.Path, e.config.ProjectDir) {
			return false, "Access to external directory not allowed"
		}
	}

	return true, ""
}

// checkAllowedRules checks if a request matches any allow rules
func (e *Engine) checkAllowedRules(req *Request) (bool, string) {
	switch req.Action {
	case ActionBash:
		if len(e.ruleSet.allowedCmdRegexes) > 0 {
			if e.ruleSet.IsCommandAllowed(req.Path) {
				return true, "Command matches allowed pattern"
			}
		}

		// Auto-allow safe commands in prompt mode
		if IsSafeCommand(req.Path) {
			return true, "Command is considered safe"
		}

	case ActionRead, ActionWrite, ActionEdit, ActionDelete:
		if len(e.ruleSet.allowedPathGlobs) > 0 {
			if e.ruleSet.IsPathAllowed(req.Path) {
				return true, "Path matches allowed pattern"
			}
		}

		// Auto-allow operations within project directory
		if !IsExternalPath(req.Path, e.config.ProjectDir) {
			// Check if path is in .git, .env, etc. (sensitive files)
			basePath := filepath.Base(req.Path)
			if !isSensitiveFile(basePath) {
				return true, "File is within project and not sensitive"
			}
		}
	}

	return false, ""
}

// cacheResponse stores a response in the cache
func (e *Engine) cacheResponse(key string, resp *Response) {
	if len(e.cache) >= e.cacheMax {
		// Simple cache eviction - clear half
		for k := range e.cache {
			delete(e.cache, k)
			if len(e.cache) < e.cacheMax/2 {
				break
			}
		}
	}
	e.cache[key] = resp
}

// isSensitiveFile checks if a file is sensitive (credentials, etc.)
func isSensitiveFile(filename string) bool {
	sensitive := []string{
		".env", ".env.local", ".env.production",
		"credentials", "secrets", "id_rsa", "id_ed25519",
		".aws/credentials", ".ssh/id_", "token",
		".npmrc", ".pypirc", ".netrc",
	}

	filenameLower := filepath.ToSlash(filename)
	for _, s := range sensitive {
		if filepath.Base(filenameLower) == s {
			return true
		}
	}

	return false
}

// ClearCache clears the permission cache
func (e *Engine) ClearCache() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cache = make(map[string]*Response)
}
