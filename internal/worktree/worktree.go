// Package worktree provides git worktree management for dcode.
// Worktrees allow running isolated coding sessions on separate git branches,
// mirroring opencode's worktree feature.
package worktree

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Worktree represents a git worktree
type Worktree struct {
	Name      string    `json:"name"`
	Branch    string    `json:"branch"`
	Path      string    `json:"path"`
	SessionID string    `json:"session_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"` // "ready", "failed", "creating"
}

// Manager manages git worktrees for dcode
type Manager struct {
	mu       sync.RWMutex
	baseDir  string // directory where worktrees are stored
	repoRoot string // root of the git repository
	trees    map[string]*Worktree
}

// NewManager creates a new worktree manager
func NewManager(repoRoot, baseDir string) *Manager {
	return &Manager{
		baseDir:  baseDir,
		repoRoot: repoRoot,
		trees:    make(map[string]*Worktree),
	}
}

// IsGitRepo checks if the given directory is a git repository
func IsGitRepo(dir string) bool {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// GetRepoRoot returns the root of the git repository for the given directory
func GetRepoRoot(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// GetCurrentBranch returns the current branch name
func GetCurrentBranch(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Create creates a new worktree for the given branch
func (m *Manager) Create(ctx context.Context, name, branch string) (*Worktree, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if worktree already exists
	if existing, ok := m.trees[name]; ok {
		return existing, nil
	}

	// Create the worktree directory
	worktreePath := filepath.Join(m.baseDir, name)

	wt := &Worktree{
		Name:      name,
		Branch:    branch,
		Path:      worktreePath,
		CreatedAt: time.Now(),
		Status:    "creating",
	}
	m.trees[name] = wt

	// Check if branch already exists
	checkCmd := exec.CommandContext(ctx, "git", "-C", m.repoRoot, "rev-parse", "--verify", branch)
	branchExists := checkCmd.Run() == nil

	var cmd *exec.Cmd
	if branchExists {
		// Checkout existing branch
		cmd = exec.CommandContext(ctx, "git", "-C", m.repoRoot, "worktree", "add", worktreePath, branch)
	} else {
		// Create new branch
		cmd = exec.CommandContext(ctx, "git", "-C", m.repoRoot, "worktree", "add", "-b", branch, worktreePath)
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		wt.Status = "failed"
		return nil, fmt.Errorf("failed to create worktree: %s\n%s", err, string(out))
	}

	wt.Status = "ready"
	return wt, nil
}

// Remove removes a worktree
func (m *Manager) Remove(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	wt, ok := m.trees[name]
	if !ok {
		return fmt.Errorf("worktree not found: %s", name)
	}

	cmd := exec.CommandContext(ctx, "git", "-C", m.repoRoot, "worktree", "remove", wt.Path, "--force")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove worktree: %s\n%s", err, string(out))
	}

	// Also try to remove the directory if it still exists
	_ = os.RemoveAll(wt.Path)

	delete(m.trees, name)
	return nil
}

// List returns all managed worktrees
func (m *Manager) List() []*Worktree {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Worktree, 0, len(m.trees))
	for _, wt := range m.trees {
		result = append(result, wt)
	}
	return result
}

// Get returns a worktree by name
func (m *Manager) Get(name string) (*Worktree, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	wt, ok := m.trees[name]
	return wt, ok
}

// GetGitWorktrees returns all git worktrees for the repository
func GetGitWorktrees(repoRoot string) ([]GitWorktreeInfo, error) {
	cmd := exec.Command("git", "-C", repoRoot, "worktree", "list", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	var trees []GitWorktreeInfo
	var current GitWorktreeInfo

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			if current.Path != "" {
				trees = append(trees, current)
				current = GitWorktreeInfo{}
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			current.Path = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "HEAD ") {
			current.HEAD = strings.TrimPrefix(line, "HEAD ")
		} else if strings.HasPrefix(line, "branch ") {
			branch := strings.TrimPrefix(line, "branch ")
			// Strip refs/heads/ prefix
			branch = strings.TrimPrefix(branch, "refs/heads/")
			current.Branch = branch
		} else if line == "bare" {
			current.IsBare = true
		}
	}

	// Don't forget the last entry
	if current.Path != "" {
		trees = append(trees, current)
	}

	return trees, nil
}

// GitWorktreeInfo contains information from git worktree list --porcelain
type GitWorktreeInfo struct {
	Path   string
	HEAD   string
	Branch string
	IsBare bool
}

// CleanupOrphanedWorktrees removes worktrees that no longer have a corresponding directory
func CleanupOrphanedWorktrees(ctx context.Context, repoRoot string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoRoot, "worktree", "prune")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to prune worktrees: %s\n%s", err, string(out))
	}
	return nil
}

// MergeWorktree merges a worktree branch back into the main branch
func MergeWorktree(ctx context.Context, repoRoot, sourceBranch, targetBranch string) error {
	// Checkout target branch
	checkoutCmd := exec.CommandContext(ctx, "git", "-C", repoRoot, "checkout", targetBranch)
	if out, err := checkoutCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to checkout %s: %s\n%s", targetBranch, err, string(out))
	}

	// Merge the source branch
	mergeCmd := exec.CommandContext(ctx, "git", "-C", repoRoot, "merge", "--no-ff", sourceBranch,
		"-m", fmt.Sprintf("Merge dcode worktree branch '%s'", sourceBranch))
	if out, err := mergeCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to merge %s into %s: %s\n%s", sourceBranch, targetBranch, err, string(out))
	}

	return nil
}

// GetDiffFromMain returns the diff of changes in the worktree compared to main
func GetDiffFromMain(ctx context.Context, worktreePath, baseBranch string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", worktreePath, "diff", baseBranch+"..HEAD", "--stat")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}
	return string(out), nil
}
