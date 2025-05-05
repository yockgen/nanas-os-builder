package main

import (
    "fmt"
    "os"
	"path/filepath"
	"go.uber.org/zap"
    "go.uber.org/zap/zapcore"
    "github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/pkgfetcher"
    "github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/config"
    "github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/provider"
    _ "github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/provider/azurelinux3" // register provider
)

// temporary placeholder for configuration
// This should be replaced with a proper configuration struct
const (
	workers = 4
	destDir = "./downloads"
)

// setupLogger initializes a zap logger with development configuration.
// It sets the encoder to use color for levels and ISO8601 for time.
func setupLogger() (*zap.Logger, error) {
    cfg := zap.NewDevelopmentConfig()
    cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
    cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
    cfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
    return cfg.Build()
}

func main() {

    logger, err := setupLogger()
    if err != nil {
        panic(err)
    }
    defer logger.Sync()
    zap.ReplaceGlobals(logger)
    sugar := zap.S()

    // check for input JSON
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <input.json>\n", os.Args[0])
		os.Exit(1)
	}
	configPath := os.Args[1]

    cfg, err := config.Load(configPath)
    if err != nil {
        sugar.Fatalf("loading config: %v", err)
    }
    
    providerName := cfg.Distro + cfg.Version

    // initialize provider
	p, ok := provider.Get(providerName)
	if !ok {
	    sugar.Fatalf("provider not found, %s", providerName)
	}

    // initialize provider
	if err := p.Init(); err != nil {
		sugar.Fatalf("provider init: %v", err)
	}

	// fetch package list
	pkgs, err := p.Packages()
	if err != nil {
		sugar.Fatalf("getting packages: %v", err)
	}

	// extract URLs
	urls := make([]string, len(pkgs))
	for i, pkg := range pkgs {
		urls[i] = pkg.URL
	}

	// start download
	absDest, err := filepath.Abs(destDir)
	if err != nil {
		sugar.Fatalf("invalid dest: %v", err)
	}
	sugar.Infof("downloading %d packages to %s", len(urls), absDest)
	if err := pkgfetcher.FetchPackages(urls, absDest, workers); err != nil {
		sugar.Fatalf("fetch failed: %v", err)
	}	
	sugar.Info("all downloads complete")

	// verify downloaded packages
	if err := p.Validate("./downloads"); err != nil {
		sugar.Fatalf("verification failed: %v", err)
	}
}