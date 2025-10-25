# Claude Code Instructions for packnplay

This file contains specific instructions for Claude Code when working on the packnplay project.

## Project Overview

packnplay is a development container tool that provides seamless AI coding agent support with intelligent user detection and Docker integration.

## Release Engineering

When preparing releases, **always follow the systematic process** documented in:

📋 **[Release Engineering Process](./docs/release-engineering.md)**

This ensures:
- Consistent versioning and tagging
- Complete documentation of changes
- Proper testing and verification
- Clear communication to users

## Key Development Practices

### Test-Driven Development
- Write tests first for all new features
- Follow RED → GREEN → REFACTOR cycle
- Ensure comprehensive test coverage

### Git Workflow
- Use descriptive commit messages
- Include "🤖 Generated with Claude Code" footer
- Follow conventional commit format when appropriate

### Code Quality
- Maintain consistency with existing code style
- Ensure all tests pass before commits
- Update documentation with new features

## Architecture Notes

### User Detection System
- Smart caching by Docker image ID
- Direct container interrogation (no username guessing)
- Priority: devcontainer.json → cache → runtime detection → fallback

### Port Mapping
- Docker-compatible `-p/--publish` syntax
- Full format support including IP binding and protocols
- Integration through RunConfig to Docker args

## Release Notes Guidelines

When updating releases:
- Focus on user impact and benefits
- Include practical usage examples
- Document any breaking changes clearly
- Acknowledge contributors and community

## Documentation Standards

- Keep README up-to-date with new features
- Include code examples in documentation
- Test all documented commands
- Link to relevant sections appropriately

## Support and Maintenance

- Address issues systematically
- Prioritize bug fixes over new features
- Maintain backward compatibility when possible
- Communicate changes clearly to users

## Contact and Collaboration

This project follows collaborative development practices with clear communication and systematic processes for quality and reliability.