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

	config, err := LoadConfig(tmpDir, "defaultuser")
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if config.Image != "mcr.microsoft.com/devcontainers/base:ubuntu" {
		t.Errorf("Image = %v, want mcr.microsoft.com/devcontainers/base:ubuntu", config.Image)
	}

	// Should use remoteUser from file, not the default
	if config.RemoteUser != "vscode" {
		t.Errorf("RemoteUser = %v, want vscode", config.RemoteUser)
	}
}

func TestLoadConfig_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	config, err := LoadConfig(tmpDir, "vscode")
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil for missing config", err)
	}

	if config != nil {
		t.Errorf("LoadConfig() = %v, want nil for missing config", config)
	}
}

func TestLoadConfig_NoRemoteUserSpecified(t *testing.T) {
	// Create temp dir with devcontainer.json that has no remoteUser field
	tmpDir := t.TempDir()
	devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
	_ = os.Mkdir(devcontainerDir, 0755)

	// Config without remoteUser field
	configContent := `{
		"image": "mcr.microsoft.com/devcontainers/base:ubuntu"
	}`

	_ = os.WriteFile(
		filepath.Join(devcontainerDir, "devcontainer.json"),
		[]byte(configContent),
		0644,
	)

	// Should use the provided default user "developer"
	config, err := LoadConfig(tmpDir, "developer")
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if config.RemoteUser != "developer" {
		t.Errorf("RemoteUser = %v, want developer (from defaultUser param)", config.RemoteUser)
	}
}

func TestGetDefaultConfig(t *testing.T) {
	// Test with empty strings - should use defaults
	config := GetDefaultConfig("", "")
	if config.Image != "ghcr.io/obra/packnplay-default:latest" {
		t.Errorf("GetDefaultConfig(\"\", \"\") Image = %v, want ghcr.io/obra/packnplay-default:latest", config.Image)
	}
	if config.RemoteUser != "vscode" {
		t.Errorf("GetDefaultConfig(\"\", \"\") RemoteUser = %v, want vscode", config.RemoteUser)
	}

	// Test with custom image
	customImage := "mycustom/image:v1"
	config = GetDefaultConfig(customImage, "")
	if config.Image != customImage {
		t.Errorf("GetDefaultConfig(%v, \"\") Image = %v, want %v", customImage, config.Image, customImage)
	}
	if config.RemoteUser != "vscode" {
		t.Errorf("GetDefaultConfig(%v, \"\") RemoteUser = %v, want vscode", customImage, config.RemoteUser)
	}

	// Test with custom user
	customUser := "developer"
	config = GetDefaultConfig("", customUser)
	if config.Image != "ghcr.io/obra/packnplay-default:latest" {
		t.Errorf("GetDefaultConfig(\"\", %v) Image = %v, want ghcr.io/obra/packnplay-default:latest", customUser, config.Image)
	}
	if config.RemoteUser != customUser {
		t.Errorf("GetDefaultConfig(\"\", %v) RemoteUser = %v, want %v", customUser, config.RemoteUser, customUser)
	}

	// Test with both custom image and user
	config = GetDefaultConfig(customImage, customUser)
	if config.Image != customImage {
		t.Errorf("GetDefaultConfig(%v, %v) Image = %v, want %v", customImage, customUser, config.Image, customImage)
	}
	if config.RemoteUser != customUser {
		t.Errorf("GetDefaultConfig(%v, %v) RemoteUser = %v, want %v", customImage, customUser, config.RemoteUser, customUser)
	}
}
