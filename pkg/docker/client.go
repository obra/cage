package docker

import (
	"fmt"
	"os"
	"os/exec"
)

// Client handles Docker CLI interactions
type Client struct {
	cmd     string
	verbose bool
}

// NewClient creates a new Docker client
func NewClient(verbose bool) (*Client, error) {
	client := &Client{verbose: verbose}
	cmd, err := client.DetectCLI()
	if err != nil {
		return nil, err
	}
	client.cmd = cmd
	return client, nil
}

// DetectCLI finds the docker command to use
func (c *Client) DetectCLI() (string, error) {
	// Check for DOCKER_CMD environment variable
	if envCmd := os.Getenv("DOCKER_CMD"); envCmd != "" {
		if _, err := exec.LookPath(envCmd); err != nil {
			return "", fmt.Errorf("DOCKER_CMD=%s not found in PATH", envCmd)
		}
		return envCmd, nil
	}

	// Try docker first
	if _, err := exec.LookPath("docker"); err == nil {
		return "docker", nil
	}

	// Try podman as fallback
	if _, err := exec.LookPath("podman"); err == nil {
		return "podman", nil
	}

	return "", fmt.Errorf("no docker-compatible CLI found (tried: docker, podman)")
}

// Run executes a docker command
func (c *Client) Run(args ...string) (string, error) {
	cmd := exec.Command(c.cmd, args...)

	if c.verbose {
		fmt.Fprintf(os.Stderr, "+ %s %v\n", c.cmd, args)
	}

	output, err := cmd.CombinedOutput()

	if c.verbose && len(output) > 0 {
		fmt.Fprintf(os.Stderr, "%s\n", output)
	}

	return string(output), err
}

// Command returns the docker command being used
func (c *Client) Command() string {
	return c.cmd
}
