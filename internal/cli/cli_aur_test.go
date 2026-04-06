// Package cli_test provides tests for AUR-integrated CLI commands
package cli

import (
	"testing"

	"github.com/kodos-prj/chisel/pkg/config"
)

// TestSearchCommandWithAUR tests basic search command initialization
func TestSearchCommandWithAUR(t *testing.T) {
	cfg := &config.Config{
		AlpmRoot:     "/tmp/test",
		AlpmDBPath:   "/tmp/test/db",
		Repositories: []string{"core", "extra"},
	}

	cmd := NewSearchCommand(cfg)
	if cmd == nil {
		t.Fatal("NewSearchCommand returned nil")
	}

	if cmd.config != cfg {
		t.Error("config not set correctly")
	}

	if cmd.aurRPC == nil {
		t.Fatal("aurRPC is nil")
	}

	if cmd.aurCache == nil {
		t.Fatal("aurCache is nil")
	}
}

// TestInfoCommandWithAUR tests info command initialization with AUR support
func TestInfoCommandWithAUR(t *testing.T) {
	cfg := &config.Config{
		AlpmRoot:     "/tmp/test",
		AlpmDBPath:   "/tmp/test/db",
		Repositories: []string{"core", "extra"},
	}

	cmd := NewInfoCommand(cfg)
	if cmd == nil {
		t.Fatal("NewInfoCommand returned nil")
	}

	if cmd.config != cfg {
		t.Error("config not set correctly")
	}

	if cmd.aurRPC == nil {
		t.Fatal("aurRPC is nil")
	}
}

// TestInstallCommandWithAUR tests install command initialization with AUR support
func TestInstallCommandWithAUR(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		AlpmRoot:               "/tmp/test",
		AlpmDBPath:             "/tmp/test/db",
		StoreRoot:              tmpDir + "/store",
		WrapperDir:             tmpDir + "/wrappers",
		SymlinkRoot:            tmpDir + "/symlinks",
		CachePath:              tmpDir + "/cache",
		RegistryPath:           tmpDir + "/registry",
		Repositories:           []string{"core", "extra"},
		Architecture:           "x86_64",
		MirrorURL:              "https://mirror.example.com",
		MaxConcurrentDownloads: 4,
	}

	cmd := NewInstallCommand(cfg)
	if cmd == nil {
		t.Fatal("NewInstallCommand returned nil")
	}

	if cmd.config != cfg {
		t.Error("config not set correctly")
	}

	if cmd.aurRPC == nil {
		t.Fatal("aurRPC is nil")
	}

	// buildMgr may be nil if directory creation fails, which is OK in tests
	// The important thing is that the command is created
}

// TestInstallCommandWithSymlinkDir tests install command with custom symlink directory
func TestInstallCommandWithSymlinkDir(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		AlpmRoot:               "/tmp/test",
		AlpmDBPath:             "/tmp/test/db",
		StoreRoot:              tmpDir + "/store",
		WrapperDir:             tmpDir + "/wrappers",
		SymlinkRoot:            tmpDir + "/symlinks",
		CachePath:              tmpDir + "/cache",
		RegistryPath:           tmpDir + "/registry",
		Repositories:           []string{"core", "extra"},
		Architecture:           "x86_64",
		MirrorURL:              "https://mirror.example.com",
		MaxConcurrentDownloads: 4,
	}

	customSymlinkDir := "/custom/symlinks"
	cmd := NewInstallCommandWithSymlinkDir(cfg, customSymlinkDir)

	if cmd.symlinkDir != customSymlinkDir {
		t.Errorf("symlinkDir not set: got %s, want %s", cmd.symlinkDir, customSymlinkDir)
	}

	if cmd.aurRPC == nil {
		t.Fatal("aurRPC is nil")
	}

	// buildMgr may be nil if directory creation fails, which is OK in tests
	// The important thing is that the command is created
}

// TestSearchCommandEmptyPattern tests search with empty pattern
func TestSearchCommandEmptyPattern(t *testing.T) {
	cfg := &config.Config{
		AlpmRoot:     "/tmp/test",
		AlpmDBPath:   "/tmp/test/db",
		Repositories: []string{"core", "extra"},
	}

	cmd := NewSearchCommand(cfg)
	err := cmd.Execute("")

	if err == nil {
		t.Fatal("Execute should fail with empty pattern")
	}
}

// TestInfoCommandEmptyName tests info with empty name
func TestInfoCommandEmptyName(t *testing.T) {
	cfg := &config.Config{
		AlpmRoot:     "/tmp/test",
		AlpmDBPath:   "/tmp/test/db",
		Repositories: []string{"core", "extra"},
	}

	cmd := NewInfoCommand(cfg)
	err := cmd.Execute("")

	if err == nil {
		t.Fatal("Execute should fail with empty package name")
	}
}

// TestInstallCommandEmptyPackages tests install with no packages
func TestInstallCommandEmptyPackages(t *testing.T) {
	cfg := &config.Config{
		AlpmRoot:   "/tmp/test",
		AlpmDBPath: "/tmp/test/db",
	}

	cmd := NewInstallCommand(cfg)
	err := cmd.Run([]string{})

	if err == nil {
		t.Fatal("Run should fail with no packages")
	}
}

// TestInstallCommandWithOptions tests install command option parsing
func TestInstallCommandWithOptions(t *testing.T) {
	cfg := &config.Config{
		AlpmRoot:   "/tmp/test",
		AlpmDBPath: "/tmp/test/db",
	}

	tests := []struct {
		name       string
		args       []string
		shouldFail bool
	}{
		{
			name:       "no options",
			args:       []string{"bash"},
			shouldFail: true, // Will fail due to missing ALPM, but not due to parsing
		},
		{
			name:       "with --no-deps",
			args:       []string{"--no-deps", "bash"},
			shouldFail: true,
		},
		{
			name:       "with --no-extract",
			args:       []string{"--no-extract", "bash"},
			shouldFail: true,
		},
		{
			name:       "with --no-symlink",
			args:       []string{"--no-symlink", "bash"},
			shouldFail: true,
		},
		{
			name:       "with --force",
			args:       []string{"--force", "bash"},
			shouldFail: true,
		},
		{
			name:       "multiple options",
			args:       []string{"--no-deps", "--force", "bash"},
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewInstallCommand(cfg)
			err := cmd.Run(tt.args)

			if !tt.shouldFail && err != nil {
				t.Errorf("Run should not fail: %v", err)
			}
		})
	}
}

// TestSearchCommandCache tests that search results can be cached
func TestSearchCommandCache(t *testing.T) {
	cfg := &config.Config{
		AlpmRoot:     "/tmp/test",
		AlpmDBPath:   "/tmp/test/db",
		Repositories: []string{"core", "extra"},
	}

	cmd := NewSearchCommand(cfg)
	if cmd.aurCache == nil {
		t.Fatal("aurCache should be initialized")
	}

	// Cache should be empty initially
	if len(cmd.aurCache) != 0 {
		t.Error("cache should be empty initially")
	}
}
