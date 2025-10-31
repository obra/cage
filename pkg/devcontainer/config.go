package devcontainer

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/obra/packnplay/pkg/userdetect"
)

// Config represents a parsed devcontainer.json
type Config struct {
	Image             string                 `json:"image"`
	DockerFile        string                 `json:"dockerFile"`
	RemoteUser        string                 `json:"remoteUser"`
	Features          map[string]interface{} `json:"features"`
	PostCreateCommand interface{}            `json:"postCreateCommand"`
	ForwardPorts      []interface{}          `json:"forwardPorts"`
	Mounts            []interface{}          `json:"mounts"`
	ContainerEnv      map[string]string      `json:"containerEnv"`
	Name              string                 `json:"name"`
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

	// If RemoteUser is not specified, detect the best user for the image
	if config.RemoteUser == "" && config.Image != "" {
		userResult, err := userdetect.DetectContainerUser(config.Image, nil)
		if err != nil {
			// If detection fails, fall back to a safe default
			config.RemoteUser = "root"
		} else {
			config.RemoteUser = userResult.User
		}
	}

	return &config, nil
}

// HasFeatures returns true if the config has features defined
func (c *Config) HasFeatures() bool {
	return c.Features != nil && len(c.Features) > 0
}

// GetFeatureList returns a list of feature identifiers
func (c *Config) GetFeatureList() []string {
	if !c.HasFeatures() {
		return nil
	}

	features := make([]string, 0, len(c.Features))
	for featureID := range c.Features {
		features = append(features, featureID)
	}
	return features
}

// GetFeatureOptions returns the options for a specific feature
func (c *Config) GetFeatureOptions(featureID string) map[string]interface{} {
	if !c.HasFeatures() {
		return nil
	}

	options, exists := c.Features[featureID]
	if !exists {
		return nil
	}

	// Handle different option formats
	switch opts := options.(type) {
	case map[string]interface{}:
		return opts
	case nil:
		return make(map[string]interface{}) // Empty options
	default:
		// Convert other types to a map with a default key
		return map[string]interface{}{"value": opts}
	}
}

// GetDefaultConfig returns the default devcontainer config
// If defaultImage is empty, uses "ghcr.io/obra/packnplay-default:latest"
func GetDefaultConfig(defaultImage string) *Config {
	if defaultImage == "" {
		defaultImage = "ghcr.io/obra/packnplay-default:latest"
	}

	// Detect the best user for this image
	userResult, err := userdetect.DetectContainerUser(defaultImage, nil)
	remoteUser := "root" // safe fallback
	if err == nil {
		remoteUser = userResult.User
	}

	return &Config{
		Image:      defaultImage,
		RemoteUser: remoteUser,
	}
}
