package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DetermineWorktreePath calculates the path for a worktree
func DetermineWorktreePath(projectPath, worktreeName string) string {
	projectName := filepath.Base(projectPath)
	sanitizedName := sanitizeBranchName(worktreeName)

	parentDir := filepath.Dir(projectPath)
	return filepath.Join(parentDir, fmt.Sprintf("%s-%s", projectName, sanitizedName))
}

// sanitizeBranchName converts branch name to filesystem-safe name
func sanitizeBranchName(name string) string {
	// Replace slashes with dashes
	name = strings.ReplaceAll(name, "/", "-")
	// Remove other problematic characters
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, " ", "-")
	return name
}

// IsGitRepo checks if a directory is a git repository
func IsGitRepo(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// GetCurrentBranch returns the current branch name
func GetCurrentBranch(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// WorktreeExists checks if a worktree with the given name exists
func WorktreeExists(worktreeName string) (bool, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	// Parse worktree list output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "branch ") {
			branch := strings.TrimPrefix(line, "branch refs/heads/")
			if branch == worktreeName {
				return true, nil
			}
		}
	}
	return false, nil
}

// CreateWorktree creates a new worktree
func CreateWorktree(path, branchName string, verbose bool) error {
	cmd := exec.Command("git", "worktree", "add", path, "-b", branchName)

	if verbose {
		fmt.Fprintf(os.Stderr, "+ git worktree add %s -b %s\n", path, branchName)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}
