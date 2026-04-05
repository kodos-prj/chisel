package alpm

import (
	"regexp"
	"strconv"
	"unicode"
)

// VerCmp compares two version strings using the RPM version scheme.
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
// This implements the exact algorithm from pacman's lib/libalpm/version.c
func VerCmp(a, b string) int {
	// Parse and compare epochs first
	epochA, releaseA := splitEpoch(a)
	epochB, releaseB := splitEpoch(b)

	cmp := compareNumeric(epochA, epochB)
	if cmp != 0 {
		return cmp
	}

	// Split release and revision
	relA, revA := splitRevision(releaseA)
	relB, revB := splitRevision(releaseB)

	// Compare releases using RPM algorithm
	cmp = compareRPMVersions(relA, relB)
	if cmp != 0 {
		return cmp
	}

	// Compare revisions
	return compareRPMVersions(revA, revB)
}

// splitEpoch splits version string into epoch and release parts.
// Format: "EPOCH:RELEASE-REVISION" → ("EPOCH", "RELEASE-REVISION")
// If no epoch: ("0", "RELEASE-REVISION")
func splitEpoch(version string) (string, string) {
	for i, ch := range version {
		if ch == ':' {
			return version[:i], version[i+1:]
		}
	}
	return "0", version
}

// splitRevision splits release and revision.
// Format: "RELEASE-REVISION" → ("RELEASE", "REVISION")
// If no revision: ("RELEASE", "0")
func splitRevision(release string) (string, string) {
	for i := len(release) - 1; i >= 0; i-- {
		if release[i] == '-' {
			return release[:i], release[i+1:]
		}
	}
	return release, "0"
}

// compareNumeric compares two numeric strings.
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
func compareNumeric(a, b string) int {
	aNum, _ := strconv.ParseInt(a, 10, 64)
	bNum, _ := strconv.ParseInt(b, 10, 64)

	if aNum < bNum {
		return -1
	}
	if aNum > bNum {
		return 1
	}
	return 0
}

// compareRPMVersions compares two version strings using RPM algorithm.
// The algorithm splits strings on transitions between digit and non-digit,
// then compares segments:
// - Numeric segments are compared numerically
// - Non-numeric segments are compared lexicographically
// - Numbers always sort before non-numbers
// - Empty/missing segments sort before any segment ("1.0" < "1.0.1")
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
func compareRPMVersions(a, b string) int {
	if a == b {
		return 0
	}

	// Tokenize both strings
	segmentsA := tokenizeVersion(a)
	segmentsB := tokenizeVersion(b)

	// Compare segments
	minLen := len(segmentsA)
	if len(segmentsB) < minLen {
		minLen = len(segmentsB)
	}

	for i := 0; i < minLen; i++ {
		cmp := compareSegments(segmentsA[i], segmentsB[i])
		if cmp != 0 {
			return cmp
		}
	}

	// If all common segments are equal, shorter string is less
	// (e.g., "1.0" < "1.0.1")
	if len(segmentsA) < len(segmentsB) {
		return -1
	}
	if len(segmentsA) > len(segmentsB) {
		return 1
	}

	return 0
}

// Segment represents a tokenized part of a version string.
type Segment struct {
	IsNumeric bool
	Value     string
}

// tokenizeVersion splits a version string into segments.
// Example: "1.0rc1" → [Segment{true, "1"}, Segment{false, "."}, Segment{true, "0"}, Segment{false, "rc"}, Segment{true, "1"}]
func tokenizeVersion(version string) []Segment {
	var segments []Segment
	var current string
	var isNumeric bool
	var started bool

	for _, ch := range version {
		charIsDigit := unicode.IsDigit(ch)

		if !started {
			current = string(ch)
			isNumeric = charIsDigit
			started = true
		} else if charIsDigit == isNumeric {
			current += string(ch)
		} else {
			segments = append(segments, Segment{isNumeric, current})
			current = string(ch)
			isNumeric = charIsDigit
		}
	}

	if started && current != "" {
		segments = append(segments, Segment{isNumeric, current})
	}

	return segments
}

// compareSegments compares two version segments.
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
func compareSegments(a, b Segment) int {
	// If one is numeric and the other isn't, numeric comes first
	if a.IsNumeric && !b.IsNumeric {
		return -1
	}
	if !a.IsNumeric && b.IsNumeric {
		return 1
	}

	// Both numeric: compare numerically
	if a.IsNumeric {
		aNum, _ := strconv.ParseInt(a.Value, 10, 64)
		bNum, _ := strconv.ParseInt(b.Value, 10, 64)

		if aNum < bNum {
			return -1
		}
		if aNum > bNum {
			return 1
		}
		return 0
	}

	// Both non-numeric: compare lexicographically
	if a.Value < b.Value {
		return -1
	}
	if a.Value > b.Value {
		return 1
	}
	return 0
}

// isZeroSegment checks if a segment represents zero.
func isZeroSegment(seg Segment) bool {
	if !seg.IsNumeric {
		return false
	}
	num, _ := strconv.ParseInt(seg.Value, 10, 64)
	return num == 0
}

// ParseDependency parses a dependency string with optional version constraint.
// Examples: "pkg", "pkg>=1.0", "pkg=2.0", "pkg<3.0", "pkg<=1.0", "pkg>1.0"
// Returns: (packageName, constraint, error)
func ParseDependency(depStr string) (string, Constraint, error) {
	// Check for version constraints
	patterns := []struct {
		pattern string
		ctype   ConstraintType
	}{
		{`>=`, ConstraintGreaterEqual},
		{`<=`, ConstraintLessEqual},
		{`>`, ConstraintGreater},
		{`<`, ConstraintLess},
		{`=`, ConstraintEqual},
	}

	for _, p := range patterns {
		re := regexp.MustCompile(`(.+?)` + p.pattern + `(.+)`)
		match := re.FindStringSubmatch(depStr)
		if match != nil {
			return match[1], Constraint{Type: p.ctype, Value: match[2]}, nil
		}
	}

	// No constraint
	return depStr, Constraint{Type: ConstraintNone}, nil
}

// CheckVersionConstraint checks if a version satisfies a constraint.
// Returns true if the version satisfies the constraint.
func CheckVersionConstraint(version string, constraint Constraint) bool {
	if constraint.Type == ConstraintNone {
		return true
	}

	cmp := VerCmp(version, constraint.Value)

	switch constraint.Type {
	case ConstraintEqual:
		return cmp == 0
	case ConstraintGreaterEqual:
		return cmp >= 0
	case ConstraintGreater:
		return cmp > 0
	case ConstraintLessEqual:
		return cmp <= 0
	case ConstraintLess:
		return cmp < 0
	case ConstraintNone:
		return true
	}

	return false
}
