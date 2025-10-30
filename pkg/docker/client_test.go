package docker

import (
	"os"
	"testing"
)

func TestDetectDockerCLI(t *testing.T) {
	tests := []struct {
		name    string
		envVar  string
		want    string
		wantErr bool
	}{
		{
			name:    "detect docker in PATH",
			envVar:  "",
			want:    "docker",
			wantErr: false,
		},
		{
			name:    "use DOCKER_CMD override",
			envVar:  "docker",
			want:    "docker",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVar != "" {
				if err := os.Setenv("DOCKER_CMD", tt.envVar); err != nil {
					t.Fatalf("Failed to set DOCKER_CMD: %v", err)
				}
				defer func() {
					if err := os.Unsetenv("DOCKER_CMD"); err != nil {
						t.Errorf("Failed to unset DOCKER_CMD: %v", err)
					}
				}()
			}

			client := &Client{}
			cmd, err := client.DetectCLI()

			if (err != nil) != tt.wantErr {
				t.Errorf("DetectCLI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if cmd != tt.want {
				t.Errorf("DetectCLI() = %v, want %v", cmd, tt.want)
			}
		})
	}
}
