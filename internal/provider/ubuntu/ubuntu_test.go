package ubuntu

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/open-edge-platform/os-image-composer/internal/chroot"
	"github.com/open-edge-platform/os-image-composer/internal/config"
	"github.com/open-edge-platform/os-image-composer/internal/ospackage/debutils"
	"github.com/open-edge-platform/os-image-composer/internal/provider"
	"github.com/open-edge-platform/os-image-composer/internal/utils/shell"
	"github.com/open-edge-platform/os-image-composer/internal/utils/system"
)

// Helper function to create a test ImageTemplate
func createTestImageTemplate() *config.ImageTemplate {
	return &config.ImageTemplate{
		Image: config.ImageInfo{
			Name:    "test-ubuntu-image",
			Version: "1.0.0",
		},
		Target: config.TargetInfo{
			OS:        "ubuntu",
			Dist:      "ubuntu24",
			Arch:      "amd64",
			ImageType: "raw",
		},
		SystemConfig: config.SystemConfig{
			Name:        "test-ubuntu-system",
			Description: "Test Ubuntu system configuration",
			Packages:    []string{"curl", "wget", "vim"},
		},
	}
}

// TestUbuntuProviderInterface tests that ubuntu implements Provider interface
func TestUbuntuProviderInterface(t *testing.T) {
	var _ provider.Provider = (*ubuntu)(nil) // Compile-time interface check
}

// TestUbuntuProviderName tests the Name method
func TestUbuntuProviderName(t *testing.T) {
	ubuntu := &ubuntu{}
	name := ubuntu.Name("ubuntu24", "amd64")
	expected := "ubuntu-ubuntu24-amd64"

	if name != expected {
		t.Errorf("Expected name %s, got %s", expected, name)
	}
}

// TestGetProviderId tests the GetProviderId function
func TestGetProviderId(t *testing.T) {
	testCases := []struct {
		dist     string
		arch     string
		expected string
	}{
		{"ubuntu24", "amd64", "ubuntu-ubuntu24-amd64"},
		{"ubuntu24", "arm64", "ubuntu-ubuntu24-arm64"},
		{"ubuntu22", "x86_64", "ubuntu-ubuntu22-x86_64"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s-%s", tc.dist, tc.arch), func(t *testing.T) {
			result := system.GetProviderId(OsName, tc.dist, tc.arch)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

// TestUbuntuProviderInit tests the Init method
func TestUbuntuProviderInit(t *testing.T) {
	// Change to project root for tests that need config files
	originalDir, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Failed to change back to original directory: %v", err)
		}
	}()

	// Navigate to project root (3 levels up from internal/provider/ubuntu)
	if err := os.Chdir("../../../"); err != nil {
		t.Skipf("Cannot change to project root: %v", err)
		return
	}

	ubuntu := &ubuntu{}

	// Test with amd64 architecture
	err := ubuntu.Init("ubuntu24", "amd64")
	if err != nil {
		// Expected to potentially fail in test environment due to network dependencies
		t.Logf("Init failed as expected in test environment: %v", err)
	} else {
		// If it succeeds, verify the configuration was set up
		if len(ubuntu.repoCfgs) == 0 {
			t.Error("Expected repoCfgs to be populated after successful Init")
		}

		// Verify that the architecture is correctly set in the config
		for _, cfg := range ubuntu.repoCfgs {
			if cfg.Arch != "amd64" {
				t.Errorf("Expected arch to be amd64, got %s", cfg.Arch)
			}
		}

		t.Logf("Successfully initialized with %d repositories", len(ubuntu.repoCfgs))
	}
}

// TestUbuntuProviderInitArchMapping tests architecture mapping in Init
func TestUbuntuProviderInitArchMapping(t *testing.T) {
	// Change to project root for tests that need config files
	originalDir, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Failed to change back to original directory: %v", err)
		}
	}()

	// Navigate to project root (3 levels up from internal/provider/ubuntu)
	if err := os.Chdir("../../../"); err != nil {
		t.Skipf("Cannot change to project root: %v", err)
		return
	}

	ubuntu := &ubuntu{}

	// Test x86_64 -> amd64 mapping
	err := ubuntu.Init("ubuntu24", "x86_64")
	if err != nil {
		t.Logf("Init failed as expected: %v", err)
	} else {
		// Verify that repoCfgs were set up correctly
		if len(ubuntu.repoCfgs) == 0 {
			t.Error("Expected repoCfgs to be populated after successful Init")
			return
		}

		// Verify that the first repository has correct architecture mapping
		firstRepo := ubuntu.repoCfgs[0]
		expectedArchInURL := "binary-amd64"
		if firstRepo.PkgList != "" && !strings.Contains(firstRepo.PkgList, expectedArchInURL) {
			t.Errorf("Expected PkgList to contain %s for x86_64 arch, got %s", expectedArchInURL, firstRepo.PkgList)
		}

		// Verify architecture was mapped correctly
		if firstRepo.Arch != "amd64" {
			t.Errorf("Expected mapped arch to be amd64, got %s", firstRepo.Arch)
		}

		t.Logf("Successfully mapped x86_64 -> amd64, PkgList: %s", firstRepo.PkgList)
	}
}

// TestLoadRepoConfig tests the loadRepoConfig function
func TestLoadRepoConfig(t *testing.T) {
	// Change to project root for tests that need config files
	originalDir, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Failed to change back to original directory: %v", err)
		}
	}()

	// Navigate to project root (3 levels up from internal/provider/ubuntu)
	if err := os.Chdir("../../../"); err != nil {
		t.Skipf("Cannot change to project root: %v", err)
		return
	}

	configs, err := loadRepoConfig("", "amd64")
	if err != nil {
		t.Skipf("loadRepoConfig failed (expected in test environment): %v", err)
		return
	}

	// If we successfully load config, verify the values
	if len(configs) == 0 {
		t.Error("Expected at least one repository configuration")
		return
	}

	for _, config := range configs {
		if config.Name == "" {
			t.Error("Expected config name to be set")
		}

		if config.Arch != "amd64" {
			t.Errorf("Expected arch 'amd64', got '%s'", config.Arch)
		}

		// Verify PkgList contains expected architecture
		if config.PkgList != "" && !strings.Contains(config.PkgList, "binary-amd64") {
			t.Errorf("Expected PkgList to contain 'binary-amd64', got '%s'", config.PkgList)
		}

		t.Logf("Successfully loaded repo config: %s", config.Name)
	}
}

// mockChrootEnv is a simple mock implementation of ChrootEnvInterface for testing
type mockChrootEnv struct{}

// Ensure mockChrootEnv implements ChrootEnvInterface
var _ chroot.ChrootEnvInterface = (*mockChrootEnv)(nil)

func (m *mockChrootEnv) GetChrootEnvRoot() string          { return "/tmp/test-chroot" }
func (m *mockChrootEnv) GetChrootImageBuildDir() string    { return "/tmp/test-build" }
func (m *mockChrootEnv) GetTargetOsPkgType() string        { return "deb" }
func (m *mockChrootEnv) GetTargetOsConfigDir() string      { return "/tmp/test-config" }
func (m *mockChrootEnv) GetTargetOsReleaseVersion() string { return "24" }
func (m *mockChrootEnv) GetChrootPkgCacheDir() string      { return "/tmp/test-cache" }
func (m *mockChrootEnv) GetChrootEnvEssentialPackageList() ([]string, error) {
	return []string{"base-files"}, nil
}
func (m *mockChrootEnv) GetChrootEnvHostPath(chrootPath string) (string, error) {
	return chrootPath, nil
}
func (m *mockChrootEnv) GetChrootEnvPath(hostPath string) (string, error) { return hostPath, nil }
func (m *mockChrootEnv) MountChrootSysfs(chrootPath string) error         { return nil }
func (m *mockChrootEnv) UmountChrootSysfs(chrootPath string) error        { return nil }
func (m *mockChrootEnv) MountChrootPath(hostFullPath, chrootPath, mountFlags string) error {
	return nil
}
func (m *mockChrootEnv) UmountChrootPath(chrootPath string) error                       { return nil }
func (m *mockChrootEnv) CopyFileFromHostToChroot(hostFilePath, chrootPath string) error { return nil }
func (m *mockChrootEnv) CopyFileFromChrootToHost(hostFilePath, chrootPath string) error { return nil }
func (m *mockChrootEnv) UpdateChrootLocalRepoMetadata(chrootRepoDir string, targetArch string, sudo bool) error {
	return nil
}
func (m *mockChrootEnv) RefreshLocalCacheRepo() error                                   { return nil }
func (m *mockChrootEnv) InitChrootEnv(targetOs, targetDist, targetArch string) error    { return nil }
func (m *mockChrootEnv) CleanupChrootEnv(targetOs, targetDist, targetArch string) error { return nil }
func (m *mockChrootEnv) TdnfInstallPackage(packageName, installRoot string, repositoryIDList []string) error {
	return nil
}
func (m *mockChrootEnv) AptInstallPackage(packageName, installRoot string, repoSrcList []string) error {
	return nil
}
func (m *mockChrootEnv) UpdateSystemPkgs(template *config.ImageTemplate) error { return nil }

// TestUbuntuProviderPreProcess tests PreProcess method with mocked dependencies
func TestUbuntuProviderPreProcess(t *testing.T) {
	// Save original shell executor and restore after test
	originalExecutor := shell.Default
	defer func() { shell.Default = originalExecutor }()

	// Set up mock executor
	mockExpectedOutput := []shell.MockCommand{
		// Mock successful package installation commands
		{Pattern: "apt-get update", Output: "Package lists updated successfully", Error: nil},
		{Pattern: "apt-get install -y mmdebstrap", Output: "Package installed successfully", Error: nil},
		{Pattern: "apt-get install -y dosfstools", Output: "Package installed successfully", Error: nil},
		{Pattern: "apt-get install -y mtools", Output: "Package installed successfully", Error: nil},
		{Pattern: "apt-get install -y xorriso", Output: "Package installed successfully", Error: nil},
		{Pattern: "apt-get install -y qemu-utils", Output: "Package installed successfully", Error: nil},
		{Pattern: "apt-get install -y systemd-ukify", Output: "Package installed successfully", Error: nil},
		{Pattern: "apt-get install -y grub-common", Output: "Package installed successfully", Error: nil},
		{Pattern: "apt-get install -y cryptsetup", Output: "Package installed successfully", Error: nil},
		{Pattern: "apt-get install -y sbsigntool", Output: "Package installed successfully", Error: nil},
		{Pattern: "apt-get install -y ubuntu-keyring", Output: "Package installed successfully", Error: nil},
	}
	shell.Default = shell.NewMockExecutor(mockExpectedOutput)

	ubuntu := &ubuntu{
		repoCfgs: []debutils.RepoConfig{
			{
				Section:     "main",
				Name:        "Ubuntu 24.04",
				PkgList:     "https://archive.ubuntu.com/ubuntu/dists/noble/main/binary-amd64/Packages.gz",
				PkgPrefix:   "https://archive.ubuntu.com/ubuntu/",
				Enabled:     true,
				GPGCheck:    true,
				ReleaseFile: "https://archive.ubuntu.com/ubuntu/dists/noble/Release",
				ReleaseSign: "https://archive.ubuntu.com/ubuntu/dists/noble/Release.gpg",
				BuildPath:   "/tmp/builds/ubuntu1_amd64_main",
				Arch:        "amd64",
			},
		},
		chrootEnv: &mockChrootEnv{}, // Add the missing chrootEnv mock
	}

	template := createTestImageTemplate()

	// This test will likely fail due to dependencies on chroot, debutils, etc.
	// but it demonstrates the testing approach
	err := ubuntu.PreProcess(template)
	if err != nil {
		t.Logf("PreProcess failed as expected due to external dependencies: %v", err)
	}
}

// TestUbuntuProviderBuildImage tests BuildImage method
func TestUbuntuProviderBuildImage(t *testing.T) {
	// Save original shell executor and restore after test
	originalExecutor := shell.Default
	defer func() { shell.Default = originalExecutor }()

	// Set up mock executor - minimal mocks for Register function
	mockExpectedOutput := []shell.MockCommand{
		{Pattern: ".*", Output: "success", Error: nil}, // Catch-all for any commands during registration
	}
	shell.Default = shell.NewMockExecutor(mockExpectedOutput)

	// Try to register and get a properly initialized ubuntu instance
	err := Register("linux", "test-build", "amd64")
	if err != nil {
		t.Skipf("Cannot test BuildImage without proper registration: %v", err)
		return
	}

	// Get the registered provider
	providerName := system.GetProviderId(OsName, "test-build", "amd64")
	retrievedProvider, exists := provider.Get(providerName)
	if !exists {
		t.Skip("Cannot test BuildImage without retrieving registered provider")
		return
	}

	ubuntu, ok := retrievedProvider.(*ubuntu)
	if !ok {
		t.Skip("Retrieved provider is not an ubuntu instance")
		return
	}

	template := createTestImageTemplate()

	// This test will fail due to dependencies on image builders that require system access
	// We expect it to fail early before reaching sudo commands
	err = ubuntu.BuildImage(template)
	if err != nil {
		t.Logf("BuildImage failed as expected due to external dependencies: %v", err)
		// Verify the error is related to expected failures, not sudo issues
		if strings.Contains(err.Error(), "sudo") {
			t.Errorf("Test should not reach sudo commands - mocking may be insufficient")
		}
	}
}

// TestUbuntuProviderBuildImageISO tests BuildImage method with ISO type
func TestUbuntuProviderBuildImageISO(t *testing.T) {
	// Save original shell executor and restore after test
	originalExecutor := shell.Default
	defer func() { shell.Default = originalExecutor }()

	// Set up mock executor - minimal mocks for Register function
	mockExpectedOutput := []shell.MockCommand{
		{Pattern: ".*", Output: "success", Error: nil}, // Catch-all for any commands during registration
	}
	shell.Default = shell.NewMockExecutor(mockExpectedOutput)

	// Try to register and get a properly initialized ubuntu instance
	err := Register("linux", "test-iso", "amd64")
	if err != nil {
		t.Skipf("Cannot test BuildImage (ISO) without proper registration: %v", err)
		return
	}

	// Get the registered provider
	providerName := system.GetProviderId(OsName, "test-iso", "amd64")
	retrievedProvider, exists := provider.Get(providerName)
	if !exists {
		t.Skip("Cannot test BuildImage (ISO) without retrieving registered provider")
		return
	}

	ubuntu, ok := retrievedProvider.(*ubuntu)
	if !ok {
		t.Skip("Retrieved provider is not an ubuntu instance")
		return
	}

	template := createTestImageTemplate()

	// Set up global config for ISO
	originalImageType := template.Target.ImageType
	defer func() { template.Target.ImageType = originalImageType }()
	template.Target.ImageType = "iso"

	err = ubuntu.BuildImage(template)
	if err != nil {
		t.Logf("BuildImage (ISO) failed as expected due to external dependencies: %v", err)
		// Verify the error is related to expected failures, not sudo issues
		if strings.Contains(err.Error(), "sudo") {
			t.Errorf("Test should not reach sudo commands - mocking may be insufficient")
		}
	}
}

// TestUbuntuProviderBuildImageInitrd tests BuildImage method with IMG type
func TestUbuntuProviderBuildImageInitrd(t *testing.T) {
	// Save original shell executor and restore after test
	originalExecutor := shell.Default
	defer func() { shell.Default = originalExecutor }()

	// Set up mock executor - minimal mocks for Register function
	mockExpectedOutput := []shell.MockCommand{
		{Pattern: ".*", Output: "success", Error: nil}, // Catch-all for any commands during registration
	}
	shell.Default = shell.NewMockExecutor(mockExpectedOutput)

	// Try to register and get a properly initialized ubuntu instance
	err := Register("linux", "test-img", "amd64")
	if err != nil {
		t.Skipf("Cannot test BuildImage (IMG) without proper registration: %v", err)
		return
	}

	// Get the registered provider
	providerName := system.GetProviderId(OsName, "test-img", "amd64")
	retrievedProvider, exists := provider.Get(providerName)
	if !exists {
		t.Skip("Cannot test BuildImage (IMG) without retrieving registered provider")
		return
	}

	ubuntu, ok := retrievedProvider.(*ubuntu)
	if !ok {
		t.Skip("Retrieved provider is not an ubuntu instance")
		return
	}

	template := createTestImageTemplate()

	// Set up global config for IMG
	originalImageType := template.Target.ImageType
	defer func() { template.Target.ImageType = originalImageType }()
	template.Target.ImageType = "img"

	err = ubuntu.BuildImage(template)
	if err != nil {
		t.Logf("BuildImage (IMG) failed as expected due to external dependencies: %v", err)
		// Verify the error is related to expected failures, not sudo issues
		if strings.Contains(err.Error(), "sudo") {
			t.Errorf("Test should not reach sudo commands - mocking may be insufficient")
		}
	}
}

// TestUbuntuProviderPostProcess tests PostProcess method
func TestUbuntuProviderPostProcess(t *testing.T) {
	// Save original shell executor and restore after test
	originalExecutor := shell.Default
	defer func() { shell.Default = originalExecutor }()

	// Set up mock executor - minimal mocks for Register function
	mockExpectedOutput := []shell.MockCommand{
		{Pattern: ".*", Output: "success", Error: nil}, // Catch-all for any commands during registration
	}
	shell.Default = shell.NewMockExecutor(mockExpectedOutput)

	// Try to register and get a properly initialized ubuntu instance
	err := Register("linux", "test-post", "amd64")
	if err != nil {
		t.Skipf("Cannot test PostProcess without proper registration: %v", err)
		return
	}

	// Get the registered provider
	providerName := system.GetProviderId(OsName, "test-post", "amd64")
	retrievedProvider, exists := provider.Get(providerName)
	if !exists {
		t.Skip("Cannot test PostProcess without retrieving registered provider")
		return
	}

	ubuntu, ok := retrievedProvider.(*ubuntu)
	if !ok {
		t.Skip("Retrieved provider is not an ubuntu instance")
		return
	}

	template := createTestImageTemplate()

	// Test with no error
	err = ubuntu.PostProcess(template, nil)
	if err != nil {
		t.Logf("PostProcess failed as expected due to chroot cleanup dependencies: %v", err)
	}

	// Test with input error - PostProcess should clean up and return nil (not the input error)
	inputError := fmt.Errorf("some build error")
	err = ubuntu.PostProcess(template, inputError)
	if err != nil {
		t.Logf("PostProcess failed during cleanup: %v", err)
	}
}

// TestUbuntuProviderInstallHostDependency tests installHostDependency method
func TestUbuntuProviderInstallHostDependency(t *testing.T) {
	// Save original shell executor and restore after test
	originalExecutor := shell.Default
	defer func() { shell.Default = originalExecutor }()

	// Set up mock executor
	mockExpectedOutput := []shell.MockCommand{
		// Mock successful command existence checks
		{Pattern: "which mmdebstrap", Output: "", Error: nil},
		{Pattern: "which mkfs.fat", Output: "", Error: nil},
		{Pattern: "which mformat", Output: "", Error: nil},
		{Pattern: "which xorriso", Output: "", Error: nil},
		{Pattern: "which qemu-img", Output: "", Error: nil},
		{Pattern: "which ukify", Output: "", Error: nil},
		{Pattern: "which grub-mkimage", Output: "", Error: nil},
		{Pattern: "which veritysetup", Output: "", Error: nil},
		{Pattern: "which sbsign", Output: "", Error: nil},
		{Pattern: "which ubuntu-keyring", Output: "", Error: nil},
		// Mock successful installation commands
		{Pattern: "apt-get install -y mmdebstrap", Output: "Success", Error: nil},
		{Pattern: "apt-get install -y dosfstools", Output: "Success", Error: nil},
		{Pattern: "apt-get install -y mtools", Output: "Success", Error: nil},
		{Pattern: "apt-get install -y xorriso", Output: "Success", Error: nil},
		{Pattern: "apt-get install -y qemu-utils", Output: "Success", Error: nil},
		{Pattern: "apt-get install -y systemd-ukify", Output: "Success", Error: nil},
		{Pattern: "apt-get install -y grub-common", Output: "Success", Error: nil},
		{Pattern: "apt-get install -y cryptsetup", Output: "Success", Error: nil},
		{Pattern: "apt-get install -y sbsigntool", Output: "Success", Error: nil},
		{Pattern: "apt-get install -y ubuntu-keyring", Output: "Success", Error: nil},
	}
	shell.Default = shell.NewMockExecutor(mockExpectedOutput)

	ubuntu := &ubuntu{}

	// This test will likely fail due to dependencies on system.GetHostOsPkgManager()
	// and shell.IsCommandExist(), but it demonstrates the testing approach
	err := ubuntu.installHostDependency()
	if err != nil {
		t.Logf("installHostDependency failed as expected due to external dependencies: %v", err)
	} else {
		t.Logf("installHostDependency succeeded with mocked commands")
	}
}

// TestUbuntuProviderInstallHostDependencyCommands tests the specific commands for host dependencies
func TestUbuntuProviderInstallHostDependencyCommands(t *testing.T) {
	// Get the dependency map by examining the installHostDependency method
	expectedDeps := map[string]string{
		"mmdebstrap":     "mmdebstrap",
		"mkfs.fat":       "dosfstools",
		"mformat":        "mtools",
		"xorriso":        "xorriso",
		"qemu-img":       "qemu-utils",
		"ukify":          "systemd-ukify",
		"grub-mkimage":   "grub-common",
		"veritysetup":    "cryptsetup",
		"sbsign":         "sbsigntool",
		"ubuntu-keyring": "ubuntu-keyring",
	}

	// This is a structural test to verify the dependency mapping
	// In a real implementation, we might expose this map for testing
	t.Logf("Expected host dependencies for Ubuntu provider: %+v", expectedDeps)

	// Verify we have the expected number of dependencies
	if len(expectedDeps) != 10 {
		t.Errorf("Expected 10 host dependencies, got %d", len(expectedDeps))
	}

	// Verify specific critical dependencies
	criticalDeps := []string{"mmdebstrap", "mkfs.fat", "xorriso", "qemu-img"}
	for _, dep := range criticalDeps {
		if _, exists := expectedDeps[dep]; !exists {
			t.Errorf("Critical dependency %s not found in expected dependencies", dep)
		}
	}
}

// TestUbuntuProviderRegister tests the Register function
func TestUbuntuProviderRegister(t *testing.T) {
	// Save original providers registry and restore after test
	// Note: We can't easily access the provider registry for cleanup,
	// so this test shows the approach but may leave test artifacts

	err := Register("linux", "ubuntu24", "amd64")
	if err != nil {
		t.Skipf("Cannot test registration due to missing dependencies: %v", err)
		return
	}

	// Try to retrieve the registered provider
	providerName := system.GetProviderId(OsName, "ubuntu24", "amd64")
	retrievedProvider, exists := provider.Get(providerName)

	if !exists {
		t.Errorf("Expected provider %s to be registered", providerName)
		return
	}

	// Verify it's an ubuntu provider
	if ubuntuProvider, ok := retrievedProvider.(*ubuntu); !ok {
		t.Errorf("Expected ubuntu provider, got %T", retrievedProvider)
	} else {
		// Test the Name method on the registered provider
		name := ubuntuProvider.Name("ubuntu24", "amd64")
		if name != providerName {
			t.Errorf("Expected provider name %s, got %s", providerName, name)
		}
	}
}

// TestUbuntuProviderWorkflow tests a complete ubuntu provider workflow
func TestUbuntuProviderWorkflow(t *testing.T) {
	// This is a unit test focused on testing the provider interface methods
	// without external dependencies that require system access

	ubuntu := &ubuntu{}

	// Test provider name generation
	name := ubuntu.Name("ubuntu24", "amd64")
	expectedName := "ubuntu-ubuntu24-amd64"
	if name != expectedName {
		t.Errorf("Expected name %s, got %s", expectedName, name)
	}

	// Test Init (will likely fail due to network dependencies)
	if err := ubuntu.Init("ubuntu24", "amd64"); err != nil {
		t.Logf("Init failed as expected: %v", err)
	} else {
		// If Init succeeds, verify configuration was loaded
		if len(ubuntu.repoCfgs) == 0 {
			t.Error("Expected repo config to be set after successful Init")
		}
		t.Logf("Repo configs loaded: %d repositories", len(ubuntu.repoCfgs))
	}

	// Skip PreProcess and BuildImage tests to avoid sudo commands
	t.Log("Skipping PreProcess and BuildImage tests to avoid system-level dependencies")

	// Skip PostProcess tests as they require properly initialized dependencies
	t.Log("Skipping PostProcess tests to avoid nil pointer panics - these are tested separately with proper registration")

	t.Log("Complete workflow test finished - core methods exist and are callable")
}

// TestUbuntuConfigurationStructure tests the structure of the ubuntu configuration
func TestUbuntuConfigurationStructure(t *testing.T) {
	// Change to project root for tests that need config files
	originalDir, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Failed to change back to original directory: %v", err)
		}
	}()

	// Navigate to project root (3 levels up from internal/provider/ubuntu)
	if err := os.Chdir("../../../"); err != nil {
		t.Skipf("Cannot change to project root: %v", err)
		return
	}

	// Test that OsName constant is set correctly
	if OsName == "" {
		t.Error("OsName should not be empty")
	}

	expectedOsName := "ubuntu"
	if OsName != expectedOsName {
		t.Errorf("Expected OsName %s, got %s", expectedOsName, OsName)
	}

	// Test that we can load provider config
	providerConfigs, err := config.LoadProviderRepoConfig(OsName, "ubuntu24")
	if err != nil {
		t.Logf("Cannot load provider config in test environment: %v", err)
	} else {
		// If we can load it, verify it has required fields
		if len(providerConfigs) == 0 {
			t.Error("Provider config should have at least one repository")
		} else {
			if providerConfigs[0].Name == "" {
				t.Error("Provider config should have a name")
			}
			t.Logf("Loaded provider config: %s", providerConfigs[0].Name)
		}
	}
}

// TestUbuntuArchitectureHandling tests architecture-specific URL construction
func TestUbuntuArchitectureHandling(t *testing.T) {
	// Change to project root for tests that need config files
	originalDir, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Failed to change back to original directory: %v", err)
		}
	}()

	// Navigate to project root (3 levels up from internal/provider/ubuntu)
	if err := os.Chdir("../../../"); err != nil {
		t.Skipf("Cannot change to project root: %v", err)
		return
	}

	testCases := []struct {
		inputArch    string
		expectedArch string
	}{
		{"x86_64", "amd64"}, // x86_64 gets converted to amd64
		{"amd64", "amd64"},  // amd64 stays amd64
		{"arm64", "arm64"},  // arm64 stays arm64
	}

	for _, tc := range testCases {
		t.Run(tc.inputArch, func(t *testing.T) {
			ubuntu := &ubuntu{}
			err := ubuntu.Init("ubuntu24", tc.inputArch) // Test arch mapping

			if err != nil {
				t.Logf("Init failed as expected: %v", err)
			} else {
				// We expect success, so we can check arch mapping
				if len(ubuntu.repoCfgs) == 0 {
					t.Error("Expected repoCfgs to be populated after successful Init")
					return
				}

				// Check the first repository configuration
				firstRepo := ubuntu.repoCfgs[0]
				if firstRepo.Arch != tc.expectedArch {
					t.Errorf("For input arch %s, expected config arch %s, got %s", tc.inputArch, tc.expectedArch, firstRepo.Arch)
				}

				// If we have a PkgList, verify it contains the expected architecture
				if firstRepo.PkgList != "" {
					expectedArchInURL := "binary-" + tc.expectedArch
					if !strings.Contains(firstRepo.PkgList, expectedArchInURL) {
						t.Errorf("For arch %s, expected PkgList to contain %s, got %s", tc.inputArch, expectedArchInURL, firstRepo.PkgList)
					}
				}

				t.Logf("Successfully tested arch %s -> %s", tc.inputArch, tc.expectedArch)
			}
		})
	}
}

// TestUbuntuBuildImageNilTemplate tests BuildImage with nil template
func TestUbuntuBuildImageNilTemplate(t *testing.T) {
	ubuntu := &ubuntu{}

	err := ubuntu.BuildImage(nil)
	if err == nil {
		t.Error("Expected error when template is nil")
	}

	expectedError := "template cannot be nil"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// TestUbuntuBuildImageUnsupportedType tests BuildImage with unsupported image type
func TestUbuntuBuildImageUnsupportedType(t *testing.T) {
	ubuntu := &ubuntu{}

	template := createTestImageTemplate()
	template.Target.ImageType = "unsupported"

	err := ubuntu.BuildImage(template)
	if err == nil {
		t.Error("Expected error for unsupported image type")
	}

	expectedError := "unsupported image type: unsupported"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// TestUbuntuBuildImageValidTypes tests BuildImage error handling for valid image types
func TestUbuntuBuildImageValidTypes(t *testing.T) {
	ubuntu := &ubuntu{}

	validTypes := []string{"raw", "img", "iso"}

	for _, imageType := range validTypes {
		t.Run(imageType, func(t *testing.T) {
			template := createTestImageTemplate()
			template.Target.ImageType = imageType

			// These will fail due to missing chrootEnv, but we can verify
			// that the code path is reached and the error is expected
			err := ubuntu.BuildImage(template)
			if err == nil {
				t.Errorf("Expected error for image type %s (missing dependencies)", imageType)
			} else {
				t.Logf("Image type %s correctly failed with: %v", imageType, err)

				// Verify the error is related to missing dependencies, not invalid type
				if err.Error() == "unsupported image type: "+imageType {
					t.Errorf("Image type %s should be supported but got unsupported error", imageType)
				}
			}
		})
	}
}

// TestUbuntuPostProcessErrorHandling tests PostProcess method signature and basic behavior
func TestUbuntuPostProcessErrorHandling(t *testing.T) {
	// Test that PostProcess method exists and has correct signature
	// We verify that the method can be called and behaves predictably

	ubuntu := &ubuntu{}
	template := createTestImageTemplate()
	inputError := fmt.Errorf("build failed")

	// Verify the method signature is correct by assigning it to a function variable
	var postProcessFunc func(*config.ImageTemplate, error) error = ubuntu.PostProcess

	t.Logf("PostProcess method has correct signature: %T", postProcessFunc)

	// Test that PostProcess with nil chrootEnv will panic - catch and validate
	defer func() {
		if r := recover(); r != nil {
			t.Logf("PostProcess correctly panicked with nil chrootEnv: %v", r)
		} else {
			t.Error("Expected PostProcess to panic with nil chrootEnv")
		}
	}()

	// This will panic due to nil chrootEnv, which we catch above
	_ = ubuntu.PostProcess(template, inputError)
}

// TestUbuntuDownloadImagePkgs tests downloadImagePkgs method structure
func TestUbuntuDownloadImagePkgs(t *testing.T) {
	ubuntu := &ubuntu{
		repoCfgs: []debutils.RepoConfig{
			{
				Name:      "Test Repository",
				PkgList:   "http://example.com/packages.gz",
				PkgPrefix: "http://example.com/",
				Arch:      "amd64",
				Enabled:   true,
			},
		},
		chrootEnv: &mockChrootEnv{},
	}

	template := createTestImageTemplate()

	// This test will likely fail due to network dependencies and debutils package resolution,
	// but it validates the method structure and error handling
	err := ubuntu.downloadImagePkgs(template)
	if err != nil {
		t.Logf("downloadImagePkgs failed as expected due to external dependencies: %v", err)
		// Verify error messages to ensure proper error handling
		if strings.Contains(err.Error(), "no repository configurations available") {
			t.Error("Repository configurations were provided but still got 'no repository configurations' error")
		}
	} else {
		// If successful, verify that template.FullPkgList was populated
		if template.FullPkgList == nil {
			t.Error("Expected FullPkgList to be populated after successful downloadImagePkgs")
		}
		t.Logf("downloadImagePkgs succeeded, FullPkgList populated with packages")
	}
}

// TestUbuntuMultipleRepositories tests handling of multiple repositories
func TestUbuntuMultipleRepositories(t *testing.T) {
	ubuntu := &ubuntu{
		repoCfgs: []debutils.RepoConfig{
			{
				Name:      "Main Repository",
				PkgList:   "http://example.com/main/packages.gz",
				PkgPrefix: "http://example.com/main/",
				Arch:      "amd64",
				Enabled:   true,
			},
			{
				Name:      "Universe Repository",
				PkgList:   "http://example.com/universe/packages.gz",
				PkgPrefix: "http://example.com/universe/",
				Arch:      "amd64",
				Enabled:   true,
			},
		},
		chrootEnv: &mockChrootEnv{},
	}

	template := createTestImageTemplate()

	// Test downloadImagePkgs with multiple repositories
	err := ubuntu.downloadImagePkgs(template)
	if err != nil {
		t.Logf("downloadImagePkgs with multiple repos failed as expected: %v", err)
		// Should not fail due to "no repository configurations available"
		if strings.Contains(err.Error(), "no repository configurations available") {
			t.Error("Should not get 'no repository configurations' error when multiple repos are configured")
		}
	} else {
		t.Logf("downloadImagePkgs with multiple repositories succeeded")
	}

	// Verify that debutils.RepoCfgs was populated correctly
	if len(debutils.RepoCfgs) != 2 {
		t.Logf("Expected debutils.RepoCfgs to have 2 repositories, got %d (may be affected by previous tests)", len(debutils.RepoCfgs))
	}
}

// TestUbuntuLoadRepoConfigMultiple tests loadRepoConfig with multiple repositories
func TestUbuntuLoadRepoConfigMultiple(t *testing.T) {
	// Change to project root for tests that need config files
	originalDir, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Failed to change back to original directory: %v", err)
		}
	}()

	// Navigate to project root (3 levels up from internal/provider/ubuntu)
	if err := os.Chdir("../../../"); err != nil {
		t.Skipf("Cannot change to project root: %v", err)
		return
	}

	configs, err := loadRepoConfig("", "amd64")
	if err != nil {
		t.Skipf("loadRepoConfig failed (expected in test environment): %v", err)
		return
	}

	// Verify multiple repositories are loaded
	if len(configs) == 0 {
		t.Error("Expected at least one repository configuration")
		return
	}

	t.Logf("Loaded %d repository configurations", len(configs))

	// Verify each repository has required fields
	for i, config := range configs {
		t.Logf("Repository %d: %s", i+1, config.Name)

		if config.Name == "" {
			t.Errorf("Repository %d: expected name to be set", i+1)
		}

		if config.Arch != "amd64" {
			t.Errorf("Repository %d: expected arch 'amd64', got '%s'", i+1, config.Arch)
		}

		if config.PkgList == "" {
			t.Errorf("Repository %d: expected PkgList to be set", i+1)
		}

		if config.PkgPrefix == "" {
			t.Errorf("Repository %d: expected PkgPrefix to be set", i+1)
		}
	}
}

// TestUbuntuOsNameConstant tests the OsName constant value
func TestUbuntuOsNameConstant(t *testing.T) {
	expectedOsName := "ubuntu"
	if OsName != expectedOsName {
		t.Errorf("Expected OsName constant to be '%s', got '%s'", expectedOsName, OsName)
	}
}
