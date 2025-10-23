# Cage Default Dev Container

This is the default dev container used by cage when a project doesn't have its own `.devcontainer/devcontainer.json`.

## What's Included

- Microsoft devcontainer base (Ubuntu)
- Node.js LTS
- AI CLI tools (npm global packages):
  - `@anthropic-ai/claude-code` - Claude Code CLI
  - `@openai/codex` - OpenAI Codex CLI
  - `@google/gemini-cli` - Google Gemini CLI
- Git and common utilities
- User: `vscode` (UID 1000)

## Building the Image

To build and publish the default cage container:

```bash
# Build locally
docker build -t ghcr.io/obra/cage-default:latest .devcontainer/

# Test it
docker run -it --rm ghcr.io/obra/cage-default:latest bash

# Push to GitHub Container Registry (requires authentication)
docker push ghcr.io/obra/cage-default:latest
```

## Using in Your Project

Projects can extend this by creating their own `.devcontainer/devcontainer.json`:

```json
{
  "image": "ghcr.io/obra/cage-default:latest",
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
