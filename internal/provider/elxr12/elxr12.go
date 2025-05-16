package elxr12

import (
	"github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/config"
	"github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/debutils"
	"github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/provider"
	"go.uber.org/zap"
)

// ref: https://packages.microsoft.com/azurelinux/3.0/prod/base/
// example: https://deb.debian.org/debian/pool/main/0/0ad/0ad_0.0.26-3_amd64.deb
const (
	baseURL    = "https://deb.debian.org/debian/dists/bookworm/main/"
	configName = "Packages.gz"
	repodata   = ""
)

// repoConfig holds .repo file values
type repoConfig struct {
	Section      string // raw section header
	Name         string // human-readable name from name=
	CfgURL       string
	PkgUrl       string
	GPGCheck     bool
	RepoGPGCheck bool
	Enabled      bool
	GPGKey       string
}

// eLxr12 implements provider.Provider
type eLxr12 struct {
	repoURL string
	repoCfg repoConfig
	gzHref  string
	spec    *config.BuildSpec
}

func init() {
	provider.Register(&eLxr12{})
}

// Name returns the unique name of the provider
func (p *eLxr12) Name() string {
	logger := zap.L().Sugar()
	logger.Infof("Name() called - Placeholder: This function will return the provider's unique name.")
	return "eLxr12"
}

// Init will initialize the provider, fetching repo configuration
func (p *eLxr12) Init(spec *config.BuildSpec) error {

	logger := zap.L().Sugar()

	//todo: need to correct of how to get the arch once finalized
	if spec.Arch == "x86_64" {
		spec.Arch = "binary-amd64"
	}
	p.repoURL = baseURL + spec.Arch + "/" + configName

	cfg, err := loadRepoConfig(p.repoURL)
	if err != nil {
		logger.Errorf("parsing repo config failed: %v", err)
		return err
	}
	p.repoCfg = cfg
	p.gzHref = cfg.CfgURL

	logger.Infof("initialized eLxr provider repo section=%s", cfg.Section)
	logger.Infof("name=%s", cfg.Name)
	logger.Infof("config url=%s", cfg.CfgURL)
	logger.Infof("package download url=%s", cfg.PkgUrl)
	logger.Infof("primary.xml.gz=%s", p.gzHref)
	return nil

}

// Packages returns the list of packages
func (p *eLxr12) Packages() ([]provider.PackageInfo, error) {

	logger := zap.L().Sugar()
	logger.Infof("Packages() started")

	debutils.ParsePrimary(p.repoURL, p.gzHref)

	// zap.L().Sync() // flush logs if needed
	// panic("Stopped by yockgen.")

	return nil, nil
}

// Validate verifies the downloaded files
func (p *eLxr12) Validate(destDir string) error {
	logger := zap.L().Sugar()
	logger.Infof("Validate() called with destDir=%s - Placeholder: This function will be implemented by the respective owner.", destDir)
	return nil
}

// Resolve resolves dependencies
func (p *eLxr12) Resolve(req []provider.PackageInfo, all []provider.PackageInfo) ([]provider.PackageInfo, error) {
	logger := zap.L().Sugar()
	logger.Infof("Resolve() called with destDir=%s - Placeholder: This function will be implemented by the respective owner.")
	return nil, nil
}

// MatchRequested matches requested packages
func (p *eLxr12) MatchRequested(requests []string, all []provider.PackageInfo) ([]provider.PackageInfo, error) {
	logger := zap.L().Sugar()
	logger.Infof("MatchRequested() called - Placeholder: This function will be implemented by the respective owner.")
	return nil, nil
}

func loadRepoConfig(repoUrl string) (repoConfig, error) {
	logger := zap.L().Sugar()

	var rc repoConfig

	//wget https://deb.debian.org/debian/pool/main/0/0ad/0ad_0.0.26-3_amd64.deb
	rc.CfgURL = repoUrl
	rc.PkgUrl = "https://deb.debian.org/debian/pool/"
	rc.Name = "Debian Bookworm Main"
	rc.GPGCheck = true
	rc.RepoGPGCheck = true
	rc.Enabled = true
	rc.GPGKey = "https://ftp-master.debian.org/keys/release-12.asc"
	rc.Section = "main"

	logger.Infof("repo config: %+v", rc)

	return rc, nil
}
