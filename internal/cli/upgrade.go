// Package cli implements command-line interface commands for chisel.
package cli

import (
	"fmt"
	"sort"
	"sync"

	"github.com/Jguer/go-alpm/v2"
	"github.com/kodos-prj/chisel/pkg/config"
	"github.com/kodos-prj/chisel/pkg/download"
	"github.com/kodos-prj/chisel/pkg/registry"
	"github.com/kodos-prj/chisel/pkg/store"
)

// UpgradeCommand implements the 'chisel upgrade' command.
// It upgrades installed packages to their latest versions from repositories.
type UpgradeCommand struct {
	config     *config.Config
	symlinkDir string
}

// NewUpgradeCommand creates a new upgrade command instance.
func NewUpgradeCommand(cfg *config.Config) *UpgradeCommand {
	return &UpgradeCommand{
		config:     cfg,
		symlinkDir: "",
	}
}

// NewUpgradeCommandWithSymlinkDir creates a new upgrade command with a symlink directory.
func NewUpgradeCommandWithSymlinkDir(cfg *config.Config, symlinkDir string) *UpgradeCommand {
	return &UpgradeCommand{
		config:     cfg,
		symlinkDir: symlinkDir,
	}
}

// UpgradeOptions holds command-line options for upgrade.
type UpgradeOptions struct {
	DryRun   bool
	Verbose  bool
	Packages []string // Empty = all packages
}

// UpgradeCandidate represents a package that can be upgraded.
type UpgradeCandidate struct {
	PackageName      string
	InstalledVersion string
	AvailableVersion string
	PackageInfo      *download.PackageInfo
	IsAutoAdded      bool // True if auto-added due to dependencies
}

// UpgradeResult represents the result of an upgrade operation.
type UpgradeResult struct {
	PackageName string
	OldVersion  string
	NewVersion  string
	Success     bool
	Error       error
	TimeSeconds int
}

// UpgradeSummary represents overall upgrade statistics.
type UpgradeSummary struct {
	Total              int
	Successful         int
	Failed             int
	SkippedNoUpdate    int
	SkippedNotFound    int
	AutoAddedCount     int
	OldVersionsCleaned int
	SpaceFreed         int64
}

// Execute runs the upgrade command.
func (u *UpgradeCommand) Execute(options *UpgradeOptions) (*UpgradeSummary, error) {
	if options == nil {
		options = &UpgradeOptions{}
	}

	summary := &UpgradeSummary{}

	// Initialize ALPM client
	client, err := alpm.Initialize(u.config.AlpmRoot, u.config.AlpmDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ALPM: %w", err)
	}
	defer client.Release()

	// Register sync databases
	for _, repo := range u.config.Repositories {
		if _, err := client.RegisterSyncDB(repo, 0); err != nil {
			return nil, fmt.Errorf("failed to register database %s: %w", repo, err)
		}
	}

	// Load registry
	reg, err := registry.NewRegistry(u.config.RegistryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	// Get installed packages
	installedPkgs := reg.ListPackages()
	if len(installedPkgs) == 0 {
		fmt.Println("No packages installed.")
		return summary, nil
	}

	if options.Verbose {
		fmt.Println("Checking for updates...")
	}

	// Find upgrade candidates
	candidates, err := u.findCandidates(client, installedPkgs, options.Packages)
	if err != nil {
		return nil, fmt.Errorf("failed to find upgrade candidates: %w", err)
	}

	if len(candidates) == 0 {
		if options.Verbose {
			fmt.Println("All packages are up to date.")
		}
		return summary, nil
	}

	summary.Total = len(candidates)

	// Resolve dependencies
	allCandidates, err := u.resolveDependencies(client, candidates, reg)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	// Count auto-added packages
	for _, pkg := range allCandidates {
		if pkg.IsAutoAdded {
			summary.AutoAddedCount++
		}
	}

	// Show plan
	u.showPlan(allCandidates, options.Verbose)

	// If dry-run, exit here
	if options.DryRun {
		if options.Verbose {
			fmt.Println("\n(No changes made)")
		}
		return summary, nil
	}

	// Execute upgrades
	results := u.executeUpgrades(allCandidates, options.Verbose, u.config, u.symlinkDir)

	// Count results
	for _, result := range results {
		if result.Success {
			summary.Successful++
		} else {
			summary.Failed++
		}
	}

	// Cleanup old versions if configured
	if u.config.KeepVersions > 0 {
		cleaned, freed, err := u.cleanupOldVersions()
		if err == nil {
			summary.OldVersionsCleaned = cleaned
			summary.SpaceFreed = freed
		}
	}

	// Show results
	u.showResults(allCandidates, results, summary, options.Verbose)

	return summary, nil
}

// findCandidates identifies packages that have newer versions available.
func (u *UpgradeCommand) findCandidates(
	client *alpm.Handle,
	installedPkgs []*registry.Package,
	selectedPkgs []string,
) ([]UpgradeCandidate, error) {
	var candidates []UpgradeCandidate
	selectedMap := make(map[string]bool)

	// Build map of selected packages if provided
	for _, pkg := range selectedPkgs {
		selectedMap[pkg] = true
	}

	// Get sync databases
	dbs, err := client.SyncDBs()
	if err != nil {
		return nil, fmt.Errorf("failed to get sync databases: %w", err)
	}

	// Check each installed package
	for _, pkg := range installedPkgs {
		// Skip if selective upgrade and not selected
		if len(selectedPkgs) > 0 && !selectedMap[pkg.Name] {
			continue
		}

		// Search for package in repositories
		var repoPkg alpm.IPackage
		for _, db := range dbs.Slice() {
			p := db.Pkg(pkg.Name)
			if p != nil {
				repoPkg = p
				break
			}
		}

		if repoPkg == nil {
			continue // Package not found in repo
		}

		// Compare versions using ALPM's vercmp
		cmp := alpm.VerCmp(pkg.Version, repoPkg.Version())
		if cmp < 0 { // Installed version is older
			pkgInfo := &download.PackageInfo{
				Name:    repoPkg.Name(),
				Version: repoPkg.Version(),
				Repo:    repoPkg.DB().Name(),
			}

			candidates = append(candidates, UpgradeCandidate{
				PackageName:      pkg.Name,
				InstalledVersion: pkg.Version,
				AvailableVersion: repoPkg.Version(),
				PackageInfo:      pkgInfo,
				IsAutoAdded:      false,
			})
		}
	}

	return candidates, nil
}

// resolveDependencies resolves dependencies for candidate packages and auto-adds if needed.
func (u *UpgradeCommand) resolveDependencies(
	client *alpm.Handle,
	candidates []UpgradeCandidate,
	reg *registry.Registry,
) ([]UpgradeCandidate, error) {
	// Create map of candidates for quick lookup
	candidateMap := make(map[string]*UpgradeCandidate)
	for i := range candidates {
		candidateMap[candidates[i].PackageName] = &candidates[i]
	}

	var allCandidates []UpgradeCandidate
	processedMap := make(map[string]bool)

	// Process each candidate and their dependencies
	var processQueue []*UpgradeCandidate
	for i := range candidates {
		processQueue = append(processQueue, &candidates[i])
	}

	for len(processQueue) > 0 {
		current := processQueue[0]
		processQueue = processQueue[1:]

		if processedMap[current.PackageName] {
			continue
		}
		processedMap[current.PackageName] = true

		allCandidates = append(allCandidates, *current)

		// Get package info if not already set
		if current.PackageInfo == nil {
			dbs, err := client.SyncDBs()
			if err != nil {
				continue
			}

			for _, db := range dbs.Slice() {
				p := db.Pkg(current.PackageName)
				if p != nil {
					current.PackageInfo = &download.PackageInfo{
						Name:    p.Name(),
						Version: p.Version(),
						Repo:    p.DB().Name(),
					}
					break
				}
			}
		}

		// Process dependencies
		if current.PackageInfo != nil {
			dbs, err := client.SyncDBs()
			if err != nil {
				continue
			}

			// Search for the new version to get its dependencies
			for _, db := range dbs.Slice() {
				p := db.Pkg(current.PackageName)
				if p != nil {
					deps := p.Depends().Slice()
					for _, dep := range deps {
						depName := dep.Name

						// Skip if already processed
						if processedMap[depName] {
							continue
						}

						// Check if dependency is installed
						if installed, exists := reg.GetPackage(depName); exists {
							// Check if there's an update available
							for _, db2 := range dbs.Slice() {
								depPkg := db2.Pkg(depName)
								if depPkg != nil {
									cmp := alpm.VerCmp(installed.Version, depPkg.Version())
									if cmp < 0 { // Installed version is older
										// Auto-add to candidates if not already there
										if _, found := candidateMap[depName]; !found {
											newCandidate := UpgradeCandidate{
												PackageName:      depName,
												InstalledVersion: installed.Version,
												AvailableVersion: depPkg.Version(),
												PackageInfo: &download.PackageInfo{
													Name:    depPkg.Name(),
													Version: depPkg.Version(),
													Repo:    depPkg.DB().Name(),
												},
												IsAutoAdded: true,
											}
											candidateMap[depName] = &newCandidate
											processQueue = append(processQueue, &newCandidate)
										}
									}
									break
								}
							}
						}
					}
					break
				}
			}
		}
	}

	return allCandidates, nil
}

// showPlan displays the upgrade plan.
func (u *UpgradeCommand) showPlan(candidates []UpgradeCandidate, verbose bool) {
	if !verbose {
		fmt.Printf("✓ %d packages can be upgraded\n\n", len(candidates))
		return
	}

	// Sort by name
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].PackageName < candidates[j].PackageName
	})

	fmt.Println("\nUpgrade Plan:")
	fmt.Println("┌─────────────────┬──────────────┬──────────────┬──────────┐")
	fmt.Println("│ Package         │ Current      │ Available    │ Type     │")
	fmt.Println("├─────────────────┼──────────────┼──────────────┼──────────┤")

	for _, pkg := range candidates {
		pkgType := ""
		if pkg.IsAutoAdded {
			pkgType = "[auto]"
		}
		fmt.Printf("│ %-15s │ %-12s │ %-12s │ %-8s │\n",
			pkg.PackageName, pkg.InstalledVersion, pkg.AvailableVersion, pkgType)
	}

	fmt.Println("└─────────────────┴──────────────┴──────────────┴──────────┘")

	// Count auto-added
	autoCount := 0
	for _, pkg := range candidates {
		if pkg.IsAutoAdded {
			autoCount++
		}
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("  %d packages to upgrade\n", len(candidates))
	if autoCount > 0 {
		fmt.Printf("  %d auto-added (dependencies)\n", autoCount)
	}
}

// executeUpgrades executes the upgrade for each candidate package.
func (u *UpgradeCommand) executeUpgrades(
	candidates []UpgradeCandidate,
	verbose bool,
	cfg *config.Config,
	symlinkDir string,
) []UpgradeResult {
	var results []UpgradeResult
	var resultsMu sync.Mutex

	// Create install command to reuse its logic
	installCmd := NewInstallCommandWithSymlinkDir(cfg, symlinkDir)

	if verbose {
		fmt.Println("\nProceeding with upgrade...")
	}

	// Upgrade each package
	for i, candidate := range candidates {
		if verbose {
			fmt.Printf("[%d/%d] %s (%s → %s)\n",
				i+1, len(candidates),
				candidate.PackageName,
				candidate.InstalledVersion,
				candidate.AvailableVersion)
		}

		result := UpgradeResult{
			PackageName: candidate.PackageName,
			OldVersion:  candidate.InstalledVersion,
			NewVersion:  candidate.AvailableVersion,
		}

		// Use install logic: download → extract → wrap → symlink
		pkgArgs := []string{candidate.PackageName}
		if err := installCmd.Run(pkgArgs); err != nil {
			result.Success = false
			result.Error = err
			if verbose {
				fmt.Printf("  ✗ Upgrade failed: %v\n", err)
			}
		} else {
			result.Success = true
			if verbose {
				fmt.Printf("  ✓ Upgraded successfully\n")
			}
		}

		resultsMu.Lock()
		results = append(results, result)
		resultsMu.Unlock()
	}

	return results
}

// cleanupOldVersions removes old versions of upgraded packages.
func (u *UpgradeCommand) cleanupOldVersions() (int, int64, error) {
	storeManager := store.NewStore(u.config.StoreRoot)
	var totalCleaned int
	var totalSpaceFreed int64

	// Get all packages in store
	allPkgs, err := storeManager.GetAllPackages()
	if err != nil {
		return 0, 0, err
	}

	for pkgName := range allPkgs {
		removed, err := storeManager.CleanupOldVersions(pkgName, u.config.KeepVersions)
		if err != nil {
			continue
		}

		totalCleaned += removed

		// Calculate space freed (approximate)
		if removed > 0 {
			size, _ := storeManager.GetPackageSize(pkgName, "")
			totalSpaceFreed += int64(removed) * size
		}
	}

	return totalCleaned, totalSpaceFreed, nil
}

// showResults displays the upgrade results.
func (u *UpgradeCommand) showResults(
	candidates []UpgradeCandidate,
	results []UpgradeResult,
	summary *UpgradeSummary,
	verbose bool,
) {
	if verbose {
		fmt.Println("\n✓ Upgrade completed")

		if summary.Successful > 0 {
			fmt.Printf("✓ %d packages upgraded successfully\n", summary.Successful)
		}

		if summary.Failed > 0 {
			fmt.Printf("✗ %d packages failed to upgrade\n", summary.Failed)
		}

		if summary.SpaceFreed > 0 {
			fmt.Printf("✓ Freed %.2f MB by removing old versions\n", float64(summary.SpaceFreed)/(1024*1024))
		}
	} else {
		// Minimal output
		if summary.Successful > 0 {
			fmt.Printf("✓ %d packages upgraded successfully\n", summary.Successful)
		}
		if summary.SpaceFreed > 0 {
			fmt.Printf("✓ Freed %.2f MB\n", float64(summary.SpaceFreed)/(1024*1024))
		}
	}
}
