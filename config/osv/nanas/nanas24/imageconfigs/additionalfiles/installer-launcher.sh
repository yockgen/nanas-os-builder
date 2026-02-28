#!/bin/bash
# Wrapper script to launch installer with proper logging and mount waiting

LOG_FILE="/tmp/installer.log"

# Redirect all output to log file
exec > "$LOG_FILE" 2>&1

echo "=== Nanas OS Installer Launcher ==="
echo "Started at: $(date)"

# Wait for /cdrom to be mounted (up to 60 seconds)
echo "Waiting for /cdrom to be mounted..."
for i in {1..60}; do
    if mountpoint -q /cdrom || [ -d "/cdrom/images" ]; then
        echo "/cdrom is ready after $i seconds"
        break
    fi
    if [ $i -eq 60 ]; then
        echo "WARNING: /cdrom not mounted after 60 seconds!"
        echo "Checking mount status:"
        mount | grep cdrom || echo "No cdrom mount found"
        echo "Continuing anyway..."
    fi
    sleep 1
done

# Wait for X display to be ready
echo "Waiting for X display..."
for i in {1..30}; do
    if xset q &>/dev/null; then
        echo "X display is ready after $i seconds"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "WARNING: X display not ready after 30 seconds!"
    fi
    sleep 1
done

# Launch installer
echo "Launching installer GUI..."
echo "DISPLAY=$DISPLAY"
echo "Python version: $(python3 --version)"

# Run installer
python3 /usr/local/bin/installer-gui.py

echo "Installer exited at: $(date)"
