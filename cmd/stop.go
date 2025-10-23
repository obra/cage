package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jessedrelick/cage/pkg/container"
	"github.com/jessedrelick/cage/pkg/docker"
	"github.com/spf13/cobra"
)

var (
	stopPath     string
	stopWorktree string
)

var stopCmd = &cobra.Command{
	Use:   "stop [flags]",
	Short: "Stop container",
	Long:  `Stop the container for the specified project/worktree.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine working directory
		workDir := stopPath
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
		worktreeName := stopWorktree
		if worktreeName == "" {
			return fmt.Errorf("--worktree flag is required for stop")
		}

		// Generate container name
		containerName := container.GenerateContainerName(workDir, worktreeName)

		// Initialize Docker client
		dockerClient, err := docker.NewClient(false)
		if err != nil {
			return fmt.Errorf("failed to initialize docker: %w", err)
		}

		// Stop container
		fmt.Printf("Stopping container %s...\n", containerName)
		_, err = dockerClient.Run("stop", containerName)
		if err != nil {
			return fmt.Errorf("failed to stop container: %w", err)
		}

		// Remove container
		_, err = dockerClient.Run("rm", containerName)
		if err != nil {
			return fmt.Errorf("failed to remove container: %w", err)
		}

		fmt.Printf("Container %s stopped and removed\n", containerName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)

	stopCmd.Flags().StringVar(&stopPath, "path", "", "Project path (default: pwd)")
	stopCmd.Flags().StringVar(&stopWorktree, "worktree", "", "Worktree name")
}
