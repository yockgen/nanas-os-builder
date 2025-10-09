package rpmutils

import (
	"testing"

	"github.com/open-edge-platform/os-image-composer/internal/ospackage"
)

func TestExtractRepoBase(t *testing.T) {
	tests := []struct {
		name     string
		rawURL   string
		expected string
		wantErr  bool
	}{
		{
			name:     "Debian pool URL",
			rawURL:   "https://example.com/debian/pool/main/a/acct/acct_6.6.4-5+b1_amd64.deb",
			expected: "https://example.com/debian/pool/",
			wantErr:  false,
		},
		{
			name:     "RPM Packages URL",
			rawURL:   "https://example.com/rpm/Packages/curl-8.8.0-2.azl3.x86_64.rpm",
			expected: "https://example.com/rpm/Packages/",
			wantErr:  false,
		},
		{
			name:     "RPM file direct URL",
			rawURL:   "https://example.com/repo/x86_64/curl-8.8.0-2.azl3.x86_64.rpm",
			expected: "https://example.com/repo/x86_64/",
			wantErr:  false,
		},
		{
			name:     "DEB file direct URL",
			rawURL:   "https://example.com/repo/binary-amd64/acct_6.6.4-5+b1_amd64.deb",
			expected: "https://example.com/repo/binary-amd64/",
			wantErr:  false,
		},
		{
			name:     "URL without recognized pattern",
			rawURL:   "https://example.com/some/path",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Invalid URL",
			rawURL:   "not-a-url",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractRepoBase(tt.rawURL)
			if tt.wantErr {
				if err == nil {
					t.Errorf("extractRepoBase() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("extractRepoBase() unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("extractRepoBase() = %q, want %q", result, tt.expected)
				}
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{
			name:     "v1 greater than v2",
			v1:       "acct_6.6.5-1_amd64.deb",
			v2:       "acct_6.6.4-5+b1_amd64.deb",
			expected: 1,
		},
		{
			name:     "v1 less than v2",
			v1:       "acct_6.6.4-1_amd64.deb",
			v2:       "acct_6.6.5-1_amd64.deb",
			expected: -1,
		},
		{
			name:     "v1 equal to v2",
			v1:       "acct_6.6.4-5+b1_amd64.deb",
			v2:       "acct_6.6.4-5+b1_amd64.deb",
			expected: 0,
		},
		{
			name:     "simple version comparison",
			v1:       "1.0.0",
			v2:       "2.0.0",
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareVersions(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestExtractBasePackageNameFromFile(t *testing.T) {
	tests := []struct {
		name     string
		fullName string
		expected string
	}{
		{
			name:     "RPM with version",
			fullName: "curl-8.8.0-2.azl3.x86_64.rpm",
			expected: "curl",
		},
		{
			name:     "RPM with devel suffix",
			fullName: "curl-devel-8.8.0-1.azl3.x86_64.rpm",
			expected: "curl-devel",
		},
		{
			name:     "RPM without .rpm extension",
			fullName: "curl-8.8.0-2.azl3.x86_64",
			expected: "curl",
		},
		{
			name:     "Package with multiple dashes",
			fullName: "python3-some-package-1.2.3-4.el8.noarch.rpm",
			expected: "python3-some-package",
		},
		{
			name:     "Simple package name without version",
			fullName: "simple-package",
			expected: "simple-package",
		},
		{
			name:     "Single word package",
			fullName: "curl",
			expected: "curl",
		},
		{
			name:     "Package name with no version part",
			fullName: "some-package-name-without-version",
			expected: "some-package-name-without-version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBasePackageNameFromFile(tt.fullName)
			if result != tt.expected {
				t.Errorf("extractBasePackageNameFromFile(%q) = %q, want %q", tt.fullName, result, tt.expected)
			}
		})
	}
}

func TestExtractBaseNameFromDep(t *testing.T) {
	tests := []struct {
		name     string
		req      string
		expected string
	}{
		{
			name:     "Simple requirement",
			req:      "curl",
			expected: "curl",
		},
		{
			name:     "Requirement with parentheses and space",
			req:      "(python3 >= 3.6)",
			expected: "python3",
		},
		{
			name:     "Requirement with complex expression",
			req:      "systemd (= 0:255-29.emt3)",
			expected: "systemd",
		},
		{
			name:     "Empty requirement",
			req:      "",
			expected: "",
		},
		{
			name:     "Requirement with only spaces",
			req:      "   ",
			expected: "",
		},
		{
			name:     "Package with version constraint",
			req:      "glibc >= 2.17",
			expected: "glibc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBaseNameFromDep(tt.req)
			if result != tt.expected {
				t.Errorf("extractBaseNameFromDep(%q) = %q, want %q", tt.req, result, tt.expected)
			}
		})
	}
}

func TestExtractVersionRequirement(t *testing.T) {
	tests := []struct {
		name          string
		reqVers       []string
		depName       string
		expectedOp    string
		expectedVer   string
		expectedFound bool
	}{
		{
			name:          "Exact version requirement",
			reqVers:       []string{"systemd (= 0:255-29.emt3)"},
			depName:       "systemd",
			expectedOp:    "=",
			expectedVer:   "0:255-29.emt3",
			expectedFound: true,
		},
		{
			name:          "Greater than requirement",
			reqVers:       []string{"glibc (>= 2.17)"},
			depName:       "glibc",
			expectedOp:    ">=",
			expectedVer:   "2.17",
			expectedFound: true,
		},
		{
			name:          "Alternative dependencies",
			reqVers:       []string{"curl (>= 7.0) | wget"},
			depName:       "curl",
			expectedOp:    ">=",
			expectedVer:   "7.0",
			expectedFound: true,
		},
		{
			name:          "Dependency not found",
			reqVers:       []string{"other-package (>= 1.0)"},
			depName:       "missing-package",
			expectedOp:    "",
			expectedVer:   "",
			expectedFound: false,
		},
		{
			name:          "Dependency without version constraint",
			reqVers:       []string{"curl"},
			depName:       "curl",
			expectedOp:    "",
			expectedVer:   "",
			expectedFound: false,
		},
		{
			name:          "Empty requirements",
			reqVers:       []string{},
			depName:       "curl",
			expectedOp:    "",
			expectedVer:   "",
			expectedFound: false,
		},
		{
			name:          "Multiple version parts",
			reqVers:       []string{"package (>= 1.2.3-4.el8)"},
			depName:       "package",
			expectedOp:    ">=",
			expectedVer:   "1.2.3-4.el8",
			expectedFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op, ver, found := extractVersionRequirement(tt.reqVers, tt.depName)
			if op != tt.expectedOp {
				t.Errorf("extractVersionRequirement() op = %q, want %q", op, tt.expectedOp)
			}
			if ver != tt.expectedVer {
				t.Errorf("extractVersionRequirement() ver = %q, want %q", ver, tt.expectedVer)
			}
			if found != tt.expectedFound {
				t.Errorf("extractVersionRequirement() found = %v, want %v", found, tt.expectedFound)
			}
		})
	}
}

func TestComparePackageVersions(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
		wantErr  bool
	}{
		{
			name:     "Empty versions",
			a:        "",
			b:        "",
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "First empty",
			a:        "",
			b:        "1.0",
			expected: -1,
			wantErr:  false,
		},
		{
			name:     "Second empty",
			a:        "1.0",
			b:        "",
			expected: 1,
			wantErr:  false,
		},
		{
			name:     "Equal versions",
			a:        "1.0.0",
			b:        "1.0.0",
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "First greater",
			a:        "2.0.0",
			b:        "1.0.0",
			expected: 1,
			wantErr:  false,
		},
		{
			name:     "Second greater",
			a:        "1.0.0",
			b:        "2.0.0",
			expected: -1,
			wantErr:  false,
		},
		{
			name:     "With epoch - first greater",
			a:        "2:1.0.0",
			b:        "1:2.0.0",
			expected: 1,
			wantErr:  false,
		},
		{
			name:     "With epoch - second greater",
			a:        "1:1.0.0",
			b:        "2:1.0.0",
			expected: -1,
			wantErr:  false,
		},
		{
			name:     "Complex version with revision",
			a:        "1.19-1.emt3",
			b:        "1.19",
			expected: 0, // Should be treated as equal due to prefix logic
			wantErr:  false,
		},
		{
			name:     "Debian-style versions",
			a:        "6.6.4-5+b1",
			b:        "6.6.4-5",
			expected: 0, // Due to prefix logic, these are treated as equal
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := comparePackageVersions(tt.a, tt.b)
			if tt.wantErr {
				if err == nil {
					t.Errorf("comparePackageVersions() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("comparePackageVersions() unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("comparePackageVersions(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.expected)
				}
			}
		})
	}
}

func TestFindAllCandidates(t *testing.T) {
	// Create test data
	parent := ospackage.PackageInfo{
		Name: "parent-package",
		URL:  "https://example.com/repo/parent.rpm",
	}

	allPackages := []ospackage.PackageInfo{
		{
			Name:    "curl-8.8.0-2.azl3.x86_64.rpm",
			Version: "8.8.0-2.azl3",
		},
		{
			Name:    "curl-7.8.0-1.azl3.x86_64.rpm",
			Version: "7.8.0-1.azl3",
		},
		{
			Name:     "another-package-1.0-1.rpm",
			Version:  "1.0-1",
			Provides: []string{"provided-capability"},
		},
		{
			Name:  "file-provider-1.0-1.rpm",
			Files: []string{"/usr/bin/curl"},
		},
	}

	tests := []struct {
		name          string
		depName       string
		expectedCount int
		expectedFirst string // Name of the first candidate (highest version)
	}{
		{
			name:          "Direct name match",
			depName:       "curl",
			expectedCount: 2,                              // curl packages
			expectedFirst: "curl-8.8.0-2.azl3.x86_64.rpm", // Higher version
		},
		{
			name:          "Provides match",
			depName:       "provided-capability",
			expectedCount: 1, // Only the package that provides this capability
			expectedFirst: "another-package-1.0-1.rpm",
		},
		{
			name:          "File match",
			depName:       "/usr/bin/curl",
			expectedCount: 1,
			expectedFirst: "file-provider-1.0-1.rpm",
		},
		{
			name:          "No match",
			depName:       "nonexistent",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidates, err := findAllCandidates(parent, tt.depName, allPackages)
			if err != nil {
				t.Errorf("findAllCandidates() unexpected error: %v", err)
			}
			if len(candidates) != tt.expectedCount {
				t.Errorf("findAllCandidates() returned %d candidates, want %d", len(candidates), tt.expectedCount)
			}
			if tt.expectedCount > 0 && candidates[0].Name != tt.expectedFirst {
				t.Errorf("findAllCandidates() first candidate = %q, want %q", candidates[0].Name, tt.expectedFirst)
			}
		})
	}
}

func TestResolveTopPackageConflicts(t *testing.T) {
	// Save original Dist value and restore after test
	originalDist := Dist
	defer func() { Dist = originalDist }()

	allPackages := []ospackage.PackageInfo{
		{
			Name:    "acct",
			Version: "6.6.4-5+b1",
			URL:     "https://example.com/acct_6.6.4-5+b1_amd64.deb",
		},
		{
			Name:    "acct-205-25.azl3.noarch.rpm",
			Version: "205-25.azl3",
		},
		{
			Name:    "acct-tools",
			Version: "1.0-1.azl3",
		},
		{
			Name:    "acct-new",
			Version: "1.0-1.emt3",
		},
		{
			Name:    "acct-other",
			Version: "2.0-1.azl3",
		},
	}

	tests := []struct {
		name        string
		want        string
		pkgType     string
		dist        string
		expectedPkg string
		expectFound bool
	}{
		{
			name:        "Exact match with file extension",
			want:        "acct_6.6.4-5+b1_amd64",
			pkgType:     "deb",
			dist:        "",
			expectedPkg: "acct",
			expectFound: true,
		},
		{
			name:        "Base name match",
			want:        "acct",
			pkgType:     "rpm",
			dist:        "",
			expectedPkg: "acct", // Should find the first acct package
			expectFound: true,
		},
		{
			name:        "Base name match with dist filter",
			want:        "acct",
			pkgType:     "rpm",
			dist:        "azl3",
			expectedPkg: "acct", // The exact package name returned might be different due to filtering logic
			expectFound: true,
		},
		{
			name:        "No match",
			want:        "nonexistent",
			pkgType:     "rpm",
			dist:        "",
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Dist = tt.dist
			pkg, found := ResolveTopPackageConflicts(tt.want, tt.pkgType, allPackages)
			if found != tt.expectFound {
				t.Errorf("ResolveTopPackageConflicts() found = %v, want %v", found, tt.expectFound)
			}
			if tt.expectFound && pkg.Name != tt.expectedPkg {
				t.Logf("ResolveTopPackageConflicts() found pkg: Name=%q, Version=%q", pkg.Name, pkg.Version)
				t.Errorf("ResolveTopPackageConflicts() pkg.Name = %q, want %q", pkg.Name, tt.expectedPkg)
			}
		})
	}
}

func TestResolveMultiCandidates(t *testing.T) {
	tests := []struct {
		name         string
		parentPkg    ospackage.PackageInfo
		candidates   []ospackage.PackageInfo
		expectedName string
		expectError  bool
	}{
		{
			name: "No candidates",
			parentPkg: ospackage.PackageInfo{
				URL: "https://example.com/repo/parent.rpm",
			},
			candidates:  []ospackage.PackageInfo{},
			expectError: true,
		},
		{
			name: "Single candidate",
			parentPkg: ospackage.PackageInfo{
				URL: "https://example.com/repo/parent.rpm",
			},
			candidates: []ospackage.PackageInfo{
				{Name: "single-candidate", URL: "https://example.com/repo/single.rpm"},
			},
			expectedName: "single-candidate",
			expectError:  false,
		},
		{
			name: "Multiple candidates same repo",
			parentPkg: ospackage.PackageInfo{
				URL: "https://example.com/repo/parent.rpm",
			},
			candidates: []ospackage.PackageInfo{
				{Name: "candidate1", Version: "1.0", URL: "https://example.com/repo/candidate1.rpm"},
				{Name: "candidate2", Version: "2.0", URL: "https://example.com/repo/candidate2.rpm"},
			},
			expectedName: "candidate2", // Should pick the latest version
			expectError:  false,
		},
		{
			name: "Multiple candidates different repos",
			parentPkg: ospackage.PackageInfo{
				URL: "https://example.com/repo1/parent.rpm",
			},
			candidates: []ospackage.PackageInfo{
				{Name: "candidate1", Version: "1.0", URL: "https://example.com/repo1/candidate1.rpm"},
				{Name: "candidate2", Version: "2.0", URL: "https://example.com/repo2/candidate2.rpm"},
			},
			expectedName: "candidate1", // Should prefer same repo
			expectError:  false,
		},
		{
			name: "Version constraint satisfied",
			parentPkg: ospackage.PackageInfo{
				URL:         "https://example.com/repo/parent.rpm",
				RequiresVer: []string{"testpkg (>= 1.5)"},
			},
			candidates: []ospackage.PackageInfo{
				{Name: "testpkg-1.0-1.rpm", Version: "1.0", URL: "https://example.com/repo/testpkg1.rpm"},
				{Name: "testpkg-2.0-1.rpm", Version: "2.0", URL: "https://example.com/repo/testpkg2.rpm"},
			},
			expectedName: "testpkg-2.0-1.rpm", // Should pick the one that satisfies constraint
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolveMultiCandidates(tt.parentPkg, tt.candidates)
			if tt.expectError {
				if err == nil {
					t.Errorf("resolveMultiCandidates() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("resolveMultiCandidates() unexpected error: %v", err)
				}
				if result.Name != tt.expectedName {
					t.Errorf("resolveMultiCandidates() result.Name = %q, want %q", result.Name, tt.expectedName)
				}
			}
		})
	}
}
