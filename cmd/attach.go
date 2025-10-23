package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/jessedrelick/cage/pkg/container"
	"github.com/jessedrelick/cage/pkg/docker"
	"github.com/spf13/cobra"
)

var (
	attachPath     string
	attachWorktree string
)

var attachCmd = &cobra.Command{
	Use:   "attach [flags]",
	Short: "Attach to running container",
	Long:  `Attach to an existing running container with an interactive shell.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine working directory
		workDir := attachPath
		if workDir == "" {
			var err error
			workDir, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}
		}

		workDir, err := filepath.Abs(workDir)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}

		// Determine worktree name
		worktreeName := attachWorktree
		if worktreeName == "" {
			return fmt.Errorf("--worktree flag is required for attach")
		}

		// Generate container name
		containerName := container.GenerateContainerName(workDir, worktreeName)

		// Initialize Docker client
		dockerClient, err := docker.NewClient(false)
		if err != nil {
			return fmt.Errorf("failed to initialize docker: %w", err)
		}

		// Check if container is running
		output, err := dockerClient.Run("ps", "--filter", fmt.Sprintf("name=%s", containerName), "--format", "{{.Names}}")
		if err != nil {
			return fmt.Errorf("failed to check container status: %w", err)
		}

		if strings.TrimSpace(output) != containerName {
			return fmt.Errorf("no running container found for worktree '%s'", worktreeName)
		}

		// Execute docker exec with interactive shell
		cmdPath, err := exec.LookPath(dockerClient.Command())
		if err != nil {
			return fmt.Errorf("failed to find docker command: %w", err)
		}

		argv := []string{
			filepath.Base(cmdPath),
			"exec",
			"-it",
			containerName,
			"/bin/bash",
		}

		return syscall.Exec(cmdPath, argv, os.Environ())
	},
}

func init() {
	rootCmd.AddCommand(attachCmd)

	attachCmd.Flags().StringVar(&attachPath, "path", "", "Project path (default: pwd)")
	attachCmd.Flags().StringVar(&attachWorktree, "worktree", "", "Worktree name")
}
