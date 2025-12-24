package hook

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/open-edge-platform/os-image-composer/internal/config"
	"github.com/open-edge-platform/os-image-composer/internal/utils/logger"
	"github.com/open-edge-platform/os-image-composer/internal/utils/shell"
)

func HookPostRootfs(installRoot string, template *config.ImageTemplate) error {
    log := logger.Logger()
	log.Infof("Applying post-rootfs hooks...")

	//TODO: this is just a temporary branding hook for Madani OS, will be replaced with a more generic solution later
    log.Debug("Applying Madani OS branding Placeholder...")
    
    // Ensure the etc directory exists
    etcDir := filepath.Join(installRoot, "etc")
    if err := os.MkdirAll(etcDir, 0755); err != nil {
        log.Errorf("Failed to create etc directory: %v", err)
        return fmt.Errorf("failed to create etc directory: %w", err)
    }
    
    // 1. Create os-release with Madani OS branding
    osReleaseContent := `NAME="Madani OS"
VERSION="0.1 (Alpha)"
ID=madanios
ID_LIKE=madanios
PRETTY_NAME="Madani OS 0.1"
VERSION_ID="0.1"
HOME_URL="https://madanios.com"
BUG_REPORT_URL="https://madanios.com/support"`
    
    osReleaseFile := fmt.Sprintf("/tmp/os-release-%s", uuid.New().String()[:8])
    if err := os.WriteFile(osReleaseFile, []byte(osReleaseContent), 0644); err != nil {
        log.Errorf("Failed to create os-release temp file: %v", err)
        return fmt.Errorf("failed to create os-release temp file: %w", err)
    }
    defer os.Remove(osReleaseFile)
    
    copyCmd := fmt.Sprintf("cp %s %s", osReleaseFile, filepath.Join(installRoot, "etc/os-release"))
    if _, err := shell.ExecCmd(copyCmd, true, shell.HostPath, nil); err != nil {
        log.Errorf("Failed to copy os-release: %v", err)
        return fmt.Errorf("failed to copy os-release: %w", err)
    }
    
    // 2. Create lsb-release with Madani OS branding
    lsbReleaseContent := `DISTRIB_ID=madanios
DISTRIB_RELEASE=1.0
DISTRIB_CODENAME=noble
DISTRIB_DESCRIPTION="Madani OS 0.1 experimental"`
    
    lsbReleaseFile := fmt.Sprintf("/tmp/lsb-release-%s", uuid.New().String()[:8])
    if err := os.WriteFile(lsbReleaseFile, []byte(lsbReleaseContent), 0644); err != nil {
        log.Errorf("Failed to create lsb-release temp file: %v", err)
        return fmt.Errorf("failed to create lsb-release temp file: %w", err)
    }
    defer os.Remove(lsbReleaseFile)
    
    copyCmd = fmt.Sprintf("cp %s %s", lsbReleaseFile, filepath.Join(installRoot, "etc/lsb-release"))
    if _, err := shell.ExecCmd(copyCmd, true, shell.HostPath, nil); err != nil {
        log.Errorf("Failed to copy lsb-release: %v", err)
        return fmt.Errorf("failed to copy lsb-release: %w", err)
    }
    
    // 3. Create issue files (login welcome messages)
    issueContent := "Welcome to Madani OS 0.1 experimental\n"
    
    issueFile := fmt.Sprintf("/tmp/issue-%s", uuid.New().String()[:8])
    if err := os.WriteFile(issueFile, []byte(issueContent), 0644); err != nil {
        log.Errorf("Failed to create issue temp file: %v", err)
        return fmt.Errorf("failed to create issue temp file: %w", err)
    }
    defer os.Remove(issueFile)
    
    // Copy to /etc/issue
    copyCmd = fmt.Sprintf("cp %s %s", issueFile, filepath.Join(installRoot, "etc/issue"))
    if _, err := shell.ExecCmd(copyCmd, true, shell.HostPath, nil); err != nil {
        log.Errorf("Failed to copy issue: %v", err)
        return fmt.Errorf("failed to copy issue: %w", err)
    }
    
    // Copy to /etc/issue.net
    copyCmd = fmt.Sprintf("cp %s %s", issueFile, filepath.Join(installRoot, "etc/issue.net"))
    if _, err := shell.ExecCmd(copyCmd, true, shell.HostPath, nil); err != nil {
        log.Errorf("Failed to copy issue.net: %v", err)
        return fmt.Errorf("failed to copy issue.net: %w", err)
    }
    
    // Set proper permissions on all created files
    files := []string{
        filepath.Join(installRoot, "etc/os-release"),
        filepath.Join(installRoot, "etc/lsb-release"),
        filepath.Join(installRoot, "etc/issue"),
        filepath.Join(installRoot, "etc/issue.net"),
    }
    
    for _, file := range files {
        chmodCmd := fmt.Sprintf("chmod 644 %s", file)
        if _, err := shell.ExecCmd(chmodCmd, true, shell.HostPath, nil); err != nil {
            log.Errorf("Failed to set permissions on %s: %v", file, err)
            return fmt.Errorf("failed to set permissions on %s: %w", file, err)
        }
    }
    
    log.Infof("Successfully applied Madani OS branding")
    return nil
}