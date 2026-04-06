// Package aur provides Arch User Repository (AUR) support.
// pkgbuild.go implements parsing of PKGBUILD shell scripts.
package aur

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// PKGBUILDParser parses PKGBUILD shell scripts to extract package metadata
type PKGBUILDParser struct{}

// NewPKGBUILDParser creates a new PKGBUILD parser
func NewPKGBUILDParser() *PKGBUILDParser {
	return &PKGBUILDParser{}
}

// Parse parses a PKGBUILD file and extracts metadata
func (p *PKGBUILDParser) Parse(filePath string) (*PKGBUILDInfo, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PKGBUILD: %w", err)
	}
	defer file.Close()

	info := &PKGBUILDInfo{
		Architecture: []string{},
		Depends:      []string{},
		MakeDepends:  []string{},
		OptDepends:   []string{},
		CheckDepends: []string{},
		Conflicts:    []string{},
		Provides:     []string{},
		Replaces:     []string{},
		Options:      []string{},
		SHA256Sums:   []string{},
		MD5Sums:      []string{},
		Sources:      []string{},
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments and empty lines
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse different fields
		if strings.HasPrefix(line, "pkgname=") {
			info.Name = p.extractSimpleValue(line)
		} else if strings.HasPrefix(line, "pkgver=") {
			info.Version = p.extractSimpleValue(line)
		} else if strings.HasPrefix(line, "arch=") {
			info.Architecture = p.extractArrayValues(line)
		} else if strings.HasPrefix(line, "depends=") {
			info.Depends = p.extractArrayValues(line)
		} else if strings.HasPrefix(line, "makedepends=") {
			info.MakeDepends = p.extractArrayValues(line)
		} else if strings.HasPrefix(line, "optdepends=") {
			info.OptDepends = p.extractArrayValues(line)
		} else if strings.HasPrefix(line, "checkdepends=") {
			info.CheckDepends = p.extractArrayValues(line)
		} else if strings.HasPrefix(line, "conflicts=") {
			info.Conflicts = p.extractArrayValues(line)
		} else if strings.HasPrefix(line, "provides=") {
			info.Provides = p.extractArrayValues(line)
		} else if strings.HasPrefix(line, "replaces=") {
			info.Replaces = p.extractArrayValues(line)
		} else if strings.HasPrefix(line, "options=") {
			info.Options = p.extractArrayValues(line)
		} else if strings.HasPrefix(line, "sha256sums=") {
			info.SHA256Sums = p.extractArrayValues(line)
		} else if strings.HasPrefix(line, "md5sums=") {
			info.MD5Sums = p.extractArrayValues(line)
		} else if strings.HasPrefix(line, "source=") {
			info.Sources = p.extractArrayValues(line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading PKGBUILD: %w", err)
	}

	// Validate required fields
	if info.Name == "" {
		return nil, fmt.Errorf("PKGBUILD missing pkgname field")
	}

	if info.Version == "" {
		return nil, fmt.Errorf("PKGBUILD missing pkgver field")
	}

	return info, nil
}

// extractSimpleValue extracts a simple string value from a line like: field=value
func (p *PKGBUILDParser) extractSimpleValue(line string) string {
	// Find the '=' character
	idx := strings.Index(line, "=")
	if idx == -1 {
		return ""
	}

	value := strings.TrimSpace(line[idx+1:])

	// Remove surrounding quotes
	value = strings.TrimSpace(value)
	if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
		(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
		value = value[1 : len(value)-1]
	}

	return value
}

// extractArrayValues extracts array values from a line like: field=(value1 value2 ...)
func (p *PKGBUILDParser) extractArrayValues(line string) []string {
	// Find the '=' character
	idx := strings.Index(line, "=")
	if idx == -1 {
		return []string{}
	}

	value := strings.TrimSpace(line[idx+1:])

	// Check if it's an array (starts with '(')
	if !strings.HasPrefix(value, "(") {
		// Might be a simple value wrapped in array syntax that spans multiple lines
		// For now, just return empty
		return []string{}
	}

	// Remove array brackets
	value = strings.TrimPrefix(value, "(")
	value = strings.TrimSuffix(value, ")")
	value = strings.TrimSpace(value)

	// Split by spaces, handling quoted values
	var results []string
	var current string
	inQuote := false
	quoteChar := rune(0)

	for _, r := range value {
		switch {
		case !inQuote && (r == '"' || r == '\''):
			inQuote = true
			quoteChar = r
		case inQuote && r == quoteChar:
			inQuote = false
		case !inQuote && r == ' ':
			if current != "" {
				results = append(results, current)
				current = ""
			}
		default:
			current += string(r)
		}
	}

	if current != "" {
		results = append(results, current)
	}

	// Clean up results - remove quotes and parse version constraints
	var cleaned []string
	for _, item := range results {
		item = strings.Trim(item, "\"'")
		item = strings.TrimSpace(item)
		if item != "" {
			// For dependency arrays, remove version constraints for now
			// Extract just the package name if it has a version constraint
			item = p.extractPackageName(item)
			cleaned = append(cleaned, item)
		}
	}

	return cleaned
}

// extractPackageName extracts the package name from a dependency string
// Examples:
//
//	"bash" → "bash"
//	"go>=1.21" → "go"
//	"python<4.0" → "python"
//	"bash: runtime" → "bash"  (for optdepends)
func (p *PKGBUILDParser) extractPackageName(depString string) string {
	if depString == "" {
		return ""
	}

	// For optdepends, format is "package: description"
	if idx := strings.Index(depString, ":"); idx > 0 {
		depString = depString[:idx]
	}

	// Remove version constraints
	constraints := []string{">=", "<=", "==", "=", ">", "<"}
	for _, constraint := range constraints {
		if idx := strings.Index(depString, constraint); idx > 0 {
			return depString[:idx]
		}
	}

	return depString
}

// ExtractMultilineDependencies handles cases where dependencies span multiple lines
// This is a more robust parser for complex PKGBUILD files
func (p *PKGBUILDParser) ExtractMultilineDependencies(filePath string, fieldName string) ([]string, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PKGBUILD: %w", err)
	}
	defer file.Close()

	var results []string
	scanner := bufio.NewScanner(file)
	inArray := false
	arrayContent := ""

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(strings.TrimSpace(line), fieldName+"=") {
			// Start of array
			inArray = true
			arrayContent = line

			// If complete on one line, parse it
			if strings.Contains(line, ")") && strings.Contains(line, "(") {
				results = p.extractArrayValues(line)
				inArray = false
				break
			}
		} else if inArray {
			arrayContent += "\n" + line

			// Check if array ends
			if strings.Contains(line, ")") {
				// Parse complete array
				results = p.extractArrayFromMultiline(arrayContent)
				inArray = false
				break
			}
		}
	}

	return results, scanner.Err()
}

// extractArrayFromMultiline extracts array values from multiline content
func (p *PKGBUILDParser) extractArrayFromMultiline(content string) []string {
	// Remove newlines and extra spaces
	content = regexp.MustCompile(`\n\s*`).ReplaceAllString(content, " ")
	return p.extractArrayValues(content)
}

// ValidatePKGBUILD performs basic validation on parsed PKGBUILD info
func (p *PKGBUILDParser) ValidatePKGBUILD(info *PKGBUILDInfo) error {
	if info == nil {
		return fmt.Errorf("PKGBUILD info cannot be nil")
	}

	if info.Name == "" {
		return fmt.Errorf("package name is required")
	}

	if info.Version == "" {
		return fmt.Errorf("package version is required")
	}

	// Check for invalid characters in name/version
	if !isValidPackageName(info.Name) {
		return fmt.Errorf("invalid package name: %s", info.Name)
	}

	return nil
}

// isValidPackageName checks if a package name is valid (alphanumeric, dash, underscore)
func isValidPackageName(name string) bool {
	if name == "" {
		return false
	}

	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}

	return true
}
