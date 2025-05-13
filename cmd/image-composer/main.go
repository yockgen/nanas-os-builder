package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/config"
	"github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/pkgfetcher"
	"github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/provider"
	"github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/validate"
	_ "github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/provider/azurelinux3" // register provider
	_ "github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/provider/elxr12"      // register provider
	_ "github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/provider/emt3_0"      // register provider
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
)

// Version information
var (
	Version   = "0.1.0"
	BuildDate = "unknown"
	CommitSHA = "unknown"
)

// Config options with defaults
var (
	workers  int    = 8
	cacheDir string = "./downloads"
	verbose  bool
)

// nopSyncer wraps an io.Writer but its Sync() does nothing.
type nopSyncer struct{ io.Writer }
func (n nopSyncer) Sync() error { return nil }

// setupLogger initializes a zap logger with development config,
// but replaces the usual fsyncing writer with one whose Sync() is a no-op.
func setupLogger() (*zap.Logger, error) {
    // start from DevConfig so we get console output, color, ISO8601 time, etc.
    cfg := zap.NewDevelopmentConfig()
    cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
    cfg.EncoderConfig.EncodeTime  = zapcore.ISO8601TimeEncoder
    cfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

    // create a console encoder using your EncoderConfig
    encoder := zapcore.NewConsoleEncoder(cfg.EncoderConfig)
    // wrap stderr in our nopSyncer
    writer  := nopSyncer{os.Stderr}
    // build a core that writes to that writer
    core    := zapcore.NewCore(encoder, writer, cfg.Level)

    // mirror the options NewDevelopmentConfig() would have added
    opts := []zap.Option{
        zap.AddCaller(),
        zap.Development(),
        zap.AddStacktrace(zapcore.ErrorLevel),
    }

    return zap.New(core, opts...), nil
}

// executeBuild handles the build command execution logic
func executeBuild(cmd *cobra.Command, args []string) error {
	logger := zap.L()
	sugar := logger.Sugar()

	// Check if spec file is provided as first positional argument
	if len(args) < 1 {
		return fmt.Errorf("no spec file provided, usage: image-composer build [flags] SPEC_FILE")
	}
	specFile := args[0]

	// Load and validate the configuration
	bc, err := config.Load(specFile)
	if err != nil {
		return fmt.Errorf("loading spec file: %v", err)
	}

	providerName := bc.Distro + bc.Version

	// Get provider by name
	p, ok := provider.Get(providerName)
	if !ok {
		return fmt.Errorf("provider not found: %s", providerName)
	}

	// Initialize provider
	if err := p.Init(bc); err != nil {
		return fmt.Errorf("provider init: %v", err)
	}

	// Fetch the entire package list
	all, err := p.Packages()
	if err != nil {
		return fmt.Errorf("getting packages: %v", err)
	}

	// Match the packages in the build spec against all the packages
	req, err := p.MatchRequested(bc.Packages, all)
	if err != nil {
		return fmt.Errorf("matching packages: %v", err)
	}
	sugar.Infof("matched a total of %d packages", len(req))
	if verbose {
		for _, pkg := range req {
			sugar.Infof("-> %s", pkg.Name)
		}
	}
	
	// Resolve the dependencies of the requested packages
	needed, err := p.Resolve(req, all)
	if err != nil {
		return fmt.Errorf("resolving packages: %v", err)
	}
	sugar.Infof("resolved %d packages", len(needed))

	// Extract URLs
	urls := make([]string, len(needed))
	for i, pkg := range needed {
		urls[i] = pkg.URL
	}

	// Populate the cache download
	absCacheDir, err := filepath.Abs(cacheDir)
	if err != nil {
		return fmt.Errorf("invalid cache directory: %v", err)
	}
	sugar.Infof("downloading %d packages to %s", len(urls), absCacheDir)
	if err := pkgfetcher.FetchPackages(urls, absCacheDir, workers); err != nil {
		return fmt.Errorf("fetch failed: %v", err)
	}
	sugar.Info("all downloads complete")

	// Verify downloaded packages
	if err := p.Validate(cacheDir); err != nil {
		return fmt.Errorf("verification failed: %v", err)
	}

	sugar.Info("build completed successfully")
	return nil
}

// executeValidate handles the validate command execution logic
func executeValidate(cmd *cobra.Command, args []string) error {
	logger := zap.L()
	sugar := logger.Sugar()

	// Check if spec file is provided as first positional argument
	if len(args) < 1 {
		return fmt.Errorf("no spec file provided, usage: image-composer validate SPEC_FILE")
	}
	specFile := args[0]

	sugar.Infof("Validating spec file: %s", specFile)
	
	// Read the file
	data, err := os.ReadFile(specFile)
	if err != nil {
		return fmt.Errorf("reading spec file: %v", err)
	}
	
	// Validate the JSON against schema
	if err := validate.ValidateJSON(data); err != nil {
		return fmt.Errorf("validation failed: %v", err)
	}
	
	sugar.Info("Spec file is valid")
	return nil
}

// executeVersion handles the version command logic
func executeVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("Image Composer Tool v%s\n", Version)
	fmt.Printf("Build Date: %s\n", BuildDate)
	fmt.Printf("Commit: %s\n", CommitSHA)
}

func main() {
	// Initialize logger
	logger, err := setupLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set up logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()
	zap.ReplaceGlobals(logger)

	// Root command
	rootCmd := &cobra.Command{
		Use:   "image-composer",
		Short: "Image Composer Tool (ICT) for building Linux distributions",
		Long: `Image Composer Tool (ICT) is a toolchain that enables building immutable
Linux distributions using a simple toolchain from pre-built packages emanating
from different Operating System Vendors (OSVs).`,
	}

	// Build command
	buildCmd := &cobra.Command{
		Use:   "build [flags] SPEC_FILE",
		Short: "Build a Linux distribution image",
		Long: `Build a Linux distribution image based on the specified spec file.
The spec file should be in JSON format according to the schema.`,
		Args: cobra.ExactArgs(1),
		RunE: executeBuild,
	}

	// Validate command
	validateCmd := &cobra.Command{
		Use:   "validate SPEC_FILE",
		Short: "Validate a spec file against the schema",
		Long:  `Validate that the given JSON spec file conforms to the schema.`,
		Args:  cobra.ExactArgs(1),
		RunE:  executeValidate,
	}

	// Version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Display version information",
		Run:   executeVersion,
	}

	// Add flags to build command
	buildCmd.Flags().IntVarP(&workers, "workers", "w", workers, "Number of concurrent download workers")
	buildCmd.Flags().StringVarP(&cacheDir, "cache-dir", "d", cacheDir, "Package cache directory")
	buildCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Add commands to root command
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(versionCmd)

	// Add global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
