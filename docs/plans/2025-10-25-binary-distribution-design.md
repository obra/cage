# Binary Distribution Design

**Date:** 2025-10-25
**Status:** Approved

## Overview

Implement automated binary distribution for packnplay using GoReleaser, supporting macOS and Linux on amd64 and arm64 architectures, with GitHub Releases and Homebrew tap integration.

## Requirements

### Distribution Channels
- GitHub Releases (primary)
- Homebrew tap (obra/homebrew-tap)

### Supported Platforms
- linux/amd64
- linux/arm64
- darwin/amd64 (Intel Macs)
- darwin/arm64 (Apple Silicon)

### Trigger Mechanism
- Automated on git tag push (e.g., `v1.0.1`)

## Architecture

### Components

1. **GoReleaser Configuration** (`.goreleaser.yml`)
   - Defines build matrix for all platforms
   - Configures archive format and contents
   - Generates checksums
   - Manages GitHub Release creation
   - Auto-updates Homebrew formula

2. **GitHub Actions Workflow** (`.github/workflows/release.yml`)
   - Triggers on tag push matching `v*` pattern
   - Sets up Go environment
   - Runs GoReleaser with release flag
   - Uses GITHUB_TOKEN for authentication

3. **Version Embedding** (`version.go`)
   - Ensures version information is compiled into binaries
   - Supports `-ldflags` injection during build

4. **Homebrew Tap Repository** (`obra/homebrew-tap`)
   - Separate repository for Homebrew formulas
   - Auto-managed by GoReleaser
   - Contains Ruby formula for packnplay

### Release Flow

```
Developer pushes tag
  ↓
GitHub Actions detects tag
  ↓
Checkout code & setup Go
  ↓
GoReleaser builds for all platforms
  ↓
Create archives (.tar.gz)
  ↓
Generate SHA256 checksums
  ↓
Create/update GitHub Release
  ↓
Update Homebrew formula in tap repo
  ↓
Users can install via GitHub or Homebrew
```

### Integration Points

- **Existing CI:** Separate workflow from `.github/workflows/ci.yml`
- **Version source:** Uses git tags as source of truth
- **Authentication:** GitHub token with repo and contents permissions
- **Dependencies:** Leverages existing `go.mod` and project structure

## Implementation Tasks

1. Create `.goreleaser.yml` configuration
2. Create `.github/workflows/release.yml` workflow
3. Verify `version.go` supports ldflags injection
4. Create `homebrew-tap` repository
5. Configure GitHub token permissions
6. Test release process with a test tag
7. Document release process in project docs

## Benefits

- **Automated:** Tag push triggers entire release process
- **Multi-platform:** Single command builds for all targets
- **Checksums:** Automatic SHA256 generation for security
- **Homebrew:** Easy installation for macOS/Linux users
- **Changelog:** Auto-generated from commit history
- **Standard tooling:** GoReleaser is industry standard for Go CLIs

## Future Enhancements

- Additional package managers (apt, yum) if needed
- Docker image publishing to GHCR
- Notarization for macOS binaries
- Windows support (if demand exists)
