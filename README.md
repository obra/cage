# Cage

> **⚠️ WARNING: This code is untested and experimental. Use at your own risk. It has not been validated in production environments.**

Cage launches commands (like Claude Code, Codex, Gemini) inside isolated Docker containers with automated worktree and dev container management.

## Features

- **Sandboxed Execution**: Run AI coding assistants in isolated Docker containers
- **Automatic Worktree Management**: Creates git worktrees in XDG-compliant locations (`~/.local/share/cage/worktrees`)
- **Dev Container Support**: Uses project's `.devcontainer/devcontainer.json` or feature-rich default with AI CLIs pre-installed
- **Credential Management**: Interactive first-run setup for git, GitHub CLI, GPG, and npm credentials
- **Clean Environment**: Only passes safe environment variables (terminal/locale), no host pollution
- **macOS Keychain Integration**: Automatically extracts Claude and GitHub CLI credentials from macOS Keychain

## Installation

```bash
go build -o cage .
sudo mv cage /usr/local/bin/
```

Or install directly:

```bash
go install github.com/obra/cage@latest
```

## Quick Start

On first run, cage will prompt you to configure which credentials to mount (git, GitHub CLI, GPG, npm). Your choices are saved to `~/.config/cage/config.json`.

```bash
# Run Claude Code in a sandboxed container (creates worktree automatically)
cage run claude

# Run in a specific worktree
cage run --worktree=feature-auth claude

# Run with all credentials enabled
cage run --all-creds claude

# List running containers
cage list

# Stop all containers
cage stop --all
```

## Usage

### Basic Commands

```bash
# Run command in container (auto-creates worktree from current branch)
cage run <command>

# Use specific worktree (creates if doesn't exist, uses if exists)
cage run --worktree=<name> <command>

# Skip worktree, use current directory
cage run --no-worktree <command>

# Pass arguments to the command
cage run bash -c "echo hello && ls"

# Attach to running container
cage attach --worktree=<name>

# Stop specific container
cage stop --worktree=<name>

# Stop all cage containers
cage stop --all

# List all running containers
cage list
```

### Credential Flags

Override default credential settings per-invocation:

```bash
# Enable specific credentials
cage run --git-creds claude           # Mount git config and SSH keys
cage run --gh-creds claude            # Mount GitHub CLI credentials
cage run --gpg-creds claude           # Mount GPG keys for signing
cage run --npm-creds claude           # Mount npm credentials
cage run --all-creds claude           # Mount all available credentials
```

### Environment Variables

```bash
# Set specific environment variable
cage run --env DEBUG=1 claude

# Pass through variable from host
cage run --env EDITOR bash

# Multiple variables
cage run --env DEBUG=1 --env EDITOR bash
```

## How It Works

### Worktree Management

Cage creates git worktrees in XDG-compliant locations for isolation:

- **Location**: `~/.local/share/cage/worktrees/<project>/<worktree>` (or `$XDG_DATA_HOME/cage/worktrees`)
- **Auto-create**: If you're in a git repo without `--worktree` flag, uses current branch name
- **Explicit**: `--worktree=<name>` creates new or connects to existing worktree
- **Skip**: `--no-worktree` uses current directory without git worktree
- **Auto-connect**: If container already running for a worktree, automatically connects to it
- **Git integration**: Main repo's `.git` directory mounted so git commands work correctly

### Dev Container Discovery

1. Checks for `.devcontainer/devcontainer.json` in project
2. Falls back to `ghcr.io/obra/cage-default:latest` if not found
3. Supports both `image` (pulls) and `dockerFile` (builds) fields
4. Auto-pulls/builds images as needed

**Default container includes:**
- Node.js v22 LTS
- AI CLI tools: Claude Code (`claude`), OpenAI Codex (`codex`), Google Gemini (`gemini`)
- GitHub CLI (`gh`)
- Git and common development utilities

## Rebuilding the Default Container

See [.devcontainer/README.md](.devcontainer/README.md) for instructions on building and publishing the default container image.

### Credential Handling

**Interactive Setup (first run):**
On first run, cage prompts you to choose which credentials to enable by default using a beautiful terminal UI.

**Credentials are mounted read-only for security:**
- **Git**: `~/.gitconfig` and `~/.ssh` (for git operations and SSH keys)
- **GitHub CLI**: `~/.config/gh` (copied from Keychain on macOS, mounted on Linux)
- **GPG**: `~/.gnupg` (for commit signing)
- **npm**: `~/.npmrc` (for authenticated package operations)

**macOS Keychain Integration:**
- Claude credentials automatically extracted from Keychain (`Claude Code-credentials`)
- GitHub CLI credentials extracted and base64-decoded from Keychain (`gh:github.com`)
- Credentials copied into container (not mounted) to avoid file locking

### File Mounts

- `~/.claude` → mounted read-write (skills, plugins, history)
- `~/.claude.json` → copied into container (avoids file lock conflicts)
- Worktree → mounted at `/workspace`
- Main repo `.git` → mounted at its real path (git commands work)

### Environment Variables

**Safe whitelist approach:**
- Only `TERM`, `LANG`, `LC_*`, `COLORTERM` passed from host
- `HOME=/home/vscode` set in container
- `IS_SANDBOX=1` marker added
- `PATH` uses container default (not polluted from host)
- Use `--env KEY=value` or `--env KEY` to pass additional variables

### Container Lifecycle

- **Persistent containers**: Started with `cage run`, stay running after command exits
- **Auto-attach**: Running `cage run` again connects to existing container
- **Labeled**: All containers tagged with `managed-by=cage` for tracking
- **Clean**: Use `cage stop --all` to stop and remove all cage containers

## Requirements

- **Docker**: Docker Desktop on macOS, or Docker Engine on Linux
- **Git**: For worktree functionality
- **Go 1.23+**: For building from source
- **Optional**: GitHub CLI (`gh`) for GitHub operations

## Configuration

### Config File

`~/.config/cage/config.json` (XDG-compliant):

```json
{
  "default_credentials": {
    "git": true,
    "gh": true,
    "gpg": false,
    "npm": false
  }
}
```

Created interactively on first run. Edit manually or delete to reconfigure.

### Environment Variables

- `DOCKER_CMD`: Override docker command (e.g., `DOCKER_CMD=podman cage run ...`)
- `XDG_DATA_HOME`: Override data directory (default: `~/.local/share`)
- `XDG_CONFIG_HOME`: Override config directory (default: `~/.config`)

## Examples

```bash
# First run - interactive credential setup, then run Claude
cage run claude

# Run in specific worktree with all credentials
cage run --worktree=bug-fix --all-creds claude

# Run with custom environment variables
cage run --env DEBUG=1 --env EDITOR bash -c "echo \$EDITOR"

# Get a shell in the container
cage run --worktree=feature bash

# Run command in existing container (auto-connects)
cage run --worktree=feature npm test

# Attach with interactive shell
cage attach --worktree=feature

# List all running containers
cage list

# Stop specific container
cage stop --worktree=feature

# Stop all cage containers
cage stop --all
```

## License

MIT
