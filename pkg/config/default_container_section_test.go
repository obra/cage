package config

import (
	"strings"
	"testing"
)

func TestDefaultContainerSection(t *testing.T) {
	// Test that Default Container section is added to settings modal

	existing := &Config{
		DefaultContainer: DefaultContainerConfig{
			Image:               "custom/image:latest",
			CheckForUpdates:     true,
			AutoPullUpdates:     false,
			CheckFrequencyHours: 12,
		},
	}

	modal := createSettingsModal(existing)

	// Should have 3 sections now (runtime, credentials, default-container)
	if len(modal.sections) != 3 {
		t.Errorf("Should have 3 sections (runtime, credentials, default-container), got %d", len(modal.sections))
	}

	// Third section should be default container
	containerSection := modal.sections[2]
	if containerSection.name != "default-container" {
		t.Errorf("Third section name = %v, want default-container", containerSection.name)
	}

	if containerSection.title != "Default Container" {
		t.Errorf("Container section title = %v, want Default Container", containerSection.title)
	}

	// Should have 4 fields in default container section
	if len(containerSection.fields) != 4 {
		t.Errorf("Default container section should have 4 fields, got %d", len(containerSection.fields))
	}
}

func TestDefaultContainerFields(t *testing.T) {
	// Test that default container fields are configured correctly

	existing := &Config{
		DefaultContainer: DefaultContainerConfig{
			Image:               "my-org/dev:v1.2.3",
			CheckForUpdates:     false,
			AutoPullUpdates:     true,
			CheckFrequencyHours: 48,
		},
	}

	modal := createSettingsModal(existing)
	containerSection := modal.sections[2] // Default container section

	// Test image field
	imageField := containerSection.fields[0]
	if imageField.name != "container-image" {
		t.Errorf("First field name = %v, want container-image", imageField.name)
	}

	if imageField.fieldType != "text" {
		t.Errorf("Image field type = %v, want text", imageField.fieldType)
	}

	if imageField.value != "my-org/dev:v1.2.3" {
		t.Errorf("Image field value = %v, want my-org/dev:v1.2.3", imageField.value)
	}

	// Test check for updates field
	checkField := containerSection.fields[1]
	if checkField.name != "check-updates" || checkField.fieldType != "toggle" {
		t.Error("Second field should be check-updates toggle")
	}

	if checkField.value != false {
		t.Error("Check updates field should have current value (false)")
	}

	// Test auto-pull field
	autoPullField := containerSection.fields[2]
	if autoPullField.name != "auto-pull" || autoPullField.fieldType != "toggle" {
		t.Error("Third field should be auto-pull toggle")
	}

	if autoPullField.value != true {
		t.Error("Auto-pull field should have current value (true)")
	}

	// Test frequency field
	freqField := containerSection.fields[3]
	if freqField.name != "check-frequency" || freqField.fieldType != "select" {
		t.Error("Fourth field should be check-frequency select")
	}

	if freqField.value != "48h" {
		t.Errorf("Frequency field value = %v, want 48h", freqField.value)
	}

	if len(freqField.options) != 5 {
		t.Errorf("Frequency field should have 5 options, got %d", len(freqField.options))
	}
}

func TestTextFieldSupport(t *testing.T) {
	// Test that settings modal supports text field type

	modal := createSettingsModal(&Config{})

	// Should support text field rendering
	textField := SettingsField{
		name:        "test-text",
		fieldType:   "text",
		title:       "Test Input",
		description: "Test text input field",
		value:       "test value",
	}

	// Should render text field without error
	rendered := modal.renderField(textField, false)
	if rendered == "" {
		t.Error("Text field should render successfully")
	}

	// Should show current value in rendered output
	if !strings.Contains(rendered, "test value") {
		t.Error("Rendered text field should show current value")
	}

	// Should support text editing mode
	if !modal.supportsTextEditing() {
		t.Error("Modal should support text editing for text fields")
	}
}

func TestDefaultContainerConfigUpdates(t *testing.T) {
	// Test that default container changes are applied to config safely

	modal := createSettingsModal(&Config{
		DefaultContainer: DefaultContainerConfig{
			Image: "original:latest",
		},
	})

	// Simulate changing default container settings
	containerSection := &modal.sections[2]
	containerSection.fields[0].value = "new-image:v2"      // Image
	containerSection.fields[1].value = true               // Check updates
	containerSection.fields[2].value = false              // Auto-pull
	containerSection.fields[3].value = "12h"              // Frequency

	// Should extract values correctly for config update
	updates := extractDefaultContainerUpdates(modal)

	if updates.DefaultContainer == nil {
		t.Error("Should generate DefaultContainer updates")
	}

	if updates.DefaultContainer.Image != "new-image:v2" {
		t.Errorf("Image update = %v, want new-image:v2", updates.DefaultContainer.Image)
	}

	if !updates.DefaultContainer.CheckForUpdates {
		t.Error("Check updates should be true")
	}

	if updates.DefaultContainer.AutoPullUpdates {
		t.Error("Auto-pull should be false")
	}

	if updates.DefaultContainer.CheckFrequencyHours != 12 {
		t.Errorf("Frequency = %v, want 12", updates.DefaultContainer.CheckFrequencyHours)
	}
}