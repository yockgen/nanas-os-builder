@echo off
SETLOCAL EnableDelayedExpansion

:: ==========================================
:: SETTINGS & VARIABLES
:: ==========================================
SET "REPO_URL=https://github.com/open-edge-platform/os-image-composer.git"
SET "REPO_DIR=os-image-composer"
SET "GO_VERSION=go1.25.5"
SET "DISTRO=Ubuntu-24.04"
:: ==========================================

echo [1/5] Checking WSL and %DISTRO% status...

:: 1. Check if WSL feature is enabled
dism.exe /online /get-features /format:table | findstr /I "Microsoft-Windows-Subsystem-Linux" | findstr /I "Enabled" >nul
if %errorlevel% neq 0 (
    echo [!] WSL feature is not enabled. Enabling now...
    dism.exe /online /enable-feature /featurename:Microsoft-Windows-Subsystem-Linux /all /norestart
    dism.exe /online /enable-feature /featurename:VirtualMachinePlatform /all /norestart
    echo [!] WSL features enabled. Please RESTART your computer and run this script again.
    pause
    exit
)

:: 2. Check if Ubuntu-24.04 is already installed
wsl --list --quiet | findstr /C:"%DISTRO%" >nul
if %errorlevel% neq 0 (
    echo [!] %DISTRO% not found. Installing now...
    echo [!] AFTER the Ubuntu window finishes its setup, CLOSE it and RUN THIS SCRIPT AGAIN.
    wsl --install -d %DISTRO%
    pause
    exit
)

echo [2/5] Ensuring Go %GO_VERSION% and Git are installed...
:: 3. Run internal setup as root
:: We use 'bash -l' to ensure we have a login shell environment
wsl -d %DISTRO% -u root bash -lc "apt-get update && apt-get install -y wget tar git; if ! go version | grep -q '%GO_VERSION%'; then echo '[!] Installing Go...'; wget -q https://go.dev/dl/%GO_VERSION%.linux-amd64.tar.gz; rm -rf /usr/local/go && tar -C /usr/local -xzf %GO_VERSION%.linux-amd64.tar.gz; rm %GO_VERSION%.linux-amd64.tar.gz; fi"

:: 4. Ensure Go is in PATH for the user (run as standard user)
wsl -d %DISTRO% bash -lc "if ! grep -q '/usr/local/go/bin' ~/.bashrc; then echo 'export PATH=\$PATH:/usr/local/go/bin' >> ~/.bashrc; fi"

echo [3/5] Checking repository: %REPO_DIR%...
:: 5. Clone the repository using the variable (using $HOME to be safe)
wsl -d %DISTRO% bash -lc "if [ ! -d \"\$HOME/%REPO_DIR%\" ]; then echo '[!] Cloning %REPO_URL%...'; cd \$HOME && git clone %REPO_URL%; else echo '[OK] %REPO_DIR% already exists.'; fi"

echo [4/5] Environment verification...
wsl -d %DISTRO% bash -lc "echo 'Go Version: ' \$(/usr/local/go/bin/go version); echo 'Git Version: ' \$(git --version)"

echo [5/5] Success! Launching %DISTRO%...
timeout /t 2 >nul
wsl -d %DISTRO%
