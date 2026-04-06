package alpm

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// parsePackageDatabase parses an Arch Linux sync database tar.gz format.
// The database contains package directories, each with metadata files.
// Returns a map of package names to Package objects.
// Note: The data may be gzipped or uncompressed (curl/http.Client may auto-decompress)
func parsePackageDatabase(data []byte, arch, repoName string) (map[string]*Package, error) {
	var tr *tar.Reader

	// Check if data is gzipped
	if len(data) > 2 && data[0] == 0x1f && data[1] == 0x8b {
		// Data is gzipped - decompress first
		gr, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gr.Close()
		tr = tar.NewReader(gr)
	} else {
		// Data is already uncompressed tar archive
		tr = tar.NewReader(bytes.NewReader(data))
	}

	packages := make(map[string]*Package)
	currentPkg := make(map[string]string) // Store as "pkgpath\x00filename" -> content to avoid issues with colons in version

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar header: %w", err)
		}

		content, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("failed to read tar content: %w", err)
		}

		// Parse file path: "pkgname/filename" or "pkgname-version/filename"
		parts := strings.Split(header.Name, "/")
		if len(parts) == 2 && parts[1] != "" { // Skip directory entries (parts[1] would be empty for directories)
			pkgPath := parts[0]
			fileName := parts[1]

			// Store file content with its name using a null separator to avoid issues with colons in version
			key := pkgPath + "\x00" + fileName
			currentPkg[key] = string(content)
		}
	}

	// Now process package data and build Package objects
	// Group files by package directory
	pkgDirs := make(map[string]map[string]string)
	for fullKey, content := range currentPkg {
		// Split on null byte to get pkgDir and fileName
		parts := strings.Split(fullKey, "\x00")
		if len(parts) != 2 {
			continue
		}
		pkgDir := parts[0]
		fileName := parts[1]

		if _, exists := pkgDirs[pkgDir]; !exists {
			pkgDirs[pkgDir] = make(map[string]string)
		}

		pkgDirs[pkgDir][fileName] = content
	}

	// Build Package objects
	for _, files := range pkgDirs {
		pkg, err := parsePackageEntry(files, arch)
		if err != nil {
			continue // Skip packages that fail to parse
		}

		// Set repository name if not already set from descriptor
		if pkg.Repository == "" {
			pkg.Repository = repoName
		}

		// Only keep packages matching the architecture
		if pkg.Architecture != "any" && pkg.Architecture != arch {
			continue
		}

		// Keep only the latest version if we have duplicates
		if existing, has := packages[pkg.Name]; has {
			if VerCmp(pkg.Version, existing.Version) > 0 {
				packages[pkg.Name] = pkg
			}
		} else {
			packages[pkg.Name] = pkg
		}
	}

	return packages, nil
}

// parsePackageEntry parses a single package directory's metadata files.
func parsePackageEntry(files map[string]string, arch string) (*Package, error) {
	pkg := &Package{
		Architecture: arch,
	}

	// Parse DESC file first - it contains the metadata
	if desc, ok := files["desc"]; ok {
		// Parse NAME from desc
		pkg.Name = parseMetadata("NAME", desc)
		// Parse VERSION from desc
		pkg.Version = parseMetadata("VERSION", desc)
		// Parse DESCRIPTION from desc
		pkg.Description = parseMetadata("DESC", desc)
		// Parse REPOSITORY from desc if available
		repo := parseMetadata("REPO", desc)
		if repo != "" {
			pkg.Repository = repo
		}

		// Parse SIZE fields
		if csize := parseMetadata("CSIZE", desc); csize != "" {
			if val, err := strconv.ParseInt(csize, 10, 64); err == nil {
				pkg.CompressedSize = val
			}
		}
		if isize := parseMetadata("ISIZE", desc); isize != "" {
			if val, err := strconv.ParseInt(isize, 10, 64); err == nil {
				pkg.InstalledSize = val
			}
		}

		// Parse ARCH from desc
		if arch := parseMetadata("ARCH", desc); arch != "" {
			pkg.Architecture = arch
		}
	}

	if pkg.Name == "" {
		return nil, fmt.Errorf("package name not found")
	}

	// Parse other metadata files
	if content, ok := files["depends"]; ok {
		deps := strings.Split(strings.TrimSpace(content), "\n")
		for _, dep := range deps {
			if dep = strings.TrimSpace(dep); dep != "" && dep != "%DEPENDS%" {
				pkg.DependsOn = append(pkg.DependsOn, dep)
			}
		}
	}

	if content, ok := files["optdepends"]; ok {
		deps := strings.Split(strings.TrimSpace(content), "\n")
		for _, dep := range deps {
			if dep = strings.TrimSpace(dep); dep != "" && dep != "%OPTDEPENDS%" {
				pkg.OptDepends = append(pkg.OptDepends, dep)
			}
		}
	}

	if content, ok := files["provides"]; ok {
		provides := strings.Split(strings.TrimSpace(content), "\n")
		for _, prov := range provides {
			if prov = strings.TrimSpace(prov); prov != "" && prov != "%PROVIDES%" {
				pkg.Provides = append(pkg.Provides, prov)
			}
		}
	}

	if content, ok := files["conflicts"]; ok {
		conflicts := strings.Split(strings.TrimSpace(content), "\n")
		for _, conf := range conflicts {
			if conf = strings.TrimSpace(conf); conf != "" && conf != "%CONFLICTS%" {
				pkg.Conflicts = append(pkg.Conflicts, conf)
			}
		}
	}

	if content, ok := files["replaces"]; ok {
		replaces := strings.Split(strings.TrimSpace(content), "\n")
		for _, repl := range replaces {
			if repl = strings.TrimSpace(repl); repl != "" && repl != "%REPLACES%" {
				pkg.Replaces = append(pkg.Replaces, repl)
			}
		}
	}

	return pkg, nil
}

// parseMetadata extracts a value from key-value metadata format.
// Format is like "%KEY%\nvalue" or multiline key-value pairs.
func parseMetadata(key, content string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if line == fmt.Sprintf("%%%s%%", key) && i+1 < len(lines) {
			return strings.TrimSpace(lines[i+1])
		}
	}
	return ""
}

// parseFilesMetadata extracts a list of files from FILES metadata.
func parseFilesMetadata(content string) []string {
	var files []string
	lines := strings.Split(content, "\n")
	inFiles := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "%FILES%" {
			inFiles = true
			continue
		}
		if strings.HasPrefix(line, "%") && line != "%FILES%" {
			inFiles = false
		}
		if inFiles && line != "" {
			files = append(files, line)
		}
	}

	return files
}

// DownloadDatabase downloads and parses a sync database from a repository URL.
// Currently not implemented - requires HTTP support.
func (c *Client) DownloadDatabase(repoName, repoURL string) (*Database, error) {
	// Construct URL: repoURL/repoName.db.tar.gz
	_ = fmt.Sprintf("%s/%s.db.tar.gz", strings.TrimSuffix(repoURL, "/"), repoName)

	// For now, we'll assume databases are already cached locally
	// In a full implementation, we would download using http.Get()
	// and cache to disk at c.DbPath/repoName.db.tar.gz

	return nil, fmt.Errorf("download not yet implemented; use cached databases")
}

// LoadCachedDatabase loads a database from the disk cache.
func (c *Client) LoadCachedDatabase(repoName string) (*Database, error) {
	// Construct path: DbPath/repoName.db
	// Arch mirrors serve databases as .db (which is gzip compressed tar)
	dbPath := fmt.Sprintf("%s/%s.db", strings.TrimSuffix(c.DbPath, "/"), repoName)

	// Read file
	data, err := readFileToBytes(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read database %s: %w", dbPath, err)
	}

	// Parse database
	packages, err := parsePackageDatabase(data, c.Arch, repoName)
	if err != nil {
		return nil, err
	}

	// Build Provides index
	provides := make(map[string][]*Package)
	for _, pkg := range packages {
		for _, prov := range pkg.Provides {
			// Parse version constraint if present: "virtual-name=1.0"
			provName := strings.Split(prov, "=")[0]
			provides[provName] = append(provides[provName], pkg)
		}
	}

	db := &Database{
		Name:     repoName,
		Path:     dbPath,
		Packages: packages,
		Provides: provides,
		Arch:     c.Arch,
	}

	return db, nil
}

// readFileToBytes reads an entire file into a byte slice.
// This is a helper for loading local cache files.
func readFileToBytes(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}
