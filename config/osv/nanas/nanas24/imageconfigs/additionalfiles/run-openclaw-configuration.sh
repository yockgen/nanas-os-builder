#!/bin/bash

# Redirect all output to a custom log
exec > >(tee -a /var/log/openclaw-config-trace.log) 2>&1

echo "--- OpenClaw Configuration started at $(date) ---"

# Set up environment
export HOME=/root
export PATH="$HOME/.npm-global/bin:$PATH"

# 1. Install OpenClaw gateway
echo "Installing OpenClaw gateway..."
openclaw gateway install
if [ $? -eq 0 ]; then
    echo "✓ OpenClaw gateway installed successfully"
else
    echo "✗ OpenClaw gateway installation failed"
    exit 1
fi

# 2. Configure OpenClaw
echo "Configuring OpenClaw..."
openclaw configure
if [ $? -eq 0 ]; then
    echo "✓ OpenClaw configured successfully"
else
    echo "✗ OpenClaw configuration failed"
    exit 1
fi

# 3. Start OpenClaw dashboard
echo "Starting OpenClaw dashboard..."
openclaw dashboard
if [ $? -eq 0 ]; then
    echo "✓ OpenClaw dashboard started successfully"
else
    echo "✗ OpenClaw dashboard failed to start"
    exit 1
fi

echo "--- OpenClaw Configuration completed at $(date) ---"