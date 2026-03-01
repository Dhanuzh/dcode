package permission

import (
	"context"
	"fmt"
)

// Mode defines how permissions are handled
type Mode string

const (
	// ModeAuto automatically approves all operations
	ModeAuto Mode = "auto"
	// ModePrompt asks the user for approval
	ModePrompt Mode = "prompt"
	// ModeDeny rejects all operations
	ModeDeny Mode = "deny"
)

// Action represents a type of operation that requires permission
type Action string

const (
	// ActionBash executes a bash command
	ActionBash Action = "bash"
	// ActionRead reads a file
	ActionRead Action = "read"
	// ActionWrite creates or overwrites a file
	ActionWrite Action = "write"
	// ActionEdit modifies an existing file
	ActionEdit Action = "edit"
	// ActionDelete removes a file
	ActionDelete Action = "delete"
	// ActionExecute runs an executable
	ActionExecute Action = "execute"
	// ActionNetwork makes a network request
	ActionNetwork Action = "network"
	// ActionExternalDir accesses files outside the project
	ActionExternalDir Action = "external_dir"
)

// Request represents a permission request
type Request struct {
	Action      Action
	Path        string // File path or command
	Description string // Human-readable description
	Metadata    map[string]interface{}
}

// Response represents the result of a permission check
type Response struct {
	Allowed bool
	Mode    Mode   // Which mode produced this decision
	Reason  string // Why this decision was made
}

// Checker is the interface for permission checking
type Checker interface {
	// Check verifies if an action is allowed
	Check(ctx context.Context, req *Request) (*Response, error)

	// SetMode changes the permission mode
	SetMode(mode Mode)

	// GetMode returns the current permission mode
	GetMode() Mode
}

// PromptFunc is called when user approval is needed
type // PromptFunc handles user approval for permission requests
PromptFunc func(ctx context.Context, req *Request) (bool, error)

// Config holds permission configuration
type Config struct {
	// DefaultMode is the fallback permission mode
	DefaultMode Mode

	// BashMode overrides the default for bash commands
	BashMode Mode

	// EditMode overrides the default for file edits
	EditMode Mode

	// WriteMode overrides the default for file writes
	WriteMode Mode

	// AllowedPaths are glob patterns for always-allowed paths
	AllowedPaths []string

	// DeniedPaths are glob patterns for always-denied paths
	DeniedPaths []string

	// AllowedCommands are regex patterns for allowed bash commands
	AllowedCommands []string

	// DeniedCommands are regex patterns for denied bash commands
	DeniedCommands []string

	// AllowExternalDir permits access outside the project directory
	AllowExternalDir bool

	// ProjectDir is the root project directory
	ProjectDir string

	// PromptFunc is called for user approval (required for ModePrompt)
	PromptFunc PromptFunc
}

// DefaultConfig returns a default permission configuration
func DefaultConfig(projectDir string) *Config {
	return &Config{
		DefaultMode:      ModePrompt,
		BashMode:         ModePrompt,
		EditMode:         ModePrompt,
		WriteMode:        ModePrompt,
		AllowedPaths:     []string{},
		DeniedPaths:      []string{},
		AllowedCommands:  []string{},
		DeniedCommands:   []string{},
		AllowExternalDir: false,
		ProjectDir:       projectDir,
		PromptFunc:       nil,
	}
}

// Error types
var (
	ErrPermissionDenied = fmt.Errorf("permission denied")
	ErrNoPromptFunc     = fmt.Errorf("prompt mode requires PromptFunc to be set")
)
