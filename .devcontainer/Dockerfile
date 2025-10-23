# Based on Microsoft's devcontainer base image with Claude installed
FROM mcr.microsoft.com/devcontainers/base:ubuntu

# Install Claude Code
RUN apt-get update && \
    apt-get install -y curl && \
    curl -fsSL https://claude.com/install.sh | sh && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Ensure claude is in PATH for all users
ENV PATH="/usr/local/bin:${PATH}"

# Set the default user (matches devcontainer convention)
USER vscode
