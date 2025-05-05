package azurelinux3

import (
    "go.uber.org/zap"
    "strings"
    "bufio"
	"fmt"
	"time"
	"io"
	"os"
	"net/http"
	"path/filepath"
    "github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/provider"
	"github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/rpmverify"
)

// repoConfig holds .repo file values
type repoConfig struct {
	Section      string // raw section header
	Name         string // human-readable name from name=
	BaseURL      string
	GPGCheck     bool
	RepoGPGCheck bool
	Enabled      bool
	GPGKey       string
}

// AzureLinux3 implements provider.Provider
type AzureLinux3 struct { 
	repo repoConfig
}

func init() {
    provider.Register(&AzureLinux3{})
}

// Name returns the unique name of the provider
func (p *AzureLinux3) Name() string { return "AzureLinux3" }

// Init will initialize the provider, fetching repo configuration
func (p *AzureLinux3) Init() error {
    
    logger := zap.L().Sugar()
    configURL := "https://packages.microsoft.com/azurelinux/3.0/prod/base/x86_64/config.repo"
	resp, err := http.Get(configURL)
	if err != nil {
		logger.Errorf("downloading repo config %s failed: %v", configURL, err)
		return err
	}
	defer resp.Body.Close()

	cfg, err := loadRepoConfig(resp.Body)
	if err != nil {
		logger.Errorf("parsing repo config failed: %v", err)
		return err
	}
	p.repo = cfg
	logger.Infof("Initialized AzureLinux3 provider repo section=%s", p.repo.Section)
    logger.Infof("name=%s", p.repo.Name)
    logger.Infof("baseurl=%s", p.repo.BaseURL)
	return nil
}
func (p *AzureLinux3) Packages() ([]provider.PackageInfo, error) {
    // get sugar logger from zap
	logger := zap.L().Sugar()
    logger.Infof("fetching packages from %s", p.repo.BaseURL)
    var pkgs []provider.PackageInfo

	// directories are under BaseURL/Packages/A, BaseURL/Packages/B, ...
	for c := 'a'; c <= 'z'; c++ {
		sub := fmt.Sprintf("%s/Packages/%c/", p.repo.BaseURL, c)
		err := crawlDirectory(sub, &pkgs)
		if err != nil {
			continue // skip missing or empty dirs
		}
	}
    logger.Infof("found %d packages in AzureLinux3 repo", len(pkgs))
	return pkgs, nil
}
func (p *AzureLinux3) Validate(destDir string) error {
    // get sugar logger from zap
	logger := zap.L().Sugar()

	// read the GPG key from the repo config
	resp, err := http.Get(p.repo.GPGKey)
    if err != nil {
        return fmt.Errorf("fetch GPG key %s: %w", p.repo.GPGKey, err)
    }
    defer resp.Body.Close()

    keyBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return fmt.Errorf("read GPG key body: %w", err)
	}
	logger.Infof("fetched GPG key \n%s", keyBytes)

	// store in a temp file
	tmp, err := os.CreateTemp("", "azurelinux-gpg-*.asc")
    if err != nil {
        return fmt.Errorf("create temp key file: %w", err)
    }
    defer func() {
        tmp.Close()
        os.Remove(tmp.Name())
    }()

    if _, err := tmp.Write(keyBytes); err != nil {
        return fmt.Errorf("write key to temp file: %w", err)
    }
	
	// get all RPMs in the destDir
	rpmPattern := filepath.Join(destDir, "*.rpm")
    rpmPaths, err := filepath.Glob(rpmPattern)
    if err != nil {
        return fmt.Errorf("glob %q: %w", rpmPattern, err)
    }
    if len(rpmPaths) == 0 {
        logger.Warn("no RPMs found to verify")
        return nil
    }

	start := time.Now()
    results := rpmverify.VerifyAll(rpmPaths, tmp.Name(), 4)
    logger.Infof("RPM verification took %s", time.Since(start))

    // Check results
    for _, r := range results {
        if !r.OK {
            return fmt.Errorf("RPM %s failed verification: %v", r.Path, r.Error)
        }
    }
    logger.Info("all RPMs verified successfully")

    return nil
}

// loadRepoConfig parses the repo configuration data
func loadRepoConfig(r io.Reader) (repoConfig, error) {
	s := bufio.NewScanner(r)
	var rc repoConfig
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		// skip comments or empty
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		// section header
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			rc.Section = strings.Trim(line, "[]")
			continue
		}
		// key=value lines
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch key {
		case "name":
			rc.Name = val
		case "baseurl":
			rc.BaseURL = val
		case "gpgcheck":
			rc.GPGCheck = (val == "1")
		case "repo_gpgcheck":
			rc.RepoGPGCheck = (val == "1")
		case "enabled":
			rc.Enabled = (val == "1")
		case "gpgkey":
			rc.GPGKey = val
		}
	}
	if err := s.Err(); err != nil {
		return rc, err
	}
	return rc, nil
}

// crawlDirectory fetches a directory listing and appends RPM entries
func crawlDirectory(url string, pkgs *[]provider.PackageInfo) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	s := bufio.NewScanner(resp.Body)
	for s.Scan() {
		line := s.Text()
		// simplistic HTML href parse
		if idx := strings.Index(line, "href=\""); idx != -1 {
			part := line[idx+6:]
			if end := strings.Index(part, "\""); end != -1 {
				fname := part[:end]
				if strings.HasSuffix(fname, ".rpm") {
					full := url + fname
					*pkgs = append(*pkgs, provider.PackageInfo{Name: strings.TrimSuffix(fname, ".rpm"), URL: full, Checksum: ""})
				}
			}
		}
	}
    
	return s.Err()
}