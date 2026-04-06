// Package aur provides Arch User Repository (AUR) support.
// aur_test.go contains unit tests for AUR package functionality.
package aur

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Test ParseDependency function
func TestParseDependency(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		expectName string
	}{
		{
			name:       "simple package",
			input:      "bash",
			expectName: "bash",
		},
		{
			name:       "package with version constraint",
			input:      "go>=1.21",
			expectName: "go",
		},
		{
			name:       "package with less than constraint",
			input:      "python<4.0",
			expectName: "python",
		},
		{
			name:       "empty string",
			input:      "",
			expectName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dep := ParseDependency(tt.input)
			if dep.Name != tt.expectName {
				t.Errorf("got %s, want %s", dep.Name, tt.expectName)
			}
		})
	}
}

// Test PKGBUILDParser.extractSimpleValue
func TestExtractSimpleValue(t *testing.T) {
	parser := NewPKGBUILDParser()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple value",
			input:    "pkgname=bash",
			expected: "bash",
		},
		{
			name:     "quoted value",
			input:    `pkgname="bash"`,
			expected: "bash",
		},
		{
			name:     "single quoted value",
			input:    "pkgname='bash'",
			expected: "bash",
		},
		{
			name:     "value with spaces",
			input:    `pkgname="my package"`,
			expected: "my package",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.extractSimpleValue(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

// Test PKGBUILDParser.extractArrayValues
func TestExtractArrayValues(t *testing.T) {
	parser := NewPKGBUILDParser()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple array",
			input:    "depends=(bash coreutils)",
			expected: []string{"bash", "coreutils"},
		},
		{
			name:     "array with quotes",
			input:    `depends=("bash" "coreutils")`,
			expected: []string{"bash", "coreutils"},
		},
		{
			name:     "empty array",
			input:    "depends=()",
			expected: []string{},
		},
		{
			name:     "single element",
			input:    "depends=(bash)",
			expected: []string{"bash"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.extractArrayValues(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("got %d elements, want %d", len(result), len(tt.expected))
				return
			}
			for i, val := range result {
				if val != tt.expected[i] {
					t.Errorf("element %d: got %q, want %q", i, val, tt.expected[i])
				}
			}
		})
	}
}

// Test PKGBUILDParser.extractPackageName
func TestExtractPackageName(t *testing.T) {
	parser := NewPKGBUILDParser()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "bash",
			expected: "bash",
		},
		{
			name:     "with version constraint",
			input:    "go>=1.21",
			expected: "go",
		},
		{
			name:     "with less than",
			input:    "python<4.0",
			expected: "python",
		},
		{
			name:     "optdepends format",
			input:    "bash: shell",
			expected: "bash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.extractPackageName(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

// Test isValidPackageName
func TestIsValidPackageName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid name",
			input:    "bash",
			expected: true,
		},
		{
			name:     "valid with dash",
			input:    "perl-module",
			expected: true,
		},
		{
			name:     "valid with underscore",
			input:    "bash_utils",
			expected: true,
		},
		{
			name:     "valid with numbers",
			input:    "bash2",
			expected: true,
		},
		{
			name:     "empty",
			input:    "",
			expected: false,
		},
		{
			name:     "invalid characters",
			input:    "bash@utils",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidPackageName(tt.input)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test PKGBUILDParser.Parse with a real PKGBUILD file
func TestPKGBUILDParse(t *testing.T) {
	// Create a temporary PKGBUILD file for testing
	tmpDir := t.TempDir()
	pkgbuildPath := filepath.Join(tmpDir, "PKGBUILD")

	pkgbuildContent := `#!/bin/bash
# Sample PKGBUILD for testing
pkgname=test-package
pkgver=1.0.0
pkgrel=1
arch=(x86_64)
depends=(bash coreutils)
makedepends=(gcc make)
optdepends=("git: for version control")
conflicts=(old-package)
provides=(test-package)
replaces=(legacy-package)
`

	err := os.WriteFile(pkgbuildPath, []byte(pkgbuildContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test PKGBUILD: %v", err)
	}

	parser := NewPKGBUILDParser()
	info, err := parser.Parse(pkgbuildPath)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// Validate parsed values
	if info.Name != "test-package" {
		t.Errorf("name: got %q, want %q", info.Name, "test-package")
	}

	if info.Version != "1.0.0" {
		t.Errorf("version: got %q, want %q", info.Version, "1.0.0")
	}

	if len(info.Depends) != 2 {
		t.Errorf("depends count: got %d, want 2", len(info.Depends))
	}

	if len(info.MakeDepends) != 2 {
		t.Errorf("makedepends count: got %d, want 2", len(info.MakeDepends))
	}
}

// Test PKGBUILDParser.ValidatePKGBUILD
func TestValidatePKGBUILD(t *testing.T) {
	parser := NewPKGBUILDParser()

	tests := []struct {
		name    string
		info    *PKGBUILDInfo
		wantErr bool
	}{
		{
			name: "valid info",
			info: &PKGBUILDInfo{
				Name:    "test-package",
				Version: "1.0.0",
			},
			wantErr: false,
		},
		{
			name:    "nil info",
			info:    nil,
			wantErr: true,
		},
		{
			name: "missing name",
			info: &PKGBUILDInfo{
				Name:    "",
				Version: "1.0.0",
			},
			wantErr: true,
		},
		{
			name: "missing version",
			info: &PKGBUILDInfo{
				Name:    "test",
				Version: "",
			},
			wantErr: true,
		},
		{
			name: "invalid name characters",
			info: &PKGBUILDInfo{
				Name:    "test@package",
				Version: "1.0.0",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.ValidatePKGBUILD(tt.info)
			if (err != nil) != tt.wantErr {
				t.Errorf("got error=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

// Test CachedAURPackage.IsCacheValid
func TestCachedAURPackageIsCacheValid(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		cached   *CachedAURPackage
		expected bool
	}{
		{
			name: "valid cache",
			cached: &CachedAURPackage{
				CachedAt:  now.Add(-1 * time.Hour),
				ExpiresAt: now.Add(1 * time.Hour),
			},
			expected: true,
		},
		{
			name: "expired cache",
			cached: &CachedAURPackage{
				CachedAt:  now.Add(-2 * time.Hour),
				ExpiresAt: now.Add(-1 * time.Hour),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cached.IsCacheValid()
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test RPCClient initialization
func TestNewRPCClient(t *testing.T) {
	client := NewRPCClient()

	if client == nil {
		t.Fatal("NewRPCClient returned nil")
	}

	if client.baseURL != "https://aur.archlinux.org/rpc" {
		t.Errorf("baseURL: got %q", client.baseURL)
	}

	if client.cacheTTL != 24*time.Hour {
		t.Errorf("cacheTTL: got %v, want 24h", client.cacheTTL)
	}

	if client.httpClient == nil {
		t.Fatal("httpClient is nil")
	}
}

// Test GitHandler initialization
func TestNewGitHandler(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewGitHandler(tmpDir)

	if handler == nil {
		t.Fatal("NewGitHandler returned nil")
	}

	if handler.baseURL != "https://aur.archlinux.org" {
		t.Errorf("baseURL: got %q", handler.baseURL)
	}

	if handler.cacheDir != tmpDir {
		t.Errorf("cacheDir: got %q, want %q", handler.cacheDir, tmpDir)
	}

	if handler.timeout != 30*time.Second {
		t.Errorf("timeout: got %v, want 30s", handler.timeout)
	}
}

// Test GitHandler.VerifyPKGBUILD
func TestGitHandlerVerifyPKGBUILD(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewGitHandler(tmpDir)

	// Test with non-existent path
	err := handler.VerifyPKGBUILD(filepath.Join(tmpDir, "nonexistent"))
	if err == nil {
		t.Error("expected error for non-existent PKGBUILD")
	}

	// Create a valid PKGBUILD file
	pkgbuildPath := filepath.Join(tmpDir, "PKGBUILD")
	err = os.WriteFile(pkgbuildPath, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("failed to create test PKGBUILD: %v", err)
	}

	// Test with valid PKGBUILD
	err = handler.VerifyPKGBUILD(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test with empty PKGBUILD
	err = os.WriteFile(pkgbuildPath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("failed to truncate PKGBUILD: %v", err)
	}

	err = handler.VerifyPKGBUILD(tmpDir)
	if err == nil {
		t.Error("expected error for empty PKGBUILD")
	}
}

// Test GitHandler.GetPKGBUILDPath
func TestGitHandlerGetPKGBUILDPath(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewGitHandler(tmpDir)

	path := handler.GetPKGBUILDPath(tmpDir)
	expected := filepath.Join(tmpDir, "PKGBUILD")

	if path != expected {
		t.Errorf("got %q, want %q", path, expected)
	}
}
