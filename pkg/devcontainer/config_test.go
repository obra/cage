package devcontainer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create temp dir with devcontainer.json
	tmpDir := t.TempDir()
	devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
	_ = os.Mkdir(devcontainerDir, 0755)

	configContent := `{
		"image": "mcr.microsoft.com/devcontainers/base:ubuntu",
		"remoteUser": "vscode"
	}`

	_ = os.WriteFile(
		filepath.Join(devcontainerDir, "devcontainer.json"),
		[]byte(configContent),
		0644,
	)

	config, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if config.Image != "mcr.microsoft.com/devcontainers/base:ubuntu" {
		t.Errorf("Image = %v, want mcr.microsoft.com/devcontainers/base:ubuntu", config.Image)
	}

	if config.RemoteUser != "vscode" {
		t.Errorf("RemoteUser = %v, want vscode", config.RemoteUser)
	}
}

func TestLoadConfig_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	config, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil for missing config", err)
	}

	if config != nil {
		t.Errorf("LoadConfig() = %v, want nil for missing config", config)
	}
}

func TestGetDefaultConfig(t *testing.T) {
	// Test with empty string - should use default image and detect user
	config := GetDefaultConfig("")
	if config.Image != "ghcr.io/obra/packnplay-default:latest" {
		t.Errorf("GetDefaultConfig(\"\") Image = %v, want ghcr.io/obra/packnplay-default:latest", config.Image)
	}
	// RemoteUser should be detected, not hardcoded. For non-existent images, should fall back to "root"
	if config.RemoteUser == "" {
		t.Errorf("GetDefaultConfig(\"\") RemoteUser should not be empty")
	}

	// Test with existing image (ubuntu should work)
	ubuntuImage := "ubuntu:22.04"
	config = GetDefaultConfig(ubuntuImage)
	if config.Image != ubuntuImage {
		t.Errorf("GetDefaultConfig(%v) Image = %v, want %v", ubuntuImage, config.Image, ubuntuImage)
	}
	// For ubuntu, should detect and use "root" as fallback since no better user found
	if config.RemoteUser == "" {
		t.Errorf("GetDefaultConfig(%v) RemoteUser should not be empty", ubuntuImage)
	}
}

func TestLoadConfigWithFeatures(t *testing.T) {
	// Create temp dir with devcontainer.json that includes features
	tmpDir := t.TempDir()
	devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
	_ = os.Mkdir(devcontainerDir, 0755)

	configContent := `{
		"image": "mcr.microsoft.com/devcontainers/base:ubuntu",
		"remoteUser": "vscode",
		"features": {
			"ghcr.io/devcontainers/features/node:1": {
				"version": "lts"
			},
			"ghcr.io/devcontainers/features/go:1": {
				"version": "1.20"
			},
			"ghcr.io/devcontainers/features/github-cli:1": {}
		},
		"postCreateCommand": "npm install",
		"forwardPorts": [3000, 8080],
		"containerEnv": {
			"NODE_ENV": "development"
		},
		"name": "Test Container"
	}`

	_ = os.WriteFile(
		filepath.Join(devcontainerDir, "devcontainer.json"),
		[]byte(configContent),
		0644,
	)

	config, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Test basic properties
	if config.Image != "mcr.microsoft.com/devcontainers/base:ubuntu" {
		t.Errorf("Image = %v, want mcr.microsoft.com/devcontainers/base:ubuntu", config.Image)
	}

	if config.Name != "Test Container" {
		t.Errorf("Name = %v, want Test Container", config.Name)
	}

	// Test features
	if !config.HasFeatures() {
		t.Error("HasFeatures() = false, want true")
	}

	features := config.GetFeatureList()
	if len(features) != 3 {
		t.Errorf("GetFeatureList() length = %v, want 3", len(features))
	}

	// Test specific feature options
	nodeOptions := config.GetFeatureOptions("ghcr.io/devcontainers/features/node:1")
	if nodeOptions == nil {
		t.Error("Node feature options should not be nil")
	}
	if nodeOptions["version"] != "lts" {
		t.Errorf("Node version = %v, want lts", nodeOptions["version"])
	}

	goOptions := config.GetFeatureOptions("ghcr.io/devcontainers/features/go:1")
	if goOptions["version"] != "1.20" {
		t.Errorf("Go version = %v, want 1.20", goOptions["version"])
	}

	ghOptions := config.GetFeatureOptions("ghcr.io/devcontainers/features/github-cli:1")
	if ghOptions == nil {
		t.Error("GitHub CLI feature options should not be nil (should be empty map)")
	}

	// Test postCreateCommand
	if config.PostCreateCommand != "npm install" {
		t.Errorf("PostCreateCommand = %v, want npm install", config.PostCreateCommand)
	}

	// Test containerEnv
	if config.ContainerEnv["NODE_ENV"] != "development" {
		t.Errorf("ContainerEnv[NODE_ENV] = %v, want development", config.ContainerEnv["NODE_ENV"])
	}

	// Test forwardPorts
	if len(config.ForwardPorts) != 2 {
		t.Errorf("ForwardPorts length = %v, want 2", len(config.ForwardPorts))
	}
}

func TestConfigHelperMethods(t *testing.T) {
	// Test with no features
	config := &Config{}
	if config.HasFeatures() {
		t.Error("HasFeatures() = true for empty config, want false")
	}

	if features := config.GetFeatureList(); features != nil {
		t.Errorf("GetFeatureList() = %v for empty config, want nil", features)
	}

	if opts := config.GetFeatureOptions("test"); opts != nil {
		t.Errorf("GetFeatureOptions() = %v for empty config, want nil", opts)
	}

	// Test with features
	config.Features = map[string]interface{}{
		"test-feature": map[string]interface{}{
			"option1": "value1",
		},
		"simple-feature": nil,
		"string-feature": "simple-value",
	}

	if !config.HasFeatures() {
		t.Error("HasFeatures() = false, want true")
	}

	features := config.GetFeatureList()
	if len(features) != 3 {
		t.Errorf("GetFeatureList() length = %v, want 3", len(features))
	}

	// Test different option formats
	opts := config.GetFeatureOptions("test-feature")
	if opts["option1"] != "value1" {
		t.Errorf("Option value = %v, want value1", opts["option1"])
	}

	opts = config.GetFeatureOptions("simple-feature")
	if opts == nil {
		t.Error("Options for nil feature should be empty map, not nil")
	}

	opts = config.GetFeatureOptions("string-feature")
	if opts["value"] != "simple-value" {
		t.Errorf("String feature value = %v, want simple-value", opts["value"])
	}
}
