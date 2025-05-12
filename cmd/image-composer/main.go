package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/config"
	"github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/pkgfetcher"
	"github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/provider"
	_ "github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/provider/azurelinux3" // register provider
	_ "github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/provider/elxr12"      // register provider
	_ "github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/provider/emt3_0"      // register provider
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// temporary placeholder for configuration
// This should be replaced with a proper configuration struct
const (
	workers = 8
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
	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Printf("failed to sync logger: %v\n", err)
		}
	}()
	zap.ReplaceGlobals(logger)
	sugar := zap.S()

	// check for input JSON
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <input.json>\n", os.Args[0])
		os.Exit(1)
	}
	configPath := os.Args[1]

	bc, err := config.Load(configPath)
	if err != nil {
		sugar.Fatalf("loading config: %v", err)
	}

	providerName := bc.Distro + bc.Version

	// get provider by name
	p, ok := provider.Get(providerName)
	if !ok {
		sugar.Fatalf("provider not found, %s", providerName)
	}

	// initialize provider
	if err := p.Init(bc); err != nil {
		sugar.Fatalf("provider init: %v", err)
	}

	// fetch the entire package list
	all, err := p.Packages()
	if err != nil {
		sugar.Fatalf("getting packages: %v", err)
	}

	// match the packages in the build spec against all the packages
	req, err := p.MatchRequested(bc.Packages, all)
	if err != nil {
		sugar.Fatalf("matching packages: %v", err)
	}
	sugar.Infof("matched al total of %d packages", len(req))
	for _, pkg := range req {
		sugar.Infof("-> %s", pkg.Name)
	}
	
	// resolve the dependencies of the requested packages
	needed, err := p.Resolve(req, all)
	if err != nil {
		sugar.Fatalf("resolving packages: %v", err)
	}
	sugar.Infof("resolved %d packages", len(needed))


	// extract URLs
	urls := make([]string, len(needed))
	for i, pkg := range needed {
		urls[i] = pkg.URL
	}

	// populate the cache download
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
	if err := p.Validate(destDir); err != nil {
		sugar.Fatalf("verification failed: %v", err)
	}

}
