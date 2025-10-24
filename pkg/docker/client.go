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
	return NewClientWithRuntime("", verbose)
}

// NewClientWithRuntime creates a client with a specific runtime preference
func NewClientWithRuntime(preferredRuntime string, verbose bool) (*Client, error) {
	client := &Client{verbose: verbose}

	var cmd string
	var err error

	if preferredRuntime != "" {
		cmd, err = client.UseSpecificRuntime(preferredRuntime)
	} else {
		cmd, err = client.DetectCLI()
	}

	if err != nil {
		return nil, err
	}
	client.cmd = cmd
	return client, nil
}

// UseSpecificRuntime uses a specific container runtime
func (c *Client) UseSpecificRuntime(runtime string) (string, error) {
	if _, err := exec.LookPath(runtime); err != nil {
		return "", fmt.Errorf("container runtime '%s' not found in PATH", runtime)
	}
	return runtime, nil
}

// DetectCLI finds the docker command to use
func (c *Client) DetectCLI() (string, error) {
	// Check for DOCKER_CMD environment variable (legacy support)
	if envCmd := os.Getenv("DOCKER_CMD"); envCmd != "" {
		if _, err := exec.LookPath(envCmd); err != nil {
			return "", fmt.Errorf("DOCKER_CMD=%s not found in PATH", envCmd)
		}
		return envCmd, nil
	}

	// Try in order: docker, podman, container
	runtimes := []string{"docker", "podman", "container"}
	for _, runtime := range runtimes {
		if _, err := exec.LookPath(runtime); err == nil {
			return runtime, nil
		}
	}

	return "", fmt.Errorf("no container runtime found (tried: docker, podman, container)")
}

// Run executes a docker command
func (c *Client) Run(args ...string) (string, error) {
	// Translate Docker commands to Apple Container CLI if needed
	if c.cmd == "container" {
		args = c.translateToAppleContainer(args)
	}

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

// translateToAppleContainer translates Docker CLI args to Apple Container CLI
func (c *Client) translateToAppleContainer(args []string) []string {
	if len(args) == 0 {
		return args
	}

	// Translate: ps -> ls
	if args[0] == "ps" {
		newArgs := []string{"ls"}
		newArgs = append(newArgs, args[1:]...)
		return newArgs
	}

	return args
}

// Command returns the docker command being used
func (c *Client) Command() string {
	return c.cmd
}
