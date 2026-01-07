@echo off
SETLOCAL EnableDelayedExpansion

:: ==========================================
:: SETTINGS & VARIABLES
:: ==========================================
SET "REPO_URL=https://github.com/yockgen/madani-os-builder.git"
SET "REPO_DIR=madani-os-builder"
SET "GO_VERSION=go1.25.5"
:: ==========================================

echo [1/5] Checking WSL and Ubuntu 24.04 status...

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
wsl --list --quiet | findstr /C:"Ubuntu-24.04" >nul
if %errorlevel% neq 0 (
    echo [!] Ubuntu 24.04 not found. Installing now...
    wsl --install -d Ubuntu-24.04
    echo [!] Ubuntu installation started. Complete the user setup in the new window, then run this script again.
    pause
    exit
)

echo [2/5] Ensuring Go %GO_VERSION% is installed...
:: 3. Check and Install Go
wsl -d Ubuntu-24.04 -u root bash -c "if go version | grep -q '%GO_VERSION%'; then echo '[OK] Go %GO_VERSION% already installed.'; else echo '[!] Installing Go %GO_VERSION%...'; apt-get update && apt-get install -y wget tar git; wget -q https://go.dev/dl/%GO_VERSION%.linux-amd64.tar.gz; rm -rf /usr/local/go && tar -C /usr/local -xzf %GO_VERSION%.linux-amd64.tar.gz; rm %GO_VERSION%.linux-amd64.tar.gz; fi"

:: Ensure Go is in PATH
wsl -d Ubuntu-24.04 bash -c "if ! grep -q '/usr/local/go/bin' ~/.bashrc; then echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc; fi"

echo [3/5] Checking repository: %REPO_DIR%...
:: 4. Clone the repository using the variable
wsl -d Ubuntu-24.04 bash -c "if [ ! -d '~/%REPO_DIR%' ]; then echo '[!] Cloning %REPO_URL%...'; cd ~ && git clone %REPO_URL%; else echo '[OK] %REPO_DIR% already exists.'; fi"

echo [4/5] Environment verification...
wsl -d Ubuntu-24.04 bash -c "go version; git --version"

echo [5/5] Success! Launching Ubuntu 24.04 shell...
timeout /t 2 >nul
wsl -d Ubuntu-24.04