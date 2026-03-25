package alpm

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kodos-prj/chisel/pkg/database"
)

// TestSearchPackageIntegration tests searching with a real database
func TestSearchPackageIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if !isLibalpmAvailable() {
		t.Skip("libalpm not available on this system")
	}

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "chisel-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	root := filepath.Join(tmpDir, "root")
	dbPath := filepath.Join(root, "var/lib/pacman")
	syncPath := filepath.Join(dbPath, "sync")

	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatalf("failed to create root dir: %v", err)
	}
	if err := os.MkdirAll(syncPath, 0755); err != nil {
		t.Fatalf("failed to create sync path: %v", err)
	}

	// Download database
	t.Log("Downloading database from Arch mirror...")
	syncer := database.NewSyncer(
		"https://mirror.rackspace.com/archlinux",
		syncPath,
		"x86_64",
		30*time.Second,
	)

	if err := syncer.Sync([]string{"core"}); err != nil {
		t.Fatalf("failed to sync database: %v", err)
	}
	t.Log("Database synced successfully")

	// Create ALPM client
	client, err := NewClient(root, dbPath)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Register database
	if err := client.RegisterSyncDB("core"); err != nil {
		t.Fatalf("failed to register sync database: %v", err)
	}

	// Test SearchPackage
	t.Run("SearchPackage", func(t *testing.T) {
		pkg, err := client.SearchPackage("bash")
		if err != nil {
			t.Fatalf("SearchPackage failed: %v", err)
		}

		if pkg == nil {
			t.Fatal("SearchPackage returned nil")
		}

		if pkg.Name() != "bash" {
			t.Errorf("expected package name 'bash', got '%s'", pkg.Name())
		}

		t.Logf("Found package: %s %s", pkg.Name(), pkg.Version())
	})

	// Test SearchPackageNotFound
	t.Run("SearchPackageNotFound", func(t *testing.T) {
		_, err := client.SearchPackage("nonexistent-package-xyz-123")
		if err == nil {
			t.Error("expected error for non-existent package, got nil")
		}
	})

	// Test SearchPackages
	t.Run("SearchPackages", func(t *testing.T) {
		packages, err := client.SearchPackages("core")
		if err != nil {
			t.Fatalf("SearchPackages failed: %v", err)
		}

		if len(packages) == 0 {
			t.Error("expected at least one package matching 'core'")
		}

		t.Logf("Found %d packages matching 'core'", len(packages))

		// Verify first package
		if len(packages) > 0 {
			pkg := packages[0]
			t.Logf("First package: %s %s - %s", pkg.Name(), pkg.Version(), pkg.Description())
		}
	})

	// Test GetPackageInfo
	t.Run("GetPackageInfo", func(t *testing.T) {
		info, err := client.GetPackageInfo("bash")
		if err != nil {
			t.Fatalf("GetPackageInfo failed: %v", err)
		}

		if info == nil {
			t.Fatal("GetPackageInfo returned nil")
		}

		// Verify all fields
		if info.Name != "bash" {
			t.Errorf("expected name 'bash', got '%s'", info.Name)
		}
		if info.Version == "" {
			t.Error("version is empty")
		}
		if info.Description == "" {
			t.Error("description is empty")
		}
		if info.Architecture == "" {
			t.Error("architecture is empty")
		}
		if info.Repository != "core" {
			t.Errorf("expected repository 'core', got '%s'", info.Repository)
		}

		t.Logf("Package info:")
		t.Logf("  Name: %s", info.Name)
		t.Logf("  Version: %s", info.Version)
		t.Logf("  Description: %s", info.Description)
		t.Logf("  Repository: %s", info.Repository)
		t.Logf("  Download Size: %.2f MB", float64(info.DownloadSize)/(1024*1024))
		t.Logf("  Installed Size: %.2f MB", float64(info.Size)/(1024*1024))
	})

	// Test ResolveDependencies
	t.Run("ResolveDependencies", func(t *testing.T) {
		deps, err := client.ResolveDependencies("bash")
		if err != nil {
			t.Fatalf("ResolveDependencies failed: %v", err)
		}

		if len(deps) == 0 {
			t.Error("expected at least one dependency")
		}

		// Bash should be in the list
		found := false
		for _, dep := range deps {
			if dep == "bash" {
				found = true
			}
		}
		if !found {
			t.Error("bash not found in its own dependency list")
		}

		t.Logf("Dependencies for bash (%d total):", len(deps))
		for i, dep := range deps {
			t.Logf("  %d. %s", i+1, dep)
		}
	})

	// Test GetDownloadURL
	t.Run("GetDownloadURL", func(t *testing.T) {
		pkg, err := client.SearchPackage("bash")
		if err != nil {
			t.Fatalf("SearchPackage failed: %v", err)
		}

		mirrorURL := "https://mirror.example.com/archlinux"
		arch := "x86_64"

		url := client.GetDownloadURL(pkg, mirrorURL, arch)

		if url == "" {
			t.Error("GetDownloadURL returned empty string")
		}

		// Verify URL format
		expectedPrefix := mirrorURL + "/core/os/" + arch + "/"
		if len(url) < len(expectedPrefix) || url[:len(expectedPrefix)] != expectedPrefix {
			t.Errorf("URL doesn't have expected prefix.\nExpected prefix: %s\nGot: %s", expectedPrefix, url)
		}

		expectedSuffix := ".pkg.tar.zst"
		if len(url) < len(expectedSuffix) || url[len(url)-len(expectedSuffix):] != expectedSuffix {
			t.Errorf("URL doesn't end with .pkg.tar.zst: %s", url)
		}

		t.Logf("Download URL: %s", url)
	})
}

// TestSearchMultipleRepositories tests searching across multiple repositories
func TestSearchMultipleRepositories(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if !isLibalpmAvailable() {
		t.Skip("libalpm not available on this system")
	}

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "chisel-multi-repo-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	root := filepath.Join(tmpDir, "root")
	dbPath := filepath.Join(root, "var/lib/pacman")
	syncPath := filepath.Join(dbPath, "sync")

	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatalf("failed to create root dir: %v", err)
	}
	if err := os.MkdirAll(syncPath, 0755); err != nil {
		t.Fatalf("failed to create sync path: %v", err)
	}

	// Download databases (core and extra)
	t.Log("Downloading databases...")
	syncer := database.NewSyncer(
		"https://mirror.rackspace.com/archlinux",
		syncPath,
		"x86_64",
		60*time.Second,
	)

	if err := syncer.Sync([]string{"core", "extra"}); err != nil {
		t.Fatalf("failed to sync databases: %v", err)
	}
	t.Log("Databases synced successfully")

	// Create ALPM client
	client, err := NewClient(root, dbPath)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Register both databases
	if err := client.RegisterAllSyncDBs([]string{"core", "extra"}); err != nil {
		t.Fatalf("failed to register sync databases: %v", err)
	}

	// Search for a package that might be in either repository
	packages, err := client.SearchPackages("vim")
	if err != nil {
		t.Fatalf("SearchPackages failed: %v", err)
	}

	if len(packages) == 0 {
		t.Error("expected at least one package matching 'vim'")
	}

	t.Logf("Found %d packages matching 'vim':", len(packages))
	for _, pkg := range packages {
		t.Logf("  %s/%s %s", pkg.DB().Name(), pkg.Name(), pkg.Version())
	}
}
