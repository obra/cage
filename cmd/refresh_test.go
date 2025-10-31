package cmd

import (
	"testing"
)

func TestRefreshCommand(t *testing.T) {
	// Test that the refresh command exists and has proper help text

	// The refresh command should be available
	if refreshCmd == nil {
		t.Error("refreshCmd should be defined")
	}

	if refreshCmd.Use != "refresh-default-container" {
		t.Errorf("refresh command Use = %v, want refresh-default-container", refreshCmd.Use)
	}

	if refreshCmd.Short == "" {
		t.Error("refresh command should have Short description")
	}
}

func TestRefreshCommandFlags(t *testing.T) {
	// Test that the refresh command has appropriate flags

	// Should have verbose flag for detailed output
	flag := refreshCmd.Flags().Lookup("verbose")
	if flag == nil {
		t.Error("refresh command should have --verbose flag")
	}
}