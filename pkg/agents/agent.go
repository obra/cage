package agents

import (
	"path/filepath"
)

// Agent defines the interface for AI coding agents
type Agent interface {
	Name() string
	ConfigDir() string           // e.g., ".claude", ".codex", ".gemini"
	DefaultAPIKeyEnv() string    // e.g., "ANTHROPIC_API_KEY", "OPENAI_API_KEY"
	RequiresSpecialHandling() bool // Claude needs credential overlay, others don't
	GetMounts(homeDir string) []Mount
}

// Mount represents a directory or file mount
type Mount struct {
	HostPath      string
	ContainerPath string
	ReadOnly      bool
}

// GetSupportedAgents returns all supported AI coding agents
func GetSupportedAgents() []Agent {
	return []Agent{
		&ClaudeAgent{},
		&CodexAgent{},
		&GeminiAgent{},
	}
}

// ClaudeAgent implements Claude Code specific requirements
type ClaudeAgent struct{}

func (c *ClaudeAgent) Name() string                { return "claude" }
func (c *ClaudeAgent) ConfigDir() string           { return ".claude" }
func (c *ClaudeAgent) DefaultAPIKeyEnv() string    { return "ANTHROPIC_API_KEY" }
func (c *ClaudeAgent) RequiresSpecialHandling() bool { return true } // Needs credential overlay

func (c *ClaudeAgent) GetMounts(homeDir string) []Mount {
	return []Mount{
		{
			HostPath:      filepath.Join(homeDir, ".claude"),
			ContainerPath: "/home/vscode/.claude",
			ReadOnly:      false, // Needs write for plugins, etc.
		},
	}
}

// CodexAgent implements OpenAI Codex specific requirements
type CodexAgent struct{}

func (c *CodexAgent) Name() string                { return "codex" }
func (c *CodexAgent) ConfigDir() string           { return ".codex" }
func (c *CodexAgent) DefaultAPIKeyEnv() string    { return "OPENAI_API_KEY" }
func (c *CodexAgent) RequiresSpecialHandling() bool { return false } // Simple config mount

func (c *CodexAgent) GetMounts(homeDir string) []Mount {
	return []Mount{
		{
			HostPath:      filepath.Join(homeDir, ".codex"),
			ContainerPath: "/home/vscode/.codex",
			ReadOnly:      false,
		},
	}
}

// GeminiAgent implements Google Gemini CLI specific requirements
type GeminiAgent struct{}

func (g *GeminiAgent) Name() string                { return "gemini" }
func (g *GeminiAgent) ConfigDir() string           { return ".gemini" }
func (g *GeminiAgent) DefaultAPIKeyEnv() string    { return "GEMINI_API_KEY" }
func (g *GeminiAgent) RequiresSpecialHandling() bool { return false } // Simple config mount

func (g *GeminiAgent) GetMounts(homeDir string) []Mount {
	return []Mount{
		{
			HostPath:      filepath.Join(homeDir, ".gemini"),
			ContainerPath: "/home/vscode/.gemini",
			ReadOnly:      false,
		},
	}
}

// GetDefaultEnvVars returns default environment variables that should be proxied
func GetDefaultEnvVars() []string {
	return []string{
		"ANTHROPIC_API_KEY",
		"OPENAI_API_KEY",
		"GEMINI_API_KEY",
		"GOOGLE_API_KEY", // Gemini fallback
	}
}