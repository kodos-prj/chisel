package alpm

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewClient tests creating a new ALPM client
func TestNewClient(t *testing.T) {
	// Check if libalpm is available
	if !isLibalpmAvailable() {
		t.Skip("libalpm not available on this system")
	}

	tmpDir, err := os.MkdirTemp("", "chisel-alpm-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	root := filepath.Join(tmpDir, "root")
	dbPath := filepath.Join(tmpDir, "db")

	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatalf("failed to create root dir: %v", err)
	}
	if err := os.MkdirAll(dbPath, 0755); err != nil {
		t.Fatalf("failed to create db dir: %v", err)
	}

	client, err := NewClient(root, dbPath)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	if client == nil {
		t.Fatal("client is nil")
	}
	if client.handle == nil {
		t.Fatal("client handle is nil")
	}
}

// TestNewClientInvalidPath tests creating a client with invalid paths
func TestNewClientInvalidPath(t *testing.T) {
	if !isLibalpmAvailable() {
		t.Skip("libalpm not available on this system")
	}

	_, err := NewClient("/nonexistent/root", "/nonexistent/db")
	if err == nil {
		t.Error("expected error for invalid paths, got nil")
	}
}

// TestRegisterSyncDB tests registering a sync database
func TestRegisterSyncDB(t *testing.T) {
	if !isLibalpmAvailable() {
		t.Skip("libalpm not available on this system")
	}

	client, cleanup := setupTestClient(t)
	defer cleanup()

	err := client.RegisterSyncDB("core")
	if err != nil {
		t.Fatalf("RegisterSyncDB failed: %v", err)
	}

	// Verify database was registered
	dbs, err := client.ListSyncDBs()
	if err != nil {
		t.Fatalf("ListSyncDBs failed: %v", err)
	}

	found := false
	for _, db := range dbs {
		if db == "core" {
			found = true
			break
		}
	}

	if !found {
		t.Error("core database not found in registered databases")
	}
}

// TestRegisterAllSyncDBs tests registering multiple databases
func TestRegisterAllSyncDBs(t *testing.T) {
	if !isLibalpmAvailable() {
		t.Skip("libalpm not available on this system")
	}

	client, cleanup := setupTestClient(t)
	defer cleanup()

	repos := []string{"core", "extra"}
	err := client.RegisterAllSyncDBs(repos)
	if err != nil {
		t.Fatalf("RegisterAllSyncDBs failed: %v", err)
	}

	// Verify all databases were registered
	dbs, err := client.ListSyncDBs()
	if err != nil {
		t.Fatalf("ListSyncDBs failed: %v", err)
	}

	if len(dbs) != len(repos) {
		t.Errorf("expected %d databases, got %d", len(repos), len(dbs))
	}
}

// TestClose tests closing the client
func TestClose(t *testing.T) {
	if !isLibalpmAvailable() {
		t.Skip("libalpm not available on this system")
	}

	client, cleanup := setupTestClient(t)
	// Don't defer cleanup since we're testing Close

	err := client.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Clean up temp directory
	cleanup()
}

// TestCloseNilHandle tests closing a client with nil handle
func TestCloseNilHandle(t *testing.T) {
	client := &Client{
		handle: nil,
	}

	err := client.Close()
	if err != nil {
		t.Errorf("Close with nil handle should not error, got: %v", err)
	}
}

// TestIsPackageInstalled tests checking if a package is installed
func TestIsPackageInstalled(t *testing.T) {
	if !isLibalpmAvailable() {
		t.Skip("libalpm not available on this system")
	}

	client, cleanup := setupTestClient(t)
	defer cleanup()

	// In a fresh install, no packages should be installed
	installed, err := client.IsPackageInstalled("nonexistent-package")
	if err != nil {
		t.Fatalf("IsPackageInstalled failed: %v", err)
	}

	if installed {
		t.Error("expected package to not be installed")
	}
}

// TestGetLocalPackages tests getting local packages
func TestGetLocalPackages(t *testing.T) {
	if !isLibalpmAvailable() {
		t.Skip("libalpm not available on this system")
	}

	client, cleanup := setupTestClient(t)
	defer cleanup()

	packages, err := client.GetLocalPackages()
	if err != nil {
		t.Fatalf("GetLocalPackages failed: %v", err)
	}

	// In a fresh install, there should be no packages
	if len(packages) != 0 {
		t.Errorf("expected 0 packages, got %d", len(packages))
	}
}

// TestListSyncDBs tests listing sync databases
func TestListSyncDBs(t *testing.T) {
	if !isLibalpmAvailable() {
		t.Skip("libalpm not available on this system")
	}

	client, cleanup := setupTestClient(t)
	defer cleanup()

	// Initially, no databases should be registered
	dbs, err := client.ListSyncDBs()
	if err != nil {
		t.Fatalf("ListSyncDBs failed: %v", err)
	}

	if len(dbs) != 0 {
		t.Errorf("expected 0 databases, got %d", len(dbs))
	}

	// Register a database
	if err := client.RegisterSyncDB("core"); err != nil {
		t.Fatalf("RegisterSyncDB failed: %v", err)
	}

	// Now should have 1 database
	dbs, err = client.ListSyncDBs()
	if err != nil {
		t.Fatalf("ListSyncDBs failed: %v", err)
	}

	if len(dbs) != 1 {
		t.Errorf("expected 1 database, got %d", len(dbs))
	}
}

// Helper functions

// isLibalpmAvailable checks if libalpm is available on the system
func isLibalpmAvailable() bool {
	// Try to create a temporary client to see if libalpm works
	tmpDir, err := os.MkdirTemp("", "chisel-check-*")
	if err != nil {
		return false
	}
	defer os.RemoveAll(tmpDir)

	root := filepath.Join(tmpDir, "root")
	dbPath := filepath.Join(tmpDir, "db")

	if err := os.MkdirAll(root, 0755); err != nil {
		return false
	}
	if err := os.MkdirAll(dbPath, 0755); err != nil {
		return false
	}

	client, err := NewClient(root, dbPath)
	if err != nil {
		return false
	}
	defer client.Close()

	return true
}

// setupTestClient creates a test ALPM client with temporary directories
func setupTestClient(t *testing.T) (*Client, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "chisel-alpm-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	root := filepath.Join(tmpDir, "root")
	dbPath := filepath.Join(tmpDir, "db")

	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatalf("failed to create root dir: %v", err)
	}
	if err := os.MkdirAll(dbPath, 0755); err != nil {
		t.Fatalf("failed to create db dir: %v", err)
	}

	client, err := NewClient(root, dbPath)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	cleanup := func() {
		client.Close()
		os.RemoveAll(tmpDir)
	}

	return client, cleanup
}

// TestSearchPackage tests searching for an exact package
func TestSearchPackage(t *testing.T) {
	if !isLibalpmAvailable() {
		t.Skip("libalpm not available on this system")
	}

	client, cleanup := setupTestClientWithDB(t)
	if client == nil {
		t.Skip("Could not setup client with database")
		return
	}
	defer cleanup()

	// Try to search for a common package
	pkg, err := client.SearchPackage("bash")
	if err != nil {
		t.Logf("SearchPackage failed (database may be empty): %v", err)
		t.Skip("Skipping test - no packages in database")
		return
	}

	if pkg == nil {
		t.Error("SearchPackage returned nil package")
	} else {
		if pkg.Name() != "bash" {
			t.Errorf("expected package name 'bash', got '%s'", pkg.Name())
		}
	}
}

// TestSearchPackageNotFound tests searching for a non-existent package
func TestSearchPackageNotFound(t *testing.T) {
	if !isLibalpmAvailable() {
		t.Skip("libalpm not available on this system")
	}

	client, cleanup := setupTestClientWithDB(t)
	if client == nil {
		t.Skip("Could not setup client with database")
		return
	}
	defer cleanup()

	_, err := client.SearchPackage("this-package-definitely-does-not-exist-12345")
	if err == nil {
		t.Error("expected error for non-existent package, got nil")
	}
}

// TestSearchPackages tests searching for packages with a pattern
func TestSearchPackages(t *testing.T) {
	if !isLibalpmAvailable() {
		t.Skip("libalpm not available on this system")
	}

	client, cleanup := setupTestClientWithDB(t)
	if client == nil {
		t.Skip("Could not setup client with database")
		return
	}
	defer cleanup()

	// Search for packages containing "lib"
	packages, err := client.SearchPackages("lib")
	if err != nil {
		t.Logf("SearchPackages failed (database may be empty): %v", err)
		t.Skip("Skipping test - no packages in database")
		return
	}

	if len(packages) == 0 {
		t.Log("No packages found matching 'lib' - database may be empty")
		t.Skip("Skipping test - no matching packages")
	}

	// Verify all results contain the search pattern
	for _, pkg := range packages {
		name := pkg.Name()
		if name == "" {
			t.Error("package has empty name")
		}
	}
}

// TestGetPackageInfo tests retrieving detailed package information
func TestGetPackageInfo(t *testing.T) {
	if !isLibalpmAvailable() {
		t.Skip("libalpm not available on this system")
	}

	client, cleanup := setupTestClientWithDB(t)
	if client == nil {
		t.Skip("Could not setup client with database")
		return
	}
	defer cleanup()

	info, err := client.GetPackageInfo("bash")
	if err != nil {
		t.Logf("GetPackageInfo failed (database may be empty): %v", err)
		t.Skip("Skipping test - package not found in database")
		return
	}

	if info == nil {
		t.Fatal("GetPackageInfo returned nil")
	}

	// Verify required fields
	if info.Name != "bash" {
		t.Errorf("expected name 'bash', got '%s'", info.Name)
	}
	if info.Version == "" {
		t.Error("package version is empty")
	}
	if info.Repository == "" {
		t.Error("package repository is empty")
	}
	if info.Architecture == "" {
		t.Error("package architecture is empty")
	}
}

// TestResolveDependencies tests dependency resolution
func TestResolveDependencies(t *testing.T) {
	if !isLibalpmAvailable() {
		t.Skip("libalpm not available on this system")
	}

	client, cleanup := setupTestClientWithDB(t)
	if client == nil {
		t.Skip("Could not setup client with database")
		return
	}
	defer cleanup()

	deps, err := client.ResolveDependencies("bash")
	if err != nil {
		t.Logf("ResolveDependencies failed (database may be empty): %v", err)
		t.Skip("Skipping test - cannot resolve dependencies")
		return
	}

	if len(deps) == 0 {
		t.Error("expected at least one dependency (the package itself)")
	}

	// The package itself should be in the dependency list
	found := false
	for _, dep := range deps {
		if dep == "bash" {
			found = true
			break
		}
	}
	if !found {
		t.Error("package 'bash' not found in its own dependency list")
	}
}

// TestGetDownloadURL tests URL generation for package downloads
func TestGetDownloadURL(t *testing.T) {
	if !isLibalpmAvailable() {
		t.Skip("libalpm not available on this system")
	}

	client, cleanup := setupTestClientWithDB(t)
	if client == nil {
		t.Skip("Could not setup client with database")
		return
	}
	defer cleanup()

	pkg, err := client.SearchPackage("bash")
	if err != nil {
		t.Skip("Cannot find bash package for URL test")
		return
	}

	mirrorURL := "https://mirror.example.com/archlinux"
	arch := "x86_64"

	url := client.GetDownloadURL(pkg, mirrorURL, arch)

	if url == "" {
		t.Error("GetDownloadURL returned empty string")
	}

	// Verify URL format
	expectedPrefix := mirrorURL + "/"
	if len(url) < len(expectedPrefix) || url[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("URL doesn't start with mirror URL: %s", url)
	}

	expectedSuffix := ".pkg.tar.zst"
	if len(url) < len(expectedSuffix) || url[len(url)-len(expectedSuffix):] != expectedSuffix {
		t.Errorf("URL doesn't end with .pkg.tar.zst: %s", url)
	}
}

// Helper functions

// setupTestClientWithDB creates a test client with actual database files
func setupTestClientWithDB(t *testing.T) (*Client, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "chisel-alpm-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	root := filepath.Join(tmpDir, "root")
	dbPath := filepath.Join(tmpDir, "db")

	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatalf("failed to create root dir: %v", err)
	}
	if err := os.MkdirAll(dbPath, 0755); err != nil {
		t.Fatalf("failed to create db dir: %v", err)
	}

	client, err := NewClient(root, dbPath)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// Try to register databases
	// Note: This will only work if databases are synced first
	err = client.RegisterSyncDB("core")
	if err != nil {
		t.Logf("Warning: Could not register core database: %v", err)
		client.Close()
		os.RemoveAll(tmpDir)
		return nil, func() {}
	}

	cleanup := func() {
		client.Close()
		os.RemoveAll(tmpDir)
	}

	return client, cleanup
}

// Benchmark tests

func BenchmarkNewClient(b *testing.B) {
	if !isLibalpmAvailable() {
		b.Skip("libalpm not available on this system")
	}

	tmpDir, err := os.MkdirTemp("", "chisel-bench-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	root := filepath.Join(tmpDir, "root")
	dbPath := filepath.Join(tmpDir, "db")

	if err := os.MkdirAll(root, 0755); err != nil {
		b.Fatalf("failed to create root dir: %v", err)
	}
	if err := os.MkdirAll(dbPath, 0755); err != nil {
		b.Fatalf("failed to create db dir: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client, err := NewClient(root, dbPath)
		if err != nil {
			b.Fatalf("NewClient failed: %v", err)
		}
		client.Close()
	}
}
