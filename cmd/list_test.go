package cmd

import (
	"testing"
)

func TestParseLabels(t *testing.T) {
	tests := []struct {
		name            string
		labels          string
		wantProject     string
		wantWorktree    string
	}{
		{
			name:         "basic labels",
			labels:       "managed-by=packnplay,packnplay-project=myproject,packnplay-worktree=main",
			wantProject:  "myproject",
			wantWorktree: "main",
		},
		{
			name:         "empty labels",
			labels:       "",
			wantProject:  "",
			wantWorktree: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProject, gotWorktree := parseLabels(tt.labels)
			if gotProject != tt.wantProject {
				t.Errorf("parseLabels() project = %v, want %v", gotProject, tt.wantProject)
			}
			if gotWorktree != tt.wantWorktree {
				t.Errorf("parseLabels() worktree = %v, want %v", gotWorktree, tt.wantWorktree)
			}
		})
	}
}

func TestParseLabelsWithLaunchInfo(t *testing.T) {
	labels := "managed-by=packnplay,packnplay-project=myproject,packnplay-worktree=main,packnplay-host-path=/Users/jesse/myproject,packnplay-launch-command=packnplay run --worktree main --git-creds claude code"

	project, worktree, hostPath, launchCommand := parseLabelsWithLaunchInfo(labels)

	if project != "myproject" {
		t.Errorf("parseLabelsWithLaunchInfo() project = %v, want myproject", project)
	}

	if worktree != "main" {
		t.Errorf("parseLabelsWithLaunchInfo() worktree = %v, want main", worktree)
	}

	if hostPath != "/Users/jesse/myproject" {
		t.Errorf("parseLabelsWithLaunchInfo() hostPath = %v, want /Users/jesse/myproject", hostPath)
	}

	expectedCommand := "packnplay run --worktree main --git-creds claude code"
	if launchCommand != expectedCommand {
		t.Errorf("parseLabelsWithLaunchInfo() launchCommand = %v, want %v", launchCommand, expectedCommand)
	}
}

func TestParseLabelsWithLaunchInfoBackwardCompatibility(t *testing.T) {
	// Test with old labels that don't have launch info
	labels := "managed-by=packnplay,packnplay-project=myproject,packnplay-worktree=main"

	project, worktree, hostPath, launchCommand := parseLabelsWithLaunchInfo(labels)

	if project != "myproject" {
		t.Errorf("parseLabelsWithLaunchInfo() project = %v, want myproject", project)
	}

	if worktree != "main" {
		t.Errorf("parseLabelsWithLaunchInfo() worktree = %v, want main", worktree)
	}

	// Should return empty strings for missing labels
	if hostPath != "" {
		t.Errorf("parseLabelsWithLaunchInfo() hostPath = %v, want empty string", hostPath)
	}

	if launchCommand != "" {
		t.Errorf("parseLabelsWithLaunchInfo() launchCommand = %v, want empty string", launchCommand)
	}
}