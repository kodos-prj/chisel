// Package registry_test provides tests for AUR version tracking
package registry

import (
	"path/filepath"
	"testing"
	"time"
)

// TestPackageWithAURFields tests Package struct with new AUR fields
func TestPackageWithAURFields(t *testing.T) {
	pkg := &Package{
		Name:         "yay",
		Version:      "12.0.0",
		Source:       "aur",
		Repository:   "aur",
		Files:        []string{"/usr/bin/yay"},
		Executables:  []string{"/usr/bin/yay"},
		Dependencies: []string{"pacman"},
		InstallDate:  "2026-04-05T10:30:00Z",
		UpdateDate:   "2026-04-05T10:30:00Z",
	}

	if pkg.Source != "aur" {
		t.Errorf("Source should be 'aur', got %s", pkg.Source)
	}

	if pkg.Repository != "aur" {
		t.Errorf("Repository should be 'aur', got %s", pkg.Repository)
	}

	if pkg.UpdateDate == "" {
		t.Error("UpdateDate should not be empty")
	}
}

// TestOfficialPackageWithNewFields tests Package struct for official repo packages
func TestOfficialPackageWithNewFields(t *testing.T) {
	pkg := &Package{
		Name:         "bash",
		Version:      "5.1.16-2",
		Source:       "official",
		Repository:   "core",
		Files:        []string{"/bin/bash", "/usr/share/doc/bash/"},
		Executables:  []string{"/bin/bash"},
		Dependencies: []string{},
		InstallDate:  "2026-03-20T15:00:00Z",
		UpdateDate:   "2026-04-05T10:30:00Z",
	}

	if pkg.Source != "official" {
		t.Errorf("Source should be 'official', got %s", pkg.Source)
	}

	if pkg.Repository != "core" {
		t.Errorf("Repository should be 'core', got %s", pkg.Repository)
	}
}

// TestGetAURPackages tests GetAURPackages method
func TestGetAURPackages(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r, err := NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Add mix of official and AUR packages
	pkgs := []*Package{
		{
			Name:        "bash",
			Version:     "5.1.16-2",
			Source:      "official",
			Repository:  "core",
			InstallDate: "2026-03-20T15:00:00Z",
		},
		{
			Name:        "yay",
			Version:     "12.0.0",
			Source:      "aur",
			Repository:  "aur",
			InstallDate: "2026-04-01T10:00:00Z",
		},
		{
			Name:        "vim",
			Version:     "8.2.3455-1",
			Source:      "official",
			Repository:  "extra",
			InstallDate: "2026-03-20T16:00:00Z",
		},
		{
			Name:        "lf",
			Version:     "28",
			Source:      "aur",
			Repository:  "aur",
			InstallDate: "2026-04-02T12:00:00Z",
		},
	}

	for _, pkg := range pkgs {
		err := r.AddPackage(pkg)
		if err != nil {
			t.Fatalf("Failed to add package: %v", err)
		}
	}

	// Get AUR packages
	aurPkgs := r.GetAURPackages()
	if len(aurPkgs) != 2 {
		t.Errorf("Expected 2 AUR packages, got %d", len(aurPkgs))
	}

	// Verify all returned packages are AUR
	for _, pkg := range aurPkgs {
		if pkg.Source != "aur" {
			t.Errorf("Expected source 'aur', got %s for package %s", pkg.Source, pkg.Name)
		}
	}
}

// TestGetOfficialPackages tests GetOfficialPackages method
func TestGetOfficialPackages(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r, err := NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Add mix of official and AUR packages
	pkgs := []*Package{
		{
			Name:        "bash",
			Version:     "5.1.16-2",
			Source:      "official",
			Repository:  "core",
			InstallDate: "2026-03-20T15:00:00Z",
		},
		{
			Name:        "yay",
			Version:     "12.0.0",
			Source:      "aur",
			Repository:  "aur",
			InstallDate: "2026-04-01T10:00:00Z",
		},
		{
			Name:        "vim",
			Version:     "8.2.3455-1",
			Source:      "official",
			Repository:  "extra",
			InstallDate: "2026-03-20T16:00:00Z",
		},
	}

	for _, pkg := range pkgs {
		err := r.AddPackage(pkg)
		if err != nil {
			t.Fatalf("Failed to add package: %v", err)
		}
	}

	// Get official packages
	officialPkgs := r.GetOfficialPackages()
	if len(officialPkgs) != 2 {
		t.Errorf("Expected 2 official packages, got %d", len(officialPkgs))
	}

	// Verify all returned packages are official
	for _, pkg := range officialPkgs {
		if pkg.Source != "official" {
			t.Errorf("Expected source 'official', got %s for package %s", pkg.Source, pkg.Name)
		}
	}
}

// TestUpdatePackageVersion tests UpdatePackageVersion method
func TestUpdatePackageVersion(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r, err := NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Add initial package
	pkg := &Package{
		Name:        "yay",
		Version:     "11.0.0",
		Source:      "aur",
		Repository:  "aur",
		InstallDate: "2026-04-01T10:00:00Z",
		UpdateDate:  "2026-04-01T10:00:00Z",
	}

	err = r.AddPackage(pkg)
	if err != nil {
		t.Fatalf("Failed to add package: %v", err)
	}

	// Update version
	newDate := time.Now().Format(time.RFC3339)
	err = r.UpdatePackageVersion("yay", "12.0.0", newDate)
	if err != nil {
		t.Fatalf("Failed to update package version: %v", err)
	}

	// Verify update
	updated, ok := r.GetPackage("yay")
	if !ok {
		t.Fatal("Package not found after update")
	}

	if updated.Version != "12.0.0" {
		t.Errorf("Expected version 12.0.0, got %s", updated.Version)
	}

	if updated.UpdateDate != newDate {
		t.Errorf("Expected UpdateDate %s, got %s", newDate, updated.UpdateDate)
	}

	// InstallDate should remain unchanged
	if updated.InstallDate != pkg.InstallDate {
		t.Errorf("InstallDate should not change on version update")
	}
}

// TestUpdatePackageVersion_NotFound tests UpdatePackageVersion with non-existent package
func TestUpdatePackageVersion_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r, err := NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Try to update non-existent package
	err = r.UpdatePackageVersion("nonexistent", "1.0.0", "2026-04-05T10:00:00Z")
	if err == nil {
		t.Fatal("UpdatePackageVersion should fail for non-existent package")
	}
}

// TestRegistrySaveAndLoadWithAUR tests save/load with AUR fields
func TestRegistrySaveAndLoadWithAUR(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create registry and add packages with AUR fields
	r1, err := NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	pkgs := []*Package{
		{
			Name:         "bash",
			Version:      "5.1.16-2",
			Source:       "official",
			Repository:   "core",
			Files:        []string{"/bin/bash"},
			Executables:  []string{"/bin/bash"},
			Dependencies: []string{},
			InstallDate:  "2026-03-20T15:00:00Z",
			UpdateDate:   "2026-04-01T10:00:00Z",
		},
		{
			Name:         "yay",
			Version:      "12.0.0",
			Source:       "aur",
			Repository:   "aur",
			Files:        []string{"/usr/bin/yay"},
			Executables:  []string{"/usr/bin/yay"},
			Dependencies: []string{"pacman"},
			InstallDate:  "2026-04-01T10:00:00Z",
			UpdateDate:   "2026-04-01T10:00:00Z",
		},
	}

	for _, pkg := range pkgs {
		err := r1.AddPackage(pkg)
		if err != nil {
			t.Fatalf("Failed to add package: %v", err)
		}
	}

	// Save registry
	err = r1.Save()
	if err != nil {
		t.Fatalf("Failed to save registry: %v", err)
	}

	// Load registry in new instance
	r2, err := NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}

	// Verify packages were loaded correctly with AUR fields
	bash, ok := r2.GetPackage("bash")
	if !ok {
		t.Fatal("bash package not found after load")
	}

	if bash.Source != "official" || bash.Repository != "core" {
		t.Errorf("bash package fields incorrect after load")
	}

	yay, ok := r2.GetPackage("yay")
	if !ok {
		t.Fatal("yay package not found after load")
	}

	if yay.Source != "aur" || yay.Repository != "aur" {
		t.Errorf("yay package fields incorrect after load")
	}

	if yay.UpdateDate == "" {
		t.Error("yay UpdateDate should be preserved after load")
	}
}

// TestPackageSourceTracking tests that package source is consistently tracked
func TestPackageSourceTracking(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r, err := NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Add AUR package
	aurPkg := &Package{
		Name:       "custom-app",
		Version:    "1.0",
		Source:     "aur",
		Repository: "aur",
	}

	err = r.AddPackage(aurPkg)
	if err != nil {
		t.Fatalf("Failed to add AUR package: %v", err)
	}

	// Save and reload
	err = r.Save()
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	r2, err := NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Check source is preserved
	retrieved, ok := r2.GetPackage("custom-app")
	if !ok {
		t.Fatal("Package not found")
	}

	if retrieved.Source != "aur" {
		t.Errorf("Source should be 'aur', got %s", retrieved.Source)
	}
}

// TestListPackagesWithAUR tests that ListPackages includes AUR info
func TestListPackagesWithAUR(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r, err := NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Add packages with different sources
	pkgs := []*Package{
		{Name: "pkg1", Version: "1.0", Source: "official", Repository: "core"},
		{Name: "pkg2", Version: "2.0", Source: "aur", Repository: "aur"},
		{Name: "pkg3", Version: "3.0", Source: "official", Repository: "extra"},
	}

	for _, pkg := range pkgs {
		err := r.AddPackage(pkg)
		if err != nil {
			t.Fatalf("Failed to add package: %v", err)
		}
	}

	// List all packages
	all := r.ListPackages()
	if len(all) != 3 {
		t.Errorf("Expected 3 packages, got %d", len(all))
	}

	// Verify sources are present
	officialCount := 0
	aurCount := 0
	for _, pkg := range all {
		if pkg.Source == "official" {
			officialCount++
		} else if pkg.Source == "aur" {
			aurCount++
		}
	}

	if officialCount != 2 {
		t.Errorf("Expected 2 official packages, got %d", officialCount)
	}

	if aurCount != 1 {
		t.Errorf("Expected 1 AUR package, got %d", aurCount)
	}
}
