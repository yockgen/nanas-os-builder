package rpmutils

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"

	"github.com/open-edge-platform/os-image-composer/internal/ospackage"
	"github.com/open-edge-platform/os-image-composer/internal/utils/logger"
)

func cleanDependencyName(dep string) string {
	depName := strings.TrimSpace(dep)

	// Handle alternatives (|) - take the first option
	if idx := strings.Index(depName, "|"); idx > 0 {
		depName = strings.TrimSpace(depName[:idx])
	}

	// Handle architecture qualifiers (:)
	if idx := strings.Index(depName, ":"); idx > 0 {
		depName = depName[:idx]
	}

	// Handle version constraints - remove everything from opening parenthesis
	if idx := strings.Index(depName, "("); idx > 0 {
		depName = strings.TrimSpace(depName[:idx])
	} else if idx := strings.Index(depName, " "); idx > 0 {
		// Handle cases where there's a space but no parentheses
		depName = depName[:idx]
	}

	return strings.TrimSpace(depName)
}

func resolveMultiCandidates(parentPkg ospackage.PackageInfo, candidates []ospackage.PackageInfo) (ospackage.PackageInfo, error) {
	parentBase, err := extractRepoBase(parentPkg.URL)
	if err != nil {
		return ospackage.PackageInfo{}, fmt.Errorf("failed to extract repo base from parent package URL: %w", err)
	}

	/////////////////////////////////////
	//A: if version is specified
	/////////////////////////////////////
	// All candidates have the same .Name, so just use candidates[0].Name for version extraction
	op := ""
	ver := ""
	hasVersionConstraint := false
	if len(candidates) > 0 {
		op, ver, hasVersionConstraint = extractVersionRequirement(parentPkg.RequiresVer, candidates[0].Name)
	}

	if hasVersionConstraint {
		// First pass: look for candidates from the same repo that meet version constraint
		var sameRepoMatches []ospackage.PackageInfo
		var otherRepoMatches []ospackage.PackageInfo

		for _, candidate := range candidates {
			candidateBase, err := extractRepoBase(candidate.URL)
			if err != nil {
				continue
			}

			// Check if version constraint is satisfied
			cmp, err := comparePackageVersions(candidate.Version, ver)
			if err != nil {
				continue
			}

			versionMatches := false
			switch op {
			case "=":
				versionMatches = (cmp == 0)
			case "<<", "<":
				versionMatches = (cmp < 0)
			case "<=":
				versionMatches = (cmp <= 0)
			case ">>", ">":
				versionMatches = (cmp > 0)
			case ">=":
				versionMatches = (cmp >= 0)
			}

			if versionMatches {
				if candidateBase == parentBase {
					sameRepoMatches = append(sameRepoMatches, candidate)
				} else {
					otherRepoMatches = append(otherRepoMatches, candidate)
				}
			}
		}

		// Priority 1: return first match from same repo
		if len(sameRepoMatches) > 0 {
			return sameRepoMatches[0], nil
		}

		// Priority 2: return first match from other repos
		if len(otherRepoMatches) > 0 {
			return otherRepoMatches[0], nil
		}

		return ospackage.PackageInfo{}, fmt.Errorf("no candidates satisfy version constraint = %s%s", op, ver)
	}

	/////////////////////////////////////
	// B: if version is not specified
	//////////////////////////////////////

	// Check for empty candidates list
	if len(candidates) == 0 {
		return ospackage.PackageInfo{}, fmt.Errorf("no candidates provided for selection")
	}

	// If only one candidate, return it
	if len(candidates) == 1 {
		return candidates[0], nil
	}

	// Rule 1: find all candidates with the same base URL and return the latest version
	var sameBaseCandidates []ospackage.PackageInfo
	for _, candidate := range candidates {
		candidateBase, err := extractRepoBase(candidate.URL)
		if err != nil {
			continue
		}
		if candidateBase == parentBase {
			sameBaseCandidates = append(sameBaseCandidates, candidate)
		}
	}

	// If we found candidates with the same base URL, return the one with the latest version
	if len(sameBaseCandidates) > 0 {
		if len(sameBaseCandidates) == 1 {
			return sameBaseCandidates[0], nil
		}

		// Find the candidate with the latest version
		latest := sameBaseCandidates[0]
		for _, candidate := range sameBaseCandidates[1:] {
			cmp := compareVersions(candidate.Version, latest.Version)
			if cmp > 0 {
				latest = candidate
			}
		}

		return latest, nil
	}

	// Rule 2: If no candidate has the same repo, return the first candidate in other repos
	return candidates[0], nil
}

func extractRepoBase(rawURL string) (string, error) {
	log := logger.Logger()
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	// Split path by "/pool/" (debian) or "/Packages/" (rpm)
	parts := strings.SplitN(u.Path, "/pool/", 2)
	if len(parts) < 2 {
		parts = strings.SplitN(u.Path, "/Packages/", 2)
		if len(parts) < 2 {
			log.Errorf("URL does not contain /pool/ or /Packages/: %s", rawURL)
			return "", fmt.Errorf("URL does not contain /pool/ or /Packages/: %s", rawURL)
		}
	}

	// Rebuild base URL: scheme + host + prefix before /pool/
	base := fmt.Sprintf("%s://%s%s/", u.Scheme, u.Host, parts[0])
	return base, nil
}

func extractVersionRequirement(reqVers []string, depName string) (op string, ver string, found bool) {
	for _, reqVer := range reqVers {
		reqVer = strings.TrimSpace(reqVer)

		// Handle alternatives (|) - check if our depName is in any of the alternatives
		alternatives := strings.Split(reqVer, "|")
		for _, alt := range alternatives {
			alt = strings.TrimSpace(alt)

			// Check if this alternative starts with the dependency name we're looking for
			//cleanReqName := cleanDependencyName(alt)
			cleanReqName := alt
			if cleanReqName != depName {
				continue // Skip to next alternative
			}

			// Found our dependency in this alternative, now extract version constraint
			// Find version constraint inside parentheses
			if idx := strings.Index(alt, "("); idx != -1 {
				verConstraint := alt[idx+1:]
				if idx2 := strings.Index(verConstraint, ")"); idx2 != -1 {
					verConstraint = verConstraint[:idx2]
				}

				// Split into operator and version
				parts := strings.Fields(verConstraint)
				if len(parts) == 2 {
					op := parts[0]
					ver := parts[1]
					return op, ver, true
				}
			}

			// If we found the dependency but no version constraint, return found=false
			return "", "", false
		}
	}

	return "", "", false
}

func comparePackageVersions(a, b string) (int, error) {
	// Empty-version handling: empty < any non-empty
	if a == "" && b == "" {
		return 0, nil
	}
	if a == "" {
		return -1, nil
	}
	if b == "" {
		return 1, nil
	}

	// Helper to split epoch
	splitEpoch := func(ver string) (epoch int, rest string) {
		parts := strings.SplitN(ver, ":", 2)
		if len(parts) == 2 {
			if _, err := fmt.Sscanf(parts[0], "%d", &epoch); err != nil {
				epoch = 0
			}
			rest = parts[1]
		} else {
			epoch = 0
			rest = ver
		}
		return
	}

	// Split upstream_version and debian_revision at last hyphen
	splitRevision := func(ver string) (upstream string, debian string) {
		if i := strings.LastIndex(ver, "-"); i >= 0 {
			return ver[:i], ver[i+1:]
		}
		return ver, ""
	}

	// nextSegment returns the next contiguous numeric or non-numeric segment.
	nextSegment := func(s string) (seg string, rest string, numeric bool) {
		if s == "" {
			return "", "", false
		}
		// numeric segment
		if s[0] >= '0' && s[0] <= '9' {
			i := 0
			for i < len(s) && s[i] >= '0' && s[i] <= '9' {
				i++
			}
			return s[:i], s[i:], true
		}
		// non-numeric segment
		i := 0
		for i < len(s) && (s[i] < '0' || s[i] > '9') {
			i++
		}
		return s[:i], s[i:], false
	}

	// Character ordering per Debian: '~' < end-of-string < letters < other characters.
	// This ordering is crucial for correct Debian version comparison, as defined in
	// Debian Policy Manual section 5.6.12 ("Version"). See:
	// https://www.debian.org/doc/debian-policy/ch-controlfields.html#version
	charOrder := func(r rune) int {
		if r == '~' {
			return -2
		}
		if r == 0 {
			return -1
		}
		if unicode.IsLetter(r) {
			return int(r)
		}
		return 0x100 + int(r)
	}

	// Compare two non-digit segments using Debian ordering
	compareNonDigitSegments := func(aSeg, bSeg string) int {
		ai, bi := 0, 0
		for {
			var ra, rb rune
			if ai < len(aSeg) {
				ra = rune(aSeg[ai])
			} else {
				ra = 0
			}
			if bi < len(bSeg) {
				rb = rune(bSeg[bi])
			} else {
				rb = 0
			}
			// both ended
			if ra == 0 && rb == 0 {
				return 0
			}
			if ra != rb {
				oa := charOrder(ra)
				ob := charOrder(rb)
				if oa < ob {
					return -1
				}
				return 1
			}
			ai++
			bi++
		}
	}

	// Compare numeric segments (as dpkg: strip leading zeros, compare length, then lexicographically)
	compareNumericSegments := func(aSeg, bSeg string) int {
		aTrim := strings.TrimLeft(aSeg, "0")
		bTrim := strings.TrimLeft(bSeg, "0")
		// treat empty as zero
		if aTrim == "" && bTrim == "" {
			return 0
		}
		if aTrim == "" {
			return -1
		}
		if bTrim == "" {
			return 1
		}
		// longer numeric (more digits) is greater
		if len(aTrim) > len(bTrim) {
			return 1
		}
		if len(aTrim) < len(bTrim) {
			return -1
		}
		// same length -> lexical compare works
		if aTrim > bTrim {
			return 1
		}
		if aTrim < bTrim {
			return -1
		}
		return 0
	}

	// Handle epoch
	epochA, restA := splitEpoch(a)
	epochB, restB := splitEpoch(b)
	if epochA < epochB {
		return -1, nil
	}
	if epochA > epochB {
		return 1, nil
	}

	// Split upstream and debian revisions
	upA, debA := splitRevision(restA)
	upB, debB := splitRevision(restB)

	// Compare iterative parts (used for upstream version and debian revision)
	compareParts := func(sa, sb string) int {
		for sa != "" || sb != "" {
			// Handle tilde first: '~' sorts before everything (including end-of-string)
			if (len(sa) > 0 && sa[0] == '~') || (len(sb) > 0 && sb[0] == '~') {
				if len(sa) > 0 && sa[0] == '~' && !(len(sb) > 0 && sb[0] == '~') {
					return -1
				}
				if len(sb) > 0 && sb[0] == '~' && !(len(sa) > 0 && sa[0] == '~') {
					return 1
				}
				// both have tilde: consume and continue
				if len(sa) > 0 && sa[0] == '~' && len(sb) > 0 && sb[0] == '~' {
					sa = sa[1:]
					sb = sb[1:]
					continue
				}
			}

			// After tilde handling, if either side is exhausted, the exhausted side is less
			if sa == "" && sb == "" {
				break
			}
			if sa == "" {
				return -1
			}
			if sb == "" {
				return 1
			}

			segA, restASeg, numA := nextSegment(sa)
			segB, restBSeg, numB := nextSegment(sb)

			// both empty segments -> continue
			if segA == "" && segB == "" {
				sa, sb = restASeg, restBSeg
				continue
			}

			// numeric vs non-numeric: numeric < non-numeric
			if numA != numB {
				if numA {
					return -1
				}
				return 1
			}

			// both numeric
			if numA && numB {
				if cmp := compareNumericSegments(segA, segB); cmp != 0 {
					return cmp
				}
			} else { // both non-numeric
				if cmp := compareNonDigitSegments(segA, segB); cmp != 0 {
					return cmp
				}
			}

			sa, sb = restASeg, restBSeg
		}
		return 0
	}

	// Compare upstream versions first, then debian revisions
	if cmp := compareParts(upA, upB); cmp != 0 {
		return cmp, nil
	}
	if cmp := compareParts(debA, debB); cmp != 0 {
		return cmp, nil
	}
	return 0, nil
}

// compareVersions compares two Debian package versions
// Returns 1 if v1 > v2, -1 if v1 < v2, 0 if equal
func compareVersions(v1, v2 string) int {
	// Extract version from Debian package names like "acct_6.6.4-5+b1_amd64.deb"
	extractVersion := func(name string) string {
		parts := strings.Split(name, "_")
		if len(parts) >= 2 {
			return parts[1]
		}
		return name
	}
	ver1 := extractVersion(v1)
	ver2 := extractVersion(v2)
	cmp, _ := comparePackageVersions(ver1, ver2)
	return cmp
}
