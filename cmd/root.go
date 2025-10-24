package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "packnplay",
	Short: "Launch commands in isolated Docker containers",
	Long: `packnplay runs commands (like Claude Code) inside isolated Docker containers
with automated worktree and dev container management.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
