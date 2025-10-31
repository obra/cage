package cmd

import (
	"github.com/obra/packnplay/pkg/runner"
	"github.com/spf13/cobra"
)

var refreshVerbose bool

var refreshCmd = &cobra.Command{
	Use:   "refresh-default-container",
	Short: "Pull latest version of default container image",
	Long:  `Force pull the latest version of the packnplay default container image to get updated tools and dependencies.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runner.RefreshDefaultContainer(refreshVerbose)
	},
}

func init() {
	rootCmd.AddCommand(refreshCmd)
	refreshCmd.Flags().BoolVarP(&refreshVerbose, "verbose", "v", false, "Show detailed output")
}