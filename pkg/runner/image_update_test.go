package runner

import (
	"testing"
	"time"

	"github.com/obra/packnplay/pkg/docker"
)

func TestGetImageAge(t *testing.T) {
	// Test that we can determine how old a local image is
	// Skip if no docker available (CI environments)
	dockerClient, err := docker.NewClient(false)
	if err != nil {
		t.Skip("Docker not available for image age testing")
	}

	imageName := "ubuntu:22.04"

	// This should return the age of the image (time since it was created/pulled)
	age, err := getImageAge(dockerClient, imageName)
	if err != nil {
		// If image doesn't exist, that's fine for this test
		if isImageNotFoundError(err) {
			t.Skip("Test image not available locally")
		}
		t.Errorf("getImageAge() error = %v, want nil for existing image", err)
	}

	if age < 0 {
		t.Errorf("getImageAge() = %v, want >= 0", age)
	}
}

func TestShouldUpdateImage(t *testing.T) {
	tests := []struct {
		name         string
		imageName    string
		age          time.Duration
		forceUpdate  bool
		shouldUpdate bool
	}{
		{
			name:         "fresh latest image should not update",
			imageName:    "ubuntu:latest",
			age:          1 * time.Hour,
			forceUpdate:  false,
			shouldUpdate: false,
		},
		{
			name:         "old latest image should update",
			imageName:    "ubuntu:latest",
			age:          25 * time.Hour, // > 24 hours
			forceUpdate:  false,
			shouldUpdate: true,
		},
		{
			name:         "old tagged image should not auto-update",
			imageName:    "ubuntu:22.04",
			age:          25 * time.Hour,
			forceUpdate:  false,
			shouldUpdate: false,
		},
		{
			name:         "force update should always update",
			imageName:    "ubuntu:22.04",
			age:          1 * time.Hour,
			forceUpdate:  true,
			shouldUpdate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldUpdateImage(tt.imageName, tt.age, tt.forceUpdate)
			if result != tt.shouldUpdate {
				t.Errorf("shouldUpdateImage(%v, %v, %v) = %v, want %v",
					tt.imageName, tt.age, tt.forceUpdate, result, tt.shouldUpdate)
			}
		})
	}
}

func TestImageUpdateLogic(t *testing.T) {
	// Test the complete image update decision logic

	// Case 1: Image doesn't exist locally - should pull
	missing := imageUpdateAction("nonexistent:latest", false)
	if missing.action != "pull" {
		t.Errorf("Missing image action = %v, want pull", missing.action)
	}
	if missing.reason != "image not found locally" {
		t.Errorf("Missing image reason = %v, want 'image not found locally'", missing.reason)
	}

	// Case 2: Fresh latest image - should use existing
	// (This would be tested with mocked image age)
}

// ImageUpdateAction is defined in runner.go

func TestRefreshDefaultContainerCommand(t *testing.T) {
	// Test that the refresh command pulls the default image

	// This function should force-pull the default packnplay image
	err := RefreshDefaultContainer(true) // verbose = true for testing

	// Should not error for valid operations
	if err != nil {
		// Only fail if it's not a "image not found" type error
		// (since we might not have the default image locally in tests)
		if !isImageNotFoundError(err) {
			t.Errorf("RefreshDefaultContainer() error = %v", err)
		}
	}
}