# packnplay Release Notes

## Version 1.1.0 - Enhanced Configuration & Container Management

### 🚀 Major Features

#### Host Path Preservation
- **BREAKING CHANGE**: Containers now mount at exact host paths instead of `/workspace`
- Perfect path consistency between host and container environments
- Improves compatibility with tools that expect consistent absolute paths
- Benefits git worktrees, symlinks, and cross-container workflows

#### Professional Configuration Interface
- **NEW**: `packnplay configure` command for safe configuration editing
- Clean scrollable sections interface with stable navigation
- Professional styling with colored toggles and consistent layout
- Text field editing for container image customization
- Preserves all existing settings during configuration updates

#### Configurable Default Container
- **NEW**: Support for any container image as default (not just packnplay's)
- Configure custom images from any registry (`my-company/dev-env:latest`)
- Smart version update notifications with specific version information
- Configurable update checking and auto-pull behavior

#### Enhanced Container Information
- **NEW**: `packnplay list` shows host directory paths for all containers
- **NEW**: `packnplay list --verbose` shows original launch commands
- Verbose mode uses clean block format for better readability
- Detailed container debugging information for troubleshooting

#### Improved Error Messages
- Rich container details when conflicts occur (name, status, host path, launch command)
- Smart command suggestions with actual container names
- Clean, actionable error messages with copy-paste ready commands

### 🔧 Technical Improvements

#### Container Management
- **NEW**: `packnplay refresh-container` to update default container image
- **NEW**: `packnplay stop container-name` accepts container names directly
- Enhanced container labeling with host path and launch command information
- Automatic directory structure creation for deep host paths

#### Configuration System
- Safe configuration editing that preserves manual edits and advanced settings
- Version tracking for update notifications to avoid spam
- Default container preferences stored in `config.json`
- Backward compatibility with existing configuration files

#### Developer Experience
- Fixed duplicate error messages and command help output
- Clean navigation with arrow keys (↑/↓ for fields, ←/→ for buttons)
- Always-visible help text for better user guidance
- Professional UI components using bubbles library

### 🔄 Breaking Changes

#### Host Path Mounting
- **BREAKING**: Containers mount at host paths instead of `/workspace`
- **Migration**: Update any scripts that reference `/workspace` to use actual paths
- **Benefit**: Perfect path consistency and better tool compatibility

#### Container Stop Command Enhancement
- **NEW**: `packnplay stop container-name` now supported
- **EXISTING**: `packnplay stop --worktree=name` still works
- **BENEFIT**: Can copy container names from `packnplay list` output

### 💾 Configuration Schema Updates

```json
{
  "default_container": {
    "image": "ghcr.io/obra/packnplay-default:latest",
    "check_for_updates": true,
    "auto_pull_updates": false,
    "check_frequency_hours": 24
  }
}
```

### 🎯 Enhanced Default Container

#### Batteries-Included Development Environment
- **NEW**: Enhanced default container with comprehensive development tools
- **Languages**: Node.js LTS, Python 3.11+ with uv, Go latest, Rust latest
- **Cloud CLIs**: AWS CLI, Azure CLI, Google Cloud CLI, GitHub CLI
- **Utilities**: jq, yq, curl, wget, vim, nano, make, build-essential
- **AI Tools**: Claude Code, OpenAI Codex, Google Gemini, GitHub Copilot, etc.

### 🔍 New Commands

- `packnplay configure` - Interactive configuration editor
- `packnplay refresh-container` - Update default container image
- `packnplay list --verbose` - Detailed container information
- `packnplay stop container-name` - Stop container by name

### 🐛 Bug Fixes

- Fixed duplicate error messages during command execution
- Resolved configuration UI navigation and alignment issues
- Fixed container lifecycle management and cleanup
- Improved error handling for missing containers and invalid commands

### 📚 Documentation

- Updated README with new features and configuration options
- Enhanced help text for all commands
- Added examples for custom container configuration
- Documented host path preservation behavior and benefits

---

**Full Changelog**: Compare changes from previous release
**Installation**: Download from releases page
**Documentation**: See README.md for complete usage guide