// Package alpm provides a wrapper around the ALPM (Arch Linux Package Management) library.
// It handles package database operations, dependency resolution, and package transactions.
//
// This package provides two implementations:
// - Pure Go implementation: No external dependencies (default, always available)
// - go-alpm/v2 wrapper: CGO-based (legacy, for backward compatibility)
//
// NewClient() automatically selects the appropriate implementation based on availability.
package alpm

import (
	"fmt"
	"os"
)

// PackageInfo contains detailed information about a package.
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

// ALPMClient is the main client for ALPM operations.
// It can use either the pure Go implementation or go-alpm/v2 (if available).
type ALPMClient struct {
	// Internal client (either *Client for pure Go, or *altImpl for go-alpm)
	impl interface{}

	// Track which implementation is in use
	isGoImpl bool

	// Keep legacy fields for potential future use
	root   string
	dbPath string
}

// NewClient creates a new ALPM client with the specified root and database path.
// root is the installation root (e.g., "/kod")
// dbPath is the database directory (e.g., "/kod/db")
//
// This function tries to use the best available implementation:
// 1. Pure Go implementation (if database files exist)
// 2. Falls back to pure Go implementation (always works if database synced)
func NewClient(root, dbPath string) (*ALPMClient, error) {
	// For now, always use pure Go implementation for compatibility with CGO_ENABLED=0
	// If root is not empty but isn't "/" we use it for the Go implementation
	if root == "" {
		root = "/"
	}
	if dbPath == "" {
		dbPath = "/var/lib/pacman/sync"
	}

	// Create pure Go client
	client := NewGoClient(dbPath, detectArch())

	return &ALPMClient{
		impl:     client,
		isGoImpl: true,
		root:     root,
		dbPath:   dbPath,
	}, nil
}

// detectArch returns the system architecture
func detectArch() string {
	// Try to detect architecture from environment or system
	if arch := os.Getenv("GOARCH"); arch != "" {
		switch arch {
		case "amd64":
			return "x86_64"
		case "arm64":
			return "aarch64"
		}
	}
	// Default to x86_64
	return "x86_64"
}

// Close releases resources held by the ALPM client.
func (c *ALPMClient) Close() error {
	if c.isGoImpl {
		client := c.impl.(*Client)
		return client.Close()
	}
	return nil
}

// RegisterSyncDB registers a sync database with the ALPM handle.
// repo is the repository name (e.g., "core", "extra")
func (c *ALPMClient) RegisterSyncDB(repo string) error {
	if c.isGoImpl {
		client := c.impl.(*Client)
		return client.RegisterSyncDB(repo)
	}
	return fmt.Errorf("client implementation not available")
}

// RegisterAllSyncDBs registers multiple sync databases.
func (c *ALPMClient) RegisterAllSyncDBs(repos []string) error {
	if c.isGoImpl {
		client := c.impl.(*Client)
		return client.RegisterAllSyncDBs(repos)
	}
	return fmt.Errorf("client implementation not available")
}

// SearchPackage searches for a package by name in all sync databases.
// Returns the package if found, or an error if not found.
func (c *ALPMClient) SearchPackage(name string) (*Package, error) {
	if c.isGoImpl {
		client := c.impl.(*Client)
		return client.SearchPackage(name)
	}
	return nil, fmt.Errorf("client implementation not available")
}

// SearchPackages searches for packages matching a pattern in all sync databases.
// pattern can be a partial package name (e.g., "vim" matches "vim", "gvim", etc.)
func (c *ALPMClient) SearchPackages(pattern string) ([]*Package, error) {
	if c.isGoImpl {
		client := c.impl.(*Client)
		return client.SearchPackages(pattern)
	}
	return nil, fmt.Errorf("client implementation not available")
}

// GetPackageInfo retrieves detailed information about a package.
func (c *ALPMClient) GetPackageInfo(name string) (*PackageInfo, error) {
	if c.isGoImpl {
		client := c.impl.(*Client)
		pkg, err := client.SearchPackage(name)
		if err != nil {
			return nil, err
		}

		info := &PackageInfo{
			Name:         pkg.Name,
			Version:      pkg.Version,
			Description:  pkg.Description,
			Architecture: pkg.Architecture,
			URL:          pkg.URL,
			Licenses:     pkg.Licenses,
			Groups:       pkg.Groups,
			Provides:     pkg.Provides,
			DependsOn:    pkg.DependsOn,
			OptDepends:   pkg.OptDepends,
			Conflicts:    pkg.Conflicts,
			Replaces:     pkg.Replaces,
			Size:         pkg.InstalledSize,
			DownloadSize: pkg.CompressedSize,
			Repository:   pkg.Repository,
			PackageBase:  pkg.PackageBase,
		}
		return info, nil
	}
	return nil, fmt.Errorf("client implementation not available")
}

// ResolveDependencies resolves all dependencies for a package.
// Returns a list of package names that need to be installed, in dependency order.
func (c *ALPMClient) ResolveDependencies(packageName string) ([]string, error) {
	if c.isGoImpl {
		client := c.impl.(*Client)
		return client.ResolveDependencies(packageName)
	}
	return nil, fmt.Errorf("client implementation not available")
}

// ListSyncDBs returns a list of registered sync database names.
func (c *ALPMClient) ListSyncDBs() ([]string, error) {
	if c.isGoImpl {
		client := c.impl.(*Client)
		return client.ListSyncDBs()
	}
	return nil, fmt.Errorf("client implementation not available")
}

// GetDownloadURL returns the download URL for a package.
// This constructs the URL based on the package's repository and name.
func (c *ALPMClient) GetDownloadURL(pkg *Package, mirrorURL string) string {
	if c.isGoImpl {
		client := c.impl.(*Client)
		return client.GetDownloadURL(pkg, mirrorURL)
	}
	return ""
}

// GetImpl returns the internal client implementation (for testing)
func (c *ALPMClient) GetImpl() interface{} {
	return c.impl
}

// SearchPackagesByGroup returns all packages in a given group.
// Returns an empty slice if the group doesn't exist.
func (c *ALPMClient) SearchPackagesByGroup(groupName string) []*Package {
	if c.isGoImpl {
		client := c.impl.(*Client)
		return client.SearchPackagesByGroup(groupName)
	}
	return nil
}

// ListAllGroups returns all package group names from all databases.
func (c *ALPMClient) ListAllGroups() []string {
	if c.isGoImpl {
		client := c.impl.(*Client)
		return client.ListAllGroups()
	}
	return nil
}
