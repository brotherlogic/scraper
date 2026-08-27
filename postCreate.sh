#!/bin/zsh

export GOPATH=/go
export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin

# Update and install dependencies
sudo apt update && sudo apt install -y \
    protobuf-compiler \
    xdg-utils \
    tmux \
    emacs

go install google.golang.org/protobuf/cmd/protoc-gen-go@latest 
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Account for Ghostty
tic -x ghostty.terminfo

# Install antigravity
curl -fsSL https://antigravity.google/cli/install.sh -o install_antigravity.sh
bash install_antigravity.sh
rm install_antigravity.sh

# Set git identity
git config --global user.email "brotherlogicautomation@gmail.com"
git config --global user.name "Brotherlogic Automation"

TMUX_BLOCK=$(cat << 'EOF'
if [ -z "$TMUX" ] && [ -n "$PS1" ]; then
  cd /workspaces/scraper
  /workspaces/scraper/start-tmux.sh && tmux attach-session -t scraper
fi
EOF
)

grep -q "tmux attach-session" ~/.zshrc || echo "$TMUX_BLOCK" >> ~/.zshrc
grep -q "tmux attach-session" ~/.bashrc || echo "$TMUX_BLOCK" >> ~/.bashrc

# Ensure the session is created
/workspaces/scraper/start-tmux.sh
