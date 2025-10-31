package config

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
)

func TestBubblesListConfiguration(t *testing.T) {
	// Test that bubbles list provides stable configuration interface

	existing := &Config{
		ContainerRuntime: "docker",
		DefaultCredentials: Credentials{
			Git: true,
			SSH: false,
			GH:  true,
		},
	}

	// Create list-based config model
	model := createConfigListModel(existing)

	// Should have proper list with all config items
	items := model.getConfigItems()
	if len(items) < 6 {
		t.Errorf("Should have at least 6 config items (runtime + 5 credentials), got %d", len(items))
	}

	// Should use bubbles list for navigation
	if !model.usesBubblesList() {
		t.Error("Should use bubbles list component for navigation")
	}

	// Should have stable item rendering (no content jumping)
	if !model.hasStableRendering() {
		t.Error("Should have stable item rendering without content jumping")
	}
}

func TestConfigItemImplementation(t *testing.T) {
	// Test that config items implement list.Item interface correctly

	// Create config items for testing
	runtimeItem := &ConfigListItem{
		name:        "runtime",
		itemType:    "select",
		title:       "Container Runtime",
		description: "Choose which container CLI to use",
		value:       "docker",
		options:     []string{"docker", "podman"},
	}

	toggleItem := &ConfigListItem{
		name:        "ssh",
		itemType:    "toggle",
		title:       "SSH keys",
		description: "Mount ~/.ssh for authentication",
		value:       false,
	}

	// Should implement list.Item interface
	var _ list.Item = runtimeItem
	var _ list.Item = toggleItem

	// Should have proper string representation
	if runtimeItem.FilterValue() == "" {
		t.Error("Config item should have filter value")
	}
}

func TestProfessionalToggleRendering(t *testing.T) {
	// Test that toggles render professionally without brackets

	item := &ConfigListItem{
		name:        "ssh",
		itemType:    "toggle",
		title:       "SSH keys",
		description: "Mount ~/.ssh for authentication",
		value:       true,
	}

	delegate := &ConfigListDelegate{}

	// Render as focused item
	focusedView := delegate.renderItem(item, true, 80)

	// Should not contain old bracket format
	if containsText(focusedView, "[Yes]") || containsText(focusedView, "[No]") {
		t.Error("Should not use bracketed Yes/No format")
	}

	// Should use colored toggle format
	if !containsText(focusedView, "ON") && !containsText(focusedView, "ENABLED") {
		t.Error("Should use colored toggle format")
	}

	// Should have consistent spacing (no jumping)
	unfocusedView := delegate.renderItem(item, false, 80)
	if len(focusedView) != len(unfocusedView) {
		t.Error("Focused and unfocused views should have consistent spacing")
	}
}

func TestStableButtonRendering(t *testing.T) {
	// Test that buttons render cleanly without problematic borders

	saveItem := &ConfigListItem{
		name:        "save",
		itemType:    "button",
		title:       "Save Configuration",
		description: "Save changes to config file",
	}

	delegate := &ConfigListDelegate{}

	// Should render without borders when not focused
	unfocusedView := delegate.renderItem(saveItem, false, 80)
	if containsText(unfocusedView, "┌") || containsText(unfocusedView, "│") {
		t.Error("Unfocused buttons should not have borders")
	}

	// Should have clean highlighting when focused
	focusedView := delegate.renderItem(saveItem, true, 80)
	if !containsText(focusedView, "Save Configuration") {
		t.Error("Focused button should show title")
	}
}

// Helper function
func containsText(s, substr string) bool {
	return strings.Contains(s, substr)
}