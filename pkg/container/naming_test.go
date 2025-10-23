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
			want:         "cage-myproject-main",
		},
		{
			name:         "sanitized worktree name",
			projectPath:  "/home/user/myproject",
			worktreeName: "feature/auth",
			want:         "cage-myproject-feature-auth",
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

	if labels["managed-by"] != "cage" {
		t.Errorf("managed-by label = %v, want cage", labels["managed-by"])
	}

	if labels["cage-project"] != "myproject" {
		t.Errorf("cage-project label = %v, want myproject", labels["cage-project"])
	}

	if labels["cage-worktree"] != "feature-auth" {
		t.Errorf("cage-worktree label = %v, want feature-auth", labels["cage-worktree"])
	}
}
