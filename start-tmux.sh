#!/bin/bash

# Ensure the 'prod' session exists
if ! tmux has-session -t scraper 2>/dev/null; then
  # Create a new session named 'prod', detached
  cd /workspaces/scraper
  tmux new-session -d -s scraper
fi
