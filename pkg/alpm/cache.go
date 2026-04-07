package alpm

// NewDatabaseCache creates a new empty database cache.
func NewDatabaseCache() *DatabaseCache {
	return &DatabaseCache{
		packages:     make(map[string]*Package),
		provides:     make(map[string][]*Package),
		groups:       make(map[string][]*Package),
		databases:    []*Database{},
		repoPriority: DefaultRepositoryPriority,
	}
}

// AddDatabase adds a database to the cache.
// Respects repository precedence: if a package exists in multiple repos,
// the one from the higher-priority repo (lower precedence number) is kept.
func (dc *DatabaseCache) AddDatabase(db *Database) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.databases = append(dc.databases, db)

	// Get precedence for this repository
	repoPrecedence := dc.repoPriority[db.Name]

	// Merge packages from database into cache
	for _, pkg := range db.Packages {
		if existing, has := dc.packages[pkg.Name]; has {
			// Compare precedence: lower number = higher priority
			existingPrec := dc.repoPriority[existing.Repository]
			if repoPrecedence >= existingPrec {
				// Existing package has higher or equal priority, skip
				continue
			}
			// Also check version if repositories have same priority
			if repoPrecedence == existingPrec && VerCmp(pkg.Version, existing.Version) <= 0 {
				continue
			}
		}

		// Add or update package
		dc.packages[pkg.Name] = pkg
	}

	// Merge provides mappings
	for provName, packages := range db.Provides {
		dc.provides[provName] = append(dc.provides[provName], packages...)
	}

	// Merge groups mappings
	for groupName, packages := range db.Groups {
		dc.groups[groupName] = append(dc.groups[groupName], packages...)
	}
}

// GetPackage retrieves a package by name with architecture filtering.
// Returns nil if not found.
func (dc *DatabaseCache) GetPackage(name string, arch string) *Package {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	pkg, has := dc.packages[name]
	if !has {
		return nil
	}

	// Check architecture filter
	if pkg.Architecture != "any" && pkg.Architecture != arch {
		return nil
	}

	return pkg
}

// GetAllPackages returns all packages in the cache (unfiltered).
func (dc *DatabaseCache) GetAllPackages() []*Package {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	packages := make([]*Package, 0, len(dc.packages))
	for _, pkg := range dc.packages {
		packages = append(packages, pkg)
	}
	return packages
}

// GetProvidingPackages returns all packages that provide a given virtual package name.
func (dc *DatabaseCache) GetProvidingPackages(name string) []*Package {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	return dc.provides[name]
}

// PackageCount returns the total number of packages in the cache.
func (dc *DatabaseCache) PackageCount() int {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	return len(dc.packages)
}

// GetPackagesByGroup returns all packages in a given group.
// Returns an empty slice if the group doesn't exist.
func (dc *DatabaseCache) GetPackagesByGroup(groupName string) []*Package {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	return dc.groups[groupName]
}

// ListAllGroups returns all group names in the cache.
func (dc *DatabaseCache) ListAllGroups() []string {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	groups := make([]string, 0, len(dc.groups))
	for groupName := range dc.groups {
		groups = append(groups, groupName)
	}
	return groups
}
