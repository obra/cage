# Based on Microsoft's devcontainer base image with AI CLIs installed
# Optimized for daily autobuilds with proper layer caching
FROM mcr.microsoft.com/devcontainers/base:ubuntu

# Layer 1: Base system dependencies (most stable)
RUN apt-get update && \
    apt-get install -y curl wget && \
    # Install Node.js
    curl -fsSL https://deb.nodesource.com/setup_lts.x | bash - && \
    apt-get install -y nodejs && \
    # Install GitHub CLI
    (type -p wget >/dev/null || (apt update && apt install wget -y)) && \
    mkdir -p -m 755 /etc/apt/keyrings && \
    wget -nv -O- https://cli.github.com/packages/githubcli-archive-keyring.gpg | tee /etc/apt/keyrings/githubcli-archive-keyring.gpg > /dev/null && \
    chmod go+r /etc/apt/keyrings/githubcli-archive-keyring.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null && \
    apt update && \
    apt install gh -y && \
    # Cleanup
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Layer 2: Sourcegraph AMP (most stable - enterprise, quarterly releases)
RUN npm install -g @sourcegraph/amp

# Layer 3: Qwen Code (stable - research-based, infrequent releases)
RUN npm install -g @qwen-code/qwen-code

# Layer 4: Google Gemini CLI (moderate - monthly releases)
RUN npm install -g @google/gemini-cli

# Layer 5: GitHub Copilot (moderate - bi-weekly releases)
RUN npm install -g @github/copilot

# Layer 6: OpenAI Codex (frequent - weekly releases)
RUN npm install -g @openai/codex

# Layer 7: Cursor CLI (binary install, separate from npm)
RUN curl -fsSL https://cursor.com/install | bash

# Layer 8: Claude Code (most volatile - multiple releases per week)
RUN npm install -g @anthropic-ai/claude-code

# Disable auto-updaters for AI agents (we control updates via Docker builds)
ENV DISABLE_AUTOUPDATER=true

# Configure npm to disable update checks and audits for performance
RUN npm config set -g update-check false && \
    npm config set -g audit false && \
    npm config set -g fund false

# Ensure all tools are in PATH for all users
ENV PATH="/usr/local/bin:${PATH}"

# Set the default user (matches devcontainer convention)
USER vscode
