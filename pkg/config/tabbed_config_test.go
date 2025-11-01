package config

import (
	"strings"
	"testing"
)

func TestTabbedConfigInterface(t *testing.T) {
	// Test that tabbed config organizes settings into tabs

	existing := &Config{
		ContainerRuntime: "docker",
		DefaultCredentials: Credentials{SSH: true},
		DefaultContainer: DefaultContainerConfig{
			Image: "custom:latest",
		},
	}

	tabbed := createTabbedConfig(existing)

	// Should have multiple tabs
	if len(tabbed.tabs) < 3 {
		t.Errorf("Should have at least 3 tabs (Runtime, Credentials, Container), got %d", len(tabbed.tabs))
	}

	// Should start at first tab
	if tabbed.activeTab != 0 {
		t.Errorf("Should start at first tab, got %d", tabbed.activeTab)
	}

	// Should have tab titles
	if !tabbed.hasTab("Runtime") {
		t.Error("Should have Runtime tab")
	}

	if !tabbed.hasTab("Credentials") {
		t.Error("Should have Credentials tab")
	}

	if !tabbed.hasTab("Container") {
		t.Error("Should have Container tab")
	}
}

func TestTabNavigation(t *testing.T) {
	// Test tab switching with left/right arrows

	tabbed := createTabbedConfig(&Config{})

	// Should start at tab 0
	if tabbed.activeTab != 0 {
		t.Error("Should start at first tab")
	}

	// Right arrow should move to next tab
	tabbed = switchToNextTab(tabbed)
	if tabbed.activeTab != 1 {
		t.Error("Right arrow should move to tab 1")
	}

	// Left arrow should move to previous tab
	tabbed = switchToPrevTab(tabbed)
	if tabbed.activeTab != 0 {
		t.Error("Left arrow should move back to tab 0")
	}
}

func TestTabContentRendering(t *testing.T) {
	// Test that each tab shows appropriate content

	tabbed := createTabbedConfig(&Config{
		ContainerRuntime: "podman",
		DefaultCredentials: Credentials{SSH: true, GH: false},
		DefaultContainer: DefaultContainerConfig{
			Image: "my-image:v1",
			CheckForUpdates: false,
		},
	})

	// Runtime tab content
	tabbed.activeTab = 0
	runtimeContent := tabbed.renderActiveTabContent()
	if !strings.Contains(runtimeContent, "Container Runtime") {
		t.Error("Runtime tab should contain container runtime field")
	}

	// Credentials tab content
	tabbed.activeTab = 1
	credentialsContent := tabbed.renderActiveTabContent()
	if !strings.Contains(credentialsContent, "SSH keys") {
		t.Error("Credentials tab should contain SSH keys field")
	}

	// Container tab content
	tabbed.activeTab = 2
	containerContent := tabbed.renderActiveTabContent()
	if !strings.Contains(containerContent, "Container Image") {
		t.Error("Container tab should contain Container Image field")
	}

	if !strings.Contains(containerContent, "my-image:v1") {
		t.Error("Container tab should show current image value")
	}
}

func TestTabbedUILayout(t *testing.T) {
	// Test that tabbed interface renders properly

	tabbed := createTabbedConfig(&Config{})

	view := tabbed.renderTabbedView()

	// Should contain tab headers
	if !strings.Contains(view, "Runtime") || !strings.Contains(view, "Credentials") {
		t.Error("View should contain tab headers")
	}

	// Should show active tab indicator
	if !strings.Contains(view, "Runtime") {
		t.Error("Should show tab titles in view")
	}

	// Should have save/cancel buttons at bottom
	if !strings.Contains(view, "Save") || !strings.Contains(view, "Cancel") {
		t.Error("Should have save/cancel buttons")
	}
}

func TestFieldNavigationWithinTabs(t *testing.T) {
	// Test that field navigation works within active tab

	tabbed := createTabbedConfig(&Config{})

	// Should start at first field in first tab
	if tabbed.currentField != 0 {
		t.Error("Should start at first field")
	}

	// Should navigate between fields in tab
	tabbed = navigateDownInTab(tabbed)
	// Runtime tab only has 1 field, so should stay at field 0
	if tabbed.currentField != 0 {
		t.Error("Should stay at field 0 in runtime tab (only has 1 field)")
	}
}