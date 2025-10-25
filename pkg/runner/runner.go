package runner

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/obra/packnplay/pkg/config"
	"github.com/obra/packnplay/pkg/container"
	"github.com/obra/packnplay/pkg/devcontainer"
	"github.com/obra/packnplay/pkg/docker"
	"github.com/obra/packnplay/pkg/git"
)

type RunConfig struct {
	Path           string
	Worktree       string
	NoWorktree     bool
	Env            []string
	Verbose        bool
	Runtime        string // docker, podman, or container
	Reconnect      bool   // Allow reconnecting to existing containers
	DefaultImage   string // default container image to use
	DefaultUser    string // default username inside container
	Command        []string
	Credentials    config.Credentials
	DefaultEnvVars []string // API keys to proxy from host
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
	var mainRepoGitDir string // Path to main repo's .git directory for mounting

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
				// Worktree already exists - just use it
				actualPath, err := git.GetWorktreePath(worktreeName)
				if err != nil {
					return fmt.Errorf("failed to get worktree path: %w", err)
				}
				mountPath = actualPath
				if config.Verbose {
					fmt.Fprintf(os.Stderr, "Using existing worktree at %s\n", mountPath)
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

			// Get main repo's .git directory for mounting
			// Resolve the real path (follow symlinks) to ensure .git paths match
			realWorkDir, err := filepath.EvalSymlinks(workDir)
			if err != nil {
				realWorkDir = workDir // Fallback if can't resolve
			}
			mainRepoGitDir = filepath.Join(realWorkDir, ".git")
		}
	}

	// Step 3: Load devcontainer config
	devConfig, err := devcontainer.LoadConfig(mountPath, config.DefaultUser)
	if err != nil {
		return fmt.Errorf("failed to load devcontainer config: %w", err)
	}
	if devConfig == nil {
		devConfig = devcontainer.GetDefaultConfig(config.DefaultImage, config.DefaultUser)
	}

	// Step 4: Initialize container client
	dockerClient, err := docker.NewClientWithRuntime(config.Runtime, config.Verbose)
	if err != nil {
		return fmt.Errorf("failed to initialize container runtime: %w", err)
	}

	// Step 5: Ensure image available
	if err := ensureImage(dockerClient, devConfig, mountPath, config.Verbose); err != nil {
		return err
	}

	// Step 5.5: Validate that configured user exists in the image
	validationImageName := devConfig.Image
	if devConfig.DockerFile != "" {
		validationImageName = fmt.Sprintf("packnplay-%s-devcontainer:latest", filepath.Base(workDir))
	}
	if err := validateUserExistsInImage(dockerClient, validationImageName, devConfig.RemoteUser, config.Verbose); err != nil {
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
		// Container is running - check if user wants to reconnect
		if !config.Reconnect {
			// Error with helpful message
			worktreeFlag := ""
			if worktreeName != "no-worktree" {
				worktreeFlag = fmt.Sprintf(" --worktree=%s", worktreeName)
			}

			var cmdStr strings.Builder
			for i, arg := range config.Command {
				if i > 0 {
					cmdStr.WriteString(" ")
				}
				if strings.Contains(arg, " ") {
					cmdStr.WriteString(fmt.Sprintf("'%s'", arg))
				} else {
					cmdStr.WriteString(arg)
				}
			}

			return fmt.Errorf(`container already running for this worktree

To run your command in the existing container:
  packnplay run%s --reconnect %s

To stop the existing container:
  packnplay stop%s`, worktreeFlag, cmdStr.String(), worktreeFlag)
		}

		// User explicitly wants to reconnect
		if config.Verbose {
			fmt.Fprintf(os.Stderr, "Reconnecting to existing container %s\n", containerName)
		}

		// Get container ID
		containerID, err := getContainerID(dockerClient, containerName)
		if err != nil {
			return fmt.Errorf("failed to get container ID: %w", err)
		}

		// Exec into existing container
		cmdPath, err := exec.LookPath(dockerClient.Command())
		if err != nil {
			return fmt.Errorf("failed to find docker command: %w", err)
		}

		// Always use /workspace as working directory
		execArgs := []string{
			filepath.Base(cmdPath),
			"exec",
			"-it",
			"-w", "/workspace",
			containerID,
		}
		execArgs = append(execArgs, config.Command...)

		return syscall.Exec(cmdPath, execArgs, os.Environ())
	}

	// Remove any stopped containers with same name (required for clean start)
	if config.Verbose {
		fmt.Fprintf(os.Stderr, "Checking for stopped container with same name...\n")
	}
	// Try to remove - ignore errors if container doesn't exist
	_, _ = dockerClient.Run("rm", containerName)

	// Step 8: Get current user and detect OS
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	// Check if we're on Linux (idmap only supported on Linux)
	isLinux := os.Getenv("OSTYPE") == "linux-gnu" || fileExists("/proc/version")

	// Note: Credentials are now managed by separate per-container files and watcher daemon
	// No need for Keychain extraction during container startup

	// Build docker run command for background container
	// Apple Container doesn't support -it with -d (detached mode)
	isApple := currentUser.HomeDir != "" && !isLinux && dockerClient.Command() == "container"
	var args []string
	if isApple {
		args = []string{"run", "-d"}
	} else {
		args = []string{"run", "-d", "-it"} // -d for detached, keep -it for interactive
	}

	// Add labels
	args = append(args, container.LabelsToArgs(labels)...)

	// Add name
	args = append(args, "--name", containerName)

	// Add mounts with or without idmap based on OS
	homeDir := currentUser.HomeDir

	// Mount .claude directory, workspace, and git directory (if worktree)
	// Note: idmap support is kernel/Docker version dependent, so we don't use it for now
	// Just use simple volume mounts and run as container's default user

	// Check if we need container-managed credentials
	hostCredFile := filepath.Join(homeDir, ".claude", ".credentials.json")
	var needsCredentialOverlay bool
	var credentialFile string

	// Check if host has meaningful credentials (not just empty file)
	hostHasCredentials := false
	if fileExists(hostCredFile) {
		if stat, err := os.Stat(hostCredFile); err == nil && stat.Size() >= 20 {
			hostHasCredentials = true
		}
	}

	if !hostHasCredentials {
		needsCredentialOverlay = true
		if config.Verbose {
			if !fileExists(hostCredFile) {
				fmt.Fprintf(os.Stderr, "Host has no .credentials.json, using container-managed credentials\n")
			} else {
				fmt.Fprintf(os.Stderr, "Host .credentials.json is too small (%d bytes), using container-managed credentials\n", getFileSize(hostCredFile))
			}
		}

		var err error
		credentialFile, err = getOrCreateContainerCredentialFile(containerName)
		if err != nil {
			return fmt.Errorf("failed to get credential file: %w", err)
		}
	} else {
		if config.Verbose {
			fmt.Fprintf(os.Stderr, "Using host .credentials.json (%d bytes)\n", getFileSize(hostCredFile))
		}
	}

	// Mount .claude directory
	args = append(args, "-v", fmt.Sprintf("%s/.claude:/home/%s/.claude", homeDir, devConfig.RemoteUser))

	// Overlay mount credential file after .claude directory mount
	if needsCredentialOverlay {
		args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.claude/.credentials.json", credentialFile, devConfig.RemoteUser))
	}

	// Mount workspace at /workspace
	args = append(args, "-v", fmt.Sprintf("%s:/workspace", mountPath))

	// Mount AI agent config directories if they exist
	agentConfigDirs := []string{".codex", ".gemini", ".copilot", ".qwen", ".cursor", ".deepseek"}
	for _, configDir := range agentConfigDirs {
		agentPath := filepath.Join(homeDir, configDir)
		if fileExists(agentPath) {
			args = append(args, "-v", fmt.Sprintf("%s:/home/%s/%s", agentPath, devConfig.RemoteUser, configDir))
			if config.Verbose {
				fmt.Fprintf(os.Stderr, "Mounting %s config directory\n", configDir)
			}
		}
	}

	// Mount .config/amp directory for Sourcegraph Amp CLI if it exists
	ampConfigPath := filepath.Join(homeDir, ".config", "amp")
	if fileExists(ampConfigPath) {
		args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.config/amp", ampConfigPath, devConfig.RemoteUser))
		if config.Verbose {
			fmt.Fprintf(os.Stderr, "Mounting amp config directory\n")
		}
	}

	// If using a worktree, also mount the main repo's .git directory at its real path
	// This allows the worktree's .git file (which contains gitdir: <path>) to resolve correctly
	if mainRepoGitDir != "" {
		args = append(args, "-v", fmt.Sprintf("%s:%s", mainRepoGitDir, mainRepoGitDir))
	}

	// Mount git config
	if config.Credentials.Git {
		gitconfigPath := filepath.Join(homeDir, ".gitconfig")
		if fileExists(gitconfigPath) {
			args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.gitconfig:ro", gitconfigPath, devConfig.RemoteUser))
		}
	}

	// Mount SSH keys
	if config.Credentials.SSH {
		sshPath := filepath.Join(homeDir, ".ssh")
		if fileExists(sshPath) {
			args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.ssh:ro", sshPath, devConfig.RemoteUser))
		}
	}

	// Note: On macOS, gh credentials from Keychain are copied in after container starts
	// On Linux, mount the gh config directory if it exists
	if config.Credentials.GH && isLinux {
		ghConfigPath := filepath.Join(homeDir, ".config", "gh")
		if fileExists(ghConfigPath) {
			args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.config/gh", ghConfigPath, devConfig.RemoteUser))
		}
	}

	if config.Credentials.GPG {
		// Mount .gnupg directory (read-only for security)
		gnupgPath := filepath.Join(homeDir, ".gnupg")
		if fileExists(gnupgPath) {
			args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.gnupg:ro", gnupgPath, devConfig.RemoteUser))
		}
	}

	if config.Credentials.NPM {
		// Mount .npmrc file
		npmrcPath := filepath.Join(homeDir, ".npmrc")
		if fileExists(npmrcPath) {
			args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.npmrc:ro", npmrcPath, devConfig.RemoteUser))
		}
	}

	workingDir := "/workspace"

	// Set working directory
	args = append(args, "-w", workingDir)

	// Add environment variables
	// Only pass safe terminal/locale variables - nothing else from host
	safeEnvVars := []string{"TERM", "LANG", "LC_ALL", "LC_CTYPE", "LC_MESSAGES", "COLORTERM"}
	for _, key := range safeEnvVars {
		if value := os.Getenv(key); value != "" {
			args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
		}
	}

	// Set HOME to container user's home directory (don't use host HOME)
	args = append(args, "-e", fmt.Sprintf("HOME=/home/%s", devConfig.RemoteUser))

	// Add IS_SANDBOX marker so tools know they're in a sandbox
	args = append(args, "-e", "IS_SANDBOX=1")

	// Don't set PATH - use container's default PATH to avoid host pollution

	// Add default environment variables (API keys for AI agents)
	for _, envVar := range config.DefaultEnvVars {
		if value := os.Getenv(envVar); value != "" {
			args = append(args, "-e", fmt.Sprintf("%s=%s", envVar, value))
		}
	}

	// Add user-specified env vars from --env flags (these can override defaults)
	for _, env := range config.Env {
		// Support both --env KEY=value and --env KEY (pass through from host)
		if strings.Contains(env, "=") {
			// KEY=value format - set specific value
			args = append(args, "-e", env)
		} else {
			// KEY format - pass through current value from host
			if value := os.Getenv(env); value != "" {
				args = append(args, "-e", fmt.Sprintf("%s=%s", env, value))
			}
		}
	}

	// Add image
	imageName := devConfig.Image
	if devConfig.DockerFile != "" {
		imageName = fmt.Sprintf("packnplay-%s-devcontainer:latest", projectName)
	}
	args = append(args, imageName)

	// Add a command that keeps container alive
	args = append(args, "sleep", "infinity")

	// Step 9: Start container in background
	if config.Verbose {
		fmt.Fprintf(os.Stderr, "Starting container %s\n", containerName)
		fmt.Fprintf(os.Stderr, "Full command: docker %v\n", args)
	}

	containerID, err := dockerClient.Run(args...)
	if err != nil {
		return fmt.Errorf("failed to start container: %w\nDocker output:\n%s", err, containerID)
	}
	containerID = strings.TrimSpace(containerID)

	// Step 10: Copy config files into container

	// Copy ~/.claude.json
	claudeConfigSrc := filepath.Join(homeDir, ".claude.json")
	if _, err := os.Stat(claudeConfigSrc); err == nil {
		if err := copyFileToContainer(dockerClient, containerID, claudeConfigSrc, fmt.Sprintf("/home/%s/.claude.json", devConfig.RemoteUser), devConfig.RemoteUser, config.Verbose); err != nil {
			_, _ = dockerClient.Run("rm", "-f", containerID)
			return fmt.Errorf("failed to copy .claude.json: %w", err)
		}
	}

	// Copy container-managed credentials into place if needed (host has no .credentials.json)
	hostCredFile2 := filepath.Join(homeDir, ".claude", ".credentials.json")
	if !fileExists(hostCredFile2) {
		if config.Verbose {
			fmt.Fprintf(os.Stderr, "Copying container credentials into .claude directory...\n")
		}
		// Copy from mounted temp location to .claude directory
		_, err = dockerClient.Run("exec", containerID, "cp", "/tmp/packnplay-credentials.json", fmt.Sprintf("/home/%s/.claude/.credentials.json", devConfig.RemoteUser))
		if err != nil && config.Verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to copy credentials: %v\n", err)
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

// validateUserExistsInImage checks if the specified user exists in the container image
func validateUserExistsInImage(dockerClient *docker.Client, imageName string, username string, verbose bool) error {
	if dockerClient == nil {
		return fmt.Errorf("docker client is nil")
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Validating that user '%s' exists in image %s\n", username, imageName)
	}

	// Try to run 'id -u <username>' in the image
	// If user exists, command succeeds; if not, it fails
	output, err := dockerClient.Run("run", "--rm", "--entrypoint", "id", imageName, "-u", username)
	if err != nil {
		return fmt.Errorf("user '%s' does not exist in image %s\nTo fix this, either:\n  1. Use an image that has the '%s' user\n  2. Set 'default_user' in ~/.config/packnplay/config.json to match your image's username\n  3. Configure 'remoteUser' in your project's .devcontainer/devcontainer.json\n  4. Build a custom image with the required user\n\nDocker output: %s", username, imageName, username, output)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "User '%s' validated successfully (uid: %s)\n", username, strings.TrimSpace(output))
	}

	return nil
}

func ensureImage(dockerClient *docker.Client, config *devcontainer.Config, projectPath string, verbose bool) error {
	var imageName string

	if config.DockerFile != "" {
		// Need to build from Dockerfile
		projectName := filepath.Base(projectPath)
		imageName = fmt.Sprintf("packnplay-%s-devcontainer:latest", projectName)

		// Check if already built
		_, err := dockerClient.Run("image", "inspect", imageName)
		if err != nil {
			// Need to build
			if verbose {
				fmt.Fprintf(os.Stderr, "Building image from %s\n", config.DockerFile)
			}

			dockerfilePath := filepath.Join(projectPath, ".devcontainer", config.DockerFile)
			contextPath := filepath.Join(projectPath, ".devcontainer")

			output, err := dockerClient.Run("build", "-f", dockerfilePath, "-t", imageName, contextPath)
			if err != nil {
				return fmt.Errorf("failed to build image from %s: %w\nDocker output:\n%s", config.DockerFile, err, output)
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

			output, err := dockerClient.Run("pull", imageName)
			if err != nil {
				return fmt.Errorf("failed to pull image %s: %w\nDocker output:\n%s", imageName, err, output)
			}
		}
	}

	return nil
}

func containerIsRunning(dockerClient *docker.Client, name string) (bool, error) {
	// Apple Container doesn't support --filter, so get all and filter client-side
	isApple := dockerClient.Command() == "container"

	var output string
	var err error

	if isApple {
		output, err = dockerClient.Run("ps", "--format", "json")
	} else {
		output, err = dockerClient.Run("ps", "--filter", fmt.Sprintf("name=%s", name), "--format", "{{.Names}}")
	}

	if err != nil {
		return false, err
	}

	// For Apple Container, output is JSON array
	if isApple {
		// Check if container exists AND is running
		// Look for: "id":"<name>" followed by "status":"running"
		idMatch := fmt.Sprintf(`"id":"%s"`, name)
		if !strings.Contains(output, idMatch) {
			return false, nil
		}

		// Find the container object and check if status is running
		// Simple check: find the id, then check if "status":"running" appears before next "id"
		idIdx := strings.Index(output, idMatch)
		nextIdIdx := strings.Index(output[idIdx+len(idMatch):], `"id":"`)
		var searchRegion string
		if nextIdIdx == -1 {
			searchRegion = output[idIdx:]
		} else {
			searchRegion = output[idIdx : idIdx+len(idMatch)+nextIdIdx]
		}

		return strings.Contains(searchRegion, `"status":"running"`), nil
	}

	// Docker/Podman - simple name matching
	return strings.TrimSpace(output) == name, nil
}

// getContainerID gets the container ID by name
func getContainerID(dockerClient *docker.Client, name string) (string, error) {
	isApple := dockerClient.Command() == "container"

	var output string
	var err error

	if isApple {
		output, err = dockerClient.Run("ps", "--format", "json")
	} else {
		output, err = dockerClient.Run("ps", "--filter", fmt.Sprintf("name=%s", name), "--format", "{{.ID}}")
	}

	if err != nil {
		return "", err
	}

	// For Apple Container, search for container with matching ID in JSON
	if isApple {
		idPrefix := fmt.Sprintf(`"id":"%s"`, name)
		if !strings.Contains(output, idPrefix) {
			return "", fmt.Errorf("container not found")
		}
		// Container name IS the ID in Apple Container
		return name, nil
	}

	// Docker/Podman - ID in output
	return strings.TrimSpace(output), nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func getFileSize(path string) int64 {
	if stat, err := os.Stat(path); err == nil {
		return stat.Size()
	}
	return 0
}

// getOrCreateContainerCredentialFile manages shared credential file for all containers
func getOrCreateContainerCredentialFile(containerName string) (string, error) {
	// Get credentials directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	xdgDataHome := os.Getenv("XDG_DATA_HOME")
	if xdgDataHome == "" {
		xdgDataHome = filepath.Join(homeDir, ".local", "share")
	}

	// Use persistent shared credential file in XDG data directory
	credentialsDir := filepath.Join(xdgDataHome, "packnplay", "credentials")
	if err := os.MkdirAll(credentialsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create credentials dir: %w", err)
	}
	credentialFile := filepath.Join(credentialsDir, "claude-credentials.json")

	// If file doesn't exist, initialize it
	if !fileExists(credentialFile) {
		// Try to get initial credentials from keychain (macOS) or copy from host (Linux)
		initialCreds, err := getInitialContainerCredentials()
		if err != nil {
			// Create empty file - user will need to authenticate in container
			if err := os.WriteFile(credentialFile, []byte("{}"), 0600); err != nil {
				return "", fmt.Errorf("failed to create credential file: %w", err)
			}
		} else {
			if err := os.WriteFile(credentialFile, []byte(initialCreds), 0600); err != nil {
				return "", fmt.Errorf("failed to write initial credentials: %w", err)
			}
		}
	}

	return credentialFile, nil
}

// getInitialContainerCredentials gets initial credentials for new containers
func getInitialContainerCredentials() (string, error) {
	// Check if we're on macOS and can get from keychain
	if !fileExists("/proc/version") { // macOS detection
		cmd := exec.Command("security", "find-generic-password",
			"-s", "packnplay-containers-credentials",
			"-a", "packnplay",
			"-w")

		output, err := cmd.Output()
		if err == nil {
			return strings.TrimSpace(string(output)), nil
		}
	} else {
		// Linux: Check if host has .credentials.json we can copy
		homeDir, _ := os.UserHomeDir()
		hostCredFile := filepath.Join(homeDir, ".claude", ".credentials.json")
		if fileExists(hostCredFile) {
			content, err := os.ReadFile(hostCredFile)
			if err == nil {
				return string(content), nil
			}
		}
	}

	return "", fmt.Errorf("no initial credentials available")
}

// copyFileToContainer copies a file into container and fixes ownership
func copyFileToContainer(dockerClient *docker.Client, containerID, srcPath, dstPath, user string, verbose bool) error {
	if verbose {
		fmt.Fprintf(os.Stderr, "Copying %s to container at %s\n", srcPath, dstPath)
	}

	// Check if this is Apple Container (no cp command)
	isApple := dockerClient.Command() == "container"

	if isApple {
		// Apple Container: use exec with base64 to write file
		return copyFileViaExec(dockerClient, containerID, srcPath, dstPath, user, verbose)
	}

	// Docker/Podman: use cp command
	// Ensure parent directory exists in container
	dstDir := filepath.Dir(dstPath)
	output, err := dockerClient.Run("exec", containerID, "mkdir", "-p", dstDir)
	if err != nil {
		return fmt.Errorf("failed to create parent directory %s: %w\nDocker output:\n%s", dstDir, err, output)
	}

	// Copy file
	containerDst := fmt.Sprintf("%s:%s", containerID, dstPath)
	output, err = dockerClient.Run("cp", srcPath, containerDst)
	if err != nil {
		return fmt.Errorf("failed to copy file %s to %s: %w\nDocker output:\n%s", srcPath, dstPath, err, output)
	}

	// Fix ownership (docker cp creates as root)
	// Only chown the specific file, not the entire directory (might contain read-only mounts)
	_, err = dockerClient.Run("exec", "-u", "root", containerID, "chown", fmt.Sprintf("%s:%s", user, user), dstPath)
	if err != nil && verbose {
		fmt.Fprintf(os.Stderr, "Warning: failed to fix ownership: %v\n", err)
	}

	return nil
}

// copyFileViaExec copies a file using a temp directory mount (for Apple Container)
func copyFileViaExec(dockerClient *docker.Client, containerID, srcPath, dstPath, user string, verbose bool) error {
	// Create temp directory for file transfer
	tempDir, err := os.MkdirTemp("", "packnplay-transfer-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Copy file to temp directory
	tempFileName := filepath.Base(srcPath)
	tempFilePath := filepath.Join(tempDir, tempFileName)

	content, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	if err := os.WriteFile(tempFilePath, content, 0644); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	// This function is no longer used for Apple Container
	// Just return error for now
	return fmt.Errorf("file copying not supported for Apple Container")
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

// getGHCredentialsFromKeychain extracts GitHub CLI credentials from macOS Keychain
func getGHCredentialsFromKeychain(username string) (string, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-s", "gh:github.com",
		"-a", username,
		"-w")

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get gh credentials from keychain: %w", err)
	}

	token := strings.TrimSpace(string(output))

	// gh stores tokens with go-keyring-base64: prefix
	if strings.HasPrefix(token, "go-keyring-base64:") {
		encoded := strings.TrimPrefix(token, "go-keyring-base64:")
		// Decode base64
		decoded, err := base64Decode(encoded)
		if err != nil {
			return "", fmt.Errorf("failed to decode gh token: %w", err)
		}
		return decoded, nil
	}

	return token, nil
}

func base64Decode(s string) (string, error) {
	// Try standard base64 decoding
	var result []byte
	var err error

	// Try standard encoding first
	result, err = base64DecodeString(s, "std")
	if err == nil {
		return string(result), nil
	}

	// Try URL encoding
	result, err = base64DecodeString(s, "url")
	if err == nil {
		return string(result), nil
	}

	return "", fmt.Errorf("failed to decode base64")
}

func base64DecodeString(s, encoding string) ([]byte, error) {
	// Import encoding/base64 at package level
	// For now, use a simple implementation
	cmd := exec.Command("base64", "-d")
	cmd.Stdin = strings.NewReader(s)
	return cmd.Output()
}

// createCredentialsTempFile creates a temporary file with credentials (mode 600)
func createCredentialsTempFile(credentials string) (string, error) {
	tmpFile, err := os.CreateTemp("", "packnplay-credentials-*.json")
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

// createGHHostsYmlFromKeychain creates a gh hosts.yml with token from Keychain
func createGHHostsYmlFromKeychain(homeDir string, verbose bool) (string, error) {
	// Read existing hosts.yml to get username
	hostsYmlPath := filepath.Join(homeDir, ".config", "gh", "hosts.yml")
	hostsData, err := os.ReadFile(hostsYmlPath)
	if err != nil {
		return "", fmt.Errorf("failed to read hosts.yml: %w", err)
	}

	// Parse to extract username (simple YAML parsing for "user: <username>")
	var username string
	for _, line := range strings.Split(string(hostsData), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "user:") {
			username = strings.TrimSpace(strings.TrimPrefix(line, "user:"))
			break
		}
	}

	if username == "" {
		return "", fmt.Errorf("could not find user in hosts.yml")
	}

	// Get token from Keychain
	token, err := getGHCredentialsFromKeychain(username)
	if err != nil {
		return "", err
	}

	// Create temp hosts.yml with the token
	hostsContent := fmt.Sprintf(`github.com:
    user: %s
    oauth_token: %s
    git_protocol: https
`, username, token)

	tmpFile, err := os.CreateTemp("", "packnplay-gh-hosts-*.yml")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	if err := tmpFile.Chmod(0600); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to set file permissions: %w", err)
	}

	if _, err := tmpFile.WriteString(hostsContent); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write hosts.yml: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	return tmpFile.Name(), nil
}
