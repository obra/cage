package cmd

import (
	"fmt"
	"os"

	"github.com/obra/packnplay/pkg/config"
	"github.com/obra/packnplay/pkg/runner"
	"github.com/spf13/cobra"
)

var (
	runPath       string
	runWorktree   string
	runNoWorktree bool
	runEnv        []string
	runVerbose    bool
	// Credential flags
	runGitCreds   *bool
	runGHCreds    *bool
	runGPGCreds   *bool
	runNPMCreds   *bool
	runAllCreds   bool
)

var runCmd = &cobra.Command{
	Use:   "run [flags] [command...]",
	Short: "Run command in container",
	Long:  `Start a container and execute the specified command inside it.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration (interactive setup on first run)
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Determine which credentials to use (flags override config)
		creds := cfg.DefaultCredentials

		// Check if flags were explicitly set
		if cmd.Flags().Changed("git-creds") {
			creds.Git = *runGitCreds
		}
		if cmd.Flags().Changed("gh-creds") {
			creds.GH = *runGHCreds
		}
		if cmd.Flags().Changed("gpg-creds") {
			creds.GPG = *runGPGCreds
		}
		if cmd.Flags().Changed("npm-creds") {
			creds.NPM = *runNPMCreds
		}
		if runAllCreds {
			creds.Git = true
			creds.GH = true
			creds.GPG = true
			creds.NPM = true
		}

		runConfig := &runner.RunConfig{
			Path:        runPath,
			Worktree:    runWorktree,
			NoWorktree:  runNoWorktree,
			Env:         runEnv,
			Verbose:     runVerbose,
			Command:     args,
			Credentials: creds,
		}

		if err := runner.Run(runConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Disable flag parsing after first positional arg (the command to run)
	// This allows the command and its args to be passed through without interpretation
	runCmd.Flags().SetInterspersed(false)

	runCmd.Flags().StringVar(&runPath, "path", "", "Project path (default: pwd)")
	runCmd.Flags().StringVar(&runWorktree, "worktree", "", "Worktree name (creates if needed)")
	runCmd.Flags().BoolVar(&runNoWorktree, "no-worktree", false, "Skip worktree, use directory directly")
	runCmd.Flags().StringSliceVar(&runEnv, "env", []string{}, "Additional env vars (KEY=value)")
	runCmd.Flags().BoolVar(&runVerbose, "verbose", false, "Show all docker/git commands")

	// Credential flags (use pointers so we can detect if they were explicitly set)
	runGitCreds = runCmd.Flags().Bool("git-creds", false, "Mount git credentials (~/.gitconfig, ~/.ssh)")
	runGHCreds = runCmd.Flags().Bool("gh-creds", false, "Mount GitHub CLI credentials")
	runGPGCreds = runCmd.Flags().Bool("gpg-creds", false, "Mount GPG credentials for commit signing")
	runNPMCreds = runCmd.Flags().Bool("npm-creds", false, "Mount npm credentials")
	runCmd.Flags().BoolVar(&runAllCreds, "all-creds", false, "Mount all available credentials")
}
