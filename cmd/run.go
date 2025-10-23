package cmd

import (
	"fmt"
	"os"

	"github.com/jessedrelick/cage/pkg/runner"
	"github.com/spf13/cobra"
)

var (
	runPath       string
	runWorktree   string
	runNoWorktree bool
	runEnv        []string
	runVerbose    bool
)

var runCmd = &cobra.Command{
	Use:   "run [flags] [command...]",
	Short: "Run command in container",
	Long:  `Start a container and execute the specified command inside it.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		config := &runner.RunConfig{
			Path:       runPath,
			Worktree:   runWorktree,
			NoWorktree: runNoWorktree,
			Env:        runEnv,
			Verbose:    runVerbose,
			Command:    args,
		}

		if err := runner.Run(config); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVar(&runPath, "path", "", "Project path (default: pwd)")
	runCmd.Flags().StringVar(&runWorktree, "worktree", "", "Worktree name (creates if needed)")
	runCmd.Flags().BoolVar(&runNoWorktree, "no-worktree", false, "Skip worktree, use directory directly")
	runCmd.Flags().StringSliceVar(&runEnv, "env", []string{}, "Additional env vars (KEY=value)")
	runCmd.Flags().BoolVar(&runVerbose, "verbose", false, "Show all docker/git commands")
}
