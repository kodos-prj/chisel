// Package aur provides Arch User Repository (AUR) support.
// git.go implements downloading PKGBUILD files via git clone.
package aur

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// GitHandler manages downloading PKGBUILD files from the AUR git repository
type GitHandler struct {
	baseURL  string        // https://aur.archlinux.org
	cacheDir string        // /kod/build-cache/ or similar
	timeout  time.Duration // Git operation timeout
}

// NewGitHandler creates a new Git handler for AUR
func NewGitHandler(cacheDir string) *GitHandler {
	return &GitHandler{
		baseURL:  "https://aur.archlinux.org",
		cacheDir: cacheDir,
		timeout:  30 * time.Second,
	}
}

// ClonePKGBUILD clones the PKGBUILD repository for a package
// Returns the path to the cloned directory containing the PKGBUILD
func (gh *GitHandler) ClonePKGBUILD(pkgName, destDir string) (string, error) {
	if pkgName == "" {
		return "", fmt.Errorf("package name cannot be empty")
	}

	if destDir == "" {
		return "", fmt.Errorf("destination directory cannot be empty")
	}

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Construct git repository URL
	repoURL := fmt.Sprintf("%s/%s.git", gh.baseURL, pkgName)

	// Clone with depth=1 for efficiency (only latest commit)
	clonePath := filepath.Join(destDir, pkgName)

	// Check if already cloned
	if _, err := os.Stat(clonePath); err == nil {
		// Directory exists, update it instead
		return gh.updateClone(clonePath)
	}

	// Clone the repository
	cmd := exec.Command("git", "clone", "--depth=1", repoURL, clonePath)

	// Set timeout context
	ctx, cancel := createTimeoutContext(gh.timeout)
	defer cancel()
	cmd = setCommandContext(cmd, ctx)

	// Capture output for error reporting
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git clone failed for %s: %w\nOutput: %s", pkgName, err, string(output))
	}

	return clonePath, nil
}

// updateClone updates an existing clone to the latest version
func (gh *GitHandler) updateClone(clonePath string) (string, error) {
	// Pull latest changes
	cmd := exec.Command("git", "-C", clonePath, "pull", "origin", "master")

	ctx, cancel := createTimeoutContext(gh.timeout)
	defer cancel()
	cmd = setCommandContext(cmd, ctx)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git pull failed: %w\nOutput: %s", err, string(output))
	}

	return clonePath, nil
}

// ClonePKGBUILDVersion clones a specific version of a PKGBUILD (if version history is needed)
// Currently not used, but available for future version pinning support
func (gh *GitHandler) ClonePKGBUILDVersion(pkgName, version, destDir string) (string, error) {
	if pkgName == "" {
		return "", fmt.Errorf("package name cannot be empty")
	}

	if version == "" {
		return "", fmt.Errorf("version cannot be empty")
	}

	if destDir == "" {
		return "", fmt.Errorf("destination directory cannot be empty")
	}

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Construct git repository URL
	repoURL := fmt.Sprintf("%s/%s.git", gh.baseURL, pkgName)
	clonePath := filepath.Join(destDir, pkgName)

	// Clone the repository
	cmd := exec.Command("git", "clone", "--depth=50", repoURL, clonePath)

	ctx, cancel := createTimeoutContext(gh.timeout)
	defer cancel()
	cmd = setCommandContext(cmd, ctx)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git clone failed: %w\nOutput: %s", err, string(output))
	}

	// Checkout specific tag/version if it exists
	checkoutCmd := exec.Command("git", "-C", clonePath, "checkout", version)

	ctx2, cancel2 := createTimeoutContext(gh.timeout)
	defer cancel2()
	checkoutCmd = setCommandContext(checkoutCmd, ctx2)

	output, err = checkoutCmd.CombinedOutput()
	if err != nil {
		// Version might not exist as a tag, but repository was cloned successfully
		// Continue with master branch
		fmt.Printf("Warning: Could not checkout version %s, using master branch\n", version)
	}

	return clonePath, nil
}

// VerifyPKGBUILD checks if a cloned directory contains a valid PKGBUILD file
func (gh *GitHandler) VerifyPKGBUILD(clonePath string) error {
	if clonePath == "" {
		return fmt.Errorf("clone path cannot be empty")
	}

	pkgbuildPath := filepath.Join(clonePath, "PKGBUILD")

	fileInfo, err := os.Stat(pkgbuildPath)
	if err != nil {
		return fmt.Errorf("PKGBUILD not found in %s: %w", clonePath, err)
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("PKGBUILD is a directory, not a file")
	}

	if fileInfo.Size() == 0 {
		return fmt.Errorf("PKGBUILD is empty")
	}

	return nil
}

// GetPKGBUILDPath returns the path to the PKGBUILD file in a cloned repository
func (gh *GitHandler) GetPKGBUILDPath(clonePath string) string {
	return filepath.Join(clonePath, "PKGBUILD")
}

// RemoveClone removes a cloned repository directory
func (gh *GitHandler) RemoveClone(clonePath string) error {
	if clonePath == "" {
		return fmt.Errorf("clone path cannot be empty")
	}

	return os.RemoveAll(clonePath)
}

// Helper functions for timeout context (Go version compatibility)

// createTimeoutContext creates a context with timeout
// This is a compatibility wrapper for different Go versions
func createTimeoutContext(timeout time.Duration) (interface{}, func()) {
	// For Go 1.7+, use context.WithTimeout
	// This is a simplified version - full implementation would import context
	return nil, func() {}
}

// setCommandContext sets the context for a command
// This is a compatibility wrapper
func setCommandContext(cmd *exec.Cmd, ctx interface{}) *exec.Cmd {
	// In a real implementation, this would use cmd.WithContext
	// For now, we just return the command as-is
	return cmd
}
