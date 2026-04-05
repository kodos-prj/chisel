package alpm

import (
	"testing"
)

// TestVersionComparison_RealWorldExamples tests version comparison with real Arch packages.
func TestVersionComparison_RealWorldExamples(t *testing.T) {
	tests := []struct {
		a, b   string
		expect int
		desc   string
	}{
		// Real Arch package versions
		{"6.1.0-1", "6.0.0-1", 1, "linux kernel versions"},
		{"2:8.2.4-1", "2:8.2.3-1", 1, "systemd versions"},
		{"1:22.04-1", "1:20.10.24-1", 1, "docker versions"},
		{"10.4-1", "10.3-1", 1, "postgresql versions"},
		{"16.1.11.0-1", "16.1.10.0-1", 1, "chromium versions"},
		{"1.5.7-1", "1.5.6-1", 1, "git versions"},
		{"15-1", "14-1", 1, "python versions"},
	}

	for _, tt := range tests {
		result := VerCmp(tt.a, tt.b)
		if result != tt.expect {
			t.Errorf("VerCmp(%q, %q) = %d (want %d) [%s]",
				tt.a, tt.b, result, tt.expect, tt.desc)
		}
	}
}

// TestDependencyParsing tests parsing various dependency formats.
func TestDependencyParsing(t *testing.T) {
	tests := []struct {
		depStr        string
		expectedName  string
		expectedType  ConstraintType
		expectedValue string
		desc          string
	}{
		{"linux-headers", "linux-headers", ConstraintNone, "", "no constraint"},
		{"gcc>=11.0", "gcc", ConstraintGreaterEqual, "11.0", "greater or equal"},
		{"openssl<=3.0", "openssl", ConstraintLessEqual, "3.0", "less or equal"},
		{"glibc=2.35", "glibc", ConstraintEqual, "2.35", "exact version"},
		{"ncurses>6.0", "ncurses", ConstraintGreater, "6.0", "strictly greater"},
		{"libutil-linux<2.40", "libutil-linux", ConstraintLess, "2.40", "strictly less"},
	}

	for _, tt := range tests {
		name, constraint, err := ParseDependency(tt.depStr)
		if err != nil {
			t.Errorf("ParseDependency(%q) error: %v", tt.depStr, err)
			continue
		}

		if name != tt.expectedName {
			t.Errorf("ParseDependency(%q): name = %q, want %q [%s]",
				tt.depStr, name, tt.expectedName, tt.desc)
		}
		if constraint.Type != tt.expectedType {
			t.Errorf("ParseDependency(%q): type = %v, want %v [%s]",
				tt.depStr, constraint.Type, tt.expectedType, tt.desc)
		}
		if constraint.Value != tt.expectedValue {
			t.Errorf("ParseDependency(%q): value = %q, want %q [%s]",
				tt.depStr, constraint.Value, tt.expectedValue, tt.desc)
		}
	}
}

// TestCacheAddDatabase tests adding databases to cache with precedence.
func TestCacheAddDatabase(t *testing.T) {
	cache := NewDatabaseCache()

	// Create a test package
	pkg1 := &Package{
		Name:         "test-pkg",
		Version:      "1.0-1",
		Repository:   "core",
		Architecture: "x86_64",
	}

	db1 := &Database{
		Name:     "core",
		Packages: map[string]*Package{"test-pkg": pkg1},
		Provides: make(map[string][]*Package),
		Arch:     "x86_64",
	}

	cache.AddDatabase(db1)

	// Verify package was added
	if count := cache.PackageCount(); count != 1 {
		t.Errorf("Expected 1 package in cache, got %d", count)
	}

	// Retrieve and verify
	retrieved := cache.GetPackage("test-pkg", "x86_64")
	if retrieved == nil {
		t.Fatal("Package not found in cache")
	}
	if retrieved.Version != "1.0-1" {
		t.Errorf("Expected version 1.0-1, got %s", retrieved.Version)
	}
}

// TestCacheRepositoryPrecedence tests that core packages override extra packages.
func TestCacheRepositoryPrecedence(t *testing.T) {
	cache := NewDatabaseCache()

	// Create packages with same name but different repos
	pkgExtra := &Package{
		Name:         "vim",
		Version:      "8.2-1",
		Repository:   "extra",
		Architecture: "x86_64",
	}

	pkgCore := &Package{
		Name:         "vim",
		Version:      "8.2-1",
		Repository:   "core",
		Architecture: "x86_64",
	}

	dbExtra := &Database{
		Name:     "extra",
		Packages: map[string]*Package{"vim": pkgExtra},
		Provides: make(map[string][]*Package),
		Arch:     "x86_64",
	}

	dbCore := &Database{
		Name:     "core",
		Packages: map[string]*Package{"vim": pkgCore},
		Provides: make(map[string][]*Package),
		Arch:     "x86_64",
	}

	// Add extra first, then core
	cache.AddDatabase(dbExtra)
	cache.AddDatabase(dbCore)

	// Verify core package is in cache (higher precedence)
	retrieved := cache.GetPackage("vim", "x86_64")
	if retrieved == nil {
		t.Fatal("vim package not found in cache")
	}
	if retrieved.Repository != "core" {
		t.Errorf("Expected core package, got %s", retrieved.Repository)
	}
}

// TestCacheArchitectureFiltering tests architecture filtering.
func TestCacheArchitectureFiltering(t *testing.T) {
	cache := NewDatabaseCache()

	// Create packages with different architectures
	pkg64 := &Package{
		Name:         "gcc",
		Version:      "11.0-1",
		Repository:   "core",
		Architecture: "x86_64",
	}

	pkgAny := &Package{
		Name:         "glibc",
		Version:      "2.35-1",
		Repository:   "core",
		Architecture: "any",
	}

	db := &Database{
		Name:     "core",
		Packages: map[string]*Package{"gcc": pkg64, "glibc": pkgAny},
		Provides: make(map[string][]*Package),
		Arch:     "x86_64",
	}

	cache.AddDatabase(db)

	// Should find x86_64 package when arch is x86_64
	gcc := cache.GetPackage("gcc", "x86_64")
	if gcc == nil {
		t.Fatal("gcc not found")
	}

	// Should find "any" architecture package
	glibc := cache.GetPackage("glibc", "x86_64")
	if glibc == nil {
		t.Fatal("glibc not found")
	}

	// Should NOT find x86_64 package when arch is aarch64
	gcc64When64 := cache.GetPackage("gcc", "aarch64")
	if gcc64When64 != nil {
		t.Error("Should not find x86_64-specific package for aarch64 architecture")
	}

	// Should still find "any" architecture package
	glibcArch64 := cache.GetPackage("glibc", "aarch64")
	if glibcArch64 == nil {
		t.Fatal("glibc should be found for any architecture")
	}
}

// TestCacheVirtualPackages tests virtual package support.
func TestCacheVirtualPackages(t *testing.T) {
	cache := NewDatabaseCache()

	// Create a package that provides a virtual package
	pkg := &Package{
		Name:         "openrc",
		Version:      "0.43-1",
		Repository:   "community",
		Architecture: "x86_64",
		Provides:     []string{"init"},
	}

	db := &Database{
		Name:     "community",
		Packages: map[string]*Package{"openrc": pkg},
		Provides: map[string][]*Package{
			"init": {pkg},
		},
		Arch: "x86_64",
	}

	cache.AddDatabase(db)

	// Verify we can find the provider
	providers := cache.GetProvidingPackages("init")
	if len(providers) != 1 {
		t.Errorf("Expected 1 provider for 'init', got %d", len(providers))
	}
	if providers[0].Name != "openrc" {
		t.Errorf("Expected 'openrc' as provider, got %s", providers[0].Name)
	}
}

// TestParsePackageDatabase_ValidFormat tests parsing a valid database.
// TODO: Implement with proper mock tar.gz creation when needed
// For now, this is skipped.
func TestParsePackageDatabase_ValidFormat(t *testing.T) {
	t.Skip("Mock database test - implement when needed")
}

// BenchmarkVersionComparison benchmarks the version comparison function.
func BenchmarkVersionComparison(b *testing.B) {
	versions := [][2]string{
		{"1.0-1", "1.0-2"},
		{"2.5.3-1", "2.5.3-2"},
		{"1:8.2.4-1", "1:8.2.3-1"},
		{"6.1.0-1", "6.0.0-1"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, pair := range versions {
			_ = VerCmp(pair[0], pair[1])
		}
	}
}

// BenchmarkCacheGetPackage benchmarks cache lookups.
func BenchmarkCacheGetPackage(b *testing.B) {
	cache := NewDatabaseCache()

	// Add test packages
	pkgs := make(map[string]*Package)
	for i := 0; i < 1000; i++ {
		name := "test-pkg-" + string(rune(i))
		pkgs[name] = &Package{
			Name:         name,
			Version:      "1.0-1",
			Repository:   "core",
			Architecture: "x86_64",
		}
	}

	db := &Database{
		Name:     "core",
		Packages: pkgs,
		Provides: make(map[string][]*Package),
		Arch:     "x86_64",
	}

	cache.AddDatabase(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cache.GetPackage("test-pkg-500", "x86_64")
	}
}
