package cmd

import (
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all cage-managed containers",
	Long:  `Display all running containers managed by cage.`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: implement
		cmd.Println("list command not yet implemented")
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
