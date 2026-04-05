package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kodos-prj/chisel/pkg/alpm"
	"github.com/kodos-prj/chisel/pkg/config"
	"github.com/kodos-prj/chisel/pkg/database"
	"github.com/kodos-prj/chisel/pkg/registry"
)

// TestFullWorkflowIntegration tests the complete chisel workflow
func TestFullWorkflowIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "chisel-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set up configuration for testing
	baseDir := filepath.Join(tmpDir, "kod")
	// ALPM requires databases to be in var/lib/pacman structure
	alpmDBPath := filepath.Join(baseDir, "var/lib/pacman")
	syncPath := filepath.Join(alpmDBPath, "sync")
	storeDir := filepath.Join(baseDir, "store")
	wrapperDir := filepath.Join(baseDir, "wrappers")
	registryPath := filepath.Join(baseDir, "registry.json")
	cacheDir := filepath.Join(baseDir, "cache")

	// Create directories
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatalf("failed to create base dir: %v", err)
	}
	if err := os.MkdirAll(syncPath, 0755); err != nil {
		t.Fatalf("failed to create sync dir: %v", err)
	}
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		t.Fatalf("failed to create store dir: %v", err)
	}
	if err := os.MkdirAll(wrapperDir, 0755); err != nil {
		t.Fatalf("failed to create wrapper dir: %v", err)
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("failed to create cache dir: %v", err)
	}

	// Create configuration
	cfg := &config.Config{
		BaseDir:                baseDir,
		SymlinkRoot:            "/",
		StoreRoot:              storeDir,
		RegistryPath:           registryPath,
		AlpmRoot:               baseDir,
		AlpmDBPath:             alpmDBPath,
		DBPath:                 syncPath,
		WrapperDir:             wrapperDir,
		CachePath:              cacheDir,
		MirrorURL:              "https://mirror.rackspace.com/archlinux",
		Architecture:           "x86_64",
		Repositories:           []string{"core", "extra"},
		VerifySignatures:       false,
		MaxConcurrentDownloads: 5,
		DownloadTimeout:        300,
		KeepVersions:           3,
	}

	// Step 1: Sync databases
	t.Log("Step 1: Syncing databases...")
	// Database syncer needs to put files in the sync/ directory for ALPM to find them
	syncer := database.NewSyncer(
		cfg.MirrorURL,
		syncPath,
		cfg.Architecture,
		30*time.Second,
	)

	if err := syncer.Sync(cfg.Repositories); err != nil {
		t.Fatalf("failed to sync databases: %v", err)
	}
	t.Log("✓ Databases synced successfully")

	// Step 2: Test search functionality
	t.Log("Step 2: Testing search...")
	client, err := alpm.NewClient(cfg.AlpmRoot, cfg.AlpmDBPath)
	if err != nil {
		t.Fatalf("failed to create ALPM client: %v", err)
	}
	defer client.Close()

	if err := client.RegisterAllSyncDBs(cfg.Repositories); err != nil {
		t.Fatalf("failed to register sync databases: %v", err)
	}

	// Test searching for a package
	pkg, err := client.SearchPackage("bash")
	if err != nil {
		t.Fatalf("SearchPackage failed: %v", err)
	}
	if pkg == nil {
		t.Fatal("SearchPackage returned nil")
	}
	t.Logf("✓ Found package: %s %s", pkg.Name(), pkg.Version())

	// Step 3: Simulate install by updating registry
	t.Log("Step 3: Simulating installation in registry...")

	// Create registry and add packages
	reg, err := registry.NewRegistry(cfg.RegistryPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Add bash package to registry
	bashPkg := &registry.Package{
		Name:        "bash",
		Version:     "5.2.002-1", // Dummy version
		InstallDate: time.Now().Format(time.RFC3339),
	}

	if err := reg.AddPackage(bashPkg); err != nil {
		t.Logf("Warning: failed to add bash to registry: %v", err)
	}

	// Add coreutils package to registry
	coreutilsPkg := &registry.Package{
		Name:        "coreutils",
		Version:     "9.1-1", // Dummy version
		InstallDate: time.Now().Format(time.RFC3339),
	}

	if err := reg.AddPackage(coreutilsPkg); err != nil {
		t.Logf("Warning: failed to add coreutils to registry: %v", err)
	}

	if err := reg.Save(); err != nil {
		t.Fatalf("failed to save registry: %v", err)
	}

	t.Logf("✓ Installed packages to registry")

	// Step 4: Verify installed packages
	t.Log("Step 4: Verifying package installation...")

	// Check if packages are registered
	bashPkg, exists := reg.GetPackage("bash")
	if !exists {
		t.Error("bash package not found in registry")
	} else {
		t.Logf("✓ bash package installed (version: %s)", bashPkg.Version)
	}

	// Get coreutils package info
	coreutilsPkgInfo, exists := reg.GetPackage("coreutils")
	if !exists {
		t.Error("coreutils package not found in registry")
	} else {
		t.Logf("✓ coreutils package installed (version: %s)", coreutilsPkgInfo.Version)
	}

	// Step 5: Remove one package
	t.Log("Step 5: Removing one package...")

	// Remove from registry
	if err := reg.RemovePackage("bash"); err != nil {
		t.Fatalf("failed to remove bash from registry: %v", err)
	}

	if err := reg.Save(); err != nil {
		t.Fatalf("failed to save registry after removal: %v", err)
	}

	t.Logf("✓ Removed package: bash")

	// Step 6: Verify removal
	t.Log("Step 6: Verifying package removal...")

	// Check that bash is no longer in registry
	_, exists = reg.GetPackage("bash")
	if exists {
		t.Error("bash package still found in registry after removal")
	} else {
		t.Logf("✓ bash package successfully removed from registry")
	}

	// Check that coreutils is still present
	_, exists = reg.GetPackage("coreutils")
	if !exists {
		t.Error("coreutils package missing after removal of bash")
	} else {
		t.Logf("✓ coreutils package still present")
	}

	t.Log("✓ Full workflow test completed successfully")
}
