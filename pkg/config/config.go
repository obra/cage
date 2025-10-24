package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/charmbracelet/huh"
)

// Config represents packnplay's configuration
type Config struct {
	ContainerRuntime   string      `json:"container_runtime"`   // docker, podman, or container
	DefaultCredentials Credentials `json:"default_credentials"`
}

// Credentials specifies which credentials to mount
type Credentials struct {
	Git bool `json:"git"` // ~/.gitconfig and ~/.ssh
	GH  bool `json:"gh"`  // GitHub CLI credentials
	GPG bool `json:"gpg"` // GPG keys for commit signing
	NPM bool `json:"npm"` // npm credentials
}

// GetConfigPath returns the path to the config file
func GetConfigPath() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, _ := os.UserHomeDir()
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "packnplay", "config.json")
}

// Load loads the config file, or prompts for interactive setup if not found
func Load() (*Config, error) {
	configPath := GetConfigPath()

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// First run - interactive setup
		return interactiveSetup(configPath)
	}

	// Load existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// If container_runtime is not set, prompt for it
	if cfg.ContainerRuntime == "" {
		return interactiveSetup(configPath)
	}

	return &cfg, nil
}

// LoadWithoutRuntimeCheck loads config without prompting for runtime
func LoadWithoutRuntimeCheck() (*Config, error) {
	configPath := GetConfigPath()

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config not found")
	}

	// Load existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// Save saves the config to disk
func Save(cfg *Config) error {
	configPath := GetConfigPath()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// interactiveSetup prompts user for credential configuration
func interactiveSetup(configPath string) (*Config, error) {
	fmt.Println("\n🔐 packnplay First Run Setup")
	fmt.Println("Configure which credentials to mount in containers by default.")

	// Detect available container runtimes
	available := detectAvailableRuntimes()
	if len(available) == 0 {
		return nil, fmt.Errorf("no container runtime found (tried: docker, podman, container)")
	}

	var selectedRuntime string
	var gitCreds, ghCreds, gpgCreds, npmCreds, saveConfig bool

	// Build runtime selection options
	runtimeOptions := make([]huh.Option[string], len(available))
	for i, rt := range available {
		runtimeOptions[i] = huh.NewOption(rt, rt)
	}

	form := huh.NewForm(
		// First group: Select container runtime
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select container runtime").
				Description("Choose which container CLI to use").
				Options(runtimeOptions...).
				Value(&selectedRuntime),
		),
		// Second group: Credentials
		huh.NewGroup(
			huh.NewConfirm().
				Title("Enable git credentials?").
				Description("Mounts ~/.gitconfig (read-only) and ~/.ssh (read-only) for git operations").
				Value(&gitCreds).
				Affirmative("Yes").
				Negative("No"),

			huh.NewConfirm().
				Title("Enable GitHub CLI credentials?").
				Description("Mounts gh config for authenticated GitHub operations").
				Value(&ghCreds).
				Affirmative("Yes").
				Negative("No"),

			huh.NewConfirm().
				Title("Enable GPG credentials?").
				Description("Mounts ~/.gnupg (read-only) for commit signing").
				Value(&gpgCreds).
				Affirmative("Yes").
				Negative("No"),

			huh.NewConfirm().
				Title("Enable npm credentials?").
				Description("Mounts ~/.npmrc for authenticated npm operations").
				Value(&npmCreds).
				Affirmative("Yes").
				Negative("No"),
		),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Save as defaults?").
				Description(fmt.Sprintf("Save to %s", configPath)).
				Value(&saveConfig).
				Affirmative("Yes").
				Negative("No"),
		),
	)

	err := form.Run()
	if err != nil {
		return nil, fmt.Errorf("interactive setup failed: %w", err)
	}

	cfg := &Config{
		ContainerRuntime: selectedRuntime,
		DefaultCredentials: Credentials{
			Git: gitCreds,
			GH:  ghCreds,
			GPG: gpgCreds,
			NPM: npmCreds,
		},
	}

	if saveConfig {
		if err := Save(cfg); err != nil {
			return nil, err
		}
		fmt.Printf("\n✓ Configuration saved to %s\n", configPath)
	} else {
		fmt.Println("\n✓ Using one-time configuration (not saved)")
	}

	return cfg, nil
}

// detectAvailableRuntimes finds which container runtimes are installed
func detectAvailableRuntimes() []string {
	runtimes := []string{"docker", "podman", "container"}
	var available []string

	for _, runtime := range runtimes {
		if _, err := exec.LookPath(runtime); err == nil {
			available = append(available, runtime)
		}
	}

	return available
}
