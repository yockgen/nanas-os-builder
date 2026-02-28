#!/bin/bash

set -e

SOURCE_FILE="/etc/apt/sources.list.d/ubuntu.sources"
DEST_FILE="/etc/apt/sources.list.d/ubuntu.sources.disabled"

# Check if source file exists and move it
if [ -f "$SOURCE_FILE" ]; then
    echo "Disabling Ubuntu sources..."
    mv "$SOURCE_FILE" "$DEST_FILE"
    echo "✓ Ubuntu sources disabled"
else
    echo "Ubuntu sources file not found, nothing to disable"
fi