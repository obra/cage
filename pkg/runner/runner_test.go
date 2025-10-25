package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/obra/packnplay/pkg/config"
)

func TestGetOrCreateContainerCredentialFile(t *testing.T) {
	// Use temp directory for test
	tempDir := t.TempDir()
	os.Setenv("XDG_DATA_HOME", tempDir)
	defer os.Unsetenv("XDG_DATA_HOME")

	// Test file creation
	credFile, err := getOrCreateContainerCredentialFile("test-container")
	if err != nil {
		t.Fatalf("getOrCreateContainerCredentialFile() error = %v", err)
	}

	// Verify file exists
	if !fileExists(credFile) {
		t.Errorf("Credential file not created at %s", credFile)
	}

	// Verify file path format
	expectedDir := filepath.Join(tempDir, "packnplay", "credentials")
	expectedFile := filepath.Join(expectedDir, "claude-credentials.json")

	if credFile != expectedFile {
		t.Errorf("Credential file path = %v, want %v", credFile, expectedFile)
	}

	// Verify file permissions
	stat, err := os.Stat(credFile)
	if err != nil {
		t.Fatalf("Failed to stat credential file: %v", err)
	}

	if stat.Mode().Perm() != 0600 {
		t.Errorf("Credential file permissions = %v, want 0600", stat.Mode().Perm())
	}

	// Test second call returns same file
	credFile2, err := getOrCreateContainerCredentialFile("another-container")
	if err != nil {
		t.Fatalf("Second getOrCreateContainerCredentialFile() error = %v", err)
	}

	if credFile != credFile2 {
		t.Errorf("Second call returned different file: %v != %v", credFile, credFile2)
	}
}

func TestGetInitialContainerCredentials(t *testing.T) {
	// Test when no initial credentials available
	_, err := getInitialContainerCredentials()
	if err == nil {
		t.Skip("getInitialContainerCredentials() might find credentials on this system - skipping")
	}
}

func TestGetFileSize(t *testing.T) {
	// Create test file
	tempFile := filepath.Join(t.TempDir(), "test.txt")
	content := "test content"
	err := os.WriteFile(tempFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	size := getFileSize(tempFile)
	expectedSize := int64(len(content))

	if size != expectedSize {
		t.Errorf("getFileSize() = %v, want %v", size, expectedSize)
	}

	// Test non-existent file
	nonExistentSize := getFileSize("/non/existent/file")
	if nonExistentSize != 0 {
		t.Errorf("getFileSize() for non-existent file = %v, want 0", nonExistentSize)
	}
}

func TestRunConfig(t *testing.T) {
	// Test RunConfig struct fields
	cfg := &RunConfig{
		Path:           "/test/path",
		Worktree:       "feature-branch",
		NoWorktree:     false,
		Env:            []string{"TEST=value"},
		Verbose:        true,
		Runtime:        "docker",
		DefaultImage:   "ghcr.io/obra/packnplay-default:latest",
		DefaultUser:    "vscode",
		Command:        []string{"claude", "test"},
		DefaultEnvVars: []string{"ANTHROPIC_API_KEY"},
		Credentials: config.Credentials{
			Git: true,
			SSH: false,
		},
	}

	// Verify all fields are accessible
	if cfg.Path != "/test/path" {
		t.Errorf("RunConfig.Path = %v, want /test/path", cfg.Path)
	}

	if cfg.Worktree != "feature-branch" {
		t.Errorf("RunConfig.Worktree = %v, want feature-branch", cfg.Worktree)
	}

	if len(cfg.DefaultEnvVars) != 1 || cfg.DefaultEnvVars[0] != "ANTHROPIC_API_KEY" {
		t.Errorf("RunConfig.DefaultEnvVars = %v, want [ANTHROPIC_API_KEY]", cfg.DefaultEnvVars)
	}

	if cfg.DefaultImage != "ghcr.io/obra/packnplay-default:latest" {
		t.Errorf("RunConfig.DefaultImage = %v, want ghcr.io/obra/packnplay-default:latest", cfg.DefaultImage)
	}

	if cfg.DefaultUser != "vscode" {
		t.Errorf("RunConfig.DefaultUser = %v, want vscode", cfg.DefaultUser)
	}
}

func TestValidateUserExistsInImage(t *testing.T) {
	// Test that function exists and has correct signature
	// This test documents the expected behavior even if we can't fully test without real docker

	// Mock docker client for testing
	// In real scenario, validateUserExistsInImage would:
	// 1. Run: docker run --rm <image> id -u <username>
	// 2. If command succeeds, user exists
	// 3. If command fails, return error with message about missing user

	// Test with nil client should fail gracefully
	err := validateUserExistsInImage(nil, "test-image", "testuser", false)
	if err == nil {
		t.Error("validateUserExistsInImage with nil client should return error")
	}
}