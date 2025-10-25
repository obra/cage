# packnplay Default Dev Container

This is the default dev container used by packnplay when a project doesn't have its own `.devcontainer/devcontainer.json`.

## What's Included

- Microsoft devcontainer base (Ubuntu)
- Node.js LTS
- GitHub CLI (`gh`)
- AI CLI tools (npm global packages):
  - `@anthropic-ai/claude-code` - Claude Code CLI
  - `@openai/codex` - OpenAI Codex CLI
  - `@google/gemini-cli` - Google Gemini CLI
  - `@github/copilot` - GitHub Copilot CLI
  - `@qwen-code/qwen-code` - Qwen Code CLI
  - `@sourcegraph/amp` - Sourcegraph Amp CLI
- Cursor CLI (`cursor-agent`) - Installed via curl
- Git and common utilities
- User: `vscode` (UID 1000)

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

### Configuration Fields

**`image`**: Specifies which Docker image to use. Can be any valid Docker image tag.

**`dockerFile`**: Path to a Dockerfile for building a custom image. Use this when you need to install additional tools or customize the environment.

**`remoteUser`**: The username to use inside the container (default: `"vscode"`). This user must exist in the image.

**Important:** packnplay validates on startup that the specified user exists in the image. If you use a custom image or change `remoteUser`, ensure the user is created in your Dockerfile:

```dockerfile
# Example: Creating a custom user in your Dockerfile
RUN useradd -m -s /bin/bash -u 1000 myuser
USER myuser
```

### Advanced Example

For projects with specific requirements:

```json
{
  "dockerFile": "Dockerfile",
  "remoteUser": "developer",
  "build": {
    "args": {
      "NODE_VERSION": "20",
      "UID": "1000"
    }
  }
}
```

Make sure your Dockerfile creates the `developer` user with UID 1000.
