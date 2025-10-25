package devcontainer

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config represents a parsed devcontainer.json
type Config struct {
	Image       string `json:"image"`
	DockerFile  string `json:"dockerFile"`
	RemoteUser  string `json:"remoteUser"`
}

// LoadConfig loads and parses .devcontainer/devcontainer.json if it exists
// If the config exists but doesn't specify remoteUser, uses defaultUser parameter
func LoadConfig(projectPath string, defaultUser string) (*Config, error) {
	configPath := filepath.Join(projectPath, ".devcontainer", "devcontainer.json")

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Set default remote user if not specified
	if config.RemoteUser == "" {
		if defaultUser == "" {
			defaultUser = "vscode"
		}
		config.RemoteUser = defaultUser
	}

	return &config, nil
}

// GetDefaultConfig returns the default devcontainer config
// If defaultImage is empty, uses "ghcr.io/obra/packnplay-default:latest"
// If defaultUser is empty, uses "vscode"
func GetDefaultConfig(defaultImage string, defaultUser string) *Config {
	if defaultImage == "" {
		defaultImage = "ghcr.io/obra/packnplay-default:latest"
	}
	if defaultUser == "" {
		defaultUser = "vscode"
	}
	return &Config{
		Image:      defaultImage,
		RemoteUser: defaultUser,
	}
}
