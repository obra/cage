# Cage

Cage launches commands (like Claude Code) inside isolated Docker containers with automated worktree and dev container management.

## Features

- **Sandboxed Execution**: Run commands in isolated Docker containers
- **Automatic Worktree Management**: Creates and manages git worktrees automatically
- **Dev Container Support**: Uses project's `.devcontainer/devcontainer.json` or sensible defaults
- **UID Mapping**: Proper file ownership with idmap mounts (Linux 6.0.8+, Docker 28.5.1+)
- **Environment Proxying**: Forwards host environment with `IS_SANDBOX=1` indicator

## Installation

```bash
go build -o cage .
sudo mv cage /usr/local/bin/
```

Or install directly:

```bash
go install github.com/jessedrelick/cage@latest
```

## Usage

### Run a command in a container

```bash
cage run 'claude --dangerously-skip-permissions'
```

### Specify a worktree

```bash
cage run --worktree=feature-auth claude
```

### Use current directory without worktree

```bash
cage run --no-worktree bash
```

### Add environment variables

```bash
cage run --env DEBUG=1 --env LOG_LEVEL=trace claude
```

### Attach to running container

```bash
cage attach --worktree=feature-auth
```

### Stop a container

```bash
cage stop --worktree=feature-auth
```

### List all containers

```bash
cage list
```

## How It Works

### Worktree Management

- **Auto-create**: If you're in a git repo, cage creates a worktree based on current branch
- **Explicit**: Use `--worktree=<name>` to specify or create a worktree
- **Skip**: Use `--no-worktree` to use directory directly
- **Collision detection**: Errors if worktree already exists (prevents accidents)

### Dev Container Discovery

1. Checks for `.devcontainer/devcontainer.json` in project
2. Falls back to `mcr.microsoft.com/devcontainers/base:ubuntu` if not found
3. Supports both `image` (pulls) and `dockerFile` (builds) fields
4. Auto-pulls/builds images as needed

### File Mounts

- `~/.claude` → mounted read-write (skills, plugins, history)
- `~/.claude.json` → copied (avoids file lock conflicts)
- Project/worktree → mounted at `/workspace` with idmap

### Container Lifecycle

- Session-based: container runs until command exits
- Labeled: all containers tagged with `managed-by=cage`
- Multiple sessions can attach to running containers

## Requirements

- Linux 6.0.8+ (for idmap support)
- Docker 28.5.1+ (for idmap support)
- Git (for worktree features)
- Go 1.21+ (for building)

## Environment Variables

- `DOCKER_CMD`: Override docker command (e.g., `DOCKER_CMD=podman cage run ...`)

## Examples

```bash
# Run Claude in auto-created worktree
cd ~/myproject
cage run claude

# Run in specific worktree with debug logging
cage run --worktree=bug-fix --env DEBUG=1 --verbose claude

# Get a shell in the container
cage run --worktree=feature bash

# Attach to running container
cage attach --worktree=feature

# List all running containers
cage list

# Stop container
cage stop --worktree=feature
```

## License

MIT
