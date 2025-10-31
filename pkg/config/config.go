package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/charmbracelet/huh"
)

// Config represents packnplay's configuration
type Config struct {
	ContainerRuntime   string                   `json:"container_runtime"` // docker, podman, or container
	DefaultImage       string                   `json:"default_image"`     // deprecated: use DefaultContainer.Image
	DefaultCredentials Credentials              `json:"default_credentials"`
	DefaultEnvVars     []string                 `json:"default_env_vars"` // API keys to always proxy
	EnvConfigs         map[string]EnvConfig     `json:"env_configs"`
	DefaultContainer   DefaultContainerConfig   `json:"default_container"`
}

// DefaultContainerConfig configures the default container and update behavior
type DefaultContainerConfig struct {
	Image               string `json:"image"`                 // default container image to use
	CheckForUpdates     bool   `json:"check_for_updates"`     // whether to check for new versions
	AutoPullUpdates     bool   `json:"auto_pull_updates"`     // whether to auto-pull new versions
	CheckFrequencyHours int    `json:"check_frequency_hours"` // how often to check for updates
}

// EnvConfig defines environment variables for different setups (API configs, etc.)
type EnvConfig struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	EnvVars     map[string]string `json:"env_vars"`
}

// Credentials specifies which credentials to mount
type Credentials struct {
	Git bool `json:"git"` // ~/.gitconfig
	SSH bool `json:"ssh"` // ~/.ssh keys
	GH  bool `json:"gh"`  // GitHub CLI credentials
	GPG bool `json:"gpg"` // GPG keys for commit signing
	NPM bool `json:"npm"` // npm credentials
	AWS bool `json:"aws"` // AWS credentials
}

// GetDefaultImage returns the configured default image or fallback
func (c *Config) GetDefaultImage() string {
	if c.DefaultContainer.Image != "" {
		return c.DefaultContainer.Image
	}
	// Fallback to old field for backward compatibility
	if c.DefaultImage != "" {
		return c.DefaultImage
	}
	// Ultimate fallback
	return "ghcr.io/obra/packnplay-default:latest"
}

// GetDefaultContainerConfig returns the default configuration for DefaultContainer
func GetDefaultContainerConfig() DefaultContainerConfig {
	return DefaultContainerConfig{
		Image:               "ghcr.io/obra/packnplay-default:latest",
		CheckForUpdates:     true,
		AutoPullUpdates:     false,
		CheckFrequencyHours: 24,
	}
}

// VersionTrackingData persists notification history to avoid spam
type VersionTrackingData struct {
	LastCheck     time.Time                      `json:"last_check"`
	Notifications map[string]VersionNotification `json:"notifications"`
}

// VersionNotification tracks when we notified about a specific image version
type VersionNotification struct {
	Digest     string    `json:"digest"`
	NotifiedAt time.Time `json:"notified_at"`
	ImageName  string    `json:"image_name"`
}

// GetVersionTrackingPath returns path to version tracking file
func GetVersionTrackingPath() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, _ := os.UserHomeDir()
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "packnplay", "version-tracking.json")
}

// SaveVersionTracking saves notification history to disk
func SaveVersionTracking(data *VersionTrackingData, filePath string) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write data
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tracking data: %w", err)
	}

	return os.WriteFile(filePath, jsonData, 0644)
}

// LoadVersionTracking loads notification history from disk
func LoadVersionTracking(filePath string) (*VersionTrackingData, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Return empty tracking data
		return &VersionTrackingData{
			Notifications: make(map[string]VersionNotification),
		}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tracking file: %w", err)
	}

	var tracking VersionTrackingData
	if err := json.Unmarshal(data, &tracking); err != nil {
		return nil, fmt.Errorf("failed to parse tracking data: %w", err)
	}

	// Initialize map if nil
	if tracking.Notifications == nil {
		tracking.Notifications = make(map[string]VersionNotification)
	}

	return &tracking, nil
}

// shouldCheckForUpdates determines if we should check for updates based on config and timing
func shouldCheckForUpdates(config DefaultContainerConfig, lastCheck time.Time) bool {
	if !config.CheckForUpdates {
		return false
	}

	checkFrequency := time.Duration(config.CheckFrequencyHours) * time.Hour
	return time.Since(lastCheck) >= checkFrequency
}

// LoadOrDefault loads config or returns default config if loading fails
func LoadOrDefault() (*Config, error) {
	cfg, err := Load()
	if err != nil {
		// Return default config if loading fails
		return &Config{
			DefaultContainer: GetDefaultContainerConfig(),
		}, nil
	}
	return cfg, nil
}

// ShouldCheckForUpdates is an alias for shouldCheckForUpdates for external use
func ShouldCheckForUpdates(config DefaultContainerConfig, lastCheck time.Time) bool {
	return shouldCheckForUpdates(config, lastCheck)
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

	// Set default image if not configured (backward compatibility)
	if cfg.DefaultImage == "" {
		cfg.DefaultImage = "ghcr.io/obra/packnplay-default:latest"
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

	// Set default image if not configured (backward compatibility)
	if cfg.DefaultImage == "" {
		cfg.DefaultImage = "ghcr.io/obra/packnplay-default:latest"
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
	var sshCreds, ghCreds, gpgCreds, npmCreds, awsCreds, saveConfig bool

	// Set sensible defaults - SSH and auth credentials should be user choice
	// Git config is just identity info, not credentials
	saveConfig = true

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
		// Second group: Credentials (SSH and auth tokens only)
		huh.NewGroup(
			huh.NewConfirm().
				Title("Enable SSH keys?").
				Description("Mounts ~/.ssh (read-only) for SSH authentication to servers and repos").
				Value(&sshCreds).
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

			huh.NewConfirm().
				Title("Enable AWS credentials?").
				Description("Mounts ~/.aws and passes AWS environment variables (supports SSO, credential_process, static)").
				Value(&awsCreds).
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
		DefaultImage:     "ghcr.io/obra/packnplay-default:latest",
		DefaultCredentials: Credentials{
			Git: true, // Always copy .gitconfig (it's config, not credentials)
			SSH: sshCreds,
			GH:  ghCreds,
			GPG: gpgCreds,
			NPM: npmCreds,
			AWS: awsCreds,
		},
		DefaultEnvVars: []string{
			"ANTHROPIC_API_KEY",
			"OPENAI_API_KEY",
			"GEMINI_API_KEY",
			"GOOGLE_API_KEY",
			"GH_TOKEN",
			"GITHUB_TOKEN",
			"QWEN_API_KEY",
			"CURSOR_API_KEY",
			"AMP_API_KEY",
			"DEEPSEEK_API_KEY",
		},
		EnvConfigs: make(map[string]EnvConfig),
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
	// Note: Apple Container support disabled due to incompatibilities
	// See: https://github.com/obra/packnplay/issues/1
	runtimes := []string{"docker", "podman"}
	var available []string

	for _, runtime := range runtimes {
		if _, err := exec.LookPath(runtime); err == nil {
			available = append(available, runtime)
		}
	}

	return available
}
