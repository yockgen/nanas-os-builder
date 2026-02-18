#!/bin/bash
set -e

# 1. Cleanup and Directory Creation
echo "Cleaning up old data..."
sudo rm -rf /home/openclaw/.openclaw
sudo mkdir -p /home/openclaw/.openclaw/{agents,canvas,completions,credentials,cron,identity,logs,workspace,devices}
sudo chown -R openclaw:openclaw /home/openclaw/.openclaw

# 2. Run Setup FIRST (to generate openclaw.json and .env)
echo "Running openclaw setup..."
sudo -u openclaw /home/openclaw/run-openclaw-podman.sh setup

# 3. FIX PERMISSIONS for Podman (The missing link)
echo "Aligning internal container permissions..."
sudo -u openclaw podman unshare chown -R 1000:1000 /home/openclaw/.openclaw
sudo -u openclaw podman unshare chmod -R 775 /home/openclaw/.openclaw

# 4. Remove and Start Container
echo "Restarting openclaw container..."
sudo -u openclaw podman rm -f openclaw || true
sudo -u openclaw podman run -d \
  --name openclaw \
  --network host \
  -v /home/openclaw/.openclaw:/home/node/.openclaw:Z \
  -e OPENCLAW_GATEWAY_HOST=0.0.0.0 \
  localhost/openclaw:local

# 5. Extract and display master token
echo "Waiting for config to settle..."
sleep 2
TOKEN=$(sudo cat /home/openclaw/.openclaw/openclaw.json | grep -oP '"token":\s*"\K[^"]+')
echo "------------------------------------------------"
echo "YOUR MASTER TOKEN: $TOKEN"
echo "URL: http://$(hostname -I | awk '{print $1}'):18789/#token=$TOKEN"
echo "------------------------------------------------"