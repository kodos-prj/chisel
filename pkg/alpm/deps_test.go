package alpm

import (
	"testing"
)

// TestResolveDependencies_SimpleDependency tests resolving a simple dependency chain.
func TestResolveDependencies_SimpleDependency(t *testing.T) {
	// Create test client
	client := NewGoClient("/var/lib/pacman/sync", "x86_64")

	// Create mock packages
	pkgA := &Package{
		Name:         "pkg-a",
		Version:      "1.0-1",
		Repository:   "core",
		Architecture: "x86_64",
		DependsOn:    []string{"pkg-b"},
	}

	pkgB := &Package{
		Name:         "pkg-b",
		Version:      "2.0-1",
		Repository:   "core",
		Architecture: "x86_64",
		DependsOn:    []string{},
	}

	db := &Database{
		Name: "core",
		Packages: map[string]*Package{
			"pkg-a": pkgA,
			"pkg-b": pkgB,
		},
		Provides: make(map[string][]*Package),
		Arch:     "x86_64",
	}

	client.Databases = append(client.Databases, db)
	client.Cache.AddDatabase(db)

	// Resolve dependencies for pkg-a
	result, err := client.ResolveDependencies("pkg-a")
	if err != nil {
		t.Fatalf("ResolveDependencies failed: %v", err)
	}

	// Should return [pkg-b, pkg-a] (dependencies first)
	if len(result) != 2 {
		t.Errorf("Expected 2 packages, got %d", len(result))
		return
	}
	if result[0] != "pkg-b" {
		t.Errorf("Expected pkg-b first, got %s", result[0])
	}
	if result[1] != "pkg-a" {
		t.Errorf("Expected pkg-a second, got %s", result[1])
	}
}

// TestResolveDependencies_MultipleDependencies tests resolving multiple dependencies.
func TestResolveDependencies_MultipleDependencies(t *testing.T) {
	client := NewGoClient("/var/lib/pacman/sync", "x86_64")

	pkgA := &Package{
		Name:       "pkg-a",
		Version:    "1.0-1",
		Repository: "core",
		Architecture: "x86_64",
		DependsOn:  []string{"pkg-b", "pkg-c"},
	}

	pkgB := &Package{
		Name:       "pkg-b",
		Version:    "1.0-1",
		Repository: "core",
		Architecture: "x86_64",
		DependsOn:  []string{},
	}

	pkgC := &Package{
		Name:       "pkg-c",
		Version:    "1.0-1",
		Repository: "core",
		Architecture: "x86_64",
		DependsOn:  []string{},
	}

	db := &Database{
		Name: "core",
		Packages: map[string]*Package{
			"pkg-a": pkgA,
			"pkg-b": pkgB,
			"pkg-c": pkgC,
		},
		Provides: make(map[string][]*Package),
		Arch:     "x86_64",
	}

	client.Databases = append(client.Databases, db)
	client.Cache.AddDatabase(db)

	result, err := client.ResolveDependencies("pkg-a")
	if err != nil {
		t.Fatalf("ResolveDependencies failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 packages, got %d: %v", len(result), result)
	}

	// pkg-a should be last
	if result[len(result)-1] != "pkg-a" {
		t.Errorf("Expected pkg-a last, got %s", result[len(result)-1])
	}
}

// TestResolveDependencies_TransitiveDependencies tests transitive dependency resolution.
func TestResolveDependencies_TransitiveDependencies(t *testing.T) {
	client := NewGoClient("/var/lib/pacman/sync", "x86_64")

	// A depends on B, B depends on C
	pkgA := &Package{
		Name:       "pkg-a",
		Version:    "1.0-1",
		Repository: "core",
		Architecture: "x86_64",
		DependsOn:  []string{"pkg-b"},
	}

	pkgB := &Package{
		Name:       "pkg-b",
		Version:    "1.0-1",
		Repository: "core",
		Architecture: "x86_64",
		DependsOn:  []string{"pkg-c"},
	}

	pkgC := &Package{
		Name:       "pkg-c",
		Version:    "1.0-1",
		Repository: "core",
		Architecture: "x86_64",
		DependsOn:  []string{},
	}

	db := &Database{
		Name: "core",
		Packages: map[string]*Package{
			"pkg-a": pkgA,
			"pkg-b": pkgB,
			"pkg-c": pkgC,
		},
		Provides: make(map[string][]*Package),
		Arch:     "x86_64",
	}

	client.Databases = append(client.Databases, db)
	client.Cache.AddDatabase(db)

	result, err := client.ResolveDependencies("pkg-a")
	if err != nil {
		t.Fatalf("ResolveDependencies failed: %v", err)
	}

	// Should be [pkg-c, pkg-b, pkg-a] (deepest dependency first)
	if len(result) != 3 {
		t.Errorf("Expected 3 packages, got %d", len(result))
		return
	}
	if result[0] != "pkg-c" {
		t.Errorf("Expected pkg-c first, got %s", result[0])
	}
	if result[1] != "pkg-b" {
		t.Errorf("Expected pkg-b second, got %s", result[1])
	}
	if result[2] != "pkg-a" {
		t.Errorf("Expected pkg-a third, got %s", result[2])
	}
}

// TestResolveDependencies_CircularDependency tests circular dependency detection.
func TestResolveDependencies_CircularDependency(t *testing.T) {
	client := NewGoClient("/var/lib/pacman/sync", "x86_64")

	// Create circular dependency: A -> B -> C -> A
	pkgA := &Package{
		Name:       "pkg-a",
		Version:    "1.0-1",
		Repository: "core",
		Architecture: "x86_64",
		DependsOn:  []string{"pkg-b"},
	}

	pkgB := &Package{
		Name:       "pkg-b",
		Version:    "1.0-1",
		Repository: "core",
		Architecture: "x86_64",
		DependsOn:  []string{"pkg-c"},
	}

	pkgC := &Package{
		Name:       "pkg-c",
		Version:    "1.0-1",
		Repository: "core",
		Architecture: "x86_64",
		DependsOn:  []string{"pkg-a"}, // Cycle
	}

	db := &Database{
		Name: "core",
		Packages: map[string]*Package{
			"pkg-a": pkgA,
			"pkg-b": pkgB,
			"pkg-c": pkgC,
		},
		Provides: make(map[string][]*Package),
		Arch:     "x86_64",
	}

	client.Databases = append(client.Databases, db)
	client.Cache.AddDatabase(db)

	_, err := client.ResolveDependencies("pkg-a")
	if err == nil {
		t.Fatal("Expected circular dependency error, got nil")
	}

	// Check that error indicates circular dependency
	resErr, ok := err.(*ResolutionError)
	if !ok {
		t.Fatalf("Expected ResolutionError, got %T", err)
	}
	if len(resErr.Cycle) == 0 {
		t.Error("Expected cycle information in error")
	}
}

// TestResolveDependencies_VersionConstraint tests version constraint validation.
func TestResolveDependencies_VersionConstraint(t *testing.T) {
	client := NewGoClient("/var/lib/pacman/sync", "x86_64")

	pkgA := &Package{
		Name:       "pkg-a",
		Version:    "1.0-1",
		Repository: "core",
		Architecture: "x86_64",
		DependsOn:  []string{"pkg-b>=2.0"}, // Requires pkg-b >= 2.0
	}

	pkgB := &Package{
		Name:       "pkg-b",
		Version:    "1.5-1", // Version too low
		Repository: "core",
		Architecture: "x86_64",
		DependsOn:  []string{},
	}

	db := &Database{
		Name: "core",
		Packages: map[string]*Package{
			"pkg-a": pkgA,
			"pkg-b": pkgB,
		},
		Provides: make(map[string][]*Package),
		Arch:     "x86_64",
	}

	client.Databases = append(client.Databases, db)
	client.Cache.AddDatabase(db)

	_, err := client.ResolveDependencies("pkg-a")
	if err == nil {
		t.Fatal("Expected version constraint error, got nil")
	}
}

// TestResolveDependencies_VirtualPackageProvider tests resolution through virtual packages.
func TestResolveDependencies_VirtualPackageProvider(t *testing.T) {
	client := NewGoClient("/var/lib/pacman/sync", "x86_64")

	pkgA := &Package{
		Name:       "pkg-a",
		Version:    "1.0-1",
		Repository: "core",
		Architecture: "x86_64",
		DependsOn:  []string{"virtual-pkg"}, // Depends on virtual package
	}

	pkgBProvider := &Package{
		Name:       "pkg-b",
		Version:    "1.0-1",
		Repository: "core",
		Architecture: "x86_64",
		DependsOn:  []string{},
		Provides:   []string{"virtual-pkg"}, // Provides the virtual package
	}

	db := &Database{
		Name: "core",
		Packages: map[string]*Package{
			"pkg-a": pkgA,
			"pkg-b": pkgBProvider,
		},
		Provides: map[string][]*Package{
			"virtual-pkg": {pkgBProvider},
		},
		Arch: "x86_64",
	}

	client.Databases = append(client.Databases, db)
	client.Cache.AddDatabase(db)

	result, err := client.ResolveDependencies("pkg-a")
	if err != nil {
		t.Fatalf("ResolveDependencies failed: %v", err)
	}

	// Should resolve to [pkg-b, pkg-a]
	if len(result) != 2 {
		t.Errorf("Expected 2 packages, got %d", len(result))
		return
	}
	if result[0] != "pkg-b" {
		t.Errorf("Expected pkg-b (provider) first, got %s", result[0])
	}
}

// TestResolveDependencies_MissingDependency tests error handling for missing dependencies.
func TestResolveDependencies_MissingDependency(t *testing.T) {
	client := NewGoClient("/var/lib/pacman/sync", "x86_64")

	pkgA := &Package{
		Name:       "pkg-a",
		Version:    "1.0-1",
		Repository: "core",
		Architecture: "x86_64",
		DependsOn:  []string{"nonexistent-pkg"},
	}

	db := &Database{
		Name: "core",
		Packages: map[string]*Package{
			"pkg-a": pkgA,
		},
		Provides: make(map[string][]*Package),
		Arch:     "x86_64",
	}

	client.Databases = append(client.Databases, db)
	client.Cache.AddDatabase(db)

	_, err := client.ResolveDependencies("pkg-a")
	if err == nil {
		t.Fatal("Expected error for missing dependency, got nil")
	}
}

// TestResolveDependencies_NoDependencies tests resolving a package with no dependencies.
func TestResolveDependencies_NoDependencies(t *testing.T) {
	client := NewGoClient("/var/lib/pacman/sync", "x86_64")

	pkgA := &Package{
		Name:       "pkg-a",
		Version:    "1.0-1",
		Repository: "core",
		Architecture: "x86_64",
		DependsOn:  []string{},
	}

	db := &Database{
		Name: "core",
		Packages: map[string]*Package{
			"pkg-a": pkgA,
		},
		Provides: make(map[string][]*Package),
		Arch:     "x86_64",
	}

	client.Databases = append(client.Databases, db)
	client.Cache.AddDatabase(db)

	result, err := client.ResolveDependencies("pkg-a")
	if err != nil {
		t.Fatalf("ResolveDependencies failed: %v", err)
	}

	// Should only return pkg-a
	if len(result) != 1 {
		t.Errorf("Expected 1 package, got %d", len(result))
		return
	}
	if result[0] != "pkg-a" {
		t.Errorf("Expected pkg-a, got %s", result[0])
	}
}

// TestResolveDependencies_SharedDependency tests handling shared dependencies (diamond pattern).
func TestResolveDependencies_SharedDependency(t *testing.T) {
	client := NewGoClient("/var/lib/pacman/sync", "x86_64")

	// Diamond pattern: A -> B,C; B,C -> D
	pkgA := &Package{
		Name:       "pkg-a",
		Version:    "1.0-1",
		Repository: "core",
		Architecture: "x86_64",
		DependsOn:  []string{"pkg-b", "pkg-c"},
	}

	pkgB := &Package{
		Name:       "pkg-b",
		Version:    "1.0-1",
		Repository: "core",
		Architecture: "x86_64",
		DependsOn:  []string{"pkg-d"},
	}

	pkgC := &Package{
		Name:       "pkg-c",
		Version:    "1.0-1",
		Repository: "core",
		Architecture: "x86_64",
		DependsOn:  []string{"pkg-d"},
	}

	pkgD := &Package{
		Name:       "pkg-d",
		Version:    "1.0-1",
		Repository: "core",
		Architecture: "x86_64",
		DependsOn:  []string{},
	}

	db := &Database{
		Name: "core",
		Packages: map[string]*Package{
			"pkg-a": pkgA,
			"pkg-b": pkgB,
			"pkg-c": pkgC,
			"pkg-d": pkgD,
		},
		Provides: make(map[string][]*Package),
		Arch:     "x86_64",
	}

	client.Databases = append(client.Databases, db)
	client.Cache.AddDatabase(db)

	result, err := client.ResolveDependencies("pkg-a")
	if err != nil {
		t.Fatalf("ResolveDependencies failed: %v", err)
	}

	// Should return [pkg-d, pkg-b, pkg-c, pkg-a] (pkg-d only once)
	if len(result) != 4 {
		t.Errorf("Expected 4 packages (with pkg-d once), got %d: %v", len(result), result)
		return
	}

	// Count occurrences of pkg-d
	count := 0
	for _, pkg := range result {
		if pkg == "pkg-d" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Expected pkg-d once, got %d times", count)
	}

	// pkg-a should be last
	if result[len(result)-1] != "pkg-a" {
		t.Errorf("Expected pkg-a last, got %s", result[len(result)-1])
	}
}
