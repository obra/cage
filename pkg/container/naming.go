package container

import (
	"fmt"
	"path/filepath"
	"strings"
)

// GenerateContainerName creates a container name from project and worktree
func GenerateContainerName(projectPath, worktreeName string) string {
	projectName := filepath.Base(projectPath)
	sanitizedWorktree := sanitizeName(worktreeName)
	return fmt.Sprintf("cage-%s-%s", projectName, sanitizedWorktree)
}

// sanitizeName converts a name to docker-compatible format
func sanitizeName(name string) string {
	// Docker container names: [a-zA-Z0-9][a-zA-Z0-9_.-]*
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, ":", "-")
	return name
}

// GenerateLabels creates Docker labels for cage-managed containers
func GenerateLabels(projectName, worktreeName string) map[string]string {
	return map[string]string{
		"managed-by":    "cage",
		"cage-project":  projectName,
		"cage-worktree": worktreeName,
	}
}

// LabelsToArgs converts label map to docker --label args
func LabelsToArgs(labels map[string]string) []string {
	args := make([]string, 0, len(labels)*2)
	for k, v := range labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", k, v))
	}
	return args
}
