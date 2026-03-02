#!/bin/bash

# Redirect all output to a custom log as well so you don't just rely on cloud-init logs
exec > >(tee -a /var/log/openclaw-setup-trace.log) 2>&1

echo "--- OpenClaw Setup started at $(date) ---"

# 0. Setup Environment Variables
# These prevent the "panic: $HOME is not defined" error
export HOME=/root
export USER=root
export XDG_CONFIG_HOME=/root/.config
# Ensure the path includes standard binary locations
export PATH=$PATH:/usr/local/bin:/usr/bin:/bin

# 1. Wait for Apt Lock (Crucial for cloud-init)
echo "Waiting for apt locks to release..."
while fuser /var/lib/dpkg/lock-frontend >/dev/null 2>&1; do
    sleep 5
done

# 2. Update package lists
echo "Updating package lists..."
apt update

# 3. Ensure curl is installed
if ! command -v curl >/dev/null 2>&1; then
    echo "Installing curl..."
    apt-get install -y curl
else
    echo "curl is already installed"
fi

# 4. Install OpenClaw natively using the official installer
echo "Installing OpenClaw natively..."
curl -fsSL https://openclaw.ai/install.sh | bash -s -- --no-onboard
if [ $? -eq 0 ]; then
    echo "✓ OpenClaw installed successfully"
else
    echo "✗ OpenClaw installation failed"
    exit 1
fi

echo "--- OpenClaw Setup completed at $(date) ---"