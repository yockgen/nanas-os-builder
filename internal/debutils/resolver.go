package debutils

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/provider"
)

// ParsePrimary parses the repodata/primary.xml.gz file from a given base URL.
func ParsePrimary(baseURL, gzHref string) ([]provider.PackageInfo, error) {

	// Download the debian repo .gz file with all components meta data
	PkgMetaFile := "/tmp/Packages.gz"
	zipFiles, err := Download(gzHref, PkgMetaFile)
	if err != nil {
		return nil, fmt.Errorf("failed to download repo file: %v", err)
	}

	// Decompress the .gz file and store the decompressed file in the same location
	if len(zipFiles) == 0 {
		return []provider.PackageInfo{}, fmt.Errorf("no files downloaded from repo URL: %s", gzHref)
	}
	files, err := Decompress(zipFiles[0])
	if err != nil {
		return []provider.PackageInfo{}, err
	}
	fmt.Printf("decompressed files: %v\n", files)

	// Parse the decompressed file
	f, err := os.Open(files[0])
	if err != nil {
		return nil, fmt.Errorf("failed to open decompressed file: %v", err)
	}
	defer f.Close()

	// packages file parser
	var pkgs []provider.PackageInfo
	scanner := bufio.NewScanner(f)
	pkg := provider.PackageInfo{}
	for scanner.Scan() {

		line := scanner.Text()

		if line == "" {
			// End of one package entry
			if pkg.Name != "" {
				pkgs = append(pkgs, pkg)
				pkg = provider.PackageInfo{}
			}
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "Package":
			pkg.Name = val
		case "Depends":
			// Split dependencies by comma and trim spaces
			deps := strings.Split(val, ",")
			for i := range deps {
				deps[i] = strings.TrimSpace(deps[i])
			}
			pkg.Requires = deps
		case "Filename":
			pkg.URL = val
		case "SHA256":
			pkg.Checksum = val
			// Add more fields as needed
		}
	}

	// Add the last package if file doesn't end with a blank line
	if pkg.Name != "" {
		pkgs = append(pkgs, pkg)
	}

	// Store the result in /tmp/Packages.trim
	outFile := "/tmp/Packages.trim"
	out, err := os.Create(outFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %v", err)
	}
	defer out.Close()

	for _, pkg := range pkgs {
		_, err := fmt.Fprintf(out, "Name: %s\nRequires: %s\nURL: %s\nChecksum: %s\n\n",
			pkg.Name, pkg.Requires, pkg.URL, pkg.Checksum)
		if err != nil {
			return nil, fmt.Errorf("failed to write to output file: %v", err)
		}
	}

	return pkgs, nil
}
