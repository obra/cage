# Binary Distribution Setup Instructions

This document contains step-by-step instructions to complete the binary distribution setup for packnplay outside the container.

## Current Status

✅ All code changes have been committed to the `binary-distribution` branch locally.

The following files have been created/modified:
- `.goreleaser.yml` - GoReleaser configuration for multi-platform builds
- `.github/workflows/release.yml` - GitHub Actions workflow for releases
- `cmd/version.go` - New version command
- `docs/plans/2025-10-25-binary-distribution-design.md` - Design documentation
- `docs/release-process.md` - Release process guide
- `version.go` - Updated to support ldflags injection

## Steps to Complete Setup

### 1. Push the Branch

```bash
cd /path/to/packnplay  # Navigate to your packnplay repository

# Verify you're on the binary-distribution branch
git branch --show-current

# Push the branch to GitHub
git push -u origin binary-distribution
```

### 2. Create the Homebrew Tap Repository

```bash
# Authenticate with GitHub CLI if not already done
gh auth login

# Create the homebrew-tap repository
gh repo create obra/homebrew-tap --public --description "Homebrew tap for packnplay"
```

Expected output:
```
✓ Created repository obra/homebrew-tap on GitHub
```

### 3. Create a Pull Request

```bash
# Create PR from the binary-distribution branch to main
gh pr create --title "feat: add automated binary distribution with GoReleaser" --body "$(cat <<'EOF'
## Summary

Implements automated binary distribution for packnplay using GoReleaser.

### Features

- **Multi-platform builds**: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64
- **GitHub Releases**: Automatically created on version tags
- **Homebrew tap**: Auto-updates `obra/homebrew-tap` with each release
- **Version command**: New `packnplay version` command with build metadata
- **Complete documentation**: Release process and design docs included

### How it works

1. Push a version tag: `git tag v1.0.1 && git push origin v1.0.1`
2. GitHub Actions triggers automatically
3. GoReleaser builds binaries for all platforms
4. Creates GitHub Release with all artifacts
5. Updates Homebrew formula in tap repository

### Testing

After merge, test with:
```bash
# Create a test tag
git tag v1.0.1
git push origin v1.0.1

# Monitor the workflow
gh workflow view release

# Once complete, test installation
brew install obra/tap/packnplay
packnplay version
\`\`\`

### Documentation

- Design: `docs/plans/2025-10-25-binary-distribution-design.md`
- Release process: `docs/release-process.md`

🤖 Generated with Claude Code
EOF
)"
```

### 4. Review and Merge

1. Review the PR on GitHub: https://github.com/obra/packnplay/pulls
2. Verify the changes look correct
3. Merge the PR to main

### 5. Test the Release Process (Optional but Recommended)

After merging, test with a real release:

```bash
# Ensure you're on main and up to date
git checkout main
git pull origin main

# Create a test tag (or your actual next version)
git tag -a v1.0.1 -m "Release v1.0.1"
git push origin v1.0.1

# Monitor the release workflow
gh workflow view release
# or visit: https://github.com/obra/packnplay/actions

# Once complete, verify the release
gh release view v1.0.1

# Test Homebrew installation
brew update
brew install obra/tap/packnplay
packnplay version
```

## Troubleshooting

### If `gh auth login` fails

Try authenticating with a personal access token:

1. Create a token at: https://github.com/settings/tokens/new
2. Required scopes: `repo`, `workflow`, `write:packages`
3. Run: `gh auth login --with-token < token.txt`

### If the homebrew-tap creation fails

You can create it manually:

1. Go to: https://github.com/new
2. Repository name: `homebrew-tap`
3. Owner: `obra`
4. Make it public
5. Click "Create repository"

### If the release workflow fails

1. Check the Actions log: https://github.com/obra/packnplay/actions
2. Common issues:
   - Missing `obra/homebrew-tap` repository (create it first)
   - Insufficient permissions on GitHub token (check workflow permissions)
   - GoReleaser configuration errors (review `.goreleaser.yml`)

### If you need to test locally before pushing

```bash
# Install GoReleaser
brew install goreleaser/tap/goreleaser

# Test the configuration (dry run)
goreleaser release --snapshot --clean --skip=publish

# This will build binaries locally in ./dist/ without publishing
```

## What Happens Next

Once everything is set up:

1. **Creating releases** is as simple as:
   ```bash
   git tag -a v1.2.3 -m "Release v1.2.3"
   git push origin v1.2.3
   ```

2. **GitHub Actions** automatically:
   - Builds binaries for all platforms
   - Creates archives and checksums
   - Publishes GitHub Release
   - Updates Homebrew formula

3. **Users can install** via:
   - `brew install obra/tap/packnplay`
   - Or download binaries directly from GitHub Releases

## Summary of Changes

**New files:**
- `.goreleaser.yml` - Build configuration
- `.github/workflows/release.yml` - CI workflow
- `cmd/version.go` - Version command
- `docs/plans/2025-10-25-binary-distribution-design.md` - Design doc
- `docs/release-process.md` - Release guide
- `SETUP-BINARY-DISTRIBUTION.md` - This file

**Modified files:**
- `version.go` - Now uses variables for ldflags injection

**Repository to create:**
- `obra/homebrew-tap` - Homebrew tap repository

## Questions?

Refer to:
- `docs/release-process.md` for ongoing release procedures
- `docs/plans/2025-10-25-binary-distribution-design.md` for design rationale
- GoReleaser docs: https://goreleaser.com/
