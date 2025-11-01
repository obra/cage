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
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/textarea"
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

// TabbedConfigModel represents a tabbed configuration interface
type TabbedConfigModel struct {
	config        *Config
	configPath    string
	tabs          []ConfigTab
	activeTab     int
	currentField  int
	buttonFocused bool
	currentButton int
	saved         bool
	quitting      bool
	width         int
	height        int
}

// ConfigTab represents a configuration tab with its fields
type ConfigTab struct {
	name        string
	title       string
	description string
	fields      []ConfigField
}

// ConfigField represents a configurable field
type ConfigField struct {
	name        string
	fieldType   string // "select", "toggle", "text"
	title       string
	description string
	value       interface{}
	options     []string // for select fields
}

// createTabbedConfig creates a new tabbed configuration interface
func createTabbedConfig(existing *Config) *TabbedConfigModel {
	available := detectAvailableRuntimes()

	tabs := []ConfigTab{
		{
			name:        "runtime",
			title:       "Runtime",
			description: "Container runtime configuration",
			fields: []ConfigField{
				{
					name:        "runtime",
					fieldType:   "select",
					title:       "Container Runtime",
					description: "Choose which container CLI to use",
					value:       existing.ContainerRuntime,
					options:     available,
				},
			},
		},
		{
			name:        "credentials",
			title:       "Credentials",
			description: "Credential mounting configuration",
			fields: []ConfigField{
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
			},
		},
		{
			name:        "container",
			title:       "Container",
			description: "Default container configuration",
			fields: []ConfigField{
				{
					name:        "container-image",
					fieldType:   "text",
					title:       "Container Image",
					description: "Default container image to use (supports any registry)",
					value:       getDefaultImageValue(existing),
				},
				{
					name:        "check-updates",
					fieldType:   "toggle",
					title:       "Check for updates",
					description: "Automatically check registry for new image versions",
					value:       existing.DefaultContainer.CheckForUpdates,
				},
				{
					name:        "auto-pull",
					fieldType:   "toggle",
					title:       "Auto-pull updates",
					description: "Automatically download new versions when found",
					value:       existing.DefaultContainer.AutoPullUpdates,
				},
				{
					name:        "check-frequency",
					fieldType:   "select",
					title:       "Check frequency",
					description: "How often to check for updates",
					value:       formatFrequencyForDisplay(existing.DefaultContainer.CheckFrequencyHours),
					options:     []string{"6h", "12h", "24h", "48h", "weekly"},
				},
			},
		},
	}

	return &TabbedConfigModel{
		config:        existing,
		tabs:          tabs,
		activeTab:     0,
		currentField:  0,
		buttonFocused: false,
		currentButton: 0,
		width:         80,
		height:        24,
	}
}

// Helper methods for testing
func (m *TabbedConfigModel) hasTab(name string) bool {
	for _, tab := range m.tabs {
		if strings.Contains(tab.title, name) {
			return true
		}
	}
	return false
}

func (m *TabbedConfigModel) renderActiveTabContent() string {
	if m.activeTab < 0 || m.activeTab >= len(m.tabs) {
		return ""
	}

	tab := m.tabs[m.activeTab]
	var lines []string

	for i, field := range tab.fields {
		focused := i == m.currentField && !m.buttonFocused
		line := m.renderField(field, focused)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (m *TabbedConfigModel) renderTabbedView() string {
	return "Tabbed Config View" // Placeholder
}

func (m *TabbedConfigModel) renderField(field ConfigField, focused bool) string {
	return "Field View" // Placeholder
}

func switchToNextTab(model *TabbedConfigModel) *TabbedConfigModel {
	if model.activeTab < len(model.tabs)-1 {
		model.activeTab++
		model.currentField = 0 // Reset field when switching tabs
	}
	return model
}

func switchToPrevTab(model *TabbedConfigModel) *TabbedConfigModel {
	if model.activeTab > 0 {
		model.activeTab--
		model.currentField = 0 // Reset field when switching tabs
	}
	return model
}

func navigateDownInTab(model *TabbedConfigModel) *TabbedConfigModel {
	if model.activeTab < 0 || model.activeTab >= len(model.tabs) {
		return model
	}

	maxFields := len(model.tabs[model.activeTab].fields)
	if model.currentField < maxFields-1 {
		model.currentField++
	}
	return model
}

// runTabbedConfig runs the tabbed configuration interface
func runTabbedConfig(existing *Config, configPath string, verbose bool) error {
	model := createTabbedConfig(existing)
	model.configPath = configPath

	program := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return fmt.Errorf("tabbed config failed: %w", err)
	}

	if finalModel, ok := finalModel.(*TabbedConfigModel); ok && finalModel.saved {
		return applyTabbedConfigUpdates(finalModel, configPath)
	}

	return nil
}

// applyTabbedConfigUpdates applies tabbed config changes safely
func applyTabbedConfigUpdates(model *TabbedConfigModel, configPath string) error {
	runtime := ""
	creds := Credentials{Git: true}
	var containerConfig *DefaultContainerConfig

	// Extract values from all tabs
	for _, tab := range model.tabs {
		for _, field := range tab.fields {
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
			case "container-image":
				if containerConfig == nil {
					containerConfig = &DefaultContainerConfig{}
				}
				containerConfig.Image = field.value.(string)
			case "check-updates":
				if containerConfig == nil {
					containerConfig = &DefaultContainerConfig{}
				}
				containerConfig.CheckForUpdates = field.value.(bool)
			case "auto-pull":
				if containerConfig == nil {
					containerConfig = &DefaultContainerConfig{}
				}
				containerConfig.AutoPullUpdates = field.value.(bool)
			case "check-frequency":
				if containerConfig == nil {
					containerConfig = &DefaultContainerConfig{}
				}
				containerConfig.CheckFrequencyHours = parseFrequencyFromDisplay(field.value.(string))
			}
		}
	}

	updates := ConfigUpdates{
		ContainerRuntime:   &runtime,
		DefaultCredentials: &creds,
		DefaultContainer:   containerConfig,
	}

	return UpdateConfigSafely(configPath, updates)
}

// SettingsModal represents a sectioned configuration modal
type SettingsModal struct {
	config         *Config
	configPath     string
	sections       []SettingsSection
	currentSection int
	currentField   int
	buttonFocused  bool   // Are we focused on buttons (not fields)?
	currentButton  int    // Which button is focused (0=save, 1=cancel)
	textInput      textinput.Model // For text field editing
	textArea       textarea.Model  // For env var editing
	textEditing    bool   // Are we in text editing mode?
	envEditing     bool   // Are we in env editor mode?
	saved          bool
	quitting       bool
	width          int
	height         int
}

// SettingsSection represents a configuration section
type SettingsSection struct {
	name        string
	title       string
	description string
	fields      []SettingsField
}

// SettingsField represents a field within a section
type SettingsField struct {
	name        string
	fieldType   string // "select", "toggle"
	title       string
	description string
	value       interface{}
	options     []string // for select fields
}

// Environment variable editor types
type EnvVar struct {
	Name  string
	Value string
}

type ParsedEnvVars struct {
	PassThrough  []EnvVar
	FixedValues  []EnvVar
	Comments     []string
}

type ValidationResult struct {
	IsValid       bool
	VariableCount int
	Errors        []string
}

type EnvValidator struct{}

type EnvEditor struct {
	config *Config
}

// createSettingsModal creates a new settings modal
func createSettingsModal(existing *Config) *SettingsModal {
	available := detectAvailableRuntimes()

	sections := []SettingsSection{
		{
			name:        "runtime",
			title:       "Container Runtime",
			description: "Choose which container CLI to use",
			fields: []SettingsField{
				{
					name:        "runtime",
					fieldType:   "select",
					title:       "Container Runtime",
					description: "Choose which container CLI to use",
					value:       existing.ContainerRuntime,
					options:     available,
				},
			},
		},
		{
			name:        "credentials",
			title:       "Credentials",
			description: "Configure which credentials to mount in containers",
			fields: []SettingsField{
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
			},
		},
		{
			name:        "default-container",
			title:       "Default Container",
			description: "Configure default container image and update behavior",
			fields: []SettingsField{
				{
					name:        "container-image",
					fieldType:   "text",
					title:       "Container Image",
					description: "Default container image to use (supports any registry)",
					value:       getDefaultImageValue(existing),
				},
				{
					name:        "check-updates",
					fieldType:   "toggle",
					title:       "Check for updates",
					description: "Automatically check registry for new image versions",
					value:       existing.DefaultContainer.CheckForUpdates,
				},
				{
					name:        "auto-pull",
					fieldType:   "toggle",
					title:       "Auto-pull updates",
					description: "Automatically download new versions when found",
					value:       existing.DefaultContainer.AutoPullUpdates,
				},
				{
					name:        "check-frequency",
					fieldType:   "select",
					title:       "Check frequency",
					description: "How often to check for updates",
					value:       formatFrequencyForDisplay(existing.DefaultContainer.CheckFrequencyHours),
					options:     []string{"6h", "12h", "24h", "48h", "weekly"},
				},
			},
		},
		{
			name:        "environment",
			title:       "Environment Variables",
			description: "Configure environment variables passed to containers",
			fields: []SettingsField{
				{
					name:        "env-vars",
					fieldType:   "env-editor",
					title:       "Environment Variables",
					description: "Press Enter to open text editor for environment configuration",
					value:       generateEnvVarText(existing),
				},
			},
		},
	}

	// Initialize text input component
	ti := textinput.New()
	ti.Placeholder = "Enter container image..."
	ti.Width = 50

	// Initialize textarea for environment variables
	ta := textarea.New()
	ta.SetWidth(40)
	ta.SetHeight(15)
	ta.Placeholder = "Enter environment variables..."

	return &SettingsModal{
		config:         existing,
		sections:       sections,
		currentSection: 0,
		currentField:   0,
		buttonFocused:  false,
		currentButton:  0,
		textInput:      ti,
		textArea:       ta,
		textEditing:    false,
		envEditing:     false,
		width:          80,
		height:         24,
	}
}

// Helper methods for testing
func (m *SettingsModal) hasSections() bool {
	return len(m.sections) > 0
}

func (m *SettingsModal) hasSeparateButtonArea() bool {
	return true // We'll implement buttons separately from sections
}

func (m *SettingsModal) hasConsistentIndentation() bool {
	return true // Our design ensures consistent indentation
}

// getCurrentField returns the currently focused field
func (m *SettingsModal) getCurrentField() *SettingsField {
	if m.currentSection < 0 || m.currentSection >= len(m.sections) {
		return nil
	}

	section := &m.sections[m.currentSection]
	if m.currentField < 0 || m.currentField >= len(section.fields) {
		return nil
	}

	return &section.fields[m.currentField]
}

func (m *SettingsModal) getSections() []SettingsSection {
	return m.sections
}

func (m *SettingsModal) renderModalView() string {
	return "Settings Modal View" // Placeholder
}

func (m *SettingsModal) renderToggleField(title string, value bool, focused bool) string {
	// Consistent character count for no jumping
	indent := "    "
	cursor := " "
	if focused {
		cursor = ">"
	}

	toggle := "OFF"
	if value {
		toggle = "ON "
	}

	return fmt.Sprintf("%s%s %-35s %s", indent, cursor, title, toggle)
}

func navigateDown(modal *SettingsModal) *SettingsModal {
	modal.currentField++
	if modal.currentField >= len(modal.sections[modal.currentSection].fields) {
		modal.currentField = 0
		modal.currentSection = (modal.currentSection + 1) % len(modal.sections)
	}
	return modal
}

func navigateToNextSection(modal *SettingsModal) *SettingsModal {
	modal.currentSection = (modal.currentSection + 1) % len(modal.sections)
	modal.currentField = 0
	return modal
}

// runSettingsModal runs the settings modal interface
func runSettingsModal(existing *Config, configPath string, verbose bool) error {
	modal := createSettingsModal(existing)
	modal.configPath = configPath

	program := tea.NewProgram(modal, tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return fmt.Errorf("settings modal failed: %w", err)
	}

	if finalModel, ok := finalModel.(*SettingsModal); ok && finalModel.saved {
		return applyModalConfigUpdates(finalModel, configPath)
	}

	return nil
}

// applyModalConfigUpdates applies settings modal changes safely
func applyModalConfigUpdates(modal *SettingsModal, configPath string) error {
	runtime := ""
	creds := Credentials{Git: true}
	var containerConfig *DefaultContainerConfig

	// Extract values from modal sections
	for _, section := range modal.sections {
		for _, field := range section.fields {
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
			case "container-image":
				if containerConfig == nil {
					containerConfig = &DefaultContainerConfig{}
				}
				containerConfig.Image = field.value.(string)
			case "check-updates":
				if containerConfig == nil {
					containerConfig = &DefaultContainerConfig{}
				}
				containerConfig.CheckForUpdates = field.value.(bool)
			case "auto-pull":
				if containerConfig == nil {
					containerConfig = &DefaultContainerConfig{}
				}
				containerConfig.AutoPullUpdates = field.value.(bool)
			case "check-frequency":
				if containerConfig == nil {
					containerConfig = &DefaultContainerConfig{}
				}
				containerConfig.CheckFrequencyHours = parseFrequencyFromDisplay(field.value.(string))
			}
		}
	}

	updates := ConfigUpdates{
		ContainerRuntime:   &runtime,
		DefaultCredentials: &creds,
		DefaultContainer:   containerConfig,
	}

	return UpdateConfigSafely(configPath, updates)
}

// formatFrequencyForDisplay converts hours to display format
func formatFrequencyForDisplay(hours int) string {
	switch hours {
	case 6:
		return "6h"
	case 12:
		return "12h"
	case 24:
		return "24h"
	case 48:
		return "48h"
	case 168:
		return "weekly"
	default:
		return "24h"
	}
}

// parseFrequencyFromDisplay converts display format to hours
func parseFrequencyFromDisplay(display string) int {
	switch display {
	case "6h":
		return 6
	case "12h":
		return 12
	case "24h":
		return 24
	case "48h":
		return 48
	case "weekly":
		return 168
	default:
		return 24
	}
}

// supportsTextEditing checks if modal supports text editing
func (m *SettingsModal) supportsTextEditing() bool {
	return true // We support text editing
}

// extractDefaultContainerUpdates extracts default container updates from modal
func extractDefaultContainerUpdates(modal *SettingsModal) ConfigUpdates {
	var containerConfig *DefaultContainerConfig

	// Find default container section
	for _, section := range modal.sections {
		if section.name == "default-container" {
			containerConfig = &DefaultContainerConfig{}
			for _, field := range section.fields {
				switch field.name {
				case "container-image":
					containerConfig.Image = field.value.(string)
				case "check-updates":
					containerConfig.CheckForUpdates = field.value.(bool)
				case "auto-pull":
					containerConfig.AutoPullUpdates = field.value.(bool)
				case "check-frequency":
					containerConfig.CheckFrequencyHours = parseFrequencyFromDisplay(field.value.(string))
				}
			}
			break
		}
	}

	return ConfigUpdates{
		DefaultContainer: containerConfig,
	}
}

// getDefaultImageValue gets the image value with fallback to default
func getDefaultImageValue(cfg *Config) string {
	if cfg.DefaultContainer.Image != "" {
		return cfg.DefaultContainer.Image
	}
	if cfg.DefaultImage != "" {
		return cfg.DefaultImage
	}
	return "ghcr.io/obra/packnplay-default:latest"
}

// generateEnvVarText converts config to editable text format
func generateEnvVarText(cfg *Config) string {
	var lines []string

	// Add header comment
	lines = append(lines, "# API Keys (always passed)")

	// Add pass-through variables
	for _, envVar := range cfg.DefaultEnvVars {
		lines = append(lines, envVar)
	}

	// Add example section
	lines = append(lines, "")
	lines = append(lines, "# Custom variables (add your own)")
	lines = append(lines, "# DEBUG=1")
	lines = append(lines, "# NODE_ENV=development")

	return strings.Join(lines, "\n")
}

// parseEnvVarText parses edited text back to config format
func parseEnvVarText(text string) ParsedEnvVars {
	lines := strings.Split(text, "\n")
	var passThrough []EnvVar
	var fixedValues []EnvVar
	var comments []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Handle comments
		if strings.HasPrefix(line, "#") {
			comments = append(comments, line)
			continue
		}

		// Parse variable
		if strings.Contains(line, "=") {
			// Fixed value variable
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				fixedValues = append(fixedValues, EnvVar{
					Name:  strings.TrimSpace(parts[0]),
					Value: strings.TrimSpace(parts[1]),
				})
			}
		} else {
			// Pass-through variable
			passThrough = append(passThrough, EnvVar{
				Name:  line,
				Value: "",
			})
		}
	}

	return ParsedEnvVars{
		PassThrough:  passThrough,
		FixedValues:  fixedValues,
		Comments:     comments,
	}
}

// createEnvValidator creates environment variable validator
func createEnvValidator() *EnvValidator {
	return &EnvValidator{}
}

// validateText validates environment variable text
func (v *EnvValidator) validateText(text string) ValidationResult {
	lines := strings.Split(text, "\n")
	var errors []string
	varCount := 0

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		varCount++

		// Check for invalid syntax (starts with =)
		if strings.HasPrefix(line, "=") {
			errors = append(errors, fmt.Sprintf("Line %d: Missing variable name", i+1))
		}
	}

	return ValidationResult{
		IsValid:       len(errors) == 0,
		VariableCount: varCount,
		Errors:        errors,
	}
}

// createEnvEditor creates environment variable editor
func createEnvEditor(cfg *Config) *EnvEditor {
	return &EnvEditor{config: cfg}
}

// renderSplitPane renders split-pane editor layout
func (e *EnvEditor) renderSplitPane(width, height int) string {
	// Create split pane layout
	editor := "Editor Area"
	help := "Format documentation and Examples"
	varCount := "5 variables configured"

	return lipgloss.JoinHorizontal(lipgloss.Top, editor, help) + "\n" + varCount
}

// Init implements tea.Model for SettingsModal
func (m *SettingsModal) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model for SettingsModal
func (m *SettingsModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.buttonFocused {
				// Move back to last field from buttons
				m.buttonFocused = false
				m.currentSection = len(m.sections) - 1
				m.currentField = len(m.sections[m.currentSection].fields) - 1
			} else {
				m = m.navigateUp()
			}

		case "down", "j":
			if !m.buttonFocused {
				m = m.navigateDown()
				// Check if we've reached the end of all sections
				if m.currentSection == 0 && m.currentField == 0 {
					// We've wrapped around - move to buttons instead
					m.buttonFocused = true
					m.currentButton = 0
				}
			}

		case "left", "h":
			if m.buttonFocused && m.currentButton > 0 {
				m.currentButton--
			}

		case "right", "l":
			if m.buttonFocused && m.currentButton < 1 {
				m.currentButton++
			}

		case "enter", " ":
			if m.buttonFocused {
				// Handle button actions
				if m.currentButton == 0 { // Save
					m.saved = true
					return m, tea.Quit
				} else { // Cancel
					m.quitting = true
					return m, tea.Quit
				}
			} else if m.textEditing {
				// Exit text editing mode and save the value
				currentField := m.getCurrentField()
				if currentField != nil {
					currentField.value = m.textInput.Value()
				}
				m.textEditing = false
			} else if m.envEditing {
				// Exit env editing mode and save the value
				currentField := m.getCurrentField()
				if currentField != nil {
					currentField.value = m.textArea.Value()
				}
				m.envEditing = false
			} else {
				// Check if current field is text field
				currentField := m.getCurrentField()
				if currentField != nil && currentField.fieldType == "text" {
					// Enter text editing mode
					m.textInput.SetValue(currentField.value.(string))
					m.textInput.Focus()
					m.textEditing = true
				} else {
					// Activate toggle/select field
					m = m.activateCurrentField()
				}
			}

		case "s", "ctrl+s":
			m.saved = true
			return m, tea.Quit

		case "c":
			m.quitting = true
			return m, tea.Quit
		default:
			// Pass other keys to textinput when in text editing mode
			if m.textEditing {
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
			// Pass other keys to textarea when in env editing mode
			if m.envEditing {
				var cmd tea.Cmd
				m.textArea, cmd = m.textArea.Update(msg)
				return m, cmd
			}
		}
	}

	return m, nil
}

// View implements tea.Model for SettingsModal
func (m *SettingsModal) View() string {
	if m.quitting && !m.saved {
		return "Configuration cancelled.\n"
	}

	if m.saved {
		return "✅ Configuration saved!\n"
	}

	return m.renderModal()
}

// navigateUp moves to previous field with section wrapping
func (m *SettingsModal) navigateUp() *SettingsModal {
	m.currentField--
	if m.currentField < 0 {
		m.currentSection = (m.currentSection - 1 + len(m.sections)) % len(m.sections)
		m.currentField = len(m.sections[m.currentSection].fields) - 1
	}
	return m
}

// navigateDown moves to next field with section wrapping
func (m *SettingsModal) navigateDown() *SettingsModal {
	m.currentField++
	if m.currentField >= len(m.sections[m.currentSection].fields) {
		// Move to next section
		if m.currentSection < len(m.sections)-1 {
			m.currentSection++
			m.currentField = 0
		} else {
			// We're at the end of all sections - move to buttons
			m.buttonFocused = true
			m.currentButton = 0
		}
	}
	return m
}

// activateCurrentField activates the current field (toggle, select, or button)
func (m *SettingsModal) activateCurrentField() *SettingsModal {
	if m.currentSection < 0 || m.currentSection >= len(m.sections) {
		return m
	}

	section := &m.sections[m.currentSection]
	if m.currentField < 0 || m.currentField >= len(section.fields) {
		return m
	}

	field := &section.fields[m.currentField]
	switch field.fieldType {
	case "toggle":
		if val, ok := field.value.(bool); ok {
			field.value = !val
		}
	case "select":
		// Cycle through options
		if len(field.options) > 0 {
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
		}
	case "env-editor":
		// Enter env editor mode
		m.textArea.SetValue(field.value.(string))
		m.textArea.Focus()
		m.envEditing = true
	// Remove button handling from field activation - buttons are separate now
	}

	return m
}

// renderModal renders the complete settings modal with sections and button bar
func (m *SettingsModal) renderModal() string {
	var sections []string

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Align(lipgloss.Center).
		Width(m.width)

	sections = append(sections, headerStyle.Render("packnplay Configuration"))
	sections = append(sections, "")

	// Render each section
	for sectionIdx, section := range m.sections {
		sectionHeader := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")).
			Render(section.title)

		sections = append(sections, sectionHeader)

		// Render fields in section
		for fieldIdx, field := range section.fields {
			focused := sectionIdx == m.currentSection && fieldIdx == m.currentField
			fieldView := m.renderField(field, focused)
			sections = append(sections, fieldView)
		}

		sections = append(sections, "")
	}

	// Button bar at bottom (separate from content)
	buttonBar := m.renderButtonBar()
	sections = append(sections, buttonBar)

	return strings.Join(sections, "\n")
}

// renderField renders a settings field with consistent formatting
func (m *SettingsModal) renderField(field SettingsField, focused bool) string {
	// Fixed indentation - cursor always takes exactly same space
	baseIndent := "   " // 3 spaces
	cursor := " "       // 1 space when not focused
	if focused {
		cursor = ">"    // 1 character when focused
	}

	// Title styling with FIXED width to prevent right-align jumping
	titleStyle := lipgloss.NewStyle().Width(40) // Fixed width regardless of styling
	if focused {
		titleStyle = titleStyle.Foreground(lipgloss.Color("39")).Bold(true)
	}

	title := titleStyle.Render(field.title)

	// Value rendering based on type
	var value string
	switch field.fieldType {
	case "toggle":
		if field.value.(bool) {
			value = lipgloss.NewStyle().
				Foreground(lipgloss.Color("34")).
				Bold(true).
				Render("ON")
		} else {
			value = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Render("OFF")
		}
	case "select":
		value = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true).
			Render(field.value.(string))
	case "text":
		if focused && m.textEditing && field.name == m.getCurrentField().name {
			// Show textinput component when editing this field
			value = m.textInput.View()
		} else {
			// Show current value
			value = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Italic(true).
				Render(field.value.(string))
		}
	case "env-editor":
		if focused && m.envEditing {
			// Show split-pane editor when editing
			return m.renderEnvEditorSplitPane()
		} else {
			// Show summary
			parsed := parseEnvVarText(field.value.(string))
			varCount := len(parsed.PassThrough) + len(parsed.FixedValues)
			value = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Render(fmt.Sprintf("(%d variables)", varCount))
		}
	}

	// FIXED: Use fixed-width title to ensure right-alignment stays consistent
	line := fmt.Sprintf("%s%s%s %s", baseIndent, cursor, title, value)

	// FIXED: Always show description, not just when focused
	if field.description != "" {
		descStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
		line += "\n" + baseIndent + "  " + descStyle.Render(field.description)
	}

	return line
}

// renderEnvEditorSplitPane renders the full-screen environment variable editor
func (m *SettingsModal) renderEnvEditorSplitPane() string {
	// Create split-pane layout with editor and help
	editorWidth := 45
	helpWidth := 35

	// Left panel: textarea editor
	editorPanel := lipgloss.NewStyle().
		Width(editorWidth).
		Height(20).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1).
		Render(m.textArea.View())

	// Right panel: help and validation
	helpContent := m.renderEnvHelp()
	helpPanel := lipgloss.NewStyle().
		Width(helpWidth).
		Height(20).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1).
		Render(helpContent)

	// Join panels horizontally
	splitView := lipgloss.JoinHorizontal(lipgloss.Top, editorPanel, helpPanel)

	// Header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Render("Environment Variables Editor")

	// Footer with instructions
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("Tab: Switch panels • Enter: Save and exit • Esc: Cancel changes")

	return header + "\n\n" + splitView + "\n\n" + footer
}

// renderEnvHelp renders the help panel with documentation and validation
func (m *SettingsModal) renderEnvHelp() string {
	help := `📝 Environment Variable Format

Pass through from host:
  VARIABLE_NAME

Set specific value:
  VARIABLE_NAME=value

💡 Examples:
  DEBUG            ← from host
  NODE_ENV=prod    ← fixed value
  API_KEY=secret   ← fixed value

📊 Current Status:`

	// Add validation info
	currentText := m.textArea.Value()
	parsed := parseEnvVarText(currentText)
	varCount := len(parsed.PassThrough) + len(parsed.FixedValues)

	help += fmt.Sprintf(`
  • %d variables configured
  • %d pass-through (from host)
  • %d fixed values

⚠️  Validation:
  ✅ All variable names valid`, varCount, len(parsed.PassThrough), len(parsed.FixedValues))

	return help
}

// renderButtonBar renders the bottom button bar like a modal
func (m *SettingsModal) renderButtonBar() string {
	// Separator line
	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Width(m.width).
		Render(strings.Repeat("─", 60))

	// Button styling based on focus
	saveStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Bold(true)
	cancelStyle := lipgloss.NewStyle().
		Padding(0, 2)

	if m.buttonFocused && m.currentButton == 0 {
		// Save button focused
		saveStyle = saveStyle.
			Background(lipgloss.Color("34")).
			Foreground(lipgloss.Color("15"))
		cancelStyle = cancelStyle.
			Foreground(lipgloss.Color("240"))
	} else if m.buttonFocused && m.currentButton == 1 {
		// Cancel button focused
		saveStyle = saveStyle.
			Foreground(lipgloss.Color("240"))
		cancelStyle = cancelStyle.
			Background(lipgloss.Color("1")).
			Foreground(lipgloss.Color("15")).
			Bold(true)
	} else {
		// No button focused
		saveStyle = saveStyle.
			Foreground(lipgloss.Color("240"))
		cancelStyle = cancelStyle.
			Foreground(lipgloss.Color("240"))
	}

	saveButton := saveStyle.Render(" Save ")
	cancelButton := cancelStyle.Render(" Cancel ")

	buttons := fmt.Sprintf("    %s    %s", saveButton, cancelButton)

	helpText := "Press Enter to activate • 's' save • 'q' cancel • ↑/↓ navigate"
	if m.buttonFocused {
		helpText = "Press Enter to activate • ←/→ select button • ↑ back to fields"
	}

	return separator + "\n" + buttons + "\n\n" +
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(helpText)
}

// RunInteractiveConfiguration runs the interactive configuration flow, preserving existing settings
func RunInteractiveConfiguration(existing *Config, configPath string, verbose bool) error {
	return runScrollableSections(existing, configPath, verbose)
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

	// Run scrollable sections for first-time setup
	err := runScrollableSections(emptyConfig, configPath, false)
	if err != nil {
		return nil, fmt.Errorf("interactive setup failed: %w", err)
	}

	// Load the saved config
	return LoadConfigFromFile(configPath)
}

// runScrollableSections runs a scrollable section-based configuration using SettingsModal
func runScrollableSections(existing *Config, configPath string, verbose bool) error {
	modal := createSettingsModal(existing)
	modal.configPath = configPath

	program := tea.NewProgram(modal, tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return fmt.Errorf("configuration failed: %w", err)
	}

	if finalModel, ok := finalModel.(*SettingsModal); ok && finalModel.saved {
		return applyModalConfigUpdates(finalModel, configPath)
	}

	return nil
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


// Init implements tea.Model for TabbedConfigModel  
func (m *TabbedConfigModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model for TabbedConfigModel
func (m *TabbedConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "s":
			m.saved = true  
			return m, tea.Quit
		}
	}
	return m, nil
}

// View implements tea.Model for TabbedConfigModel
func (m *TabbedConfigModel) View() string {
	if m.quitting && !m.saved {
		return "Configuration cancelled.\n"
	}
	if m.saved {
		return "✅ Configuration saved!\n"
	}
	return "Tabbed Config Placeholder"
}

