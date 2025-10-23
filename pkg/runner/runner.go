package runner

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/obra/cage/pkg/container"
	"github.com/obra/cage/pkg/devcontainer"
	"github.com/obra/cage/pkg/docker"
	"github.com/obra/cage/pkg/git"
)

type RunConfig struct {
	Path       string
	Worktree   string
	NoWorktree bool
	Env        []string
	Verbose    bool
	Command    []string
}

func Run(config *RunConfig) error {
	// Step 1: Determine working directory
	workDir := config.Path
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	// Make absolute
	workDir, err := filepath.Abs(workDir)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Step 2: Handle worktree logic
	var mountPath string
	var worktreeName string

	if config.NoWorktree {
		// Use directory directly
		mountPath = workDir
		worktreeName = "no-worktree"
	} else {
		// Check if git repo
		if !git.IsGitRepo(workDir) {
			if config.Worktree != "" {
				return fmt.Errorf("--worktree specified but %s is not a git repository", workDir)
			}
			// Not a git repo and no worktree flag: use directly
			mountPath = workDir
			worktreeName = "no-worktree"
		} else {
			// Is a git repo
			explicitWorktree := config.Worktree != ""
			if explicitWorktree {
				worktreeName = config.Worktree
			} else {
				// Auto-detect from current branch
				branch, err := git.GetCurrentBranch(workDir)
				if err != nil {
					return fmt.Errorf("failed to get current branch: %w", err)
				}
				worktreeName = branch
			}

			// Check if worktree exists
			exists, err := git.WorktreeExists(worktreeName)
			if err != nil {
				return fmt.Errorf("failed to check worktree: %w", err)
			}

			if exists {
				// If user explicitly specified the worktree, use the existing one
				if explicitWorktree {
					mountPath = git.DetermineWorktreePath(workDir, worktreeName)
					if config.Verbose {
						fmt.Fprintf(os.Stderr, "Using existing worktree at %s\n", mountPath)
					}
				} else {
					// Auto-detected worktree exists - error to prevent accidents
					return fmt.Errorf("worktree '%s' already exists. Use --worktree=%s to connect, or --no-worktree to use directory directly", worktreeName, worktreeName)
				}
			} else {
				// Create worktree
				mountPath = git.DetermineWorktreePath(workDir, worktreeName)
				if config.Verbose {
					fmt.Fprintf(os.Stderr, "Creating worktree at %s\n", mountPath)
				}

				if err := git.CreateWorktree(mountPath, worktreeName, config.Verbose); err != nil {
					return fmt.Errorf("failed to create worktree: %w", err)
				}
			}
		}
	}

	// Step 3: Load devcontainer config
	devConfig, err := devcontainer.LoadConfig(mountPath)
	if err != nil {
		return fmt.Errorf("failed to load devcontainer config: %w", err)
	}
	if devConfig == nil {
		devConfig = devcontainer.GetDefaultConfig()
	}

	// Step 4: Initialize Docker client
	dockerClient, err := docker.NewClient(config.Verbose)
	if err != nil {
		return fmt.Errorf("failed to initialize docker: %w", err)
	}

	// Step 5: Ensure image available
	if err := ensureImage(dockerClient, devConfig, mountPath, config.Verbose); err != nil {
		return err
	}

	// Step 6: Generate container name and labels
	projectName := filepath.Base(workDir)
	containerName := container.GenerateContainerName(workDir, worktreeName)
	labels := container.GenerateLabels(projectName, worktreeName)

	// Step 7: Check if container already running
	if isRunning, err := containerIsRunning(dockerClient, containerName); err != nil {
		return fmt.Errorf("failed to check container status: %w", err)
	} else if isRunning {
		return fmt.Errorf("container already running. Use 'cage attach --worktree=%s' or 'cage stop --worktree=%s'", worktreeName, worktreeName)
	}

	// Step 8: Get current user and detect OS
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	// Check if we're on Linux (idmap only supported on Linux)
	isLinux := os.Getenv("OSTYPE") == "linux-gnu" || fileExists("/proc/version")

	// On macOS, extract credentials from Keychain and create temp file
	var credentialsTempFile string
	if !isLinux {
		creds, err := getClaudeCredentialsFromKeychain()
		if err != nil {
			if config.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: Could not get credentials from Keychain: %v\n", err)
			}
		} else {
			credentialsTempFile, err = createCredentialsTempFile(creds)
			if err != nil {
				return fmt.Errorf("failed to create credentials temp file: %w", err)
			}
			// Clean up temp file when done
			defer os.Remove(credentialsTempFile)

			if config.Verbose {
				fmt.Fprintf(os.Stderr, "Created credentials temp file: %s\n", credentialsTempFile)
			}
		}
	}

	// Build docker run command for background container
	args := []string{"run", "-d", "-it"} // -d for detached, keep -it for interactive

	// Add labels
	args = append(args, container.LabelsToArgs(labels)...)

	// Add name
	args = append(args, "--name", containerName)

	// Add mounts with or without idmap based on OS
	homeDir := currentUser.HomeDir

	// Mount .claude directory and workspace
	// Note: idmap support is kernel/Docker version dependent, so we don't use it for now
	// Just use simple volume mounts and run as container's default user
	var workingDir string

	if isLinux {
		// Linux - simple bind mounts at /workspace
		args = append(args, "-v", fmt.Sprintf("%s/.claude:/home/%s/.claude", homeDir, devConfig.RemoteUser))
		args = append(args, "-v", fmt.Sprintf("%s:/workspace", mountPath))
		workingDir = "/workspace"
	} else {
		// macOS - Docker Desktop handles permissions automatically
		args = append(args, "-v", fmt.Sprintf("%s/.claude:/home/%s/.claude", homeDir, devConfig.RemoteUser))

		// macOS only: Mount credentials file if we extracted it from Keychain
		if credentialsTempFile != "" {
			args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.claude/.credentials.json:ro", credentialsTempFile, devConfig.RemoteUser))
		}

		// macOS - mount parent directory to preserve git worktree paths
		parentDir := filepath.Dir(mountPath)
		args = append(args, "-v", fmt.Sprintf("%s:%s", parentDir, parentDir))
		workingDir = mountPath
	}

	// Set working directory
	args = append(args, "-w", workingDir)

	// Add environment variables
	// Copy host environment, but skip HOME on macOS since it points to host path
	for _, env := range os.Environ() {
		if !isLinux && strings.HasPrefix(env, "HOME=") {
			continue
		}
		args = append(args, "-e", env)
	}

	// Set HOME to container user's home directory
	if !isLinux {
		args = append(args, "-e", fmt.Sprintf("HOME=/home/%s", devConfig.RemoteUser))
	}

	// Add IS_SANDBOX
	args = append(args, "-e", "IS_SANDBOX=1")

	// Add custom env vars
	for _, env := range config.Env {
		args = append(args, "-e", env)
	}

	// Add image
	imageName := devConfig.Image
	if devConfig.DockerFile != "" {
		imageName = fmt.Sprintf("cage-%s-devcontainer:latest", projectName)
	}
	args = append(args, imageName)

	// Add a command that keeps container alive
	args = append(args, "sleep", "infinity")

	// Step 9: Start container in background
	if config.Verbose {
		fmt.Fprintf(os.Stderr, "Starting container %s\n", containerName)
	}

	containerID, err := dockerClient.Run(args...)
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	containerID = strings.TrimSpace(containerID)

	// Step 10: Copy ~/.claude.json into container
	claudeConfigSrc := filepath.Join(homeDir, ".claude.json")
	claudeConfigDst := fmt.Sprintf("%s:/home/%s/.claude.json", containerID, devConfig.RemoteUser)

	if _, err := os.Stat(claudeConfigSrc); err == nil {
		if config.Verbose {
			fmt.Fprintf(os.Stderr, "Copying %s to container\n", claudeConfigSrc)
		}
		_, err = dockerClient.Run("cp", claudeConfigSrc, claudeConfigDst)
		if err != nil {
			// Clean up container on error
			dockerClient.Run("rm", "-f", containerID)
			return fmt.Errorf("failed to copy .claude.json: %w", err)
		}
	}

	// Step 11: Exec into container with user's command
	cmdPath, err := exec.LookPath(dockerClient.Command())
	if err != nil {
		return fmt.Errorf("failed to find docker command: %w", err)
	}

	execArgs := []string{
		filepath.Base(cmdPath),
		"exec",
		"-it",
		"-w", workingDir,
		containerID,
	}
	execArgs = append(execArgs, config.Command...)

	// Use syscall.Exec to replace current process
	return syscall.Exec(cmdPath, execArgs, os.Environ())
}

func ensureImage(dockerClient *docker.Client, config *devcontainer.Config, projectPath string, verbose bool) error {
	var imageName string

	if config.DockerFile != "" {
		// Need to build from Dockerfile
		projectName := filepath.Base(projectPath)
		imageName = fmt.Sprintf("cage-%s-devcontainer:latest", projectName)

		// Check if already built
		_, err := dockerClient.Run("image", "inspect", imageName)
		if err != nil {
			// Need to build
			if verbose {
				fmt.Fprintf(os.Stderr, "Building image from %s\n", config.DockerFile)
			}

			dockerfilePath := filepath.Join(projectPath, ".devcontainer", config.DockerFile)
			contextPath := filepath.Join(projectPath, ".devcontainer")

			_, err := dockerClient.Run("build", "-f", dockerfilePath, "-t", imageName, contextPath)
			if err != nil {
				return fmt.Errorf("failed to build image: %w", err)
			}
		}
	} else {
		// Use pre-built image
		imageName = config.Image

		// Check if exists locally
		_, err := dockerClient.Run("image", "inspect", imageName)
		if err != nil {
			// Need to pull
			if verbose {
				fmt.Fprintf(os.Stderr, "Pulling image %s\n", imageName)
			}

			_, err := dockerClient.Run("pull", imageName)
			if err != nil {
				return fmt.Errorf("failed to pull image: %w", err)
			}
		}
	}

	return nil
}

func containerIsRunning(dockerClient *docker.Client, name string) (bool, error) {
	output, err := dockerClient.Run("ps", "--filter", fmt.Sprintf("name=%s", name), "--format", "{{.Names}}")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) == name, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// getClaudeCredentialsFromKeychain extracts Claude credentials from macOS Keychain
func getClaudeCredentialsFromKeychain() (string, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-s", "Claude Code-credentials",
		"-w")

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get credentials from keychain: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// createCredentialsTempFile creates a temporary file with credentials (mode 600)
func createCredentialsTempFile(credentials string) (string, error) {
	tmpFile, err := os.CreateTemp("", "cage-credentials-*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	// Set mode 600 (owner read/write only)
	if err := tmpFile.Chmod(0600); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to set file permissions: %w", err)
	}

	// Write credentials
	if _, err := tmpFile.WriteString(credentials); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write credentials: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	return tmpFile.Name(), nil
}
