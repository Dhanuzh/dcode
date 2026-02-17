package session

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Snapshot manages a separate git repository for tracking file changes
// This enables undo/revert by taking snapshots before and after tool execution
type Snapshot struct {
	gitDir  string // Path to the snapshot .git directory
	workDir string // Path to the working tree
}

// FileDiff represents a file diff between two snapshots
type FileDiff struct {
	File      string `json:"file"`
	Before    string `json:"before"`
	After     string `json:"after"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Status    string `json:"status"` // "added", "deleted", "modified"
}

// NewSnapshot creates a new snapshot manager
func NewSnapshot(dataDir, workDir string) *Snapshot {
	return &Snapshot{
		gitDir:  filepath.Join(dataDir, "snapshot"),
		workDir: workDir,
	}
}

// Init initializes the snapshot git repository if it doesn't exist
func (s *Snapshot) Init() error {
	if err := os.MkdirAll(s.gitDir, 0755); err != nil {
		return fmt.Errorf("failed to create snapshot dir: %w", err)
	}

	// Check if already initialized
	if _, err := os.Stat(filepath.Join(s.gitDir, "HEAD")); err == nil {
		return nil // Already initialized
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = s.workDir
	cmd.Env = append(os.Environ(),
		"GIT_DIR="+s.gitDir,
		"GIT_WORK_TREE="+s.workDir,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git init failed: %s: %w", string(out), err)
	}

	// Disable autocrlf
	cmd2 := exec.Command("git", "--git-dir", s.gitDir, "config", "core.autocrlf", "false")
	cmd2.Dir = s.workDir
	_ = cmd2.Run()

	return nil
}

// Track stages all files and writes a tree object, returning the tree hash
func (s *Snapshot) Track() (string, error) {
	if err := s.Init(); err != nil {
		return "", err
	}

	// Stage all files
	cmd := exec.Command("git", "--git-dir", s.gitDir, "--work-tree", s.workDir, "add", ".")
	cmd.Dir = s.workDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git add failed: %s: %w", string(out), err)
	}

	// Write tree
	cmd = exec.Command("git", "--git-dir", s.gitDir, "--work-tree", s.workDir, "write-tree")
	cmd.Dir = s.workDir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git write-tree failed: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

// Patch returns the list of files changed since the given snapshot hash
func (s *Snapshot) Patch(hash string) ([]string, error) {
	// Stage current state
	cmd := exec.Command("git", "--git-dir", s.gitDir, "--work-tree", s.workDir, "add", ".")
	cmd.Dir = s.workDir
	_ = cmd.Run()

	// Get changed files
	cmd = exec.Command("git",
		"-c", "core.autocrlf=false",
		"-c", "core.quotepath=false",
		"--git-dir", s.gitDir,
		"--work-tree", s.workDir,
		"diff", "--no-ext-diff", "--name-only", hash, "--", ".",
	)
	cmd.Dir = s.workDir
	out, err := cmd.Output()
	if err != nil {
		return nil, nil // graceful failure
	}

	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, filepath.Join(s.workDir, line))
		}
	}
	return files, nil
}

// Restore restores the working tree to a specific snapshot via read-tree + checkout-index
func (s *Snapshot) Restore(hash string) error {
	script := fmt.Sprintf(
		"git --git-dir %s --work-tree %s read-tree %s && git --git-dir %s --work-tree %s checkout-index -a -f",
		s.gitDir, s.workDir, hash, s.gitDir, s.workDir,
	)
	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = s.workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("restore failed: %s: %w", string(out), err)
	}
	return nil
}

// Revert reverts specific files from a list of patches
func (s *Snapshot) Revert(patches []SnapshotPatch) error {
	reverted := make(map[string]bool)

	for _, patch := range patches {
		for _, file := range patch.Files {
			if reverted[file] {
				continue
			}

			cmd := exec.Command("git",
				"--git-dir", s.gitDir,
				"--work-tree", s.workDir,
				"checkout", patch.Hash, "--", file,
			)
			cmd.Dir = s.workDir
			if err := cmd.Run(); err != nil {
				// Check if file existed in the snapshot
				relPath, _ := filepath.Rel(s.workDir, file)
				checkCmd := exec.Command("git",
					"--git-dir", s.gitDir,
					"--work-tree", s.workDir,
					"ls-tree", patch.Hash, "--", relPath,
				)
				checkCmd.Dir = s.workDir
				checkOut, checkErr := checkCmd.Output()
				if checkErr == nil && strings.TrimSpace(string(checkOut)) != "" {
					// File existed but checkout failed - keep it
					continue
				}
				// File didn't exist in snapshot - delete it
				os.Remove(file)
			}

			reverted[file] = true
		}
	}
	return nil
}

// Diff returns the full diff text between a snapshot and current state
func (s *Snapshot) Diff(hash string) (string, error) {
	// Stage current state
	cmd := exec.Command("git", "--git-dir", s.gitDir, "--work-tree", s.workDir, "add", ".")
	cmd.Dir = s.workDir
	_ = cmd.Run()

	cmd = exec.Command("git",
		"-c", "core.autocrlf=false",
		"-c", "core.quotepath=false",
		"--git-dir", s.gitDir,
		"--work-tree", s.workDir,
		"diff", "--no-ext-diff", hash, "--", ".",
	)
	cmd.Dir = s.workDir
	out, err := cmd.Output()
	if err != nil {
		return "", nil // graceful failure
	}
	return strings.TrimSpace(string(out)), nil
}

// DiffFull returns structured file diffs between two snapshots
func (s *Snapshot) DiffFull(from, to string) ([]FileDiff, error) {
	var result []FileDiff

	// Get file statuses
	statusMap := make(map[string]string)
	cmd := exec.Command("git",
		"-c", "core.autocrlf=false",
		"-c", "core.quotepath=false",
		"--git-dir", s.gitDir,
		"--work-tree", s.workDir,
		"diff", "--no-ext-diff", "--name-status", "--no-renames", from, to, "--", ".",
	)
	cmd.Dir = s.workDir
	statusOut, err := cmd.Output()
	if err != nil {
		return nil, nil
	}

	for _, line := range strings.Split(strings.TrimSpace(string(statusOut)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		code := parts[0]
		file := parts[1]
		if strings.HasPrefix(code, "A") {
			statusMap[file] = "added"
		} else if strings.HasPrefix(code, "D") {
			statusMap[file] = "deleted"
		} else {
			statusMap[file] = "modified"
		}
	}

	// Get numstat for additions/deletions counts
	cmd = exec.Command("git",
		"-c", "core.autocrlf=false",
		"-c", "core.quotepath=false",
		"--git-dir", s.gitDir,
		"--work-tree", s.workDir,
		"diff", "--no-ext-diff", "--no-renames", "--numstat", from, to, "--", ".",
	)
	cmd.Dir = s.workDir
	numstatOut, err := cmd.Output()
	if err != nil {
		return nil, nil
	}

	for _, line := range strings.Split(strings.TrimSpace(string(numstatOut)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}

		additions, _ := strconv.Atoi(parts[0])
		deletions, _ := strconv.Atoi(parts[1])
		file := parts[2]
		isBinary := parts[0] == "-" && parts[1] == "-"

		var before, after string
		if !isBinary {
			// Get before content
			showCmd := exec.Command("git",
				"-c", "core.autocrlf=false",
				"--git-dir", s.gitDir,
				"--work-tree", s.workDir,
				"show", from+":"+file,
			)
			showCmd.Dir = s.workDir
			if out, err := showCmd.Output(); err == nil {
				before = string(out)
			}

			// Get after content
			showCmd = exec.Command("git",
				"-c", "core.autocrlf=false",
				"--git-dir", s.gitDir,
				"--work-tree", s.workDir,
				"show", to+":"+file,
			)
			showCmd.Dir = s.workDir
			if out, err := showCmd.Output(); err == nil {
				after = string(out)
			}
		}

		status := statusMap[file]
		if status == "" {
			status = "modified"
		}

		result = append(result, FileDiff{
			File:      file,
			Before:    before,
			After:     after,
			Additions: additions,
			Deletions: deletions,
			Status:    status,
		})
	}

	return result, nil
}

// Cleanup runs git gc on the snapshot repository
func (s *Snapshot) Cleanup() error {
	cmd := exec.Command("git",
		"--git-dir", s.gitDir,
		"--work-tree", s.workDir,
		"gc", "--prune=7.days",
	)
	cmd.Dir = s.workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gc failed: %s: %w", string(out), err)
	}
	return nil
}

// SnapshotPatch represents a recorded change set
type SnapshotPatch struct {
	Hash  string   `json:"hash"`
	Files []string `json:"files"`
}
