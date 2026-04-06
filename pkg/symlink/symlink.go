// Package symlink manages symlink operations between the system and the package store.
// It creates, removes, and verifies symlinks for installed packages.
package symlink

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StripPrefix removes the given prefix from the beginning of a path.
// If stripPrefix is empty or "/", the original path is returned unchanged.
// If the path does not start with the prefix, an error is returned.
// Examples:
//
//	StripPrefix("/tmp/kod/store/app/v1", "/tmp") -> "/kod/store/app/v1"
//	StripPrefix("/tmp/kod/store/app/v1", "") -> "/tmp/kod/store/app/v1" (no error)
//	StripPrefix("/tmp/kod/store/app/v1", "/") -> "/tmp/kod/store/app/v1" (no error)
//	StripPrefix("/other/path", "/tmp") -> error (path doesn't start with prefix)
func StripPrefix(path, stripPrefix string) (string, error) {
	// If no prefix specified, return path unchanged
	if stripPrefix == "" || stripPrefix == "/" {
		return path, nil
	}

	// Normalize the prefix to ensure it ends with / for proper prefix matching
	// This prevents "/tmp" from matching "/tmp2"
	if !strings.HasSuffix(stripPrefix, "/") {
		stripPrefix = stripPrefix + "/"
	}

	// Check if path starts with the prefix
	if !strings.HasPrefix(path, stripPrefix) {
		// Path doesn't start with prefix - this is an error
		return "", fmt.Errorf("path %q does not start with prefix %q", path, strings.TrimSuffix(stripPrefix, "/"))
	}

	// Remove the prefix (including the trailing /)
	result := path[len(stripPrefix):]

	// Ensure result starts with / for absolute paths
	if !strings.HasPrefix(result, "/") {
		result = "/" + result
	}

	return result, nil
}

// Manager handles symlink operations for package files.
type Manager struct {
	storeRoot   string // Root of the package store (e.g., /kod/store)
	symlinkRoot string // Root where symlinks are created (e.g., /)
	stripPrefix string // Prefix to strip from symlink targets (e.g., /tmp for chroot)
}

// NewManager creates a new symlink manager.
// storeRoot is where packages are stored (e.g., /kod/store)
// symlinkRoot is where symlinks are created (e.g., / for system root)
func NewManager(storeRoot, symlinkRoot string) *Manager {
	if symlinkRoot == "" {
		symlinkRoot = "/"
	}
	return &Manager{
		storeRoot:   storeRoot,
		symlinkRoot: symlinkRoot,
		stripPrefix: "",
	}
}

// NewManagerWithPrefix creates a new symlink manager with optional prefix stripping.
// stripPrefix is the prefix to remove from symlink targets (e.g., /tmp for chroot scenarios)
func NewManagerWithPrefix(storeRoot, symlinkRoot, stripPrefix string) *Manager {
	if symlinkRoot == "" {
		symlinkRoot = "/"
	}
	return &Manager{
		storeRoot:   storeRoot,
		symlinkRoot: symlinkRoot,
		stripPrefix: stripPrefix,
	}
}

// CreateSymlinks creates symlinks for all files in a package.
// It skips files that already exist unless they are existing symlinks pointing elsewhere.
// If stripPrefix is configured, it will be removed from symlink targets.
func (m *Manager) CreateSymlinks(pkgName, version string, files []string) error {
	if len(files) == 0 {
		return nil // Nothing to do
	}

	var failedFiles []string
	var skippedFiles []string

	for _, file := range files {
		// Skip directories and special files
		if filepath.Base(file) == "." || filepath.Base(file) == ".." {
			continue
		}

		symlinkPath := m.GetSymlinkPath(file)
		storePath := m.GetStorePath(pkgName, version, file)

		// Apply prefix stripping if configured
		targetPath := storePath
		if m.stripPrefix != "" && m.stripPrefix != "/" {
			stripped, err := StripPrefix(storePath, m.stripPrefix)
			if err != nil {
				failedFiles = append(failedFiles, fmt.Sprintf("%s (prefix strip failed: %v)", file, err))
				continue
			}
			targetPath = stripped
		}

		// Create parent directories if needed
		symlinkDir := filepath.Dir(symlinkPath)
		if err := os.MkdirAll(symlinkDir, 0755); err != nil {
			failedFiles = append(failedFiles, fmt.Sprintf("%s (mkdir failed: %v)", file, err))
			continue
		}

		// Check if symlink already exists
		if stat, err := os.Lstat(symlinkPath); err == nil {
			// File/symlink exists
			if stat.Mode()&os.ModeSymlink == os.ModeSymlink {
				// It's a symlink, check if it points to the same location
				target, err := os.Readlink(symlinkPath)
				if err == nil && target == targetPath {
					// Symlink already points to correct location, skip
					continue
				}
				// Symlink points elsewhere, skip with warning
				skippedFiles = append(skippedFiles, fmt.Sprintf("%s (symlink exists, pointing to %s)", file, target))
				continue
			}
			// Regular file exists, skip with warning
			skippedFiles = append(skippedFiles, fmt.Sprintf("%s (regular file exists)", file))
			continue
		}
		// Path doesn't exist, create symlink

		if err := os.Symlink(targetPath, symlinkPath); err != nil {
			failedFiles = append(failedFiles, fmt.Sprintf("%s (symlink creation failed: %v)", file, err))
			continue
		}
	}

	// Return error if any symlinks failed to create
	if len(failedFiles) > 0 {
		errMsg := fmt.Sprintf("failed to create %d symlinks:", len(failedFiles))
		for _, f := range failedFiles {
			errMsg += fmt.Sprintf("\n  - %s", f)
		}
		return errors.New(errMsg)
	}

	return nil
}

// RemoveSymlinks removes symlinks for all files in a package.
// It only removes symlinks, not regular files.
func (m *Manager) RemoveSymlinks(files []string) error {
	if len(files) == 0 {
		return nil // Nothing to do
	}

	var failedFiles []string
	var skippedFiles []string

	for _, file := range files {
		symlinkPath := m.GetSymlinkPath(file)

		// Check if path exists
		stat, err := os.Lstat(symlinkPath)
		if err != nil {
			if os.IsNotExist(err) {
				// File doesn't exist, skip
				skippedFiles = append(skippedFiles, fmt.Sprintf("%s (not found)", file))
				continue
			}
			failedFiles = append(failedFiles, fmt.Sprintf("%s (stat failed: %v)", file, err))
			continue
		}

		// Only remove if it's a symlink
		if stat.Mode()&os.ModeSymlink != os.ModeSymlink {
			// Regular file, don't remove
			skippedFiles = append(skippedFiles, fmt.Sprintf("%s (not a symlink, skipped)", file))
			continue
		}

		// Remove the symlink
		if err := os.Remove(symlinkPath); err != nil {
			failedFiles = append(failedFiles, fmt.Sprintf("%s (removal failed: %v)", file, err))
			continue
		}
	}

	// Return error if any symlinks failed to remove
	if len(failedFiles) > 0 {
		errMsg := fmt.Sprintf("failed to remove %d symlinks:", len(failedFiles))
		for _, f := range failedFiles {
			errMsg += fmt.Sprintf("\n  - %s", f)
		}
		return errors.New(errMsg)
	}

	return nil
}

// VerifySymlinks checks that all symlinks point to the correct locations.
// It handles both absolute and prefix-stripped symlink targets.
func (m *Manager) VerifySymlinks(pkgName, version string, files []string) error {
	if len(files) == 0 {
		return nil // Nothing to verify
	}

	var issues []string

	for _, file := range files {
		symlinkPath := m.GetSymlinkPath(file)
		expectedStorePath := m.GetStorePath(pkgName, version, file)

		// Apply prefix stripping to expected path if configured
		expectedTarget := expectedStorePath
		if m.stripPrefix != "" && m.stripPrefix != "/" {
			stripped, err := StripPrefix(expectedStorePath, m.stripPrefix)
			if err != nil {
				issues = append(issues, fmt.Sprintf("%s: prefix strip failed: %v", file, err))
				continue
			}
			expectedTarget = stripped
		}

		stat, err := os.Lstat(symlinkPath)
		if err != nil {
			if os.IsNotExist(err) {
				issues = append(issues, fmt.Sprintf("%s: symlink not found", file))
				continue
			}
			issues = append(issues, fmt.Sprintf("%s: stat failed: %v", file, err))
			continue
		}

		// Check if it's a symlink
		if stat.Mode()&os.ModeSymlink == 0 {
			issues = append(issues, fmt.Sprintf("%s: not a symlink (regular file)", file))
			continue
		}

		// Check where symlink points
		target, err := os.Readlink(symlinkPath)
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: readlink failed: %v", file, err))
			continue
		}

		if target != expectedTarget {
			issues = append(issues, fmt.Sprintf("%s: points to %s, expected %s", file, target, expectedTarget))
			continue
		}
	}

	if len(issues) > 0 {
		errMsg := fmt.Sprintf("verification failed for %d symlinks:", len(issues))
		for _, issue := range issues {
			errMsg += fmt.Sprintf("\n  - %s", issue)
		}
		return errors.New(errMsg)
	}

	return nil
}

// GetSymlinkPath returns the system path where a symlink should be created.
func (m *Manager) GetSymlinkPath(file string) string {
	return filepath.Join(m.symlinkRoot, file)
}

// GetStorePath returns the store path that a symlink should point to.
func (m *Manager) GetStorePath(pkgName, version, file string) string {
	return filepath.Join(m.storeRoot, pkgName, version, file)
}
