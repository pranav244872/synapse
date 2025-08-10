FROM golang

# Create user (optional)
RUN adduser --disabled-password --gecos '' devuser && \
    chown -R devuser:devuser /home/devuser

USER devuser
WORKDIR /home/devuser/learnGo

# Optional: Install your favorite editor/tools (nvim example)
USER root
RUN apt update && apt install -y git curl ripgrep fd-find unzip && apt clean

USER devuser
CMD [ "bash" ]
