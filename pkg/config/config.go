package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// ConfigUpdates represents partial config updates that preserve unshown settings
type ConfigUpdates struct {
	ContainerRuntime   *string      `json:"container_runtime,omitempty"`
	DefaultCredentials *Credentials `json:"default_credentials,omitempty"`
	DefaultContainer   *DefaultContainerConfig `json:"default_container,omitempty"`
}

// LoadExistingOrEmpty loads config from file or returns empty config if file doesn't exist
func LoadExistingOrEmpty(configPath string) (*Config, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return empty config with defaults
		return &Config{
			DefaultContainer: GetDefaultContainerConfig(),
			DefaultEnvVars:   []string{},
			EnvConfigs:       make(map[string]EnvConfig),
		}, nil
	}

	return LoadConfigFromFile(configPath)
}

// LoadConfigFromFile loads config from specified file
func LoadConfigFromFile(configPath string) (*Config, error) {
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

// UpdateConfigSafely updates only specified fields, preserving others
func UpdateConfigSafely(configPath string, updates ConfigUpdates) error {
	// Load existing config
	cfg, err := LoadExistingOrEmpty(configPath)
	if err != nil {
		return fmt.Errorf("failed to load existing config: %w", err)
	}

	// Apply updates only to specified fields
	if updates.ContainerRuntime != nil {
		cfg.ContainerRuntime = *updates.ContainerRuntime
	}

	if updates.DefaultCredentials != nil {
		cfg.DefaultCredentials = *updates.DefaultCredentials
	}

	if updates.DefaultContainer != nil {
		cfg.DefaultContainer = *updates.DefaultContainer
	}

	// Save updated config
	return SaveConfig(cfg, configPath)
}

// applyCredentialUpdates applies credential updates to config, preserving other settings
func applyCredentialUpdates(original *Config, credUpdates Credentials) *Config {
	// Create copy to avoid modifying original
	updated := *original
	updated.DefaultCredentials = credUpdates
	return &updated
}

// SaveConfig saves config to file
func SaveConfig(cfg *Config, configPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal and save
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configPath, data, 0644)
}

// ConfigTUIModel represents the state of the configuration TUI
type ConfigTUIModel struct {
	config       *Config
	configPath   string
	currentField int
	fields       []ConfigField
	saved        bool
	quitting     bool
	width        int
	height       int
}

// ConfigField represents a configurable field in the TUI
type ConfigField struct {
	name        string
	fieldType   string // "select", "toggle"
	title       string
	description string
	value       interface{}
	options     []string // for select fields
}

// createConfigTUIModel creates a new configuration TUI model
func createConfigTUIModel(existing *Config) *ConfigTUIModel {
	// Detect available runtimes
	available := detectAvailableRuntimes()

	fields := []ConfigField{
		{
			name:        "runtime",
			fieldType:   "select",
			title:       "Container Runtime",
			description: "Choose which container CLI to use",
			value:       existing.ContainerRuntime,
			options:     available,
		},
		{
			name:        "ssh",
			fieldType:   "toggle",
			title:       "SSH keys",
			description: "Mount ~/.ssh (read-only) for SSH authentication",
			value:       existing.DefaultCredentials.SSH,
		},
		{
			name:        "github",
			fieldType:   "toggle",
			title:       "GitHub CLI credentials",
			description: "Mount gh config for GitHub operations",
			value:       existing.DefaultCredentials.GH,
		},
		{
			name:        "gpg",
			fieldType:   "toggle",
			title:       "GPG credentials",
			description: "Mount ~/.gnupg (read-only) for commit signing",
			value:       existing.DefaultCredentials.GPG,
		},
		{
			name:        "npm",
			fieldType:   "toggle",
			title:       "npm credentials",
			description: "Mount ~/.npmrc for authenticated npm operations",
			value:       existing.DefaultCredentials.NPM,
		},
		{
			name:        "aws",
			fieldType:   "toggle",
			title:       "AWS credentials",
			description: "Mount ~/.aws and AWS environment variables",
			value:       existing.DefaultCredentials.AWS,
		},
		{
			name:        "save",
			fieldType:   "button",
			title:       "Save Configuration",
			description: "Save changes to config file",
		},
		{
			name:        "cancel",
			fieldType:   "button",
			title:       "Cancel",
			description: "Exit without saving changes",
		},
	}

	return &ConfigTUIModel{
		config:       existing,
		currentField: 0,
		fields:       fields,
		width:        80,
		height:       24,
	}
}

// getFieldCount returns number of configurable fields
func (m *ConfigTUIModel) getFieldCount() int {
	return len(m.fields)
}

// hasRuntimeField checks if model has runtime field
func (m *ConfigTUIModel) hasRuntimeField() bool {
	for _, field := range m.fields {
		if field.name == "runtime" {
			return true
		}
	}
	return false
}

// hasCredentialFields checks if model has credential fields
func (m *ConfigTUIModel) hasCredentialFields() bool {
	credentialCount := 0
	for _, field := range m.fields {
		if field.fieldType == "toggle" {
			credentialCount++
		}
	}
	return credentialCount >= 5 // SSH, GH, GPG, npm, AWS
}

// moveDown navigates to next field
func moveDown(model *ConfigTUIModel) *ConfigTUIModel {
	model.currentField = (model.currentField + 1) % len(model.fields)
	return model
}

// moveUp navigates to previous field
func moveUp(model *ConfigTUIModel) *ConfigTUIModel {
	model.currentField = (model.currentField - 1 + len(model.fields)) % len(model.fields)
	return model
}

// findFieldIndex finds the index of a field by name
func (m *ConfigTUIModel) findFieldIndex(name string) int {
	for i, field := range m.fields {
		if strings.Contains(field.name, strings.ToLower(name)) {
			return i
		}
	}
	return -1
}

// getFieldValue gets the value of a field by index
func (m *ConfigTUIModel) getFieldValue(index int) interface{} {
	if index < 0 || index >= len(m.fields) {
		return nil
	}
	return m.fields[index].value
}

// toggleCurrentField toggles the current field (if it's a toggle)
func toggleCurrentField(model *ConfigTUIModel) *ConfigTUIModel {
	if model.currentField < 0 || model.currentField >= len(model.fields) {
		return model
	}

	field := &model.fields[model.currentField]
	if field.fieldType == "toggle" {
		if val, ok := field.value.(bool); ok {
			field.value = !val
		}
	}

	return model
}

// Init implements tea.Model
func (m *ConfigTUIModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m *ConfigTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			m = moveUp(m)

		case "down", "j":
			m = moveDown(m)

		case "enter", " ":
			// Handle different field types
			currentField := m.fields[m.currentField]
			switch currentField.fieldType {
			case "toggle":
				m = toggleCurrentField(m)
			case "select":
				m = cycleSelectOption(m)
			case "button":
				if currentField.name == "save" {
					m.saved = true
					return m, tea.Quit
				} else if currentField.name == "cancel" {
					m.quitting = true
					return m, tea.Quit
				}
			}

		case "s", "ctrl+s":
			// Save configuration
			m.saved = true
			return m, tea.Quit
		}
	}

	return m, nil
}

// View implements tea.Model
func (m *ConfigTUIModel) View() string {
	if m.quitting && !m.saved {
		return "Configuration cancelled.\n"
	}

	if m.saved {
		return "✅ Configuration saved!\n"
	}

	return m.renderView()
}

// cycleSelectOption cycles through options for select fields
func cycleSelectOption(model *ConfigTUIModel) *ConfigTUIModel {
	field := &model.fields[model.currentField]
	if field.fieldType != "select" || len(field.options) == 0 {
		return model
	}

	currentValue := field.value.(string)
	currentIndex := 0
	for i, option := range field.options {
		if option == currentValue {
			currentIndex = i
			break
		}
	}

	nextIndex := (currentIndex + 1) % len(field.options)
	field.value = field.options[nextIndex]

	return model
}

// renderView renders the TUI view with proper layout
func (m *ConfigTUIModel) renderView() string {
	var lines []string

	// Header with clean styling
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	lines = append(lines, headerStyle.Render("packnplay Configuration"))
	lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Use ↑/↓ to navigate • Enter/Space to select • 's' save • 'q' cancel"))
	lines = append(lines, "")

	// Runtime section
	lines = append(lines, headerStyle.Render("Container Runtime"))
	runtimeField := m.fields[0] // Runtime is always first
	runtimeLine := m.renderSelectField(0, runtimeField)
	lines = append(lines, runtimeLine)
	lines = append(lines, "")

	// Credentials section
	lines = append(lines, headerStyle.Render("Credentials"))
	for i := 1; i < len(m.fields); i++ { // Skip runtime field
		field := m.fields[i]
		if field.fieldType == "toggle" {
			line := m.renderToggleField(i, field)
			lines = append(lines, line)
		}
	}
	lines = append(lines, "")

	// Action buttons at bottom
	lines = append(lines, headerStyle.Render("Actions"))
	for i, field := range m.fields {
		if field.fieldType == "button" {
			line := m.renderButtonField(i, field)
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}

// renderToggleField renders a toggle field with colored toggle widget
func (m *ConfigTUIModel) renderToggleField(index int, field ConfigField) string {
	focused := index == m.currentField
	value := field.value.(bool)

	// Use consistent spacing - cursor goes where space would be
	cursor := " "
	if focused {
		cursor = "●"
	}

	// Create colored toggle widget
	var toggle string
	if value {
		toggle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ff00")).
			Bold(true).
			Render("ON ")
	} else {
		toggle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Render("OFF")
	}

	// Title styling
	titleStyle := lipgloss.NewStyle()
	if focused {
		titleStyle = titleStyle.Foreground(lipgloss.Color("39")).Bold(true)
	}

	// Create consistent layout with no jumping
	title := titleStyle.Render(field.title)
	line := fmt.Sprintf("%s%-44s %s", cursor, title, toggle)

	// Always show description (consistent spacing)
	if field.description != "" {
		descStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			MarginLeft(2)
		line += "\n" + descStyle.Render(field.description)
	}

	return line
}

// renderSelectField renders a select field
func (m *ConfigTUIModel) renderSelectField(index int, field ConfigField) string {
	focused := index == m.currentField
	value := field.value.(string)

	// Consistent cursor positioning
	cursor := " "
	if focused {
		cursor = "●"
	}

	// Value styling
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true)

	// Title styling
	titleStyle := lipgloss.NewStyle()
	if focused {
		titleStyle = titleStyle.Foreground(lipgloss.Color("39")).Bold(true)
	}

	title := titleStyle.Render(field.title)
	valueText := valueStyle.Render(value)
	line := fmt.Sprintf("%s%-44s %s", cursor, title, valueText)

	// Always show description
	if field.description != "" {
		descStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			MarginLeft(2)
		line += "\n" + descStyle.Render(field.description)
	}

	return line
}

// renderButtonField renders a button field
func (m *ConfigTUIModel) renderButtonField(index int, field ConfigField) string {
	focused := index == m.currentField

	// Consistent cursor positioning
	cursor := " "
	if focused {
		cursor = "●"
	}

	// Button styling
	buttonStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Margin(0, 1)

	if focused {
		if field.name == "save" {
			buttonStyle = buttonStyle.
				Background(lipgloss.Color("34")).
				Foreground(lipgloss.Color("15")).
				Bold(true)
		} else {
			buttonStyle = buttonStyle.
				Background(lipgloss.Color("1")).
				Foreground(lipgloss.Color("15")).
				Bold(true)
		}
	} else {
		buttonStyle = buttonStyle.
			Foreground(lipgloss.Color("240")).
			Border(lipgloss.RoundedBorder())
	}

	button := buttonStyle.Render(field.title)
	line := fmt.Sprintf("%s%s", cursor, button)

	// Always show description
	if field.description != "" {
		descStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			MarginLeft(2)
		line += "\n" + descStyle.Render(field.description)
	}

	return line
}

// runCustomConfigTUI runs the custom configuration TUI
func runCustomConfigTUI(existing *Config, configPath string, verbose bool) error {
	model := createConfigTUIModel(existing)
	model.configPath = configPath

	program := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return fmt.Errorf("TUI failed: %w", err)
	}

	// Check if user saved changes
	if finalModel, ok := finalModel.(*ConfigTUIModel); ok && finalModel.saved {
		// Apply safe config updates
		return applySafeConfigUpdates(finalModel, configPath)
	}

	return nil // User cancelled
}

// applySafeConfigUpdates applies TUI changes to config file safely
func applySafeConfigUpdates(model *ConfigTUIModel, configPath string) error {
	// Extract values from TUI model
	runtime := ""
	creds := Credentials{Git: true} // Always copy .gitconfig

	for _, field := range model.fields {
		switch field.name {
		case "runtime":
			runtime = field.value.(string)
		case "ssh":
			creds.SSH = field.value.(bool)
		case "github":
			creds.GH = field.value.(bool)
		case "gpg":
			creds.GPG = field.value.(bool)
		case "npm":
			creds.NPM = field.value.(bool)
		case "aws":
			creds.AWS = field.value.(bool)
		}
	}

	// Use safe update system
	updates := ConfigUpdates{
		ContainerRuntime:   &runtime,
		DefaultCredentials: &creds,
	}

	return UpdateConfigSafely(configPath, updates)
}

// RunInteractiveConfiguration runs the interactive configuration flow, preserving existing settings
func RunInteractiveConfiguration(existing *Config, configPath string, verbose bool) error {
	return runCustomConfigTUI(existing, configPath, verbose)
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

// interactiveSetup prompts user for credential configuration using custom TUI
func interactiveSetup(configPath string) (*Config, error) {
	// Create empty config for first-time setup
	emptyConfig := &Config{
		DefaultContainer: GetDefaultContainerConfig(),
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

	// Run custom TUI for first-time setup
	err := runCustomConfigTUI(emptyConfig, configPath, false)
	if err != nil {
		return nil, fmt.Errorf("interactive setup failed: %w", err)
	}

	// Load the saved config
	return LoadConfigFromFile(configPath)
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
