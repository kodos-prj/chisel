package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.BaseDir != DefaultBaseDir {
		t.Errorf("Expected BaseDir %s, got %s", DefaultBaseDir, cfg.BaseDir)
	}

	if cfg.SymlinkRoot != DefaultSymlinkRoot {
		t.Errorf("Expected SymlinkRoot %s, got %s", DefaultSymlinkRoot, cfg.SymlinkRoot)
	}

	expectedStore := filepath.Join(DefaultBaseDir, "store")
	if cfg.StoreRoot != expectedStore {
		t.Errorf("Expected StoreRoot %s, got %s", expectedStore, cfg.StoreRoot)
	}

	expectedRegistry := filepath.Join(DefaultBaseDir, "registry.json")
	if cfg.RegistryPath != expectedRegistry {
		t.Errorf("Expected RegistryPath %s, got %s", expectedRegistry, cfg.RegistryPath)
	}

	if cfg.KeepVersions != 3 {
		t.Errorf("Expected KeepVersions 3, got %d", cfg.KeepVersions)
	}
}

func TestConfigNormalize(t *testing.T) {
	// Test empty config gets normalized to defaults
	cfg := &Config{}
	cfg.Normalize()

	if cfg.BaseDir != DefaultBaseDir {
		t.Errorf("Expected BaseDir to be normalized to %s, got %s", DefaultBaseDir, cfg.BaseDir)
	}

	if cfg.SymlinkRoot != DefaultSymlinkRoot {
		t.Errorf("Expected SymlinkRoot to be normalized to %s, got %s", DefaultSymlinkRoot, cfg.SymlinkRoot)
	}

	if cfg.KeepVersions != 3 {
		t.Errorf("Expected KeepVersions to be normalized to 3, got %d", cfg.KeepVersions)
	}
}

func TestConfigNormalizeCustomBaseDir(t *testing.T) {
	// Test that custom base_dir is respected and paths are derived
	cfg := &Config{
		BaseDir: "/custom/base",
	}
	cfg.Normalize()

	if cfg.BaseDir != "/custom/base" {
		t.Errorf("Expected BaseDir /custom/base, got %s", cfg.BaseDir)
	}

	expectedStore := "/custom/base/store"
	if cfg.StoreRoot != expectedStore {
		t.Errorf("Expected StoreRoot %s, got %s", expectedStore, cfg.StoreRoot)
	}

	expectedRegistry := "/custom/base/registry.json"
	if cfg.RegistryPath != expectedRegistry {
		t.Errorf("Expected RegistryPath %s, got %s", expectedRegistry, cfg.RegistryPath)
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create a custom config
	cfg1 := &Config{
		BaseDir:      "/custom/kod",
		SymlinkRoot:  "/custom/root",
		StoreRoot:    "/custom/kod/store",
		RegistryPath: "/custom/kod/registry.json",
		AlpmRoot:     "/",
		AlpmDBPath:   "/var/lib/pacman",
		KeepVersions: 5,
	}

	// Save it
	err := cfg1.Save(configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Check file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load it back
	cfg2, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify values
	if cfg2.BaseDir != cfg1.BaseDir {
		t.Errorf("BaseDir mismatch: expected %s, got %s", cfg1.BaseDir, cfg2.BaseDir)
	}

	if cfg2.SymlinkRoot != cfg1.SymlinkRoot {
		t.Errorf("SymlinkRoot mismatch: expected %s, got %s", cfg1.SymlinkRoot, cfg2.SymlinkRoot)
	}

	if cfg2.KeepVersions != cfg1.KeepVersions {
		t.Errorf("KeepVersions mismatch: expected %d, got %d", cfg1.KeepVersions, cfg2.KeepVersions)
	}
}

func TestLoadNonExistentConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent.json")

	// Loading a non-existent config should return default config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Expected no error for non-existent config, got: %v", err)
	}

	// Should have default values
	if cfg.BaseDir != DefaultBaseDir {
		t.Errorf("Expected default BaseDir %s, got %s", DefaultBaseDir, cfg.BaseDir)
	}
}

func TestLoadPartialConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "partial.json")

	// Write a partial config (only base_dir)
	partialJSON := `{
  "base_dir": "/opt/packmgr"
}`
	err := os.WriteFile(configPath, []byte(partialJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write partial config: %v", err)
	}

	// Load it
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load partial config: %v", err)
	}

	// base_dir should be set
	if cfg.BaseDir != "/opt/packmgr" {
		t.Errorf("Expected BaseDir /opt/packmgr, got %s", cfg.BaseDir)
	}

	// Other fields should be normalized to defaults based on base_dir
	expectedStore := "/opt/packmgr/store"
	if cfg.StoreRoot != expectedStore {
		t.Errorf("Expected StoreRoot %s, got %s", expectedStore, cfg.StoreRoot)
	}

	expectedRegistry := "/opt/packmgr/registry.json"
	if cfg.RegistryPath != expectedRegistry {
		t.Errorf("Expected RegistryPath %s, got %s", expectedRegistry, cfg.RegistryPath)
	}

	if cfg.KeepVersions != 3 {
		t.Errorf("Expected KeepVersions 3, got %d", cfg.KeepVersions)
	}
}
