package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cage",
	Short: "Launch commands in isolated Docker containers",
	Long: `Cage runs commands (like Claude Code) inside isolated Docker containers
with automated worktree and dev container management.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
