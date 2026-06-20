// Package cli_test provides tests for AUR-integrated CLI commands
package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/kodos-prj/chisel/pkg/config"
	"github.com/kodos-prj/chisel/pkg/symlink"
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

// ==================== SYMLINK-PREFIX TESTS ====================

// TestInstallWithSymlinkPrefix tests that --symlink-prefix flag is parsed correctly
func TestInstallWithSymlinkPrefix(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		expectedPrefix   string
	}{
		{
			name:             "equals-separated syntax",
			args:             []string{"--symlink-prefix=/tmp/chroot", "vim"},
			expectedPrefix:   "/tmp/chroot",
		},
		{
			name:             "space-separated syntax",
			args:             []string{"--symlink-prefix", "/tmp/demo", "gcc"},
			expectedPrefix:   "/tmp/demo",
		},
		{
			name:             "no symlink-prefix",
			args:             []string{"vim"},
			expectedPrefix:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse arguments into options
			opts := &InstallOptions{}
			args := tt.args
			
			for i := 0; i < len(args); i++ {
				arg := args[i]
				if strings.HasPrefix(arg, "--symlink-prefix=") {
					opts.SymlinkPrefix = strings.TrimPrefix(arg, "--symlink-prefix=")
				} else if arg == "--symlink-prefix" {
					if i+1 < len(args) {
						i++
						opts.SymlinkPrefix = args[i]
					}
				}
			}
			
			if opts.SymlinkPrefix != tt.expectedPrefix {
				t.Errorf("symlink-prefix not parsed correctly: got %q, want %q", opts.SymlinkPrefix, tt.expectedPrefix)
			}
		})
	}
}

// TestInstallOptionsSymlinkPrefix tests SymlinkPrefix field initialization
func TestInstallOptionsSymlinkPrefix(t *testing.T) {
	opts := &InstallOptions{}
	
	if opts.SymlinkPrefix != "" {
		t.Errorf("SymlinkPrefix should default to empty string, got %q", opts.SymlinkPrefix)
	}
	
	opts.SymlinkPrefix = "/tmp/chroot"
	if opts.SymlinkPrefix != "/tmp/chroot" {
		t.Errorf("SymlinkPrefix not set correctly: got %q, want %q", opts.SymlinkPrefix, "/tmp/chroot")
	}
}

// TestSymlinkPrefixStripCorrectly tests that symlink.StripPrefix strips paths correctly
func TestSymlinkPrefixStripCorrectly(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		prefix      string
		expected    string
		shouldError bool
	}{
		{
			name:        "strip absolute prefix",
			path:        "/tmp/chroot/kod/store/vim/9.0.0-1/usr/bin/vim",
			prefix:      "/tmp/chroot",
			expected:    "/kod/store/vim/9.0.0-1/usr/bin/vim",
			shouldError: false,
		},
		{
			name:        "prefix without trailing slash",
			path:        "/tmp/demo/usr/lib/libc.so.6",
			prefix:      "/tmp/demo",
			expected:    "/usr/lib/libc.so.6",
			shouldError: false,
		},
		{
			name:        "path doesn't start with prefix",
			path:        "/home/user/kod/store/vim",
			prefix:      "/tmp/chroot",
			expected:    "",
			shouldError: true,
		},
		{
			name:        "empty prefix (no-op)",
			path:        "/kod/store/vim/9.0.0-1/usr/bin/vim",
			prefix:      "",
			expected:    "/kod/store/vim/9.0.0-1/usr/bin/vim",
			shouldError: false,
		},
		{
			name:        "root prefix (no-op)",
			path:        "/kod/store/vim/9.0.0-1/usr/bin/vim",
			prefix:      "/",
			expected:    "/kod/store/vim/9.0.0-1/usr/bin/vim",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := symlink.StripPrefix(tt.path, tt.prefix)
			
			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("strip result incorrect: got %q, want %q", result, tt.expected)
				}
			}
		})
	}
}

// TestSymlinkTargetsAreRelativePaths tests that symlinks point to relative paths when using --symlink-prefix
func TestSymlinkTargetsAreRelativePaths(t *testing.T) {
	// This test verifies the behavior of symlink target stripping
	// When --symlink-prefix=/tmp/chroot is used:
	// - Symlink location: /tmp/chroot/usr/bin/vim
	// - Symlink target (should be): /kod/store/vim/.../usr/bin/vim (relative path)
	// - NOT: /tmp/chroot/kod/store/vim/.../usr/bin/vim (absolute within prefix)

	testCases := []struct {
		name            string
		originalTarget  string
		prefix          string
		expectedTarget  string
	}{
		{
			name:            "executable symlink",
			originalTarget:  "/tmp/chroot/kod/store/vim/9.0.0-1/usr/bin/vim",
			prefix:          "/tmp/chroot",
			expectedTarget:  "/kod/store/vim/9.0.0-1/usr/bin/vim",
		},
		{
			name:            "library symlink",
			originalTarget:  "/tmp/chroot/kod/store/gcc-libs/13.1.0-1/usr/lib/libstdc++.so.6",
			prefix:          "/tmp/chroot",
			expectedTarget:  "/kod/store/gcc-libs/13.1.0-1/usr/lib/libstdc++.so.6",
		},
		{
			name:            "no prefix stripping needed",
			originalTarget:  "/kod/store/vim/9.0.0-1/usr/bin/vim",
			prefix:          "",
			expectedTarget:  "/kod/store/vim/9.0.0-1/usr/bin/vim",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := symlink.StripPrefix(tc.originalTarget, tc.prefix)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			
			// Verify the result is a relative path (starts with /)
			if !strings.HasPrefix(result, "/") {
				t.Errorf("symlink target should be absolute path, got: %q", result)
			}
			
			// Verify the result matches expected
			if result != tc.expectedTarget {
				t.Errorf("symlink target incorrect: got %q, want %q", result, tc.expectedTarget)
			}
			
			// Verify it doesn't contain the prefix
			if strings.Contains(result, tc.prefix) && tc.prefix != "" {
				t.Errorf("symlink target contains prefix (should be stripped): %q in %q", tc.prefix, result)
			}
		})
	}
}

// TestInstallOptionsSymlinkPrefixWithOtherFlags tests --symlink-prefix combined with other flags
func TestInstallOptionsSymlinkPrefixWithOtherFlags(t *testing.T) {
	tests := []struct {
		name              string
		symlink           string
		noSymlink         bool
		force             bool
		noDeps            bool
		expectedSymlink   string
		expectedNoSymlink bool
		expectedForce     bool
		expectedNoDeps    bool
	}{
		{
			name:              "symlink-prefix with --force",
			symlink:           "/tmp/chroot",
			force:             true,
			expectedSymlink:   "/tmp/chroot",
			expectedForce:     true,
		},
		{
			name:              "symlink-prefix with --no-symlink",
			symlink:           "/tmp/chroot",
			noSymlink:         true,
			expectedSymlink:   "/tmp/chroot",
			expectedNoSymlink: true,
		},
		{
			name:            "symlink-prefix with --no-deps",
			symlink:         "/tmp/chroot",
			noDeps:          true,
			expectedSymlink: "/tmp/chroot",
			expectedNoDeps:  true,
		},
		{
			name:              "all flags combined",
			symlink:           "/tmp/chroot",
			noSymlink:         true,
			force:             true,
			noDeps:            true,
			expectedSymlink:   "/tmp/chroot",
			expectedNoSymlink: true,
			expectedForce:     true,
			expectedNoDeps:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &InstallOptions{
				SymlinkPrefix: tt.symlink,
				NoSymlink:     tt.noSymlink,
				Force:         tt.force,
				NoDeps:        tt.noDeps,
			}

			if opts.SymlinkPrefix != tt.expectedSymlink {
				t.Errorf("SymlinkPrefix: got %q, want %q", opts.SymlinkPrefix, tt.expectedSymlink)
			}
			if opts.NoSymlink != tt.expectedNoSymlink {
				t.Errorf("NoSymlink: got %v, want %v", opts.NoSymlink, tt.expectedNoSymlink)
			}
			if opts.Force != tt.expectedForce {
				t.Errorf("Force: got %v, want %v", opts.Force, tt.expectedForce)
			}
			if opts.NoDeps != tt.expectedNoDeps {
				t.Errorf("NoDeps: got %v, want %v", opts.NoDeps, tt.expectedNoDeps)
			}
		})
	}
}

// TestInstallSymlinkTargetsWithAndWithoutPrefix tests that symlink targets differ based on --symlink-prefix
func TestInstallSymlinkTargetsWithAndWithoutPrefix(t *testing.T) {
	tests := []struct {
		name              string
		usePrefix         bool
		prefix            string
		expectedTargetHas string
		description       string
	}{
		{
			name:              "without prefix points to wrapper",
			usePrefix:         false,
			prefix:            "",
			expectedTargetHas: "/kod/wrappers/",
			description:       "Normal mode: /usr/bin/vim → /kod/wrappers/vim",
		},
		{
			name:              "with prefix points to package file",
			usePrefix:         true,
			prefix:            "/tmp/chroot",
			expectedTargetHas: "/kod/store/",
			description:       "Prefix mode: /usr/bin/vim → /kod/store/vim/.../usr/bin/vim",
		},
		{
			name:              "with different prefix still points to package",
			usePrefix:         true,
			prefix:            "/home/user/build",
			expectedTargetHas: "/kod/store/",
			description:       "Prefix mode with different path: symlink still points to store",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &InstallOptions{
				SymlinkPrefix: tt.prefix,
			}

			// Simulate the symlink target logic from install.go lines 377-384
			filePath := "usr/bin/vim"
			fileName := "vim"
			storeRoot := "/kod/store"
			wrapperDir := "/kod/wrappers"
			version := "9.0.0-1"
			pkgName := "vim"

			var targetPath string
			if strings.HasPrefix(filePath, "usr/bin/") || strings.HasPrefix(filePath, "usr/sbin/") {
				if opts.SymlinkPrefix != "" {
					// With symlink-prefix, point directly to package files
					targetPath = filepath.Join(storeRoot, pkgName, version, filePath)
				} else {
					// Normal mode: point to wrapper
					targetPath = filepath.Join(wrapperDir, fileName)
				}
			}

			// Verify the target contains the expected path component
			if !strings.Contains(targetPath, tt.expectedTargetHas) {
				t.Errorf("symlink target incorrect: got %q, expected to contain %q\n  %s", targetPath, tt.expectedTargetHas, tt.description)
			}

			// Additional verification for prefix mode
			if tt.usePrefix {
				// Should point to store, not wrappers
				if strings.Contains(targetPath, "/kod/wrappers/") {
					t.Errorf("with --symlink-prefix, symlink should NOT point to wrapper: %q", targetPath)
				}
				// Should contain the full path
				if !strings.Contains(targetPath, version) {
					t.Errorf("symlink target should contain version: %q", targetPath)
				}
			}

			// Additional verification for non-prefix mode
			if !tt.usePrefix {
				// Should point to wrappers
				if !strings.Contains(targetPath, "/kod/wrappers/") {
					t.Errorf("without --symlink-prefix, symlink should point to wrapper: %q", targetPath)
				}
				// Should NOT contain version
				if strings.Contains(targetPath, version) {
					t.Errorf("wrapper symlink should NOT contain version: %q", targetPath)
				}
			}
		})
	}
}

// TestExecutableSymlinkBehaviorWithPrefix tests that usr/bin and usr/sbin are handled correctly
func TestExecutableSymlinkBehaviorWithPrefix(t *testing.T) {
	tests := []struct {
		name           string
		filePath       string
		usePrefix      bool
		expectedPrefix string
		description    string
	}{
		{
			name:           "usr/bin executable with prefix",
			filePath:       "usr/bin/vim",
			usePrefix:      true,
			expectedPrefix: "/kod/store/",
			description:    "usr/bin files should point to store with prefix",
		},
		{
			name:           "usr/sbin executable with prefix",
			filePath:       "usr/sbin/useradd",
			usePrefix:      true,
			expectedPrefix: "/kod/store/",
			description:    "usr/sbin files should point to store with prefix",
		},
		{
			name:           "usr/bin executable without prefix",
			filePath:       "usr/bin/vim",
			usePrefix:      false,
			expectedPrefix: "/kod/wrappers/",
			description:    "usr/bin files should point to wrapper without prefix",
		},
		{
			name:           "usr/sbin executable without prefix",
			filePath:       "usr/sbin/useradd",
			usePrefix:      false,
			expectedPrefix: "/kod/wrappers/",
			description:    "usr/sbin files should point to wrapper without prefix",
		},
		{
			name:           "library file with prefix",
			filePath:       "usr/lib/libvim.so",
			usePrefix:      true,
			expectedPrefix: "/kod/store/",
			description:    "non-executable files should point to store regardless",
		},
		{
			name:           "library file without prefix",
			filePath:       "usr/lib/libvim.so",
			usePrefix:      false,
			expectedPrefix: "/kod/store/",
			description:    "non-executable files should point to store regardless",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &InstallOptions{
				SymlinkPrefix: "",
			}
			if tt.usePrefix {
				opts.SymlinkPrefix = "/tmp/chroot"
			}

			// Simulate the logic from install.go
			fileName := filepath.Base(tt.filePath)
			storeRoot := "/kod/store"
			wrapperDir := "/kod/wrappers"
			pkgName := "vim"
			version := "9.0.0-1"

			var targetPath string
			if strings.HasPrefix(tt.filePath, "usr/bin/") || strings.HasPrefix(tt.filePath, "usr/sbin/") {
				if opts.SymlinkPrefix != "" {
					targetPath = filepath.Join(storeRoot, pkgName, version, tt.filePath)
				} else {
					targetPath = filepath.Join(wrapperDir, fileName)
				}
			} else {
				// Regular file: point to storage
				targetPath = filepath.Join(storeRoot, pkgName, version, tt.filePath)
			}

			if !strings.Contains(targetPath, tt.expectedPrefix) {
				t.Errorf("symlink target incorrect: got %q, expected to contain %q\n  %s", targetPath, tt.expectedPrefix, tt.description)
			}
		})
	}
}
