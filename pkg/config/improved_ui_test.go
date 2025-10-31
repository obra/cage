package config

import (
	"testing"
)

func TestSingleScreenConfigurationForm(t *testing.T) {
	// Test that configuration form uses single group for better navigation

	existing := &Config{
		ContainerRuntime: "docker",
		DefaultCredentials: Credentials{
			Git: true,
			SSH: false,
			GH:  true,
		},
	}

	// Create the form that should be used in configuration
	form, _ := createConfigurationForm(existing)

	// Should have exactly one group (single screen)
	groupCount := getFormGroupCount(form)
	if groupCount != 1 {
		t.Errorf("Form should have 1 group for single-screen navigation, got %d", groupCount)
	}

	// Should have all fields in that single group
	fieldCount := getFormFieldCount(form)
	expectedFields := 7 // runtime + 5 credentials + default container = 7 minimum
	if fieldCount < expectedFields {
		t.Errorf("Form should have at least %d fields, got %d", expectedFields, fieldCount)
	}
}

func TestConfigurationFormLayout(t *testing.T) {
	// Test that form uses proper layout and styling

	existing := &Config{
		DefaultContainer: DefaultContainerConfig{
			Image: "custom/image:latest",
		},
	}

	form, _ := createConfigurationForm(existing)

	// Form should be configured for optimal layout
	if !hasOptimalLayout(form) {
		t.Error("Form should be configured with optimal layout for navigation")
	}

	// Should show current values as defaults
	if !showsCurrentValues(form, existing) {
		t.Error("Form should display current configuration values as defaults")
	}
}