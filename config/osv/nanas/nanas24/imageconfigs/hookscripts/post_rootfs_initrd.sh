#!/bin/sh

# 1. Safety Check: Ensure TARGET_ROOTFS is defined
if [ -z "$TARGET_ROOTFS" ]; then
    echo "Error: TARGET_ROOTFS environment variable is not set."
    echo "Aborting to prevent overwriting host system files."
    exit 1
fi

echo "Configuring live installer environment..."
echo "TARGET_ROOTFS=$TARGET_ROOTFS"

# Remove password for root user by editing shadow file directly
if [ -f "$TARGET_ROOTFS/etc/shadow" ]; then
    echo "Removing root password from shadow file..."
    sed -i 's/^root:[^:]*:/root::/' "$TARGET_ROOTFS/etc/shadow"
    echo "Root password removed"
else
    echo "Warning: /etc/shadow not found"
fi

# Configure lightdm autologin
echo "Creating lightdm autologin configuration..."
mkdir -p "$TARGET_ROOTFS/etc/lightdm/lightdm.conf.d"

cat <<EOF > "$TARGET_ROOTFS/etc/lightdm/lightdm.conf.d/50-autologin.conf"
[Seat:*]
autologin-user=root
autologin-user-timeout=0
autologin-session=xfce
user-session=xfce
greeter-session=lightdm-gtk-greeter
pam-service=lightdm-autologin
EOF
echo "Created lightdm autologin config"

# Ensure pam autologin service exists
echo "Creating PAM autologin service..."
mkdir -p "$TARGET_ROOTFS/etc/pam.d"
cat <<EOF > "$TARGET_ROOTFS/etc/pam.d/lightdm-autologin"
#%PAM-1.0
auth     required pam_permit.so
auth     optional pam_gnome_keyring.so
account  include  common-account
session  include  common-session
session  optional pam_gnome_keyring.so auto_start
EOF
echo "Created PAM autologin service"

# Configure XFCE as default session for root
echo "Configuring XFCE desktop session..."
mkdir -p "$TARGET_ROOTFS/root/.config"
mkdir -p "$TARGET_ROOTFS/root/Desktop"
cat <<EOF > "$TARGET_ROOTFS/root/.xsession"
#!/bin/sh
exec startxfce4
EOF
chmod +x "$TARGET_ROOTFS/root/.xsession"
echo "Created .xsession for XFCE and Desktop directory"

# Configure PATH to include sbin directories
echo "Configuring system PATH..."
cat <<'EOF' > "$TARGET_ROOTFS/etc/profile.d/sbin-path.sh"
# Add sbin directories to PATH
export PATH="/usr/local/sbin:/usr/sbin:/sbin:$PATH"
EOF
chmod +x "$TARGET_ROOTFS/etc/profile.d/sbin-path.sh"
echo "Created PATH configuration"

# Auto-run attendedinstaller on login
echo "Setting up attendedinstaller auto-launch..."
cat <<'EOF' > "$TARGET_ROOTFS/root/.bash_profile"
# Add sbin to PATH
export PATH="/usr/local/sbin:/usr/sbin:/sbin:$PATH"

# Auto-launch installer on first login
if [ -z "$INSTALLER_LAUNCHED" ] && [ -n "$DISPLAY" ]; then
    export INSTALLER_LAUNCHED=1
    /root/attendedinstaller
fi
EOF
echo "Created .bash_profile with attendedinstaller launch"

# Also create XFCE autostart for attendedinstaller
echo "Creating XFCE autostart for installer..."
mkdir -p "$TARGET_ROOTFS/etc/xdg/autostart"
cat <<'EOF' > "$TARGET_ROOTFS/etc/xdg/autostart/attendedinstaller.desktop"
[Desktop Entry]
Type=Application
Name=Nanas OS Installer
Comment=Launch system installer
Exec=/root/attendedinstaller
Icon=system-software-install
Terminal=false
NoDisplay=true
X-GNOME-Autostart-enabled=true
X-GNOME-Autostart-Delay=2
EOF
chmod +x "$TARGET_ROOTFS/root/attendedinstaller" 2>/dev/null || true
echo "Created XFCE autostart entry"

# Create systemd override to ensure lightdm starts
echo "Enabling lightdm service..."
mkdir -p "$TARGET_ROOTFS/etc/systemd/system"
if [ ! -e "$TARGET_ROOTFS/etc/systemd/system/display-manager.service" ]; then
    ln -sf /lib/systemd/system/lightdm.service "$TARGET_ROOTFS/etc/systemd/system/display-manager.service" 2>/dev/null || true
fi

# Create /cdrom directory (will be mounted by attendedinstaller script)
echo "Creating /cdrom directory for ISO mount..."
mkdir -p "$TARGET_ROOTFS/cdrom"

# Set installer permissions
echo "Setting installer executable permissions..."
if [ -f "$TARGET_ROOTFS/usr/local/bin/installer-gui.py" ]; then
    chmod +x "$TARGET_ROOTFS/usr/local/bin/installer-gui.py"
    echo "Set executable: installer-gui.py"
else
    echo "WARNING: installer-gui.py not found"
fi

if [ -f "$TARGET_ROOTFS/usr/local/bin/installer-launcher.sh" ]; then
    chmod +x "$TARGET_ROOTFS/usr/local/bin/installer-launcher.sh"
    echo "Set executable: installer-launcher.sh"
else
    echo "WARNING: installer-launcher.sh not found"
fi

echo "Live installer environment configuration complete"

