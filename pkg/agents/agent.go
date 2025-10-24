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
		&CopilotAgent{},
		&QwenAgent{},
		&CodeWhispererAgent{},
		&DeepSeekAgent{},
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

// CopilotAgent implements GitHub Copilot CLI requirements
type CopilotAgent struct{}

func (c *CopilotAgent) Name() string                { return "copilot" }
func (c *CopilotAgent) ConfigDir() string           { return ".copilot" }
func (c *CopilotAgent) DefaultAPIKeyEnv() string    { return "GH_TOKEN" } // Uses GitHub auth
func (c *CopilotAgent) RequiresSpecialHandling() bool { return false }

func (c *CopilotAgent) GetMounts(homeDir string) []Mount {
	return []Mount{
		{
			HostPath:      filepath.Join(homeDir, ".copilot"),
			ContainerPath: "/home/vscode/.copilot",
			ReadOnly:      false,
		},
	}
}

// QwenAgent implements Qwen Code CLI requirements
type QwenAgent struct{}

func (q *QwenAgent) Name() string                { return "qwen" }
func (q *QwenAgent) ConfigDir() string           { return ".qwen" }
func (q *QwenAgent) DefaultAPIKeyEnv() string    { return "QWEN_API_KEY" }
func (q *QwenAgent) RequiresSpecialHandling() bool { return false }

func (q *QwenAgent) GetMounts(homeDir string) []Mount {
	return []Mount{
		{
			HostPath:      filepath.Join(homeDir, ".qwen"),
			ContainerPath: "/home/vscode/.qwen",
			ReadOnly:      false,
		},
	}
}

// CodeWhispererAgent implements Amazon CodeWhisperer CLI requirements
type CodeWhispererAgent struct{}

func (c *CodeWhispererAgent) Name() string                { return "codewhisperer" }
func (c *CodeWhispererAgent) ConfigDir() string           { return ".aws" } // Uses AWS config
func (c *CodeWhispererAgent) DefaultAPIKeyEnv() string    { return "AWS_ACCESS_KEY_ID" }
func (c *CodeWhispererAgent) RequiresSpecialHandling() bool { return false }

func (c *CodeWhispererAgent) GetMounts(homeDir string) []Mount {
	return []Mount{
		{
			HostPath:      filepath.Join(homeDir, ".aws"),
			ContainerPath: "/home/vscode/.aws",
			ReadOnly:      true, // AWS config should be read-only for security
		},
	}
}

// DeepSeekAgent implements DeepSeek CLI requirements
type DeepSeekAgent struct{}

func (d *DeepSeekAgent) Name() string                { return "deepseek" }
func (d *DeepSeekAgent) ConfigDir() string           { return ".deepseek" }
func (d *DeepSeekAgent) DefaultAPIKeyEnv() string    { return "DEEPSEEK_API_KEY" }
func (d *DeepSeekAgent) RequiresSpecialHandling() bool { return false }

func (d *DeepSeekAgent) GetMounts(homeDir string) []Mount {
	return []Mount{
		{
			HostPath:      filepath.Join(homeDir, ".deepseek"),
			ContainerPath: "/home/vscode/.deepseek",
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
		"GH_TOKEN",       // GitHub Copilot
		"GITHUB_TOKEN",   // GitHub fallback
		"QWEN_API_KEY",
		"AWS_ACCESS_KEY_ID",    // CodeWhisperer
		"AWS_SECRET_ACCESS_KEY", // CodeWhisperer
		"AWS_REGION",           // CodeWhisperer
		"DEEPSEEK_API_KEY",
	}
}