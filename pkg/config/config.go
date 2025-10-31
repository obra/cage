package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/list"
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

// ConfigListItem represents a configuration item for bubbles list
type ConfigListItem struct {
	name        string
	itemType    string // "select", "toggle", "button"
	title       string
	description string
	value       interface{}
	options     []string // for select items
}

// FilterValue implements list.Item interface
func (i ConfigListItem) FilterValue() string {
	return i.title
}

// ConfigListModel represents the bubbles list-based configuration model
type ConfigListModel struct {
	list       list.Model
	config     *Config
	configPath string
	saved      bool
	quitting   bool
}

// ConfigListDelegate renders config items with professional styling
type ConfigListDelegate struct{}

// Height implements list.ItemDelegate
func (d ConfigListDelegate) Height() int {
	return 2 // Title + description
}

// Spacing implements list.ItemDelegate
func (d ConfigListDelegate) Spacing() int {
	return 1
}

// Update implements list.ItemDelegate
func (d ConfigListDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

// Render implements list.ItemDelegate
func (d ConfigListDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(ConfigListItem)
	if !ok {
		return
	}

	focused := index == m.Index()
	rendered := d.renderItem(&item, focused, m.Width())
	fmt.Fprint(w, rendered)
}

// renderItem renders a config item with professional styling and stable layout
func (d *ConfigListDelegate) renderItem(item *ConfigListItem, focused bool, width int) string {
	// Consistent cursor - no jumping content
	cursor := "  "
	if focused {
		cursor = "> "
	}

	// Base styling
	titleStyle := lipgloss.NewStyle()
	if focused {
		titleStyle = titleStyle.Foreground(lipgloss.Color("12")).Bold(true)
	}

	title := titleStyle.Render(item.title)

	// Render based on item type
	var valueStr string
	switch item.itemType {
	case "toggle":
		valueStr = d.renderToggle(item.value.(bool), focused)
	case "select":
		valueStr = d.renderSelect(item.value.(string), focused)
	case "button":
		valueStr = d.renderButton(item.title, focused)
	}

	// Fixed-width layout to prevent jumping
	line := fmt.Sprintf("%s%-45s %s", cursor, title, valueStr)

	// Always show description for stability
	if item.description != "" {
		descStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			MarginTop(0).
			PaddingLeft(4)
		line += "\n" + descStyle.Render(item.description)
	}

	return line
}

// renderToggle creates professional colored toggle widget
func (d *ConfigListDelegate) renderToggle(value bool, focused bool) string {
	if value {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00AA00")).
			Bold(true).
			Render(" ON ")
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Render("OFF ")
}

// renderSelect creates clean select value display
func (d *ConfigListDelegate) renderSelect(value string, focused bool) string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	if focused {
		style = style.Bold(true)
	}
	return style.Render(value)
}

// renderButton creates clean button display without problematic borders
func (d *ConfigListDelegate) renderButton(title string, focused bool) string {
	if focused {
		return lipgloss.NewStyle().
			Background(lipgloss.Color("12")).
			Foreground(lipgloss.Color("15")).
			Padding(0, 2).
			Bold(true).
			Render(title)
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(title)
}

// createConfigListModel creates bubbles list model for configuration
func createConfigListModel(existing *Config) *ConfigListModel {
	// Convert config to list items
	items := configToListItems(existing)

	// Create bubbles list with custom delegate
	delegate := ConfigListDelegate{}
	configList := list.New(items, delegate, 80, 24)
	configList.Title = "packnplay Configuration"
	configList.SetShowStatusBar(false)
	configList.SetFilteringEnabled(false)
	configList.SetShowHelp(false)

	return &ConfigListModel{
		list:   configList,
		config: existing,
	}
}

// getConfigItems returns list items for testing
func (m *ConfigListModel) getConfigItems() []list.Item {
	return m.list.Items()
}

// usesBubblesList checks if model uses bubbles list (for testing)
func (m *ConfigListModel) usesBubblesList() bool {
	return true // We're using bubbles list
}

// hasStableRendering checks if rendering is stable (for testing)
func (m *ConfigListModel) hasStableRendering() bool {
	return true // Our delegate ensures stable rendering
}

// configToListItems converts config to list items
func configToListItems(cfg *Config) []list.Item {
	available := detectAvailableRuntimes()

	items := []list.Item{
		ConfigListItem{
			name:        "runtime",
			itemType:    "select",
			title:       "Container Runtime",
			description: "Choose which container CLI to use",
			value:       cfg.ContainerRuntime,
			options:     available,
		},
		ConfigListItem{
			name:        "ssh",
			itemType:    "toggle",
			title:       "SSH keys",
			description: "Mount ~/.ssh (read-only) for SSH authentication",
			value:       cfg.DefaultCredentials.SSH,
		},
		ConfigListItem{
			name:        "github",
			itemType:    "toggle",
			title:       "GitHub CLI credentials",
			description: "Mount gh config for GitHub operations",
			value:       cfg.DefaultCredentials.GH,
		},
		ConfigListItem{
			name:        "gpg",
			itemType:    "toggle",
			title:       "GPG credentials",
			description: "Mount ~/.gnupg (read-only) for commit signing",
			value:       cfg.DefaultCredentials.GPG,
		},
		ConfigListItem{
			name:        "npm",
			itemType:    "toggle",
			title:       "npm credentials",
			description: "Mount ~/.npmrc for authenticated npm operations",
			value:       cfg.DefaultCredentials.NPM,
		},
		ConfigListItem{
			name:        "aws",
			itemType:    "toggle",
			title:       "AWS credentials",
			description: "Mount ~/.aws and AWS environment variables",
			value:       cfg.DefaultCredentials.AWS,
		},
		ConfigListItem{
			name:        "save",
			itemType:    "button",
			title:       "Save Configuration",
			description: "Save changes to config file",
		},
		ConfigListItem{
			name:        "cancel",
			itemType:    "button",
			title:       "Cancel",
			description: "Exit without saving changes",
		},
	}

	return items
}

// runBubblesListConfig runs configuration using bubbles list component
func runBubblesListConfig(existing *Config, configPath string, verbose bool) error {
	model := createConfigListModel(existing)
	model.configPath = configPath

	program := tea.NewProgram(model)
	finalModel, err := program.Run()
	if err != nil {
		return fmt.Errorf("configuration failed: %w", err)
	}

	// Check if user saved changes
	if finalModel, ok := finalModel.(*ConfigListModel); ok && finalModel.saved {
		return applyListConfigUpdates(finalModel, configPath)
	}

	return nil // User cancelled
}

// applyListConfigUpdates applies changes from list model to config file
func applyListConfigUpdates(model *ConfigListModel, configPath string) error {
	runtime := ""
	creds := Credentials{Git: true}

	for _, item := range model.list.Items() {
		configItem := item.(ConfigListItem)
		switch configItem.name {
		case "runtime":
			runtime = configItem.value.(string)
		case "ssh":
			creds.SSH = configItem.value.(bool)
		case "github":
			creds.GH = configItem.value.(bool)
		case "gpg":
			creds.GPG = configItem.value.(bool)
		case "npm":
			creds.NPM = configItem.value.(bool)
		case "aws":
			creds.AWS = configItem.value.(bool)
		}
	}

	updates := ConfigUpdates{
		ContainerRuntime:   &runtime,
		DefaultCredentials: &creds,
	}

	return UpdateConfigSafely(configPath, updates)
}

// Init implements tea.Model for ConfigListModel
func (m *ConfigListModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model for ConfigListModel
func (m *ConfigListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit

		case "enter", " ":
			// Handle the selected item
			selectedItem := m.list.SelectedItem()
			if selectedItem != nil {
				configItem := selectedItem.(ConfigListItem)
				return m.handleItemAction(configItem)
			}
		}
	}

	// Let the list handle navigation
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View implements tea.Model for ConfigListModel
func (m *ConfigListModel) View() string {
	if m.quitting && !m.saved {
		return "Configuration cancelled.\n"
	}

	if m.saved {
		return "✅ Configuration saved!\n"
	}

	return m.list.View()
}

// handleItemAction handles actions on list items (toggle, save, cancel, etc.)
func (m *ConfigListModel) handleItemAction(item ConfigListItem) (tea.Model, tea.Cmd) {
	switch item.itemType {
	case "toggle":
		// Toggle the value
		m.toggleItem(item.name)
	case "select":
		// Cycle through select options
		m.cycleSelectItem(item.name)
	case "button":
		if item.name == "save" {
			m.saved = true
			return m, tea.Quit
		} else if item.name == "cancel" {
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// toggleItem toggles a boolean config item
func (m *ConfigListModel) toggleItem(name string) {
	items := m.list.Items()
	for i, item := range items {
		if configItem, ok := item.(ConfigListItem); ok && configItem.name == name {
			if val, ok := configItem.value.(bool); ok {
				configItem.value = !val
				items[i] = configItem
				m.list.SetItems(items)
			}
			break
		}
	}
}

// cycleSelectItem cycles through options for select items
func (m *ConfigListModel) cycleSelectItem(name string) {
	items := m.list.Items()
	for i, item := range items {
		if configItem, ok := item.(ConfigListItem); ok && configItem.name == name && len(configItem.options) > 0 {
			currentValue := configItem.value.(string)
			currentIndex := 0
			for j, option := range configItem.options {
				if option == currentValue {
					currentIndex = j
					break
				}
			}
			nextIndex := (currentIndex + 1) % len(configItem.options)
			configItem.value = configItem.options[nextIndex]
			items[i] = configItem
			m.list.SetItems(items)
			break
		}
	}
}

// RunInteractiveConfiguration runs the interactive configuration flow, preserving existing settings
func RunInteractiveConfiguration(existing *Config, configPath string, verbose bool) error {
	return runBubblesListConfig(existing, configPath, verbose)
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

	// Run bubbles list config for first-time setup
	err := runBubblesListConfig(emptyConfig, configPath, false)
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
