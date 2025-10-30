# Docker Autobuilds

This document explains packnplay's automated Docker image building system, designed to keep AI agent CLI tools up-to-date with their frequent releases.

## Overview

AI agent companies release updates frequently, with Claude Code being updated multiple times per week. Our autobuild system ensures users always have access to the latest versions while minimizing build time through optimal Docker layer caching.

## Build Strategy

### Layer Optimization

The Dockerfile is structured with AI agent installs ordered from most stable to most volatile:

1. **Base system + GitHub CLI** (most stable - system packages)
2. **@sourcegraph/amp** (enterprise, quarterly releases)
3. **@qwen-code/qwen-code** (research-based, infrequent releases)
4. **@google/gemini-cli** (Google enterprise cadence, monthly)
5. **@github/copilot** (established product, bi-weekly releases)
6. **@openai/codex** (weekly releases)
7. **Cursor CLI** (binary download, separate from npm)
8. **@anthropic-ai/claude-code** (most volatile - multiple releases per week)

### Why This Order?

- **Layer caching**: Docker reuses unchanged layers. When Claude Code updates, only the final layer rebuilds
- **Build efficiency**: Most days, only 1-2 layers need rebuilding instead of all 8
- **Bandwidth savings**: Users and CI only download changed layers

## Build Workflows

### Daily Builds (`.github/workflows/daily-docker-build.yml`)

**Schedule**: Daily at 2 AM UTC

**Triggers**:
- Scheduled daily execution
- Manual dispatch
- Changes to Dockerfile or workflow files

**Process**:
1. Build multi-platform image (amd64, arm64)
2. Tag with multiple formats:
   - `latest` (for main branch)
   - `daily-YYYY-MM-DD`
   - `YYYY-MM-DD` (short form)
3. Test all AI CLI installations
4. Push to GitHub Container Registry
5. Clean up old daily builds (keep last 7)

**Registry**: `ghcr.io/obra/packnplay/devcontainer`

### PR Builds (`.github/workflows/docker-pr-build.yml`)

**Triggers**: Pull requests that modify Docker-related files

**Process**:
1. Build image for PR testing
2. Tag as `pr-{number}`
3. Test basic functionality
4. Comment on PR with test image details
5. Automatic cleanup after 7 days

## Image Usage

### In DevContainers

The daily builds are automatically used by the `.devcontainer/devcontainer.json`:

```json
{
  "image": "ghcr.io/obra/packnplay/devcontainer:latest"
}
```

### Manual Usage

```bash
# Use latest daily build
docker run -it ghcr.io/obra/packnplay/devcontainer:latest

# Use specific date
docker run -it ghcr.io/obra/packnplay/devcontainer:2024-01-15

# Test a PR build
docker run -it ghcr.io/obra/packnplay/devcontainer:pr-123
```

## Monitoring and Maintenance

### Build Status

Monitor builds via:
- GitHub Actions tab in repository
- Package registry: https://github.com/obra/packnplay/pkgs/container/devcontainer

### Failed Builds

Common failure causes:
1. **AI CLI package unavailable**: Usually temporary, will succeed on next run
2. **Network issues**: Retry the workflow
3. **Breaking changes**: May require Dockerfile updates

### Manual Intervention

To force a rebuild:
1. Go to Actions → Daily Docker Image Build
2. Click "Run workflow"
3. Select main branch and run

## Performance Benefits

### Cache Hit Rates

Expected cache hit rates by layer:
- Base system (Layer 1): ~95% (only updates with Node.js LTS changes)
- Sourcegraph AMP (Layer 2): ~90% (quarterly releases)
- Qwen Code (Layer 3): ~85% (infrequent updates)
- Gemini CLI (Layer 4): ~75% (monthly updates)
- GitHub Copilot (Layer 5): ~65% (bi-weekly updates)
- OpenAI Codex (Layer 6): ~50% (weekly updates)
- Cursor CLI (Layer 7): ~70% (irregular binary updates)
- Claude Code (Layer 8): ~30% (multiple updates per week)

### Build Time Savings

- **Without optimization**: ~10-15 minutes (all layers rebuild)
- **With optimization**: ~2-5 minutes (typically only 1-2 layers rebuild)
- **Network savings**: ~80% reduction in download size for incremental updates

## Security Considerations

### Registry Access

- Images stored in GitHub Container Registry (GHCR)
- Public read access for ease of use
- Write access limited to repository maintainers

### Supply Chain

- All packages installed from official registries (npm, apt)
- No unofficial or third-party sources
- Automated dependency updates through daily builds

### Image Scanning

GitHub automatically scans images for vulnerabilities. Review security advisories in the Packages section.

## Troubleshooting

### Common Issues

**Build fails on npm install**:
```bash
# Check if package exists
npm view @anthropic-ai/claude-code

# Test locally
docker build -f .devcontainer/Dockerfile .
```

**Image not updating**:
- Check if workflow ran successfully
- Verify triggers are properly configured
- Manual trigger may be needed

**Missing AI CLI after update**:
- Check test results in workflow logs
- Verify installation commands in Dockerfile
- Test image manually before use

### Getting Help

1. Check workflow logs in GitHub Actions
2. Review package registry for image status
3. Test problematic layers individually
4. Open issue with build logs if needed

## Future Improvements

### Planned Enhancements

1. **Notification system**: Slack/email alerts for build failures
2. **Health checks**: Periodic testing of published images
3. **Version pinning**: Option to pin specific AI CLI versions
4. **Multi-variant builds**: Separate images for different AI tool combinations
5. **Build metrics**: Track cache hit rates and build performance

### Metrics Collection

Consider adding:
- Build duration tracking
- Layer cache hit rate monitoring
- Download metrics for optimization feedback