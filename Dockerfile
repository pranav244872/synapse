FROM golang

# Create user (optional)
RUN adduser --disabled-password --gecos '' devuser && \
    chown -R devuser:devuser /home/devuser

USER devuser
WORKDIR /home/devuser/learnGo

# Optional: Install your favorite editor/tools (nvim example)
USER root
RUN apt update && apt install -y git curl ripgrep fd-find unzip && apt clean

# Install Neovim (prebuilt)
RUN curl -L -o nvim-linux64.tar.gz https://github.com/neovim/neovim/releases/download/v0.11.2/nvim-linux-x86_64.tar.gz && \
    tar xzf nvim-linux64.tar.gz && \
    mv nvim-linux-x86_64 /opt/nvim && \
    ln -s /opt/nvim/bin/nvim /usr/local/bin/nvim && \
    rm nvim-linux64.tar.gz

USER devuser
CMD [ "bash" ]
