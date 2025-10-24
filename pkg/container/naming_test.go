package container

import (
	"testing"
)

func TestGenerateContainerName(t *testing.T) {
	tests := []struct {
		name         string
		projectPath  string
		worktreeName string
		want         string
	}{
		{
			name:         "basic naming",
			projectPath:  "/home/user/myproject",
			worktreeName: "main",
			want:         "packnplay-myproject-main",
		},
		{
			name:         "sanitized worktree name",
			projectPath:  "/home/user/myproject",
			worktreeName: "feature/auth",
			want:         "packnplay-myproject-feature-auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateContainerName(tt.projectPath, tt.worktreeName)
			if got != tt.want {
				t.Errorf("GenerateContainerName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateLabels(t *testing.T) {
	labels := GenerateLabels("myproject", "feature-auth")

	if labels["managed-by"] != "packnplay" {
		t.Errorf("managed-by label = %v, want packnplay", labels["managed-by"])
	}

	if labels["packnplay-project"] != "myproject" {
		t.Errorf("packnplay-project label = %v, want myproject", labels["packnplay-project"])
	}

	if labels["packnplay-worktree"] != "feature-auth" {
		t.Errorf("packnplay-worktree label = %v, want feature-auth", labels["packnplay-worktree"])
	}
}
