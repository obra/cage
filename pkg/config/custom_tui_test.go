package config

import (
	"strings"
	"testing"
)

func TestCustomConfigTUI(t *testing.T) {
	// Test that custom config TUI implements proper navigation and layout

	existing := &Config{
		ContainerRuntime: "docker",
		DefaultCredentials: Credentials{
			Git: true,
			SSH: false,
			GH:  true,
		},
	}

	// Create config TUI model
	model := createConfigTUIModel(existing)

	// Should have proper navigation state
	if model.currentField < 0 {
		t.Error("TUI model should have valid currentField index")
	}

	// Should show all config fields in one screen
	fieldCount := model.getFieldCount()
	if fieldCount < 8 {
		t.Errorf("Should have at least 8 fields (runtime + 5 credentials + 2 buttons), got %d", fieldCount)
	}

	// Should have proper field types
	if !model.hasRuntimeField() {
		t.Error("Should have container runtime selection field")
	}

	if !model.hasCredentialFields() {
		t.Error("Should have credential toggle fields")
	}
}

func TestConfigTUINavigation(t *testing.T) {
	// Test arrow key navigation between all fields

	model := createConfigTUIModel(&Config{})

	// Should start at first field
	if model.currentField != 0 {
		t.Errorf("Should start at field 0, got %d", model.currentField)
	}

	// Should navigate down
	model = moveDown(model)
	if model.currentField != 1 {
		t.Errorf("Down arrow should move to field 1, got %d", model.currentField)
	}

	// Should navigate up
	model = moveUp(model)
	if model.currentField != 0 {
		t.Errorf("Up arrow should move back to field 0, got %d", model.currentField)
	}

	// Should wrap at boundaries
	model.currentField = model.getFieldCount() - 1
	model = moveDown(model)
	if model.currentField != 0 {
		t.Error("Should wrap to first field when going down from last")
	}
}

func TestConfigTUILayout(t *testing.T) {
	// Test that TUI renders with proper compact layout

	model := createConfigTUIModel(&Config{
		ContainerRuntime: "podman",
		DefaultCredentials: Credentials{SSH: true, GH: false},
	})

	view := model.renderView()

	// Should contain section headers
	if !containsText(view, "Container Runtime") {
		t.Error("View should contain Container Runtime section")
	}

	if !containsText(view, "Credentials") {
		t.Error("View should contain Credentials section")
	}

	// Should show current values
	if !containsText(view, "podman") {
		t.Error("View should show current runtime value")
	}

	// Should have right-aligned toggles (look for Yes/No at end of lines)
	if !containsText(view, "[Yes]") || !containsText(view, "[No]") {
		t.Error("View should contain right-aligned toggle indicators")
	}
}

func TestConfigTUIToggling(t *testing.T) {
	// Test toggling boolean values

	model := createConfigTUIModel(&Config{
		DefaultCredentials: Credentials{SSH: false},
	})

	// Find SSH field
	sshFieldIndex := model.findFieldIndex("SSH")
	if sshFieldIndex < 0 {
		t.Error("Should be able to find SSH field")
	}

	// Navigate to SSH field
	model.currentField = sshFieldIndex

	// Toggle the value
	model = toggleCurrentField(model)

	// Value should change
	if !model.getFieldValue(sshFieldIndex).(bool) {
		t.Error("SSH field should be toggled to true")
	}
}

// Helper function for tests
func containsText(s, substr string) bool {
	return strings.Contains(s, substr)
}