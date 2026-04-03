// Package cli implements command-line interface commands for chisel.
package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kodos-prj/chisel/pkg/config"
	"github.com/kodos-prj/chisel/pkg/registry"
	"github.com/kodos-prj/chisel/pkg/store"
	"github.com/kodos-prj/chisel/pkg/symlink"
	"github.com/kodos-prj/chisel/pkg/wrapper"
)

// CleanupCommand implements the 'chisel cleanup' command.
// It removes old package versions from store after verifying no symlinks or wrappers point to them.
type CleanupCommand struct {
	config     *config.Config
	symlinkDir string
}

// NewCleanupCommand creates a new cleanup command instance.
func NewCleanupCommand(cfg *config.Config) *CleanupCommand {
	return &CleanupCommand{
		config:     cfg,
		symlinkDir: "",
	}
}

// NewCleanupCommandWithSymlinkDir creates a new cleanup command with a symlink directory.
func NewCleanupCommandWithSymlinkDir(cfg *config.Config, symlinkDir string) *CleanupCommand {
	return &CleanupCommand{
		config:     cfg,
		symlinkDir: symlinkDir,
	}
}

// CleanupOptions holds command-line options for cleanup.
type CleanupOptions struct {
	DryRun       bool // Preview mode: don't make changes
	Verbose      bool // Show detailed output
	Force        bool // Skip confirmation prompt
	KeepVersions int  // Number of recent versions to keep (-1 means use config)
}

// VersionStatus tracks the state of a package version
type VersionStatus struct {
	Version          string
	HasActiveSymlink bool
	HasActiveWrapper bool
	SafeToRemove     bool
	Reason           string // Why it can't be removed
}

// CleanupResult reports cleanup results for a single package
type CleanupResult struct {
	PackageName     string
	VersionsRemoved []string
	VersionsSkipped []string
	SpaceFreed      int64
	Error           error
}

// CleanupSummary reports overall cleanup results
type CleanupSummary struct {
	TotalVersionsRemoved  int
	TotalSpaceFreed       int64
	PackagesProcessed     int
	PackagesSkipped       int
	VersionsSkipped       int
	OrphanWrappersRemoved int
	TotalResults          []CleanupResult
}

// Execute runs the cleanup command.
func (c *CleanupCommand) Execute(opts *CleanupOptions) (*CleanupSummary, error) {
	if opts == nil {
		opts = &CleanupOptions{}
	}

	summary := &CleanupSummary{
		TotalResults: []CleanupResult{},
	}

	// Use config keep_versions if not overridden
	keepVersions := opts.KeepVersions
	if keepVersions == -1 {
		keepVersions = c.config.KeepVersions
		if keepVersions <= 0 {
			keepVersions = 2 // Default to keeping 2 versions
		}
	}

	// Load registry
	reg, err := registry.NewRegistry(c.config.RegistryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	// Create store and symlink managers
	storeManager := store.NewStore(c.config.StoreRoot)
	symlinkMgr := symlink.NewManager(c.config.StoreRoot, c.symlinkDir)

	// Find old versions
	oldVersions, err := c.findOldVersions(keepVersions)
	if err != nil {
		return nil, fmt.Errorf("failed to identify old versions: %w", err)
	}

	if len(oldVersions) == 0 {
		if opts.Verbose {
			fmt.Println("All packages are at their current versions. No cleanup needed.")
		}
		return summary, nil
	}

	// Check status of each old version
	var toRemove []struct {
		pkgName string
		version string
		status  *VersionStatus
		result  *CleanupResult
	}

	for pkgName, versions := range oldVersions {
		result := CleanupResult{
			PackageName:     pkgName,
			VersionsRemoved: []string{},
			VersionsSkipped: []string{},
		}

		for _, version := range versions {
			status, err := c.checkVersionStatus(pkgName, version, reg, symlinkMgr, storeManager)
			if err != nil {
				result.VersionsSkipped = append(result.VersionsSkipped, fmt.Sprintf("%s (error: %v)", version, err))
				summary.VersionsSkipped++
				continue
			}

			if status.SafeToRemove {
				toRemove = append(toRemove, struct {
					pkgName string
					version string
					status  *VersionStatus
					result  *CleanupResult
				}{pkgName, version, status, &result})
			} else {
				result.VersionsSkipped = append(result.VersionsSkipped, fmt.Sprintf("%s (%s)", version, status.Reason))
				summary.VersionsSkipped++
			}
		}

		if len(result.VersionsRemoved) > 0 || len(result.VersionsSkipped) > 0 {
			summary.TotalResults = append(summary.TotalResults, result)
		}
	}

	if len(toRemove) == 0 {
		c.showCleanupResults(summary, opts.Verbose, false)
		return summary, nil
	}

	// Show cleanup plan
	c.showCleanupPlan(toRemove, opts.Verbose)

	// Ask for confirmation if not --force
	if !opts.Force && !opts.DryRun {
		if !c.askForConfirmation(len(toRemove)) {
			fmt.Println("Cleanup cancelled.")
			return summary, nil
		}
	}

	// If dry-run, stop here
	if opts.DryRun {
		return summary, nil
	}

	// Execute cleanup
	for _, item := range toRemove {
		spaceFreed, err := c.removeVersion(item.pkgName, item.version, storeManager)
		if err != nil {
			if opts.Verbose {
				fmt.Printf("  ✗ Failed to remove %s/%s: %v\n", item.pkgName, item.version, err)
			}
			item.result.Error = err
		} else {
			item.result.VersionsRemoved = append(item.result.VersionsRemoved, item.version)
			item.result.SpaceFreed += spaceFreed
			summary.TotalVersionsRemoved++
			summary.TotalSpaceFreed += spaceFreed
			if opts.Verbose {
				fmt.Printf("  ✓ Removed %s/%s (%.2f MB)\n", item.pkgName, item.version, float64(spaceFreed)/(1024*1024))
			}
		}
	}

	// Remove orphaned wrappers
	orphanedCount, err := c.removeOrphanedWrappers(reg)
	if err == nil && orphanedCount > 0 {
		summary.OrphanWrappersRemoved = orphanedCount
		if opts.Verbose {
			fmt.Printf("✓ Removed %d orphaned wrapper(s)\n", orphanedCount)
		}
	}

	// Show results
	c.showCleanupResults(summary, opts.Verbose, len(toRemove) > 0)

	return summary, nil
}

// findOldVersions identifies versions that can be removed (keeps N most recent)
func (c *CleanupCommand) findOldVersions(keepCount int) (map[string][]string, error) {
	storeManager := store.NewStore(c.config.StoreRoot)

	allPackages, err := storeManager.GetAllPackages()
	if err != nil {
		return nil, fmt.Errorf("failed to list packages: %w", err)
	}

	oldVersions := make(map[string][]string)

	for pkgName, versions := range allPackages {
		if len(versions) > keepCount {
			// Sort versions (keep most recent ones)
			sort.Strings(versions)
			// Reverse sort (newest first)
			sort.Sort(sort.Reverse(sort.StringSlice(versions)))
			// Take old ones (beyond keepCount)
			oldVersions[pkgName] = versions[keepCount:]
		}
	}

	return oldVersions, nil
}

// checkVersionStatus checks if a version can be safely removed
func (c *CleanupCommand) checkVersionStatus(pkgName, version string, reg *registry.Registry, symlinkMgr *symlink.Manager, storeManager *store.Store) (*VersionStatus, error) {
	status := &VersionStatus{
		Version:      version,
		SafeToRemove: true,
	}

	// Check if symlinks point to this version
	hasSymlink, err := c.isSymlinkActive(pkgName, version, reg, symlinkMgr)
	if err != nil {
		return status, err
	}
	if hasSymlink {
		status.HasActiveSymlink = true
		status.SafeToRemove = false
		status.Reason = "active symlink points to this version"
		return status, nil
	}

	// Check if wrapper references this version
	hasWrapper, err := c.isWrapperActive(pkgName, version)
	if err != nil {
		return status, err
	}
	if hasWrapper {
		status.HasActiveWrapper = true
		status.SafeToRemove = false
		status.Reason = "wrapper script references this version"
		return status, nil
	}

	return status, nil
}

// isSymlinkActive checks if any symlinks point to a specific version
func (c *CleanupCommand) isSymlinkActive(pkgName, version string, reg *registry.Registry, symlinkMgr *symlink.Manager) (bool, error) {
	pkg, ok := reg.GetPackage(pkgName)
	if !ok {
		return false, nil // Package not installed, no active symlinks
	}

	// Check each executable for active symlinks
	for _, exe := range pkg.Executables {
		symlinkPath := symlinkMgr.GetSymlinkPath(exe)

		// Check if symlink exists
		stat, err := os.Lstat(symlinkPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue // Symlink doesn't exist, not active
			}
			continue // Skip on other errors
		}

		// Check if it's a symlink
		if stat.Mode()&os.ModeSymlink == 0 {
			continue // Not a symlink, skip
		}

		// Read symlink target
		target, err := os.Readlink(symlinkPath)
		if err != nil {
			continue // Skip on error
		}

		// Check if target contains this version path
		expectedPath := filepath.Join(c.config.StoreRoot, pkgName, version)
		if strings.Contains(target, expectedPath) {
			return true, nil // Found active symlink to this version
		}
	}

	return false, nil
}

// isWrapperActive checks if wrapper script references a specific version
func (c *CleanupCommand) isWrapperActive(pkgName, version string) (bool, error) {
	wrapperGen := wrapper.NewGenerator(c.config.StoreRoot, filepath.Join(c.config.BaseDir, "wrappers"), c.symlinkDir)
	wrapperPath := wrapperGen.GetWrapperPath(pkgName)

	// Check if wrapper exists
	stat, err := os.Stat(wrapperPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // No wrapper, not active
		}
		return false, err
	}

	// Must be a regular file
	if stat.IsDir() {
		return false, nil
	}

	// Read wrapper content
	content, err := os.ReadFile(wrapperPath)
	if err != nil {
		return false, err
	}

	// Check if version appears in wrapper (in LD_LIBRARY_PATH)
	versionPath := filepath.Join(c.config.StoreRoot, pkgName, version)
	if strings.Contains(string(content), versionPath) {
		return true, nil // Wrapper references this version
	}

	return false, nil
}

// removeVersion removes a package version from store
func (c *CleanupCommand) removeVersion(pkgName, version string, storeManager *store.Store) (int64, error) {
	// Get size before deletion
	size, err := storeManager.GetPackageSize(pkgName, version)
	if err != nil {
		size = 0
	}

	// Remove package
	err = storeManager.RemovePackage(pkgName, version)
	if err != nil {
		return 0, fmt.Errorf("failed to remove %s/%s: %w", pkgName, version, err)
	}

	return size, nil
}

// removeOrphanedWrappers removes wrappers for packages no longer in registry
func (c *CleanupCommand) removeOrphanedWrappers(reg *registry.Registry) (int, error) {
	wrapperDir := filepath.Join(c.config.BaseDir, "wrappers")

	// Check if wrapper directory exists
	entries, err := os.ReadDir(wrapperDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	var removedCount int

	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip directories
		}

		wrapperName := entry.Name()

		// Check if package exists in registry
		_, ok := reg.GetPackage(wrapperName)
		if !ok {
			// Orphaned wrapper - try to remove it
			wrapperPath := filepath.Join(wrapperDir, wrapperName)
			if err := os.Remove(wrapperPath); err == nil {
				removedCount++
			}
		}
	}

	return removedCount, nil
}

// showCleanupPlan displays what will be cleaned
func (c *CleanupCommand) showCleanupPlan(toRemove []struct {
	pkgName string
	version string
	status  *VersionStatus
	result  *CleanupResult
}, verbose bool) {
	if !verbose {
		fmt.Printf("✓ Found %d versions to remove\n\n", len(toRemove))
		return
	}

	fmt.Println("\nCleanup Plan:")
	fmt.Println("┌─────────────────┬──────────────┐")
	fmt.Println("│ Package         │ Version      │")
	fmt.Println("├─────────────────┼──────────────┤")

	for _, item := range toRemove {
		fmt.Printf("│ %-15s │ %-12s │\n", item.pkgName, item.version)
	}

	fmt.Println("└─────────────────┴──────────────┘")
}

// showCleanupResults displays summary after cleanup
func (c *CleanupCommand) showCleanupResults(summary *CleanupSummary, verbose bool, executed bool) {
	if verbose || executed {
		fmt.Println()
		if summary.TotalVersionsRemoved > 0 {
			fmt.Printf("✓ Cleanup Summary:\n")
			fmt.Printf("  Versions removed:     %d\n", summary.TotalVersionsRemoved)
			fmt.Printf("  Space freed:          %.2f GB\n", float64(summary.TotalSpaceFreed)/(1024*1024*1024))
			if summary.VersionsSkipped > 0 {
				fmt.Printf("  Versions skipped:     %d (still in use)\n", summary.VersionsSkipped)
			}
			if summary.OrphanWrappersRemoved > 0 {
				fmt.Printf("  Orphaned wrappers:    %d\n", summary.OrphanWrappersRemoved)
			}
		} else {
			fmt.Println("✓ No versions needed cleanup")
		}
	} else {
		if summary.TotalVersionsRemoved == 0 && summary.VersionsSkipped == 0 {
			fmt.Println("✓ All packages are at their current versions")
		} else if summary.TotalVersionsRemoved > 0 {
			fmt.Printf("✓ Cleanup complete: %d versions removed, %.2f GB freed\n",
				summary.TotalVersionsRemoved,
				float64(summary.TotalSpaceFreed)/(1024*1024*1024))
		}
	}
}

// askForConfirmation prompts user before cleanup
func (c *CleanupCommand) askForConfirmation(versionCount int) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Proceed with cleanup of %d versions? (y/n) ", versionCount)

	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
