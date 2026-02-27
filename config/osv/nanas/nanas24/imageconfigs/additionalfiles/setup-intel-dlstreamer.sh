#!/bin/bash
#---------------------------------------------
# Script: setup-intel-dlstreamer.sh
# Purpose: Download DLStreamer / OpenVINO models in virtual environment
#---------------------------------------------

set -e

# Define where to store the models and virtual environment
: "${MODELS_PATH:=/data/intel/models}"
: "${VENV_PATH:=/data/intel/venv}"

echo "Using MODELS_PATH=${MODELS_PATH}"
echo "Using VENV_PATH=${VENV_PATH}"

# Check if python3.12-venv is installed, install if not
echo "Checking for python3.12-venv..."
if ! dpkg -l | grep -q python3.12-venv; then
    echo "Installing python3.12-venv..."
    apt update
    apt install -y python3.12-venv
else
    echo "python3.12-venv is already installed"
fi

# Create virtual environment if it doesn't exist or is incomplete
if [ ! -f "$VENV_PATH/bin/activate" ]; then
    echo "Creating Python virtual environment..."
    rm -rf "$VENV_PATH"  # Remove incomplete venv if it exists
    python3 -m venv "$VENV_PATH"
fi

# Activate virtual environment
echo "Activating virtual environment..."
source "$VENV_PATH/bin/activate"

# Upgrade pip in virtual environment
echo "Upgrading pip..."
python -m pip install --upgrade pip

# Install OpenVINO dev with extras
echo "Installing OpenVINO dev packages..."
python -m pip install --upgrade openvino-dev[onnx,tensorflow,pytorch]

# Create models directory
mkdir -p "$MODELS_PATH"

# Download models using OMZ Downloader
echo "Downloading models..."
omz_downloader --name person-vehicle-bike-detection-2004,vehicle-attributes-recognition-barrier-0039,face-detection-adas-0001,emotions-recognition-retail-0003 -o "$MODELS_PATH"

echo "Models downloaded to: $MODELS_PATH"

# List downloaded files
echo "Downloaded files:"
ls -lR "$MODELS_PATH"

# Deactivate virtual environment
deactivate

# Check if podman is available
echo "Checking for podman..."
if ! command -v podman >/dev/null 2>&1; then
    echo "Warning: podman not found, skipping DLStreamer container image pull"
else
    echo "Pulling Intel DLStreamer container image..."
    if podman pull intel/dlstreamer:latest; then
        echo "Successfully pulled intel/dlstreamer:latest"
    else
        echo "Warning: Failed to pull intel/dlstreamer:latest container image"
    fi
fi

echo "Done."