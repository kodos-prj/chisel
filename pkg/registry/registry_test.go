package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create a new registry
	r, err := NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	if r == nil {
		t.Fatal("Registry is nil")
	}

	if r.path != registryPath {
		t.Errorf("Expected path %s, got %s", registryPath, r.path)
	}
}

func TestAddAndGetPackage(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r, err := NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Add a package
	pkg := &Package{
		Name:         "test-package",
		Version:      "1.0.0",
		Files:        []string{"/usr/bin/test"},
		Dependencies: []string{"dep1", "dep2"},
		InstallDate:  "2026-03-21",
	}

	err = r.AddPackage(pkg)
	if err != nil {
		t.Fatalf("Failed to add package: %v", err)
	}

	// Retrieve the package
	retrieved, ok := r.GetPackage("test-package")
	if !ok {
		t.Fatal("Package not found")
	}

	if retrieved.Name != pkg.Name {
		t.Errorf("Expected name %s, got %s", pkg.Name, retrieved.Name)
	}

	if retrieved.Version != pkg.Version {
		t.Errorf("Expected version %s, got %s", pkg.Version, retrieved.Version)
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create registry and add a package
	r1, err := NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	pkg := &Package{
		Name:    "test-package",
		Version: "1.0.0",
		Files:   []string{"/usr/bin/test"},
	}

	r1.AddPackage(pkg)

	// Save the registry
	err = r1.Save()
	if err != nil {
		t.Fatalf("Failed to save registry: %v", err)
	}

	// Check that file was created
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		t.Fatal("Registry file was not created")
	}

	// Load the registry in a new instance
	r2, err := NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}

	// Verify the package was loaded
	retrieved, ok := r2.GetPackage("test-package")
	if !ok {
		t.Fatal("Package not found after load")
	}

	if retrieved.Name != pkg.Name || retrieved.Version != pkg.Version {
		t.Error("Package data mismatch after load")
	}
}

func TestRemovePackage(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	r, err := NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Add a package
	pkg := &Package{
		Name:    "test-package",
		Version: "1.0.0",
	}
	r.AddPackage(pkg)

	// Remove the package
	err = r.RemovePackage("test-package")
	if err != nil {
		t.Fatalf("Failed to remove package: %v", err)
	}

	// Verify it's gone
	_, ok := r.GetPackage("test-package")
	if ok {
		t.Error("Package still exists after removal")
	}
}
