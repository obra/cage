package cmd

import (
	"fmt"
	"strings"

	"github.com/obra/packnplay/pkg/config"
	"github.com/spf13/cobra"
)

var (
	configureSection string
	configureVerbose bool
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Edit packnplay configuration",
	Long: `Interactive configuration editor for packnplay settings.

Safely edits configuration while preserving any existing settings
not shown in the user interface (custom env configs, advanced settings, etc.).

Use --section to edit specific configuration sections:
  --section=runtime      Container runtime selection
  --section=credentials  Default credential mounting
  --section=container    Default container and update settings
  --section=all          All sections (default)

This command preserves all existing configuration values not displayed
in the interactive forms, ensuring manual edits and advanced settings
are never lost during configuration updates.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInteractiveConfigure(configureSection, configureVerbose)
	},
}

func runInteractiveConfigure(section string, verbose bool) error {
	configPath := config.GetConfigPath()

	if verbose {
		fmt.Printf("Editing config: %s\n", configPath)
		fmt.Printf("Section: %s\n", section)
	}

	// Load existing config to show current values
	existingConfig, err := config.LoadExistingOrEmpty(configPath)
	if err != nil {
		return fmt.Errorf("failed to load existing config: %w", err)
	}

	// Run appropriate configuration sections
	switch strings.ToLower(section) {
	case "runtime":
		return configureRuntime(existingConfig, configPath, verbose)
	case "credentials":
		return configureCredentials(existingConfig, configPath, verbose)
	case "container":
		return configureContainer(existingConfig, configPath, verbose)
	case "all", "":
		return configureAll(existingConfig, configPath, verbose)
	default:
		return fmt.Errorf("unknown section: %s (valid: runtime, credentials, container, all)", section)
	}
}

// Placeholder functions to be implemented
func configureRuntime(existing *config.Config, configPath string, verbose bool) error {
	return fmt.Errorf("not implemented")
}

func configureCredentials(existing *config.Config, configPath string, verbose bool) error {
	return fmt.Errorf("not implemented")
}

func configureContainer(existing *config.Config, configPath string, verbose bool) error {
	return fmt.Errorf("not implemented")
}

func configureAll(existing *config.Config, configPath string, verbose bool) error {
	return fmt.Errorf("not implemented")
}

func init() {
	rootCmd.AddCommand(configureCmd)
	configureCmd.Flags().StringVar(&configureSection, "section", "all", "Configuration section to edit (runtime, credentials, container, all)")
	configureCmd.Flags().BoolVarP(&configureVerbose, "verbose", "v", false, "Show detailed output")
}