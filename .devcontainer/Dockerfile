# Based on Microsoft's devcontainer base image with Claude and Codex installed
FROM mcr.microsoft.com/devcontainers/base:ubuntu

# Install Claude Code and OpenAI Codex
RUN apt-get update && \
    apt-get install -y curl wget && \
    # Install Claude Code
    curl -fsSL https://claude.com/install.sh | sh && \
    # Install OpenAI Codex CLI
    curl -fsSL https://openai.com/install-codex.sh | sh && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Ensure claude and codex are in PATH for all users
ENV PATH="/usr/local/bin:${PATH}"

# Set the default user (matches devcontainer convention)
USER vscode
