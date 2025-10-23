package cmd

import (
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
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: implement
		cmd.Println("stop command not yet implemented")
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)

	stopCmd.Flags().StringVar(&stopPath, "path", "", "Project path (default: pwd)")
	stopCmd.Flags().StringVar(&stopWorktree, "worktree", "", "Worktree name")
}
