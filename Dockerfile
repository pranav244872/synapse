FROM golang

# Create user with UID 1000
RUN useradd -m -u 1000 devuser

USER root

# Install dev tools
RUN apt-get update && \
    apt-get install -y \
        git curl ripgrep fd-find unzip \
        && apt-get clean && rm -rf /var/lib/apt/lists/*

# Symlink fd
RUN ln -s $(which fdfind) /usr/local/bin/fd || true

# Install Neovim (optional dev environment)
RUN curl -L -o nvim-linux64.tar.gz https://github.com/neovim/neovim/releases/download/v0.11.2/nvim-linux-x86_64.tar.gz && \
    tar xzf nvim-linux64.tar.gz && \
    mv nvim-linux-x86_64 /opt/nvim && \
    ln -s /opt/nvim/bin/nvim /usr/local/bin/nvim && \
    rm nvim-linux64.tar.gz

# Switch to dev user
USER devuser

# Set working directory
WORKDIR /home/devuser/synapse

# Default command
CMD ["bash"]
