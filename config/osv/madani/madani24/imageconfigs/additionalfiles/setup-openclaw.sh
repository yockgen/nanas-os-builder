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

# 3. Ensure Git is installed
if ! command -v git >/dev/null 2>&1; then
    echo "Installing Git..."
    apt-get install -y git
else
    echo "Git is already installed"
fi

# 4. Create data directory and navigate to it
echo "Creating /data directory..."
mkdir -p /data
cd /data/

# 5. Clone OpenClaw repository
echo "Cloning OpenClaw repository..."
if [ -d "openclaw" ]; then
    echo "OpenClaw directory already exists, removing it first..."
    rm -rf openclaw
fi

git clone https://github.com/openclaw/openclaw
if [ $? -eq 0 ]; then
    echo "✓ OpenClaw repository cloned successfully"
else
    echo "✗ Failed to clone OpenClaw repository"
    exit 1
fi

# 6. Navigate to openclaw directory
cd openclaw

# 7. Make the setup script executable
echo "Making setup-podman.sh executable..."
if [ -f "setup-podman.sh" ]; then
    chmod +x setup-podman.sh
    echo "✓ setup-podman.sh made executable"
else
    echo "✗ setup-podman.sh not found in repository"
    exit 1
fi

# 8. Run the setup script (already running as root, so no need for sudo)
echo "Running OpenClaw setup script..."
./setup-podman.sh
if [ $? -eq 0 ]; then
    echo "✓ OpenClaw setup completed successfully"
else
    echo "✗ OpenClaw setup failed"
    exit 1
fi

echo "--- OpenClaw Setup completed at $(date) ---"