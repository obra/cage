# Based on Microsoft's devcontainer base image with AI CLIs installed
FROM mcr.microsoft.com/devcontainers/base:ubuntu

# Install Node.js (required for npm packages)
RUN apt-get update && \
    apt-get install -y curl && \
    curl -fsSL https://deb.nodesource.com/setup_lts.x | bash - && \
    apt-get install -y nodejs && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Install AI CLI tools globally
RUN npm install -g @anthropic-ai/claude-code \
                   @openai/codex \
                   @google/gemini-cli

# Ensure all tools are in PATH for all users
ENV PATH="/usr/local/bin:${PATH}"

# Set the default user (matches devcontainer convention)
USER vscode
