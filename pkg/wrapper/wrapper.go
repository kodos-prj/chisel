// Package wrapper manages wrapper script generation for isolated package execution.
// It discovers library paths in extracted packages and generates shell wrapper scripts
// that set LD_LIBRARY_PATH to enable library isolation.
package wrapper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Generator handles wrapper script creation for packages.
type Generator struct {
	storeRoot   string // Root of the package store (e.g., /kod/store)
	wrapperRoot string // Root where wrapper scripts are created (e.g., /kod/wrappers)
	symlinkRoot string // Root where symlinks are created (e.g., /)
}

// NewGenerator creates a new wrapper script generator.
// storeRoot is where packages are stored (e.g., /kod/store)
// wrapperRoot is where wrapper scripts are created (e.g., /kod/wrappers)
// symlinkRoot is where symlinks are created (e.g., /)
func NewGenerator(storeRoot, wrapperRoot, symlinkRoot string) *Generator {
	if symlinkRoot == "" {
		symlinkRoot = "/"
	}
	return &Generator{
		storeRoot:   storeRoot,
		wrapperRoot: wrapperRoot,
		symlinkRoot: symlinkRoot,
	}
}

// DiscoverLibraries finds all .so files in a package's extracted files.
// Returns a map of library directory -> list of library files.
func (g *Generator) DiscoverLibraries(pkgName, version string) (map[string][]string, error) {
	pkgPath := filepath.Join(g.storeRoot, pkgName, version)

	// Check if package directory exists
	if _, err := os.Stat(pkgPath); err != nil {
		return nil, fmt.Errorf("package directory not found: %s", pkgPath)
	}

	libraries := make(map[string][]string)

	// Walk through all files in the package
	err := filepath.Walk(pkgPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file is a shared library (.so or .so.*)
		if strings.Contains(info.Name(), ".so") {
			relPath, err := filepath.Rel(pkgPath, path)
			if err != nil {
				return err
			}

			dir := filepath.Dir(relPath)
			libraries[dir] = append(libraries[dir], info.Name())
		}

		return nil
	})

	return libraries, err
}

// shouldIncludeInLD_LIBRARY_PATH determines if a dependency's libraries should be added to LD_LIBRARY_PATH.
// Uses a whitelist of essential C/C++ runtime libraries and excludes scripting runtimes
// that cause path pollution issues when combined with other applications.
func shouldIncludeInLD_LIBRARY_PATH(pkgName string) bool {
	// Packages whose libraries should NOT be included in LD_LIBRARY_PATH
	// Focus on scripting language runtimes and dev tools that cause conflicts
	excludedPackages := map[string]bool{
		// Scripting language runtimes - their lib directories conflict with app expectations
		"python":       true,
		"python2":      true,
		"python3":      true,
		"ruby":         true,
		"perl":         true,
		"php":          true,
		"nodejs":       true,
		"lua":          true,
		"guile":        true,
		"tcl":          true,
		"java-runtime": true,
		"jre":          true,
		"jdk":          true,

		// Development tools and compilers - not needed at runtime
		"gcc":      true,
		"clang":    true,
		"binutils": true,
		"gdb":      true,
		"lldb":     true,

		// Build systems - not needed at runtime
		"cmake":      true,
		"meson":      true,
		"ninja":      true,
		"autoconf":   true,
		"automake":   true,
		"libtool":    true,
		"pkg-config": true,

		// Terminfo and shell integration (these are data files, not libraries)
		"kitty-terminfo":          true,
		"kitty-shell-integration": true,
	}

	return !excludedPackages[pkgName]
}

// GenerateWrapper creates a wrapper script for a command that uses isolated libraries.
// The wrapper sets LD_LIBRARY_PATH to point to the package's lib directories and all dependency lib directories.
func (g *Generator) GenerateWrapper(cmdName, pkgName, version string, libDirs []string) error {
	return g.GenerateWrapperWithDeps(cmdName, pkgName, version, libDirs, nil, nil)
}

// GenerateWrapperWithDeps creates a wrapper script for a command with dependency library paths.
// depPkgs is a list of dependency package names, depVersions maps package names to versions.
// Dependencies that are known to cause conflicts are excluded (e.g., Python, Ruby, dev tools).
func (g *Generator) GenerateWrapperWithDeps(cmdName, pkgName, version string, libDirs []string, depPkgs []string, depVersions map[string]string) error {
	// Create wrapper directory if it doesn't exist
	if err := os.MkdirAll(g.wrapperRoot, 0755); err != nil {
		return fmt.Errorf("failed to create wrapper directory: %w", err)
	}

	wrapperPath := filepath.Join(g.wrapperRoot, cmdName)

	// Build LD_LIBRARY_PATH
	var ldLibraryPath []string

	// Add libraries from the main package first
	for _, libDir := range libDirs {
		// Convert to absolute path in store
		absLibPath := filepath.Join(g.storeRoot, pkgName, version, libDir)
		ldLibraryPath = append(ldLibraryPath, absLibPath)
	}

	// Add libraries from dependencies, but skip packages that cause conflicts
	if depVersions != nil && len(depPkgs) > 0 {
		for _, depName := range depPkgs {
			// Skip packages known to cause conflicts
			if !shouldIncludeInLD_LIBRARY_PATH(depName) {
				continue
			}

			if depVersion, ok := depVersions[depName]; ok {
				depLibDirs, err := g.DiscoverLibraries(depName, depVersion)
				if err != nil {
					// Log warning but continue with other dependencies
					fmt.Fprintf(os.Stderr, "Warning: Failed to discover libraries for dependency %s: %v\n", depName, err)
					continue
				}
				for dir := range depLibDirs {
					absLibPath := filepath.Join(g.storeRoot, depName, depVersion, dir)
					ldLibraryPath = append(ldLibraryPath, absLibPath)
				}
			}
		}
	}

	// Build the wrapper script content
	script := g.buildWrapperScript(cmdName, pkgName, version, ldLibraryPath)

	// Write the wrapper script
	if err := os.WriteFile(wrapperPath, []byte(script), 0755); err != nil {
		return fmt.Errorf("failed to write wrapper script: %w", err)
	}

	return nil
}

// buildWrapperScript constructs the content of a wrapper script.
func (g *Generator) buildWrapperScript(cmdName, pkgName, version string, ldLibraryPath []string) string {
	// Get the actual command path from the store
	// The binary is in usr/bin or usr/sbin, try usr/bin first
	cmdPath := filepath.Join(g.storeRoot, pkgName, version, "usr/bin", cmdName)

	// Build LD_LIBRARY_PATH value
	ldPath := strings.Join(ldLibraryPath, ":")
	if ldPath != "" {
		ldPath = ldPath + ":$LD_LIBRARY_PATH"
	}

	script := fmt.Sprintf(`#!/bin/bash
# Wrapper script for %s (from package %s-%s)
# Sets LD_LIBRARY_PATH to enable library isolation

export LD_LIBRARY_PATH="%s"
exec "%s" "$@"
`, cmdName, pkgName, version, ldPath, cmdPath)

	return script
}

// RemoveWrapper removes a wrapper script.
func (g *Generator) RemoveWrapper(cmdName string) error {
	wrapperPath := filepath.Join(g.wrapperRoot, cmdName)

	// Check if wrapper exists
	if _, err := os.Stat(wrapperPath); err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, no error
			return nil
		}
		return fmt.Errorf("failed to stat wrapper: %w", err)
	}

	// Remove the wrapper
	if err := os.Remove(wrapperPath); err != nil {
		return fmt.Errorf("failed to remove wrapper: %w", err)
	}

	return nil
}

// GetWrapperPath returns the path where a wrapper script should be created.
func (g *Generator) GetWrapperPath(cmdName string) string {
	return filepath.Join(g.wrapperRoot, cmdName)
}
