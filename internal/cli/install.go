package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kodos-prj/chisel/pkg/alpm"
	"github.com/kodos-prj/chisel/pkg/aur"
	"github.com/kodos-prj/chisel/pkg/build"
	"github.com/kodos-prj/chisel/pkg/config"
	"github.com/kodos-prj/chisel/pkg/download"
	"github.com/kodos-prj/chisel/pkg/extract"
	"github.com/kodos-prj/chisel/pkg/registry"
	"github.com/kodos-prj/chisel/pkg/store"
	"github.com/kodos-prj/chisel/pkg/wrapper"
)

// InstallCommand handles installing packages from official repos or AUR.
type InstallCommand struct {
	config     *config.Config
	symlinkDir string
	aurRPC     *aur.RPCClient
	buildMgr   *build.BuildManager
}

// NewInstallCommand creates a new install command.
func NewInstallCommand(cfg *config.Config) *InstallCommand {
	buildMgr, _ := build.NewBuildManager("/kod/build-cache/", "/kod/build-logs/")
	return &InstallCommand{
		config:     cfg,
		symlinkDir: "",
		aurRPC:     aur.NewRPCClient(),
		buildMgr:   buildMgr,
	}
}

// NewInstallCommandWithSymlinkDir creates a new install command with a symlink directory.
func NewInstallCommandWithSymlinkDir(cfg *config.Config, symlinkDir string) *InstallCommand {
	buildMgr, _ := build.NewBuildManager("/kod/build-cache/", "/kod/build-logs/")
	return &InstallCommand{
		config:     cfg,
		symlinkDir: symlinkDir,
		aurRPC:     aur.NewRPCClient(),
		buildMgr:   buildMgr,
	}
}

// InstallOptions holds command-line options for install.
type InstallOptions struct {
	NoDeps    bool
	NoExtract bool
	NoSymlink bool
	Force     bool
	Source    string // "", "aur", or "official"
}

// Run executes the install command.
// Usage: chisel install [options] <package> [package2] ...
//
//	--source=aur        Install from AUR only (skip official repositories)
//	--source=official   Install from official repositories only (skip AUR)
//	--no-deps           Skip dependency resolution
//	--no-extract        Skip extraction (assume already in store)
//	--no-symlink        Skip symlink creation
//	--force             Force overwrite of existing symlinks
//
// Source Constraint Behavior:
//   - Root packages: Respect --source= constraint
//   - Dependencies: Always auto-detect from both sources
//   - Conflicts: Using both --source=aur and --source=official is an error
//   - Default: Without --source=, official repos checked first, AUR as fallback
//
// Examples:
//
//	chisel install yay                     # Auto-detect (official first, then AUR)
//	chisel install --source=aur yay        # AUR only
//	chisel install --source=official firefox  # Official only
func (i *InstallCommand) Run(args []string) error {
	// Parse options and package names
	opts := InstallOptions{Source: ""}
	var pkgNames []string

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--source="):
			// Parse --source= flag
			source := strings.TrimPrefix(arg, "--source=")
			if source != "aur" && source != "official" {
				return fmt.Errorf("invalid source: %s (must be 'aur' or 'official')", source)
			}
			if opts.Source != "" {
				return fmt.Errorf("cannot specify multiple --source flags")
			}
			opts.Source = source
		case arg == "--no-deps":
			opts.NoDeps = true
		case arg == "--no-extract":
			opts.NoExtract = true
		case arg == "--no-symlink":
			opts.NoSymlink = true
		case arg == "--force":
			opts.Force = true
		default:
			pkgNames = append(pkgNames, arg)
		}
	}

	if len(pkgNames) == 0 {
		return fmt.Errorf("package name required")
	}

	// Initialize ALPM client
	client, err := alpm.NewClient(i.config.AlpmRoot, i.config.AlpmDBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize ALPM: %w", err)
	}
	defer client.Close()

	// Register sync databases
	if err := client.RegisterAllSyncDBs(i.config.Repositories); err != nil {
		return fmt.Errorf("failed to register databases: %w", err)
	}

	// Resolve package dependencies using MixedResolver (official + AUR)
	if opts.Source != "" {
		fmt.Printf("Resolving package dependencies (%s only)...\n", opts.Source)
	} else {
		fmt.Println("Resolving package dependencies...")
	}
	resolver := build.NewMixedResolver(client, i.config.AlpmDBPath)
	defer resolver.Close()

	toInstall, err := i.resolveMixedDependencies(resolver, pkgNames, opts.NoDeps, opts.Source)
	if err != nil {
		return fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	if len(toInstall) == 0 {
		return fmt.Errorf("no packages to install")
	}

	fmt.Printf("Will install %d package(s)\n", len(toInstall))
	for _, pkg := range toInstall {
		fmt.Printf("  - %s/%s\n", pkg.Name, pkg.Version)
	}

	// Map to track extracted files per package (for registry and symlink creation)
	// Structure: pkgName -> version -> {allFiles: []string, executables: []string}
	type PackageFiles struct {
		AllExtractedFiles []extract.ExtractedFile
		AllFiles          []string
		Executables       []string
	}
	extractedFilesMap := make(map[string]map[string]PackageFiles) // pkgName -> version -> PackageFiles

	// Download packages
	if !opts.NoExtract {
		fmt.Println("\nDownloading packages...")
		downloader := download.NewDownloader(
			i.config.MirrorURL,
			i.config.CachePath,
			i.config.Architecture,
			i.config.MaxConcurrentDownloads,
			0,
		)

		results, err := downloader.DownloadPackages(toInstall)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Download warnings: %v\n", err)
		}

		if len(results) == 0 {
			return fmt.Errorf("no packages were successfully downloaded")
		}

		fmt.Printf("✓ Downloaded %d/%d package(s)\n", len(results), len(toInstall))

		// Extract packages
		fmt.Println("\nExtracting packages...")
		storeManager := store.NewStore(i.config.StoreRoot)

		for _, pkgInfo := range toInstall {
			// Construct cache file path
			fileName := fmt.Sprintf("%s-%s-x86_64.pkg.tar.zst", pkgInfo.Name, pkgInfo.Version)
			cachePath := filepath.Join(i.config.CachePath, fileName)

			// Check if file exists
			if _, err := os.Stat(cachePath); err != nil {
				fmt.Fprintf(os.Stderr, "✗ Cache file not found: %s\n", cachePath)
				continue
			}

			// Extract package
			extractedFileObjs, err := storeManager.ExtractPackage(cachePath, pkgInfo.Name, pkgInfo.Version)
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Failed to extract %s/%s: %v\n", pkgInfo.Name, pkgInfo.Version, err)
				continue
			}

			fmt.Printf("  ✓ Extracted %d files\n", len(extractedFileObjs))

			// Process extracted files
			if _, exists := extractedFilesMap[pkgInfo.Name]; !exists {
				extractedFilesMap[pkgInfo.Name] = make(map[string]PackageFiles)
			}

			var allFiles []string
			var executables []string

			for _, file := range extractedFileObjs {
				// Collect all files (except directories)
				if !file.IsDirectory {
					allFiles = append(allFiles, file.Path)

					// Also track executables in /usr/bin and /usr/sbin
					if strings.HasPrefix(file.Path, "usr/bin/") || strings.HasPrefix(file.Path, "usr/sbin/") {
						executables = append(executables, file.Path)
					}
				}
			}

			extractedFilesMap[pkgInfo.Name][pkgInfo.Version] = PackageFiles{
				AllExtractedFiles: extractedFileObjs,
				AllFiles:          allFiles,
				Executables:       executables,
			}

			// Set as current version
			_ = storeManager.SetLatestVersion(pkgInfo.Name, pkgInfo.Version)
		}
	}

	// Create symlinks
	symlinkDir := i.symlinkDir
	if symlinkDir == "" {
		symlinkDir = i.config.SymlinkRoot
	}

	if !opts.NoSymlink && symlinkDir != "" {
		fmt.Println("\nCreating symlinks...")

		symlinkCount := 0

		// Create symlink hierarchy pointing to storage and wrappers
		for _, pkg := range toInstall {
			pkgFileInfo, ok := extractedFilesMap[pkg.Name][pkg.Version]
			if !ok || len(pkgFileInfo.AllFiles) == 0 {
				fmt.Fprintf(os.Stderr, "  ! Skipping symlinks for %s (not extracted)\n", pkg.Name)
				continue
			}

			// Build a map of extracted symlinks with their targets
			extractedSymlinksMap := make(map[string]string) // path -> target
			for _, extractedFile := range pkgFileInfo.AllExtractedFiles {
				if extractedFile.IsSymlink {
					extractedSymlinksMap[extractedFile.Path] = extractedFile.LinkTarget
				}
			}

			// Create symlinks for all extracted files
			for _, filePath := range pkgFileInfo.AllFiles {
				// Skip Arch package metadata files
				fileName := filepath.Base(filePath)
				if fileName == ".PKGINFO" || fileName == ".BUILDINFO" || fileName == ".MTREE" || fileName == ".INSTALL" {
					continue
				}

				symlinkPath := filepath.Join(symlinkDir, filePath)

				// Create parent directories if needed
				symlinkParentDir := filepath.Dir(symlinkPath)
				if err := os.MkdirAll(symlinkParentDir, 0755); err != nil {
					fmt.Fprintf(os.Stderr, "  ! Warning: Failed to create directory %s: %v\n", symlinkParentDir, err)
					continue
				}

				// Determine target path
				var targetPath string

				// Check if this file was originally extracted as a symlink
				if originalTarget, isSymlink := extractedSymlinksMap[filePath]; isSymlink {
					// This is a symlink from the package
					// Point it to the storage location: /stor/pkg/version/path
					symlinkTargetDir := filepath.Join(i.config.StoreRoot, pkg.Name, pkg.Version, filepath.Dir(filePath))
					targetPath = filepath.Join(symlinkTargetDir, originalTarget)
				} else if strings.HasPrefix(filePath, "usr/bin/") || strings.HasPrefix(filePath, "usr/sbin/") {
					// Regular executable: point to wrapper
					targetPath = filepath.Join(i.config.WrapperDir, fileName)
				} else {
					// Regular file: point to storage
					targetPath = filepath.Join(i.config.StoreRoot, pkg.Name, pkg.Version, filePath)
				}

				// Check if symlink already exists
				if !opts.Force {
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
							fmt.Fprintf(os.Stderr, "  ! Warning: Symlink exists at %s (pointing elsewhere), skipping\n", symlinkPath)
							continue
						}
						// Regular file exists, skip with warning
						fmt.Fprintf(os.Stderr, "  ! Warning: Regular file exists at %s, skipping\n", symlinkPath)
						continue
					}
				} else {
					// Force mode: remove existing symlink
					_ = os.Remove(symlinkPath)
				}

				// Create symlink
				if err := os.Symlink(targetPath, symlinkPath); err != nil {
					fmt.Fprintf(os.Stderr, "  ! Warning: Failed to create symlink %s: %v\n", filePath, err)
				} else {
					symlinkCount++
				}
			}
		}

		if symlinkCount > 0 {
			fmt.Printf("✓ Created %d symlink(s)\n", symlinkCount)
		} else {
			fmt.Println("! No symlinks were created")
		}
	}

	// Generate wrapper scripts
	fmt.Println("\nGenerating wrapper scripts...")
	wrapperGen := wrapper.NewGenerator(i.config.StoreRoot, i.config.WrapperDir, i.config.SymlinkRoot)

	// Build a map of package versions for dependency resolution
	depVersionMap := make(map[string]string)
	for _, pkg := range toInstall {
		depVersionMap[pkg.Name] = pkg.Version
	}

	for _, pkg := range toInstall {
		libDirs, err := wrapperGen.DiscoverLibraries(pkg.Name, pkg.Version)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ! Warning: Failed to discover libraries for %s: %v\n", pkg.Name, err)
			continue
		}

		// Convert map to slice for generating wrappers
		var libDirsList []string
		for dir := range libDirs {
			libDirsList = append(libDirsList, dir)
		}

		// Get dependencies for this package (empty for now with MixedResolver)
		var dependencies []string
		// TODO: Track dependencies from MixedResolver in future optimization

		// Generate wrappers only for standard executable locations (usr/bin, usr/sbin)
		standardExecDirs := []string{"usr/bin", "usr/sbin"}
		for _, dir := range standardExecDirs {
			pkgExecDir := filepath.Join(i.config.StoreRoot, pkg.Name, pkg.Version, dir)
			if _, err := os.Stat(pkgExecDir); err != nil {
				continue
			}

			// Get list of executables
			entries, err := os.ReadDir(pkgExecDir)
			if err != nil {
				continue
			}

			// Generate wrapper for each executable
			for _, entry := range entries {
				if !entry.IsDir() {
					cmdName := entry.Name()
					if err := wrapperGen.GenerateWrapperWithDeps(cmdName, pkg.Name, pkg.Version, libDirsList, dependencies, depVersionMap); err != nil {
						fmt.Fprintf(os.Stderr, "  ! Warning: Failed to generate wrapper for %s: %v\n", cmdName, err)
					}
				}
			}
		}
	}

	// Update registry
	fmt.Println("\nUpdating registry...")
	reg, err := registry.NewRegistry(i.config.RegistryPath)
	if err != nil {
		return fmt.Errorf("failed to open registry: %w", err)
	}

	for _, pkg := range toInstall {
		// Get file information if available
		pkgFileInfo, ok := extractedFilesMap[pkg.Name][pkg.Version]
		var files []string
		var executables []string
		if ok {
			files = pkgFileInfo.AllFiles
			executables = pkgFileInfo.Executables
		}

		// Get dependencies for this package (empty for now with MixedResolver)
		var dependencies []string
		// TODO: Track dependencies from MixedResolver in future optimization

		// Determine source: official repo or AUR
		source := "official"
		if pkg.Repo == "aur" {
			source = "aur"
		}

		regPkg := &registry.Package{
			Name:         pkg.Name,
			Version:      pkg.Version,
			Source:       source,
			Repository:   pkg.Repo,
			Files:        files,
			Executables:  executables,
			Dependencies: dependencies,
			InstallDate:  time.Now().Format(time.RFC3339),
			UpdateDate:   time.Now().Format(time.RFC3339),
		}

		if err := reg.AddPackage(regPkg); err != nil {
			fmt.Fprintf(os.Stderr, "  ! Warning: Failed to add %s to registry: %v\n", pkg.Name, err)
			continue
		}
	}

	if err := reg.Save(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	fmt.Println("\n✓ Installation complete!")
	return nil
}

// resolveDependencies resolves package dependencies.
// If skipDeps is true, only returns the requested packages.
// Otherwise, uses ALPM's ResolveDependencies() to get the full dependency tree.
func (i *InstallCommand) resolveDependencies(client *alpm.ALPMClient, pkgNames []string, skipDeps bool) ([]download.PackageInfo, error) {
	var toInstall []download.PackageInfo
	visited := make(map[string]bool)

	for _, pkgName := range pkgNames {
		var pkgDeps []string
		var err error

		if skipDeps {
			// Just the requested package
			pkgDeps = []string{pkgName}
		} else {
			// Get full dependency tree from ALPM (in correct order)
			pkgDeps, err = client.ResolveDependencies(pkgName)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve dependencies for %s: %w", pkgName, err)
			}
		}

		// Add resolved packages to install list (skip if already visited)
		for _, depName := range pkgDeps {
			if visited[depName] {
				continue // Skip if we've already added it
			}

			// Check if package is already installed (in registry or store)
			if i.isPackageInstalled(depName) {
				fmt.Printf("  ℹ %s already installed, skipping\n", depName)
				visited[depName] = true
				continue
			}

			visited[depName] = true

			// Get package info
			pkgInfo, err := client.GetPackageInfo(depName)
			if err != nil {
				return nil, fmt.Errorf("package not found: %s", depName)
			}

			toInstall = append(toInstall, download.PackageInfo{
				Name:    pkgInfo.Name,
				Version: pkgInfo.Version,
				Repo:    pkgInfo.Repository,
			})
		}
	}

	return toInstall, nil
}

// isPackageInstalled checks if a package is already installed in the store/registry
func (i *InstallCommand) isPackageInstalled(pkgName string) bool {
	// Try to open registry
	reg, err := registry.NewRegistry(i.config.RegistryPath)
	if err != nil {
		return false // If registry doesn't exist, package isn't installed
	}

	// Check if package exists in registry
	_, exists := reg.GetPackage(pkgName)
	return exists
}

// resolveDependenciesWithMap resolves package dependencies and returns a map of dependencies per package.
// Returns (toInstall, depMap, error) where depMap[pkgName] = []dependentPkgNames
func (i *InstallCommand) resolveDependenciesWithMap(client *alpm.ALPMClient, pkgNames []string, skipDeps bool) ([]download.PackageInfo, map[string][]string, error) {
	var toInstall []download.PackageInfo
	visited := make(map[string]bool)
	depMap := make(map[string][]string) // package -> list of packages that depend on it

	for _, pkgName := range pkgNames {
		var pkgDeps []string
		var err error

		if skipDeps {
			// Just the requested package
			pkgDeps = []string{pkgName}
		} else {
			// Get full dependency tree from ALPM (in correct order)
			pkgDeps, err = client.ResolveDependencies(pkgName)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to resolve dependencies for %s: %w", pkgName, err)
			}
		}

		// Track which packages depend on which
		// The last item in pkgDeps is the requested package, the others are its dependencies
		// We want: depMap[packageName] = its direct dependencies
		if len(pkgDeps) > 1 && !skipDeps {
			requestedPkg := pkgDeps[len(pkgDeps)-1]
			// Store all other packages as dependencies of the requested package
			depMap[requestedPkg] = append(depMap[requestedPkg], pkgDeps[:len(pkgDeps)-1]...)
		}

		// Add resolved packages to install list (skip if already visited)
		for _, depName := range pkgDeps {
			if visited[depName] {
				continue // Skip if we've already added it
			}

			// Check if package is already installed (in registry or store)
			if i.isPackageInstalled(depName) {
				fmt.Printf("  ℹ %s already installed, skipping\n", depName)
				visited[depName] = true
				continue
			}

			visited[depName] = true

			// Get package info
			pkgInfo, err := client.GetPackageInfo(depName)
			if err != nil {
				return nil, nil, fmt.Errorf("package not found: %s", depName)
			}

			toInstall = append(toInstall, download.PackageInfo{
				Name:    pkgInfo.Name,
				Version: pkgInfo.Version,
				Repo:    pkgInfo.Repository,
			})
		}
	}

	return toInstall, depMap, nil
}

// resolveMixedDependencies resolves dependencies using MixedResolver (official + AUR)
// Returns packages to install in proper order, handling both official and AUR packages
func (i *InstallCommand) resolveMixedDependencies(resolver *build.MixedResolver, pkgNames []string, skipDeps bool, sourceConstraint string) ([]download.PackageInfo, error) {
	var toInstall []download.PackageInfo
	visited := make(map[string]bool)

	for idx, pkgName := range pkgNames {
		var pkgSources []build.PackageSource
		var err error

		// Only apply source constraint to root packages (first package in each call)
		isRootPackage := idx == 0

		if skipDeps {
			// ResolveDependencies will return just the package itself if no deps
			pkgSources, err = resolver.ResolveDependencies(pkgName, isRootPackage, sourceConstraint)
		} else {
			// Get full dependency tree from MixedResolver (official + AUR, recursive)
			pkgSources, err = resolver.ResolveDependencies(pkgName, isRootPackage, sourceConstraint)
		}

		if err != nil {
			// Provide helpful error message if source constraint was used
			if sourceConstraint == "aur" {
				return nil, fmt.Errorf("package '%s' not found in AUR\nHint: Use 'chisel install %s' to search both sources", pkgName, pkgName)
			} else if sourceConstraint == "official" {
				return nil, fmt.Errorf("package '%s' not found in official repositories\nHint: Use 'chisel install %s' to search both sources", pkgName, pkgName)
			}
			return nil, fmt.Errorf("failed to resolve %s: %w", pkgName, err)
		}

		if len(pkgSources) == 0 {
			return nil, fmt.Errorf("no packages resolved for %s", pkgName)
		}

		// Add resolved packages to install list
		for pkgIdx, pkgSource := range pkgSources {
			if visited[pkgSource.Name] {
				continue // Skip if already added
			}

			// Check if package is already installed
			if i.isPackageInstalled(pkgSource.Name) {
				fmt.Printf("  ℹ %s already installed, skipping\n", pkgSource.Name)
				visited[pkgSource.Name] = true
				continue
			}

			visited[pkgSource.Name] = true

			// Determine how to handle the package based on its source
			if pkgSource.Source == "official" {
				// Official repository package - will be downloaded
				toInstall = append(toInstall, download.PackageInfo{
					Name:    pkgSource.Name,
					Version: pkgSource.Version,
					Repo:    pkgSource.Repo,
				})
				// Show constraint indicator only for root package
				if isRootPackage && pkgIdx == 0 && sourceConstraint != "" {
					fmt.Printf("  + %s/%s (official - forced by --source=%s)\n", pkgSource.Name, pkgSource.Version, sourceConstraint)
				} else {
					fmt.Printf("  + %s/%s (official)\n", pkgSource.Name, pkgSource.Version)
				}
			} else if pkgSource.Source == "aur" {
				// AUR package - needs to be built
				// For AUR packages, we still add them to the install list
				// but mark them as AUR so we know to build them
				toInstall = append(toInstall, download.PackageInfo{
					Name:    pkgSource.Name,
					Version: pkgSource.Version,
					Repo:    "aur", // Special marker for AUR packages
				})
				// Show constraint indicator only for root package
				if isRootPackage && pkgIdx == 0 && sourceConstraint != "" {
					fmt.Printf("  + %s/%s (AUR - will be built - forced by --source=%s)\n", pkgSource.Name, pkgSource.Version, sourceConstraint)
				} else {
					fmt.Printf("  + %s/%s (AUR - will be built)\n", pkgSource.Name, pkgSource.Version)
				}
			}
		}
	}

	return toInstall, nil
}
