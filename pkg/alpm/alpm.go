// Package alpm provides a wrapper around the ALPM (Arch Linux Package Management) library.
// It handles package database operations, dependency resolution, and package transactions.
package alpm

import (
	"fmt"

	"github.com/Jguer/go-alpm/v2"
)

// Client wraps the ALPM handle and provides high-level package management operations.
type Client struct {
	handle *alpm.Handle
	root   string
	dbPath string
}

// NewClient creates a new ALPM client with the specified root and database path.
// root is the installation root (e.g., "/kod")
// dbPath is the database directory (e.g., "/kod/db")
func NewClient(root, dbPath string) (*Client, error) {
	handle, err := alpm.Initialize(root, dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ALPM: %w", err)
	}

	return &Client{
		handle: handle,
		root:   root,
		dbPath: dbPath,
	}, nil
}

// Close releases the ALPM handle and associated resources.
func (c *Client) Close() error {
	if c.handle != nil {
		return c.handle.Release()
	}
	return nil
}

// RegisterSyncDB registers a sync database with the ALPM handle.
// repo is the repository name (e.g., "core", "extra")
func (c *Client) RegisterSyncDB(repo string) error {
	_, err := c.handle.RegisterSyncDB(repo, 0)
	if err != nil {
		return fmt.Errorf("failed to register sync database %s: %w", repo, err)
	}
	return nil
}

// RegisterAllSyncDBs registers multiple sync databases.
func (c *Client) RegisterAllSyncDBs(repos []string) error {
	for _, repo := range repos {
		if err := c.RegisterSyncDB(repo); err != nil {
			return err
		}
	}
	return nil
}

// SearchPackage searches for a package by name in all sync databases.
// Returns the package if found, or an error if not found.
func (c *Client) SearchPackage(name string) (alpm.IPackage, error) {
	dbs, err := c.handle.SyncDBs()
	if err != nil {
		return nil, fmt.Errorf("failed to get sync databases: %w", err)
	}

	for _, db := range dbs.Slice() {
		pkg := db.Pkg(name)
		if pkg != nil {
			return pkg, nil
		}
	}

	return nil, fmt.Errorf("package %s not found in any sync database", name)
}

// SearchPackages searches for packages matching a pattern in all sync databases.
// pattern can be a partial package name (e.g., "vim" matches "vim", "gvim", etc.)
func (c *Client) SearchPackages(pattern string) ([]alpm.IPackage, error) {
	dbs, err := c.handle.SyncDBs()
	if err != nil {
		return nil, fmt.Errorf("failed to get sync databases: %w", err)
	}

	var results []alpm.IPackage
	for _, db := range dbs.Slice() {
		pkgs := db.Search([]string{pattern})
		if pkgs != nil {
			for _, pkg := range pkgs.Slice() {
				results = append(results, pkg)
			}
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no packages found matching %s", pattern)
	}

	return results, nil
}

// GetPackageInfo returns detailed information about a package.
type PackageInfo struct {
	Name         string
	Version      string
	Description  string
	Architecture string
	URL          string
	Licenses     []string
	Groups       []string
	Provides     []string
	DependsOn    []string
	OptDepends   []string
	Conflicts    []string
	Replaces     []string
	Size         int64
	DownloadSize int64
	Repository   string
	PackageBase  string
	Maintainer   string
}

// GetPackageInfo retrieves detailed information about a package.
func (c *Client) GetPackageInfo(name string) (*PackageInfo, error) {
	pkg, err := c.SearchPackage(name)
	if err != nil {
		return nil, err
	}

	info := &PackageInfo{
		Name:         pkg.Name(),
		Version:      pkg.Version(),
		Description:  pkg.Description(),
		Architecture: pkg.Architecture(),
		URL:          pkg.URL(),
		Size:         pkg.ISize(),
		DownloadSize: pkg.Size(),
		Repository:   pkg.DB().Name(),
		PackageBase:  pkg.Base(),
	}

	// Extract licenses
	for _, license := range pkg.Licenses().Slice() {
		info.Licenses = append(info.Licenses, license)
	}

	// Extract groups
	for _, group := range pkg.Groups().Slice() {
		info.Groups = append(info.Groups, group)
	}

	// Extract provides
	for _, dep := range pkg.Provides().Slice() {
		info.Provides = append(info.Provides, dep.String())
	}

	// Extract dependencies
	for _, dep := range pkg.Depends().Slice() {
		info.DependsOn = append(info.DependsOn, dep.String())
	}

	// Extract optional dependencies
	for _, dep := range pkg.OptionalDepends().Slice() {
		info.OptDepends = append(info.OptDepends, dep.String())
	}

	// Extract conflicts
	for _, dep := range pkg.Conflicts().Slice() {
		info.Conflicts = append(info.Conflicts, dep.String())
	}

	// Extract replaces
	for _, dep := range pkg.Replaces().Slice() {
		info.Replaces = append(info.Replaces, dep.String())
	}

	return info, nil
}

// ResolveDependencies resolves all dependencies for a package.
// Returns a list of package names that need to be installed, in dependency order.
func (c *Client) ResolveDependencies(packageName string) ([]string, error) {
	pkg, err := c.SearchPackage(packageName)
	if err != nil {
		return nil, err
	}

	// Use a map to track packages we've already seen
	seen := make(map[string]bool)
	var result []string

	// Recursive function to resolve dependencies
	var resolveDeps func(p alpm.IPackage) error
	resolveDeps = func(p alpm.IPackage) error {
		// Skip if we've already processed this package
		if seen[p.Name()] {
			return nil
		}
		seen[p.Name()] = true

		// Process dependencies first
		for _, dep := range p.Depends().Slice() {
			depName := dep.Name
			depPkg, err := c.SearchPackage(depName)
			if err != nil {
				// Try to find package that provides this dependency
				depPkg, err = c.findProviding(depName)
				if err != nil {
					return fmt.Errorf("failed to resolve dependency %s: %w", depName, err)
				}
			}

			if err := resolveDeps(depPkg); err != nil {
				return err
			}
		}

		// Add this package after its dependencies
		result = append(result, p.Name())
		return nil
	}

	if err := resolveDeps(pkg); err != nil {
		return nil, err
	}

	return result, nil
}

// findProviding finds a package that provides the given dependency.
func (c *Client) findProviding(name string) (alpm.IPackage, error) {
	dbs, err := c.handle.SyncDBs()
	if err != nil {
		return nil, fmt.Errorf("failed to get sync databases: %w", err)
	}

	for _, db := range dbs.Slice() {
		pkgs := db.PkgCache()
		for _, pkg := range pkgs.Slice() {
			for _, prov := range pkg.Provides().Slice() {
				if prov.Name == name {
					return pkg, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no package provides %s", name)
}

// ListSyncDBs returns a list of registered sync database names.
func (c *Client) ListSyncDBs() ([]string, error) {
	dbs, err := c.handle.SyncDBs()
	if err != nil {
		return nil, fmt.Errorf("failed to get sync databases: %w", err)
	}

	var names []string
	for _, db := range dbs.Slice() {
		names = append(names, db.Name())
	}

	return names, nil
}

// GetLocalPackages returns a list of all locally installed packages.
func (c *Client) GetLocalPackages() ([]alpm.IPackage, error) {
	localDB, err := c.handle.LocalDB()
	if err != nil {
		return nil, fmt.Errorf("failed to get local database: %w", err)
	}

	pkgs := localDB.PkgCache()
	var results []alpm.IPackage
	for _, pkg := range pkgs.Slice() {
		results = append(results, pkg)
	}

	return results, nil
}

// IsPackageInstalled checks if a package is installed locally.
func (c *Client) IsPackageInstalled(name string) (bool, error) {
	localDB, err := c.handle.LocalDB()
	if err != nil {
		return false, fmt.Errorf("failed to get local database: %w", err)
	}

	pkg := localDB.Pkg(name)
	return pkg != nil, nil
}

// GetDownloadURL returns the download URL for a package.
// This constructs the URL based on the package's repository and name.
func (c *Client) GetDownloadURL(pkg alpm.IPackage, mirrorURL, arch string) string {
	// Format: https://mirror.example.com/archlinux/{repo}/os/{arch}/{pkgname}-{version}-{arch}.pkg.tar.zst
	return fmt.Sprintf("%s/%s/os/%s/%s-%s-%s.pkg.tar.zst",
		mirrorURL,
		pkg.DB().Name(),
		arch,
		pkg.Name(),
		pkg.Version(),
		pkg.Architecture(),
	)
}
