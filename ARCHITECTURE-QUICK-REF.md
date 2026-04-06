# Chisel Architecture - Quick Reference

## System Overview
- **Base Directory**: `/kod` (configurable)
- **Store**: `/kod/store/{package}/{version}/` (extracted packages)
- **Registry**: `/kod/registry.json` (installed packages metadata)
- **Databases**: `/kod/db/sync/{repo}.db` (Arch package metadata)
- **Cache**: `/kod/cache/` (downloaded .pkg.tar.zst files)
- **Wrappers**: `/kod/wrappers/` (shell scripts for executables)

---

## Key Component Quick Links

### 1. Dependency Resolution
**File**: `pkg/alpm/db.go` (lines 126-207)
**Entry Point**: `Client.ResolveDependencies(packageName)`
**Algorithm**: Depth-first search with cycle detection
**Output**: `[]string` (package names in install order, dependencies before dependents)

### 2. Package Database
**File**: `pkg/alpm/parse.go` (lines 18-111)
**Source**: `/kod/db/sync/*.db` (gzipped tar archives)
**Parsing**: Extract per-package metadata, parse PKGINFO/DESC files
**Cache**: `DatabaseCache` respects repo precedence (core > extra > community)

### 3. Version Comparison
**File**: `pkg/alpm/version.go` (lines 9-34)
**Algorithm**: RPM version scheme
**Example**: "1.0" < "1.0.1", "5.3.9-1" vs "5.3.10-1"

### 4. Download Manager
**File**: `pkg/download/download.go` (lines 24-177)
**URL Format**: `{mirror}/{repo}/os/{arch}/{name}-{version}-{arch}.pkg.tar.zst`
**Process**: HTTP GET → temp file (.tmp) → atomic rename
**Concurrency**: Configurable semaphore (default 5)

### 5. Package Extraction
**File**: `pkg/extract/extract.go` (lines 39-272)
**Format**: `.pkg.tar.zst` (zstd + tar)
**Output**: `[]ExtractedFile` with path, size, mode, symlink info
**Target**: `/kod/store/{pkg}/{version}/`

### 6. Symlink Management
**File**: `pkg/symlink/symlink.go` (lines 12-206)
**Strategy**: 
- Executables (usr/bin/*) → wrapper scripts
- Other files → direct store path
- Check for conflicts before creation

### 7. Registry
**File**: `pkg/registry/registry.go` (lines 35-116)
**Location**: `/kod/registry.json` (JSON format)
**Fields**: name, version, files[], executables[], dependencies[], install_date

### 8. CLI Router
**File**: `cmd/chisel/main.go` (lines 22-87)
**Commands**: sync, search, info, download, extract, install, remove, list, upgrade, cleanup, cache
**Config Priority**: CLI flags > env vars > config file > defaults

---

## Installation Flow (Simplified)

```
install → resolve deps → download → extract → create symlinks → generate wrappers → update registry
```

**Each Stage**:
1. **Resolve** - `ALPMClient.ResolveDependencies()` returns package list
2. **Download** - `Downloader.DownloadPackages()` fetches to cache
3. **Extract** - `Store.ExtractPackage()` uncompresses to `/kod/store/`
4. **Symlink** - Create system-level hierarchy pointing to store
5. **Wrapper** - Generate `LD_LIBRARY_PATH` shell scripts for execs
6. **Registry** - Record in `/kod/registry.json`

---

## Key Data Structures

### Package (alpm)
```go
type Package struct {
    Name, Version, Description, Architecture string
    DependsOn, OptDepends []string  // Dependencies
    Conflicts, Replaces []string    // Alternatives
    Provides []string               // Virtual packages
    CompressedSize, InstalledSize int64
    Repository, PackageBase string
    Licenses, Groups []string
}
```

### ExtractedFile (extract)
```go
type ExtractedFile struct {
    Path, AbsPath string      // Relative and absolute paths
    IsDirectory, IsSymlink bool
    LinkTarget string         // Symlink target
    Size int64
    Mode os.FileMode
}
```

### Registry Package
```go
type Package struct {
    Name, Version string
    Files, Executables []string
    Dependencies []string
    InstallDate string  // RFC3339 format
}
```

---

## Extension Points for AUR

### New Commands (cmd/chisel/main.go)
```go
case "aur-install": handleAURInstall()
case "aur-search": handleAURSearch()
case "aur-info": handleAURInfo()
case "aur-upgrade": handleAURUpgrade()
```

### New Packages
- `pkg/aur/client.go` - AUR API queries
- `pkg/build/builder.go` - makepkg wrapper
- `internal/cli/aur_install.go` - CLI integration

### AUR Package Metadata (extend Package struct)
```go
IsAUR bool
AURPackageBase string
AURMaintainer string
PKGBUILDRef string
AURBuildDeps []string
```

### Build Flow
```
aur-install → search AUR → clone PKGBUILD → build (makepkg) 
→ cache .pkg.tar.zst → continue normal install flow
```

---

## Config File Structure

**Location**: `/etc/chisel/config.json` or via `CHISEL_CONFIG` env var

```json
{
  "base_dir": "/kod",
  "symlink_root": "/",
  "store_root": "/kod/store",
  "registry_path": "/kod/registry.json",
  "db_path": "/kod/db/sync",
  "cache_path": "/kod/cache",
  "wrapper_dir": "/kod/wrappers",
  "mirror_url": "https://mirror.rackspace.com/archlinux",
  "architecture": "x86_64",
  "repositories": ["core", "extra", "community"],
  "verify_signatures": false,
  "max_concurrent_downloads": 5,
  "download_timeout": 300,
  "keep_versions": 3
}
```

---

## Version Precedence (Repository)

```
core (0)      - Highest priority
extra (1)     
community (2)
multilib (3)  - Lowest priority
```

When a package exists in multiple repos, the one from the highest-priority repo is used.

---

## Dependency Resolution Algorithm

1. Get package: `Cache.GetPackage(name, arch)`
2. If not found, try providers: `Cache.GetProvidingPackages(name)`
3. Check version constraint: `CheckVersionConstraint(version, constraint)`
4. Detect cycles: maintain `visiting` and `visited` sets
5. Recurse depth-first
6. Return ordered list: dependencies before dependents

---

## Common CLI Operations

```bash
# Sync databases
chisel sync

# Search packages
chisel search pattern

# Get package info
chisel info package-name

# Install (full flow)
chisel install package-name

# Install specific version (not dependencies)
chisel install --no-deps package-name

# List installed
chisel list
chisel list --verbose

# Remove package
chisel remove package-name

# Upgrade all
chisel upgrade

# Clean old versions (keep 3)
chisel cleanup

# Clean download cache
chisel cache --clean
```

---

## Error Handling

### ResolutionError
- **Circular Dependency**: Returns error with cycle list
- **Missing Dependency**: Returns error with package name
- **Version Constraint Violation**: Returns error with version details

### Download/Extract Errors
- **HTTP Errors**: Checked via status code
- **Partial Downloads**: Cleaned up (.tmp files removed)
- **Path Traversal**: Blocked via `HasPrefix` check
- **Permission Issues**: Logged as warnings, not fatal

---

## Testing Entry Points

**Dependency Resolution**: `pkg/alpm/deps_test.go`
- Simple chains
- Multiple dependencies
- Transitive dependencies
- Circular dependency detection
- Version constraints
- Virtual packages (provides)

**Version Comparison**: `pkg/alpm/version_test.go`
- Epoch handling
- Release/revision parsing
- Segment tokenization
- Complex version strings

**Database Operations**: `pkg/alpm/db_test.go`
- Database loading
- Package search
- Cache merging

---

## Future Enhancements

1. **Conflict Resolution** - Check `Package.Conflicts` during install
2. **Signature Verification** - Implement GPG checks
3. **Binary Cache** - Pre-built package repository
4. **AUR Support** - Build from source with makepkg
5. **Rollback** - Version management and downgrades
6. **Hooks** - Pre/post-install scripts
