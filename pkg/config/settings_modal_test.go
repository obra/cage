package config

import (
	"strings"
	"testing"
)

func TestSettingsModalLayout(t *testing.T) {
	// Test that settings modal has proper sectioned layout

	existing := &Config{
		ContainerRuntime: "docker",
		DefaultCredentials: Credentials{
			SSH: true,
			GH:  false,
		},
	}

	modal := createSettingsModal(existing)

	// Should have distinct sections
	if !modal.hasSections() {
		t.Error("Settings modal should have distinct sections")
	}

	// Should have separate button area (not part of content sections)
	if !modal.hasSeparateButtonArea() {
		t.Error("Settings modal should have separate button area")
	}

	// Should have consistent indentation that doesn't break
	if !modal.hasConsistentIndentation() {
		t.Error("Settings modal should have consistent indentation")
	}
}

func TestSettingsModalNavigation(t *testing.T) {
	// Test that modal navigation works within sections

	modal := createSettingsModal(&Config{})

	// Should start at first field in first section
	if modal.currentSection != 0 || modal.currentField != 0 {
		t.Error("Should start at beginning of first section")
	}

	// Should navigate to next section (runtime section only has 1 field)
	modal = modal.navigateDown()
	if modal.currentSection != 1 || modal.currentField != 0 {
		t.Error("Should move to next section when no more fields in current section")
	}

	// Should eventually move to next section
	for i := 0; i < 10; i++ { // Navigate enough to reach next section
		modal = modal.navigateDown()
	}
	if modal.currentSection == 0 {
		t.Error("Should eventually move to next section")
	}
}

func TestSettingsModalSections(t *testing.T) {
	// Test that modal organizes config into logical sections

	modal := createSettingsModal(&Config{})

	sections := modal.getSections()

	// Should have runtime section
	if !hasSectionNamed(sections, "runtime") {
		t.Error("Should have runtime section")
	}

	// Should have credentials section
	if !hasSectionNamed(sections, "credentials") {
		t.Error("Should have credentials section")
	}

	// Should be expandable for future sections (container, env)
	if len(sections) < 2 {
		t.Error("Should have at least 2 sections for expandability")
	}
}

func TestSettingsModalButtonArea(t *testing.T) {
	// Test that buttons are separate from content and feel like modal actions

	modal := createSettingsModal(&Config{})

	view := modal.renderModal()

	// Should have save button
	if !strings.Contains(view, "Save") {
		t.Error("Modal should have save button")
	}

	// Should have cancel button
	if !strings.Contains(view, "Cancel") {
		t.Error("Modal should have cancel button")
	}

	// Buttons should be visually separate from content
	if !strings.Contains(view, "────") {
		t.Error("Buttons should be visually separated from content with separator line")
	}
}

func TestConsistentToggleRendering(t *testing.T) {
	// Test that toggles render consistently without jumping

	modal := createSettingsModal(&Config{
		DefaultCredentials: Credentials{SSH: true},
	})

	// Render toggle unfocused
	unfocused := modal.renderToggleField("SSH keys", true, false)

	// Render toggle focused
	focused := modal.renderToggleField("SSH keys", true, true)

	// Should have same character count to prevent jumping
	unfocusedLines := strings.Split(unfocused, "\n")
	focusedLines := strings.Split(focused, "\n")

	if len(unfocusedLines) != len(focusedLines) {
		t.Error("Focused and unfocused toggles should have same line count")
	}

	// First line should have same structure
	if len(unfocusedLines[0]) != len(focusedLines[0]) {
		t.Error("Toggle lines should have consistent length to prevent jumping")
	}
}

// Helper functions for testing
func hasSectionNamed(sections []SettingsSection, name string) bool {
	for _, section := range sections {
		if strings.Contains(strings.ToLower(section.name), name) {
			return true
		}
	}
	return false
}