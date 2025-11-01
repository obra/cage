package config

import (
	"strings"
	"testing"
)

func TestEnvironmentVariablesSection(t *testing.T) {
	// Test that Environment Variables section is added to settings modal

	existing := &Config{
		DefaultEnvVars: []string{"ANTHROPIC_API_KEY", "DEBUG"},
	}

	modal := createSettingsModal(existing)

	// Should have 4 sections now (runtime, credentials, container, environment)
	if len(modal.sections) != 4 {
		t.Errorf("Should have 4 sections including environment, got %d", len(modal.sections))
	}

	// Fourth section should be environment variables
	envSection := modal.sections[3]
	if envSection.name != "environment" {
		t.Errorf("Fourth section name = %v, want environment", envSection.name)
	}

	if envSection.title != "Environment Variables" {
		t.Errorf("Environment section title = %v, want Environment Variables", envSection.title)
	}
}

func TestEnvEditorFieldType(t *testing.T) {
	// Test that environment section uses env-editor field type

	modal := createSettingsModal(&Config{})
	envSection := modal.sections[3]

	// Should have env-editor field
	if len(envSection.fields) != 1 {
		t.Errorf("Environment section should have 1 env-editor field, got %d", len(envSection.fields))
	}

	editorField := envSection.fields[0]
	if editorField.fieldType != "env-editor" {
		t.Errorf("Environment field type = %v, want env-editor", editorField.fieldType)
	}

	if editorField.name != "env-vars" {
		t.Errorf("Environment field name = %v, want env-vars", editorField.name)
	}
}

func TestEnvVariableTextGeneration(t *testing.T) {
	// Test converting config to editable text format

	config := &Config{
		DefaultEnvVars: []string{"API_KEY", "DEBUG", "NODE_ENV"},
	}

	text := generateEnvVarText(config)

	// Should contain API keys as comments
	if !strings.Contains(text, "# API Keys") {
		t.Error("Should contain API keys section header")
	}

	// Should contain pass-through variables
	if !strings.Contains(text, "API_KEY") {
		t.Error("Should contain API_KEY variable")
	}

	if !strings.Contains(text, "DEBUG") {
		t.Error("Should contain DEBUG variable")
	}

	// Should not have equals signs for pass-through vars
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "API_KEY") && strings.Contains(line, "=") {
			t.Error("Pass-through variables should not have equals signs")
		}
	}
}

func TestEnvVariableTextParsing(t *testing.T) {
	// Test parsing edited text back to config format

	editorText := `# API Keys (always passed)
ANTHROPIC_API_KEY
OPENAI_API_KEY

# Development settings
DEBUG=1
NODE_ENV=development
API_BASE_URL=https://api.dev.com

# Database
DATABASE_URL

# Disabled
# UNUSED_VAR=disabled`

	parsed := parseEnvVarText(editorText)

	// Should extract pass-through variables
	if !containsEnvVar(parsed.PassThrough, "ANTHROPIC_API_KEY") {
		t.Error("Should parse ANTHROPIC_API_KEY as pass-through")
	}

	if !containsEnvVar(parsed.PassThrough, "DATABASE_URL") {
		t.Error("Should parse DATABASE_URL as pass-through")
	}

	// Should extract fixed-value variables
	if !containsEnvVar(parsed.FixedValues, "DEBUG") {
		t.Error("Should parse DEBUG as fixed-value")
	}

	debugVar := getEnvVar(parsed.FixedValues, "DEBUG")
	if debugVar == nil || debugVar.Value != "1" {
		t.Error("DEBUG should have value '1'")
	}

	// Should ignore commented variables
	if containsEnvVar(parsed.PassThrough, "UNUSED_VAR") || containsEnvVar(parsed.FixedValues, "UNUSED_VAR") {
		t.Error("Should ignore commented variables")
	}
}

func TestEnvEditorValidation(t *testing.T) {
	// Test real-time validation of environment variables

	validator := createEnvValidator()

	// Valid syntax should pass
	validText := "VALID_VAR=value\nANOTHER_VAR"
	result := validator.validateText(validText)

	if !result.IsValid {
		t.Error("Valid syntax should pass validation")
	}

	if result.VariableCount != 2 {
		t.Errorf("Should count 2 variables, got %d", result.VariableCount)
	}

	// Invalid syntax should fail
	invalidText := "=invalid\nVALID_VAR=good\n=another_invalid"
	result = validator.validateText(invalidText)

	if result.IsValid {
		t.Error("Invalid syntax should fail validation")
	}

	if len(result.Errors) != 2 {
		t.Errorf("Should have 2 validation errors, got %d", len(result.Errors))
	}
}

func TestSplitPaneLayout(t *testing.T) {
	// Test that split-pane layout renders correctly

	editor := createEnvEditor(&Config{})

	view := editor.renderSplitPane(80, 24)

	// Should contain editor area
	if !strings.Contains(view, "Editor") {
		t.Error("Should contain editor area")
	}

	// Should contain help area
	if !strings.Contains(view, "Format") || !strings.Contains(view, "Examples") {
		t.Error("Should contain help documentation")
	}

	// Should show variable count
	if !strings.Contains(view, "variables configured") {
		t.Error("Should show variable count")
	}
}

// Helper functions for testing
func containsEnvVar(vars []EnvVar, name string) bool {
	for _, v := range vars {
		if v.Name == name {
			return true
		}
	}
	return false
}

func getEnvVar(vars []EnvVar, name string) *EnvVar {
	for _, v := range vars {
		if v.Name == name {
			return &v
		}
	}
	return nil
}

// Types are implemented in config.go