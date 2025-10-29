package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/obra/packnplay/pkg/docker"
	"github.com/spf13/cobra"
)

type ContainerInfo struct {
	Names  string `json:"Names"`
	Status string `json:"Status"`
	Labels string `json:"Labels"`
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all packnplay-managed containers",
	Long:  `Display all running containers managed by packnplay.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Initialize Docker client
		dockerClient, err := docker.NewClient(false)
		if err != nil {
			return fmt.Errorf("failed to initialize docker: %w", err)
		}

		// Get all packnplay-managed containers
		output, err := dockerClient.Run(
			"ps",
			"--filter", "label=managed-by=packnplay",
			"--format", "{{json .}}",
		)
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}

		if output == "" {
			fmt.Println("No packnplay-managed containers running")
			return nil
		}

		// Parse JSON output
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		_, _ = fmt.Fprintln(w, "CONTAINER\tSTATUS\tPROJECT\tWORKTREE")

		// Docker outputs one JSON object per line
		lines := splitLines(output)
		for _, line := range lines {
			if line == "" {
				continue
			}

			var info ContainerInfo
			if err := json.Unmarshal([]byte(line), &info); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to parse container info: %v\n", err)
				continue
			}

			// Parse labels to extract project and worktree
			project, worktree := parseLabels(info.Labels)

			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				info.Names,
				info.Status,
				project,
				worktree,
			)
		}

		return w.Flush()
	},
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func parseLabels(labels string) (project, worktree string) {
	// Labels format: "label1=value1,label2=value2"
	pairs := splitByComma(labels)
	for _, pair := range pairs {
		kv := splitByEquals(pair)
		if len(kv) == 2 {
			switch kv[0] {
			case "packnplay-project":
				project = kv[1]
			case "packnplay-worktree":
				worktree = kv[1]
			}
		}
	}
	return
}

func splitByComma(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

func splitByEquals(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

func init() {
	rootCmd.AddCommand(listCmd)
}
