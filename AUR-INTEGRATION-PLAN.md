# AUR Integration Plan - Phase 6

This document outlines the comprehensive plan for integrating Arch User Repository (AUR) support into Chisel.

## Executive Summary

Chisel will support installing packages from the AUR alongside official Arch repositories. AUR packages will be built from PKGBUILD files and integrated into Chisel's existing store/symlink/registry architecture identically to official packages.

### Key Requirements

1. **Download, Build & Install**: Build AUR packages from PKGBUILD and install with full dependency support
2. **Recursive Dependency Resolution**: Resolve dependencies across both official repos and AUR (Option A)
3. **Version Tracking**: Track AUR package versions, rebuild on updates if versions differ
4. **User Responsibility**: Users review PKGBUILDs before installation (Chisel doesn't sandbox)
5. **General Audience**: Support general users comfortable with building from source

### Design Decisions

- **Official repo priority**: Official packages checked first, AUR used as fallback
- **Identical integration**: AUR-built packages treated identically to official after building
- **Persistent build cache**: Reuse build directories across builds for speed
- **Build logging**: Capture all build output to files, delete logs on cleanup
- **Wrapper support**: AUR executables use same wrapper system as official packages

---

## Architecture Overview

### Module Structure

```
pkg/aur/
  ├─ types.go              # AUR package types
  ├─ rpc.go                # AUR API v5 client
  ├─ git.go                # PKGBUILD download via git
  ├─ pkgbuild.go           # PKGBUILD parsing
  └─ aur_test.go          # AUR tests

pkg/build/
  ├─ resolver.go           # Mixed repo dependency resolver
  ├─ builder.go            # makepkg build execution
  └─ builder_test.go       # Build tests

pkg/alpm/ (existing, minor enhancements)
  └─ Enhanced for mixed repository support

internal/cli/ (existing, modifications)
  ├─ search.go             # Add --aur flag
  ├─ info.go               # Add --aur flag
  └─ install.go            # Full AUR support

pkg/registry/ (existing, schema update)
  └─ Add Source field to track official vs AUR

pkg/config/ (existing, new fields)
  └─ Add build cache/logs directories
```

---

## Phase-by-Phase Implementation

### Phase 1: AUR Package Source (pkg/aur/)

#### 1.1 AUR Data Types (`pkg/aur/types.go`)

Defines AUR package structures matching AUR RPC v5 API:

```go
type AURPackage struct {
  ID           int      `json:"ID"`
  Name         string   `json:"Name"`
  PackageBase  string   `json:"PackageBase"`
  Version      string   `json:"Version"`
  Description  string   `json:"Description"`
  URL          string   `json:"URL"`
  Maintainer   string   `json:"Maintainer"`
  
  // Dependencies
  Depends      []string `json:"Depends"`       // Runtime deps
  MakeDepends  []string `json:"MakeDepends"`  // Build deps
  OptDepends   []string `json:"OptDepends"`
  Conflicts    []string `json:"Conflicts"`
  Provides     []string `json:"Provides"`
  Replaces     []string `json:"Replaces"`
  
  // Metadata
  FirstSubmit  int64 `json:"FirstSubmitted"`
  LastModified int64 `json:"LastModified"`
  OutOfDate    int   `json:"OutOfDate"`  // 0=current, unix timestamp=outdated
  Popularity   float64 `json:"Popularity"`
  Votes        int     `json:"NumVotes"`
}

type PKGBUILDInfo struct {
  Name         string
  Version      string
  Depends      []string
  MakeDepends  []string
  OptDepends   []string
  Conflicts    []string
  Provides     []string
  Replaces     []string
  Architecture []string  // x86_64, any, etc.
}
```

#### 1.2 AUR RPC Client (`pkg/aur/rpc.go`)

Implements AUR API v5 client with caching and rate limiting:

```go
type RPCClient struct {
  baseURL     string           // https://aur.archlinux.org/rpc/v5/
  httpClient  *http.Client
  cache       map[string]interface{}
  cacheTTL    time.Duration
  lastRequest time.Time
  requestCount int
}

// Search packages in AUR
func (rc *RPCClient) SearchPackages(query string, limit int) ([]AURPackage, error)

// Get detailed info for specific packages
func (rc *RPCClient) GetPackageInfo(names []string) (map[string]*AURPackage, error)

// Get single package (convenience)
func (rc *RPCClient) GetPackage(name string) (*AURPackage, error)
```

Features:
- Batches requests (max 200 packages per info query per RPC API)
- Caches results for 24 hours locally
- Rate limit tracking (warns at 3800/4000 requests/day)
- Retry logic for transient failures

#### 1.3 Git Handler (`pkg/aur/git.go`)

Downloads PKGBUILD files via git clone:

```go
type GitHandler struct {
  baseURL     string  // https://aur.archlinux.org
  cacheDir    string  // /kod/build-cache/
  httpClient  *http.Client
}

// Clone PKGBUILD for latest version
func (gh *GitHandler) ClonePKGBUILD(pkgName, destDir string) (string, error)
  // Returns path to cloned directory
  // URL: https://aur.archlinux.org/PKGNAME.git

// Clone specific version (if version history needed in future)
func (gh *GitHandler) ClonePKGBUILDVersion(pkgName, version, destDir string) (string, error)
```

Features:
- Efficient cloning (git clone --depth=1 for latest)
- Timeout handling (30s default)
- Network error handling with retries

#### 1.4 PKGBUILD Parser (`pkg/aur/pkgbuild.go`)

Parses PKGBUILD shell scripts to extract metadata:

```go
type PKGBUILDParser struct{}

// Parse PKGBUILD file and extract metadata
func (pp *PKGBUILDParser) Parse(filePath string) (*PKGBUILDInfo, error)
  // Extracts: pkgname, pkgver, depends, makedepends, etc.
  // Handles bash array syntax: depends=("pkg1" "pkg2>=1.0")
```

Features:
- Regex-based bash array parsing
- Handles version constraints (>=, <=, >, <, =)
- Handles split packages
- Robust error handling for malformed PKGBUILDs

---

### Phase 2: Dependency Resolution (`pkg/build/`)

#### 2.1 Mixed Repository Resolver (`pkg/build/resolver.go`)

Unified dependency resolver supporting official repos + AUR:

```go
type MixedResolver struct {
  alpClient *alpm.Client
  aurRPC    *aur.RPCClient
  visited   map[string]bool  // Cycle detection
}

type PackageSource struct {
  Name       string              // Package name
  Version    string
  Source     string              // "official" or "aur"
  Repo       string              // "core", "extra", "aur", etc.
  IsAUR      bool
  PKGBUILD   *aur.PKGBUILDInfo   // Only set for AUR packages
}

// Resolve dependencies with correct build order
func (mr *MixedResolver) ResolveDependencies(pkgName string) ([]PackageSource, error)
  // Returns: [dep1, dep2, ..., requested_pkg] in build order
```

Resolution Algorithm:
```
ResolveDependencies(pkgName):
  1. Check if package exists in official repos via ALPM
     → Found: return with Source="official"
  2. Not found → check AUR via RPC
     → Not found: return error "package not found"
  3. For AUR package:
     a. Get PKGBUILD via git clone
     b. Parse PKGBUILD to extract makedepends + depends
     c. For each dependency:
        i.   Recursively resolve (official or AUR)
        ii.  Add to result list if not visited
     d. Build order: all unresolved dependencies first, then package
  4. Cycle detection: track visited packages, error if cycle found
  5. Return ordered list [dep1, dep2, ..., pkg]
```

---

### Phase 3: Build System (`pkg/build/builder.go`)

#### 3.1 Build Manager

Executes `makepkg` to build AUR packages:

```go
type BuildManager struct {
  buildCacheDir  string      // /kod/build-cache/
  logsDir        string      // /kod/build-logs/
  gitHandler     *aur.GitHandler
  pkgbuildParser *aur.PKGBUILDParser
}

// Main build function
func (bm *BuildManager) BuildAURPackage(
  pkgName string,
  version string,
  pkgbuildPath string,  // path to cloned PKGBUILD directory
) (string, error)  // returns: path to built .pkg.tar.zst

// Cleanup old build artifacts
func (bm *BuildManager) CleanupBuildArtifacts(maxAge time.Duration) error
```

Build Process:
```
BuildAURPackage(pkgName, version, pkgbuildPath):
  1. Create build directory:
     /kod/build-cache/PKG-NAME-VERSION-TIMESTAMP/
  
  2. Copy PKGBUILD and all source files to build dir
  
  3. Execute build in temp directory:
     cd build-dir && makepkg -s 2>&1 | tee /kod/build-logs/PKG-NAME-VERSION.log
     
  4. Capture output .pkg.tar.zst from build directory
     (Handles split packages - use pkgbase if multiple)
  
  5. Verify checksums (if provided in PKGBUILD)
  
  6. Move artifact to cache directory:
     /kod/cache/PKG-NAME-VERSION-x86_64.pkg.tar.zst
  
  7. Return path to artifact
  
  On Error:
    - Keep build directory for debugging
    - Keep log file for user review
    - Return error, installation aborts (fail-fast)
```

Build Cache Structure:
```
/kod/build-cache/
  ├─ vim-aur-1.0-1-1234567890/
  │   ├─ PKGBUILD
  │   ├─ .git/
  │   └─ src/
  └─ neovim-1.0-1-1234567891/
      ├─ PKGBUILD
      └─ ...

/kod/build-logs/
  ├─ vim-aur-1.0-1.log
  └─ neovim-1.0-1.log
```

---

### Phase 4: CLI Integration

#### 4.1 Search Command Enhancement (`internal/cli/search.go`)

Add AUR search capability:

```go
// New method
func (s *SearchCommand) ExecuteAUR(pattern string) error
  // Query AUR RPC, display results
  // Format: [aur] name version (description)
```

Usage:
```bash
chisel search vim              # Search official repos only
chisel search --aur vim        # Search AUR only
chisel search --include-aur vi # Search official + AUR (combined)
```

#### 4.2 Info Command Enhancement (`internal/cli/info.go`)

Show AUR package information:

```go
// New method
func (i *InfoCommand) ExecuteAUR(packageName string) error
  // Display AUR package info:
  // - Name, Version, Last Modified
  // - Description
  // - Dependencies, Make Dependencies
  // - Maintainer, Popularity
  // - Conflicting/Provides/Replaces
```

#### 4.3 Install Command Major Refactoring (`internal/cli/install.go`)

Complete rewrite to support AUR:

```go
type InstallCommand struct {
  config         *config.Config
  symlinkDir     string
  aurRPC         *aur.RPCClient       // NEW
  buildManager   *build.BuildManager  // NEW
  mixedResolver  *build.MixedResolver // NEW
}

func (i *InstallCommand) Run(args []string) error
```

Installation Flow:
```
1. Parse args: --aur flag, package names
   --aur          Force AUR (even if official exists)
   --no-aur       Official only
   (default: official first, fallback to AUR)

2. Resolve dependencies
   resolveMixedDependencies(packages) → []PackageSource

3. For each PackageSource:
   a. If Source="official":
      - Download from mirror (existing flow)
      - Extract to store (existing flow)
   
   b. If Source="aur":
      - Clone PKGBUILD from git
      - Build with buildManager.BuildAURPackage()
      - Download: skip (already in cache from build)
      - Extract to store (same as official)

4. Create symlinks (existing flow, works same for both)

5. Generate wrappers (existing flow, works same for both)

6. Update registry with Source field

7. If ANY build fails: abort entire installation
```

#### 4.4 Main CLI Router Modifications (`cmd/chisel/main.go`)

```bash
chisel search --aur keyword          # Search AUR
chisel info --aur package-name       # Info from AUR
chisel install --aur package-name    # Force install from AUR
chisel install package-name          # Try official, fallback AUR
```

---

### Phase 5: Version Tracking & Upgrades

#### 5.1 Registry Schema Update (`pkg/registry/registry.go`)

Add source tracking:

```go
type Package struct {
  Name         string   `json:"name"`
  Version      string   `json:"version"`
  Source       string   `json:"source"`           // NEW: "official" or "aur"
  Files        []string `json:"files"`
  Executables  []string `json:"executables"`
  Dependencies []string `json:"dependencies"`
  InstallDate  string   `json:"install_date"`
  BuildLog     string   `json:"build_log,omitempty"`  // Path to log if AUR
}
```

#### 5.2 Upgrade Command Enhancement (`internal/cli/upgrade.go`)

Support upgrading AUR packages:

```go
func (u *UpgradeCommand) Execute(opts *UpgradeOptions) (*UpgradeSummary, error)
  // For each installed package:
  // 1. If Source="official": check official repos
  // 2. If Source="aur": 
  //    a. Query AUR RPC for latest version
  //    b. If newer AND version differs:
  //       - Build new version
  //       - Keep old version in store
  //       - Update symlinks to new version
  //       - Update registry
  //    c. If build fails: mark as failed, continue
```

#### 5.3 Version Handling

Store structure supports multiple versions:
```
/kod/stor/
  └─ vim-aur/
      ├─ 1.0-1/     # Old version (kept in store)
      └─ 2.0-1/     # New version (symlinks point here)
```

Registry: Tracks only latest installed version per package name
- `GetPackage("vim-aur")` returns current version (e.g., 2.0-1)
- Symlinks: Point to current version
- Cleanup: Allows removal of old versions

---

### Phase 6: Cleanup Command Enhancement (`internal/cli/cleanup.go`)

Extend cleanup to handle build artifacts:

```go
// Add build cache cleanup
func (c *CleanupCommand) cleanupBuildCache(maxAge time.Duration) error
  // Remove build dirs older than maxAge from /kod/build-cache/
  // Remove associated log files from /kod/build-logs/
  
// Integration with existing cleanup:
// Default: keep 2 latest versions, delete build cache > 30 days old
// --aggressive: delete all old versions, clean all build cache
```

---

## Configuration

### New Config Fields (`pkg/config/config.go`)

```go
type Config struct {
  // ... existing fields ...
  
  // AUR support
  BuildCacheDir     string  // default: /kod/build-cache/
  BuildLogsDir      string  // default: /kod/build-logs/
  EnableAUR         bool    // default: true
  AURRPCTimeout     int     // seconds, default: 30
  PreferOfficial    bool    // default: true (prefer official over AUR)
  
  // Build options
  MaxBuildAttempts  int     // default: 1 (fail fast)
  BuildTimeoutMins  int     // default: 30 (timeout if build > 30 mins)
}

// UpdateDerivedPaths() creates:
// - /kod/build-cache/
// - /kod/build-logs/
```

---

## Error Handling

| Error Case | Handling |
|------------|----------|
| Package not in official OR AUR | Error: "package not found" |
| Build failure | Log output, fail installation, abort |
| Circular dependencies | Detect cycle, error with path |
| makepkg missing | Error: "base-devel not installed" with instructions |
| Rate limit approaching | Warn user: "4000 req/day limit" |
| Git clone fails | Retry once, then error |
| PKGBUILD parse error | Error with file path + line number |
| Version conflict (AUR < official) | Prefer official unless --aur flag |

---

## Testing Strategy

### Unit Tests
- `pkg/aur/aur_test.go`: PKGBUILD parsing, RPC caching
- `pkg/build/resolver_test.go`: Dependency resolution, cycle detection
- `pkg/build/builder_test.go`: Build success/failure, logging

### Integration Tests
- Full install flow with mock AUR packages
- Recursive dependency resolution (official + AUR)
- Build logging verification
- Registry updates with source tracking

### Edge Cases
- Circular dependencies (A depends on B, B depends on A)
- Split packages
- Missing makedeps
- makepkg not installed
- Network timeouts
- Invalid PKGBUILD syntax

---

## Implementation Timeline

| Phase | Duration | Deliverables |
|-------|----------|--------------|
| Phase 1: AUR Source | 2-3 days | RPC client, PKGBUILD parser, git handler |
| Phase 2: Resolver | 2-3 days | Mixed resolver, cycle detection |
| Phase 3: Build System | 2-3 days | Build manager, makepkg wrapper, logging |
| Phase 4: CLI Integration | 1-2 days | search/info/install with AUR support |
| Phase 5: Version Tracking | 1-2 days | Registry enhancement, upgrade logic |
| Phase 6: Polish & Testing | 2-3 days | Full test suite, documentation, edge cases |
| **Total** | **~11-16 days** | **Full AUR support** |

---

## Known Limitations

1. **No PKGBUILD sandboxing**: Users must review PKGBUILDs before building (by design)
2. **Build environment**: Assumes user has base-devel toolchain installed
3. **Split packages**: Supported, but only primary package artifact installed
4. **VCS dependencies**: aur package names ending in -git, -svn, etc. are built fresh each time
5. **Rate limiting**: AUR RPC has 4000 requests/day limit (caching helps)

---

## Success Criteria

- ✅ AUR packages can be installed with full dependency resolution
- ✅ AUR packages treated identically to official packages after building
- ✅ Recursive dependency resolution works (official + AUR)
- ✅ Version tracking and upgrades work for AUR packages
- ✅ Build logs saved and cleaned on cleanup
- ✅ Official repos preferred over AUR by default
- ✅ All existing tests pass
- ✅ New tests for AUR functionality (80%+ coverage)

