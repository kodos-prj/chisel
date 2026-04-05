// Package alpm provides a pure Go implementation of Arch Linux package management.
// It parses and queries Arch sync databases without requiring libalpm C library.
package alpm

import (
	"sync"
)

// Package represents an Arch Linux package with all its metadata.
type Package struct {
	Name           string
	Version        string
	Description    string
	Architecture   string // x86_64, aarch64, any
	URL            string
	Licenses       []string
	Groups         []string
	Provides       []string
	DependsOn      []string // Regular dependencies
	OptDepends     []string // Optional dependencies
	Conflicts      []string // Conflicting packages
	Replaces       []string // Packages this replaces
	CompressedSize int64    // Download size in bytes
	InstalledSize  int64    // Installed size in bytes
	Repository     string   // e.g., "core", "extra", "community"
	PackageBase    string   // Base package name
	BuildDate      string   // Build date
	Maintainer     string   // Maintainer name
	MD5Sum         string
	SHA256Sum      string
}

// Database represents a sync database containing multiple packages.
type Database struct {
	Name     string                // e.g., "core", "extra"
	Path     string                // Disk cache path
	Packages map[string]*Package   // name → Package
	Provides map[string][]*Package // virtual name → [packages providing it]
	Arch     string                // Target architecture for filtering
	mu       sync.RWMutex
}

// DatabaseCache maintains in-memory cache of all loaded databases.
// It respects repository precedence (core > extra > community > etc.)
type DatabaseCache struct {
	mu           sync.RWMutex
	packages     map[string]*Package   // package name → latest Package
	provides     map[string][]*Package // virtual name → [packages providing it]
	databases    []*Database           // ordered list of databases (by precedence)
	repoPriority map[string]int        // repository name → priority (lower = higher priority)
}

// Client provides the main interface for package queries and operations.
type Client struct {
	DbPath       string // Path to sync databases (e.g., /var/lib/pacman/sync)
	Arch         string // System architecture (x86_64, aarch64, etc.)
	Cache        *DatabaseCache
	Databases    []*Database
	DownloadURLs map[string]string // repository → base URL for downloads
}

// Dependency represents a package dependency with optional version constraint.
type Dependency struct {
	Name       string     // Package name
	Constraint Constraint // Version constraint type and value
}

// Constraint represents version constraint in dependency string.
type Constraint struct {
	Type  ConstraintType
	Value string
}

// ConstraintType defines the type of version constraint.
type ConstraintType int

const (
	ConstraintNone ConstraintType = iota
	ConstraintEqual
	ConstraintGreaterEqual
	ConstraintGreater
	ConstraintLessEqual
	ConstraintLess
)

// ResolutionError represents an error during dependency resolution.
type ResolutionError struct {
	Reason string
	Cycle  []string // If circular dependency, list of packages in cycle
}

func (e *ResolutionError) Error() string {
	if len(e.Cycle) > 0 {
		return "circular dependency: " + e.Reason
	}
	return e.Reason
}

// DefaultRepositoryPriority defines the default precedence order.
var DefaultRepositoryPriority = map[string]int{
	"core":      0,
	"extra":     1,
	"community": 2,
	"multilib":  3,
}
