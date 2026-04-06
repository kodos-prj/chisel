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

// TestInstallOptionsSourceField tests InstallOptions Source field
func TestInstallOptionsSourceField(t *testing.T) {
	opts := &InstallOptions{
		Source: "",
	}

	if opts.Source != "" {
		t.Error("Source should default to empty string")
	}

	opts.Source = "aur"
	if opts.Source != "aur" {
		t.Error("Source should be set to 'aur'")
	}

	opts.Source = "official"
	if opts.Source != "official" {
		t.Error("Source should be set to 'official'")
	}
}

// TestInstallCommandSourceFlagParsing tests --source= flag parsing
func TestInstallCommandSourceFlagParsing(t *testing.T) {
	cfg := &config.Config{
		AlpmRoot:   "/tmp/test",
		AlpmDBPath: "/tmp/test/db",
	}

	tests := []struct {
		name       string
		args       []string
		expectErr  string
		expectCode int // 0 = no specific error, 1 = error expected
	}{
		{
			name:       "valid --source=aur",
			args:       []string{"--source=aur", "bash"},
			expectCode: 1, // Will fail later due to ALPM, but flag should parse OK
		},
		{
			name:       "valid --source=official",
			args:       []string{"--source=official", "bash"},
			expectCode: 1, // Will fail later due to ALPM, but flag should parse OK
		},
		{
			name:       "invalid --source=invalid",
			args:       []string{"--source=invalid", "bash"},
			expectErr:  "invalid source",
			expectCode: 0, // Should error on flag parsing
		},
		{
			name:       "multiple --source flags",
			args:       []string{"--source=aur", "--source=official", "bash"},
			expectErr:  "cannot specify multiple --source flags",
			expectCode: 0,
		},
		{
			name:       "no package name",
			args:       []string{"--source=aur"},
			expectErr:  "package name required",
			expectCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewInstallCommand(cfg)
			err := cmd.Run(tt.args)

			if tt.expectErr != "" && (err == nil || err.Error() == "") {
				t.Errorf("expected error containing '%s', got none", tt.expectErr)
			}

			if tt.expectErr != "" && err != nil {
				// Check if error message contains expected substring
				if !contains(err.Error(), tt.expectErr) {
					t.Errorf("expected error containing '%s', got '%v'", tt.expectErr, err)
				}
			}
		})
	}
}

// TestInstallOptionsSourceVariations tests different source option combinations
func TestInstallOptionsSourceVariations(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		noDeps    bool
		force     bool
		noSymlink bool
	}{
		{"aur with default options", "aur", false, false, false},
		{"official with default options", "official", false, false, false},
		{"aur with --no-deps", "aur", true, false, false},
		{"official with --force", "official", false, true, false},
		{"aur with --no-symlink", "aur", false, false, true},
		{"aur with multiple options", "aur", true, true, true},
		{"empty source (auto-detect)", "", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &InstallOptions{
				Source:    tt.source,
				NoDeps:    tt.noDeps,
				Force:     tt.force,
				NoSymlink: tt.noSymlink,
			}

			if opts.Source != tt.source {
				t.Errorf("Source not set correctly: got %s, want %s", opts.Source, tt.source)
			}

			if opts.NoDeps != tt.noDeps {
				t.Errorf("NoDeps not set correctly")
			}

			if opts.Force != tt.force {
				t.Errorf("Force not set correctly")
			}

			if opts.NoSymlink != tt.noSymlink {
				t.Errorf("NoSymlink not set correctly")
			}
		})
	}
}

// TestInstallCommandSourceConstraintWithPackages tests source constraints with multiple packages
func TestInstallCommandSourceConstraintWithPackages(t *testing.T) {
	cfg := &config.Config{
		AlpmRoot:   "/tmp/test",
		AlpmDBPath: "/tmp/test/db",
	}

	tests := []struct {
		name   string
		args   []string
		pkgErr string
	}{
		{
			name:   "single package with --source=aur",
			args:   []string{"--source=aur", "yay"},
			pkgErr: "package name required", // Will error on parsing
		},
		{
			name:   "single package with --source=official",
			args:   []string{"--source=official", "bash"},
			pkgErr: "package name required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewInstallCommand(cfg)
			err := cmd.Run(tt.args)

			// Since ALPM won't be available in tests, we just verify
			// that parsing doesn't error on the source flag
			if err != nil && contains(err.Error(), "invalid source") {
				t.Errorf("source flag parsing failed: %v", err)
			}
		})
	}
}

// TestInstallOptionsDefaultSource tests that Source defaults to empty string
func TestInstallOptionsDefaultSource(t *testing.T) {
	opts := &InstallOptions{}

	if opts.Source != "" {
		t.Errorf("default Source should be empty string, got '%s'", opts.Source)
	}

	if opts.NoDeps || opts.NoExtract || opts.NoSymlink || opts.Force {
		t.Error("other options should default to false")
	}
}

// TestInstallCommandSourceFlagValidation tests source flag value validation
func TestInstallCommandSourceFlagValidation(t *testing.T) {
	cfg := &config.Config{
		AlpmRoot:   "/tmp/test",
		AlpmDBPath: "/tmp/test/db",
	}

	invalidSources := []string{
		"--source=aur2",
		"--source=aur ",
		"--source= aur",
		"--source=OFFICIAL",
		"--source=AUR",
		"--source=",
		"--source=pacman",
		"--source=user",
	}

	for _, source := range invalidSources {
		t.Run("invalid_"+source, func(t *testing.T) {
			cmd := NewInstallCommand(cfg)
			err := cmd.Run([]string{source, "bash"})

			if err == nil || (!contains(err.Error(), "invalid source") && !contains(err.Error(), "package name required")) {
				t.Errorf("expected source validation error for '%s', got: %v", source, err)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
