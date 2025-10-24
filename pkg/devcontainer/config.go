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
func LoadConfig(projectPath string) (*Config, error) {
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
		config.RemoteUser = "devuser"
	}

	return &config, nil
}

// GetDefaultConfig returns the default devcontainer config
func GetDefaultConfig() *Config {
	return &Config{
		Image:      "ghcr.io/obra/packnplay-default:latest",
		RemoteUser: "vscode",
	}
}
