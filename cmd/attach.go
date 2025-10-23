package cmd

import (
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
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: implement
		cmd.Println("attach command not yet implemented")
	},
}

func init() {
	rootCmd.AddCommand(attachCmd)

	attachCmd.Flags().StringVar(&attachPath, "path", "", "Project path (default: pwd)")
	attachCmd.Flags().StringVar(&attachWorktree, "worktree", "", "Worktree name")
}
