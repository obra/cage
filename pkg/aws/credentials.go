package aws

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// AWSCredentials represents temporary AWS credentials
// Format matches AWS credential_process specification:
// https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sourcing-external.html
type AWSCredentials struct {
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
	SessionToken    string `json:"SessionToken"`
	Expiration      string `json:"Expiration"`
	Version         int    `json:"Version"`
}

// GetCredentialsFromProcess executes a credential_process and returns the credentials
func GetCredentialsFromProcess(credentialProcess string) (*AWSCredentials, error) {
	if credentialProcess == "" {
		return nil, fmt.Errorf("empty credential_process")
	}

	// Use shell to execute command - this properly handles quoted arguments
	// and shell syntax that credential_process commands may use
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", credentialProcess)

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Include stderr in error message for debugging
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("credential_process timed out after 30 seconds")
		}
		return nil, fmt.Errorf("credential_process failed: %w\nOutput: %s", err, string(output))
	}

	// Parse the JSON output
	var creds AWSCredentials
	if err := json.Unmarshal(output, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credential_process JSON output: %w\nOutput: %s", err, string(output))
	}

	// Validate required fields are present
	if creds.AccessKeyID == "" {
		return nil, fmt.Errorf("credential_process output missing required field 'AccessKeyId'")
	}
	if creds.SecretAccessKey == "" {
		return nil, fmt.Errorf("credential_process output missing required field 'SecretAccessKey'")
	}

	return &creds, nil
}

// ParseAWSConfig parses AWS config file and returns credential_process for a profile
func ParseAWSConfig(profile string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Check for AWS_CONFIG_FILE override
	configPath := os.Getenv("AWS_CONFIG_FILE")
	if configPath == "" {
		configPath = filepath.Join(homeDir, ".aws", "config")
	}

	file, err := os.Open(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to open AWS config at %s: %w", configPath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentProfile string
	var credentialProcess string
	var profileFound bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Check for profile section
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			// Extract profile name from [profile name] or [default]
			profileLine := strings.Trim(line, "[]")
			profileLine = strings.TrimSpace(profileLine) // Handle extra whitespace
			if strings.HasPrefix(profileLine, "profile ") {
				currentProfile = strings.TrimSpace(strings.TrimPrefix(profileLine, "profile "))
			} else {
				currentProfile = profileLine
			}
			if currentProfile == profile {
				profileFound = true
			}
			continue
		}

		// Check for credential_process in current profile
		if currentProfile == profile {
			if strings.HasPrefix(line, "credential_process") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					value := strings.TrimSpace(parts[1])
					// Strip inline comments (# or ;)
					if idx := strings.IndexAny(value, "#;"); idx > 0 {
						value = strings.TrimSpace(value[:idx])
					}
					credentialProcess = value
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading AWS config: %w", err)
	}

	if !profileFound {
		return "", fmt.Errorf("profile '%s' not found in %s", profile, configPath)
	}

	if credentialProcess == "" {
		return "", fmt.Errorf("profile '%s' exists but has no credential_process configured", profile)
	}

	return credentialProcess, nil
}

// GetAWSEnvVars returns all AWS_* environment variables, excluding problematic ones
func GetAWSEnvVars() map[string]string {
	envVars := make(map[string]string)

	// Variables to exclude - these reference host-specific services that won't work in container
	excludeVars := map[string]bool{
		"AWS_CONTAINER_CREDENTIALS_RELATIVE_URI": true,
		"AWS_CONTAINER_CREDENTIALS_FULL_URI":     true,
		"AWS_CONTAINER_AUTHORIZATION_TOKEN":      true,
	}

	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "AWS_") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				key := parts[0]
				if !excludeVars[key] {
					envVars[key] = parts[1]
				}
			}
		}
	}
	return envVars
}

// HasStaticCredentials checks if static AWS credentials are already set
// This includes both long-term credentials and temporary credentials with session tokens
func HasStaticCredentials() bool {
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	return accessKey != "" && secretKey != ""
}
