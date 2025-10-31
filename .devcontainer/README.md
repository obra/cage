# packnplay Batteries-Included Dev Container

This is the default dev container used by packnplay when a project doesn't have its own `.devcontainer/devcontainer.json`. It provides a comprehensive agentic development environment.

## What's Included

### 🗣️ Programming Languages & Runtimes
- **Node.js LTS** + npm - JavaScript/TypeScript development
- **Python 3.11+** + **uv** - Modern fast Python package management
- **Go latest** - CLI tools, backend development
- **Rust latest** - Systems programming, modern tools

### ☁️ Cloud CLIs (Multi-Architecture)
- **AWS CLI** - Amazon Web Services management
- **Azure CLI** - Microsoft cloud services
- **Google Cloud CLI** - Google Cloud Platform

### 🔧 Development Utilities
- **jq/yq** - JSON/YAML processing (essential for APIs)
- **curl/wget** - HTTP requests and downloads
- **vim/nano** - Text editing
- **make** - Build automation
- **build-essential** - Compilers and build tools
- **GitHub CLI (gh)** - GitHub workflows

### 🤖 AI CLI Tools
- `@anthropic-ai/claude-code` - Claude Code CLI
- `@openai/codex` - OpenAI Codex CLI
- `@google/gemini-cli` - Google Gemini CLI
- `@github/copilot` - GitHub Copilot CLI
- `@qwen-code/qwen-code` - Qwen Code CLI
- `@sourcegraph/amp` - Sourcegraph Amp CLI

### 👤 User Configuration
- User: `vscode` (UID 1000)
- All tools properly configured in PATH

## Building the Image

To build and publish the default packnplay container:

```bash
# Build locally
docker build -t ghcr.io/obra/packnplay-default:latest .devcontainer/

# Test it
docker run -it --rm ghcr.io/obra/packnplay-default:latest bash

# Push to GitHub Container Registry (requires authentication)
docker push ghcr.io/obra/packnplay-default:latest
```

## Using in Your Project

Projects can extend this by creating their own `.devcontainer/devcontainer.json`:

```json
{
  "image": "ghcr.io/obra/packnplay-default:latest",
  "remoteUser": "vscode"
}
```

Or build from this Dockerfile:

```json
{
  "dockerFile": "Dockerfile",
  "build": {
    "context": ".",
    "dockerfile": "Dockerfile"
  },
  "remoteUser": "vscode"
}
```
