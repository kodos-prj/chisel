package alpm

import (
	"testing"
)

// TestDatabaseCacheGroups tests group indexing and retrieval in DatabaseCache.
func TestDatabaseCacheGroups(t *testing.T) {
	// Create packages with groups
	pkg1 := &Package{
		Name:    "gnome-shell",
		Version: "45.0-1",
		Groups:  []string{"gnome"},
	}
	pkg2 := &Package{
		Name:    "gnome-terminal",
		Version: "3.50-1",
		Groups:  []string{"gnome"},
	}
	pkg3 := &Package{
		Name:    "nautilus",
		Version: "45-1",
		Groups:  []string{"gnome"},
	}
	pkg4 := &Package{
		Name:    "gcc",
		Version: "13.2.1-1",
		Groups:  []string{"base-devel", "development-tools"},
	}

	// Create database with packages
	db := &Database{
		Name:     "extra",
		Packages: make(map[string]*Package),
		Groups:   make(map[string][]*Package),
	}

	// Add packages to database and build groups index
	packages := []*Package{pkg1, pkg2, pkg3, pkg4}
	for _, pkg := range packages {
		db.Packages[pkg.Name] = pkg
		for _, group := range pkg.Groups {
			db.Groups[group] = append(db.Groups[group], pkg)
		}
	}

	// Create cache and add database
	cache := NewDatabaseCache()
	cache.AddDatabase(db)

	// Test SearchPackagesByGroup
	tests := []struct {
		group         string
		expectedCount int
		expectedNames []string
	}{
		{"gnome", 3, []string{"gnome-shell", "gnome-terminal", "nautilus"}},
		{"base-devel", 1, []string{"gcc"}},
		{"development-tools", 1, []string{"gcc"}},
		{"nonexistent", 0, []string{}},
	}

	for _, tt := range tests {
		result := cache.GetPackagesByGroup(tt.group)
		if len(result) != tt.expectedCount {
			t.Errorf("GetPackagesByGroup(%q) returned %d packages, expected %d",
				tt.group, len(result), tt.expectedCount)
		}

		// Verify package names (as a simple check)
		if len(result) > 0 {
			found := make(map[string]bool)
			for _, pkg := range result {
				found[pkg.Name] = true
			}
			for _, expectedName := range tt.expectedNames {
				if !found[expectedName] {
					t.Errorf("GetPackagesByGroup(%q) missing expected package %q",
						tt.group, expectedName)
				}
			}
		}
	}

	// Test ListAllGroups
	allGroups := cache.ListAllGroups()
	expectedGroups := map[string]bool{
		"gnome":             true,
		"base-devel":        true,
		"development-tools": true,
	}

	if len(allGroups) != len(expectedGroups) {
		t.Errorf("ListAllGroups() returned %d groups, expected %d",
			len(allGroups), len(expectedGroups))
	}

	for _, group := range allGroups {
		if !expectedGroups[group] {
			t.Errorf("ListAllGroups() returned unexpected group %q", group)
		}
	}
}

// TestMultipleDatabasesGroups tests group merging across multiple databases.
func TestMultipleDatabasesGroups(t *testing.T) {
	// Create packages in core repo
	corePkg := &Package{
		Name:       "linux",
		Version:    "6.5.0-1",
		Repository: "core",
		Groups:     []string{"base"},
	}

	coreDB := &Database{
		Name:     "core",
		Packages: map[string]*Package{corePkg.Name: corePkg},
		Groups:   map[string][]*Package{"base": {corePkg}},
	}

	// Create packages in extra repo
	extraPkg := &Package{
		Name:       "vim",
		Version:    "9.0.0-1",
		Repository: "extra",
		Groups:     []string{"editors"},
	}

	extraDB := &Database{
		Name:     "extra",
		Packages: map[string]*Package{extraPkg.Name: extraPkg},
		Groups:   map[string][]*Package{"editors": {extraPkg}},
	}

	// Create cache and add both databases
	cache := NewDatabaseCache()
	cache.AddDatabase(coreDB)
	cache.AddDatabase(extraDB)

	// Verify both groups are present
	baseGroups := cache.GetPackagesByGroup("base")
	if len(baseGroups) != 1 || baseGroups[0].Name != "linux" {
		t.Error("Failed to retrieve 'base' group from first database")
	}

	editorGroups := cache.GetPackagesByGroup("editors")
	if len(editorGroups) != 1 || editorGroups[0].Name != "vim" {
		t.Error("Failed to retrieve 'editors' group from second database")
	}

	// Verify all groups listed
	allGroups := cache.ListAllGroups()
	if len(allGroups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(allGroups))
	}
}

// TestGroupParsing tests that groups are correctly parsed from package metadata.
func TestGroupParsing(t *testing.T) {
	// Create a package with multiple groups (like it would be in real data)
	pkg := &Package{
		Name:    "gcc",
		Version: "13.2.1-1",
		Groups:  []string{"base-devel", "development-tools"},
	}

	if len(pkg.Groups) != 2 {
		t.Errorf("Package has %d groups, expected 2", len(pkg.Groups))
	}

	expectedGroups := map[string]bool{
		"base-devel":        true,
		"development-tools": true,
	}

	for _, group := range pkg.Groups {
		if !expectedGroups[group] {
			t.Errorf("Unexpected group: %q", group)
		}
	}
}

// TestGroupEmptyPackage tests handling of packages with no groups.
func TestGroupEmptyPackage(t *testing.T) {
	pkg := &Package{
		Name:    "some-utility",
		Version: "1.0-1",
		Groups:  []string{},
	}

	db := &Database{
		Name:     "extra",
		Packages: map[string]*Package{pkg.Name: pkg},
		Groups:   make(map[string][]*Package),
	}

	// Build groups index (should be empty for this package)
	for _, group := range pkg.Groups {
		db.Groups[group] = append(db.Groups[group], pkg)
	}

	cache := NewDatabaseCache()
	cache.AddDatabase(db)

	// Verify no groups are added for this package
	allGroups := cache.ListAllGroups()
	if len(allGroups) != 0 {
		t.Errorf("Expected no groups, got %d", len(allGroups))
	}
}

// TestPackageInMultipleGroups tests handling of packages in multiple groups.
func TestPackageInMultipleGroups(t *testing.T) {
	pkg := &Package{
		Name:    "gcc",
		Version: "13.2.1-1",
		Groups:  []string{"base", "base-devel", "development-tools"},
	}

	db := &Database{
		Name:     "core",
		Packages: map[string]*Package{pkg.Name: pkg},
		Groups:   make(map[string][]*Package),
	}

	// Build groups index
	for _, group := range pkg.Groups {
		db.Groups[group] = append(db.Groups[group], pkg)
	}

	cache := NewDatabaseCache()
	cache.AddDatabase(db)

	// Verify package appears in all groups
	for _, group := range pkg.Groups {
		result := cache.GetPackagesByGroup(group)
		if len(result) != 1 || result[0].Name != "gcc" {
			t.Errorf("Package not found in group %q", group)
		}
	}

	// Verify all groups listed
	allGroups := cache.ListAllGroups()
	if len(allGroups) != 3 {
		t.Errorf("Expected 3 groups, got %d", len(allGroups))
	}
}
