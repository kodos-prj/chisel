package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yourusername/packmgr-go/pkg/config"
	"github.com/yourusername/packmgr-go/pkg/store"
)

// ExtractCommand handles extracting packages.
type ExtractCommand struct {
	config *config.Config
}

// NewExtractCommand creates a new extract command.
func NewExtractCommand(cfg *config.Config) *ExtractCommand {
	return &ExtractCommand{
		config: cfg,
	}
}

// Run executes the extract command.
// Usage: packmgr extract <package.pkg.tar.zst> [<package2.pkg.tar.zst>] ...
func (e *ExtractCommand) Run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("package file path required")
	}

	// Create store manager
	storeManager := store.NewStore(e.config.StoreRoot)

	var totalExtracted int
	var totalSize int64

	for _, pkgPath := range args {
		// Verify file exists
		if _, err := os.Stat(pkgPath); err != nil {
			fmt.Fprintf(os.Stderr, "✗ Package not found: %s\n", pkgPath)
			continue
		}

		// Extract filename without extension
		baseName := filepath.Base(pkgPath)
		baseName = baseName[:len(baseName)-len(".pkg.tar.zst")] // Remove .pkg.tar.zst

		// Parse package name and version from filename
		// Expected format: name-version-arch.pkg.tar.zst
		// Extract name and version
		var pkgName, pkgVersion string
		parts := pathToPackageParts(baseName)
		if len(parts) >= 2 {
			pkgName = parts[0]
			pkgVersion = parts[1]
		} else {
			fmt.Fprintf(os.Stderr, "✗ Invalid package filename format: %s\n", baseName)
			continue
		}

		fmt.Printf("Extracting %s/%s to store...\n", pkgName, pkgVersion)

		// Extract package to store
		extracted, err := storeManager.ExtractPackage(pkgPath, pkgName, pkgVersion)
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ Failed to extract %s: %v\n", baseName, err)
			continue
		}

		// Calculate size
		size, _ := storeManager.GetPackageSize(pkgName, pkgVersion)

		fmt.Printf("  ✓ Extracted %d files (%d bytes)\n", len(extracted), size)
		totalExtracted += len(extracted)
		totalSize += size

		// Set as current version
		if err := storeManager.SetLatestVersion(pkgName, pkgVersion); err != nil {
			fmt.Fprintf(os.Stderr, "  ! Warning: Failed to set as current version: %v\n", err)
		} else {
			fmt.Printf("  ✓ Set as current version\n")
		}
	}

	fmt.Printf("\n✓ Extraction complete: %d files extracted (%d bytes)\n", totalExtracted, totalSize)
	return nil
}

// pathToPackageParts parses a package filename to extract name-version parts.
// Example: "bash-5.3.9-1-x86_64" -> ["bash", "5.3.9-1"]
// Format: <name>-<version>-<pkgrel>-<arch>
func pathToPackageParts(pkgName string) []string {
	// Find positions of all dashes
	var dashes []int
	for i := 0; i < len(pkgName); i++ {
		if pkgName[i] == '-' {
			dashes = append(dashes, i)
		}
	}

	// Need at least 3 dashes for name-version-pkgrel-arch format
	if len(dashes) < 3 {
		return []string{pkgName}
	}

	// The third-to-last dash separates version-pkgrel from arch
	// The first dash we find (usually after name) separates name from version
	// We want to find the minimal name that could be valid
	// Strategy: try splitting at the earliest dash, keep the rest as version-pkgrel

	// For simplicity: first dash position gives us the split between name and version-pkgrel-arch
	if len(dashes) == 0 {
		return []string{pkgName}
	}

	// Try to find the correct split
	// The arch part is usually one of: x86_64, i686, aarch64, etc.
	// Work backwards: the last dash is before arch, second-to-last is before pkgrel
	// Third-to-last is before version

	thirdLastDash := dashes[len(dashes)-3]

	name := pkgName[:thirdLastDash]
	version := pkgName[thirdLastDash+1 : dashes[len(dashes)-1]]

	return []string{name, version}
}

// Help returns help text for the extract command.
func (e *ExtractCommand) Help() string {
	return `Extract packages to the store.

Usage:
  packmgr extract <package.pkg.tar.zst> [package2.pkg.tar.zst] ...

Options:
  --no-symlink     Don't create symlinks after extraction
  --skip-current   Don't set as current version
  
Examples:
  packmgr extract bash-5.3.9-1-x86_64.pkg.tar.zst
  packmgr extract /tmp/bash-5.3.9-1-x86_64.pkg.tar.zst
  packmgr extract *.pkg.tar.zst
`
}
