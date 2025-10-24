# Based on Microsoft's devcontainer base image with AI CLIs installed
FROM mcr.microsoft.com/devcontainers/base:ubuntu

# Install Node.js and GitHub CLI
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

# Install AI CLI tools globally
RUN npm install -g @anthropic-ai/claude-code \
                   @openai/codex \
                   @google/gemini-cli

# Ensure all tools are in PATH for all users
ENV PATH="/usr/local/bin:${PATH}"

# Set the default user (matches devcontainer convention)
USER vscode
