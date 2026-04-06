# Chisel Codebase Architecture Analysis

## Overview
Chisel is a cross-distribution package manager that brings Arch Linux packages to any Linux distribution. It manages packages in a central store with symlink-based distribution to system directories.

---

## 1. CURRENT DEPENDENCY RESOLUTION

### How It Works
The dependency resolver is implemented in the ALPM (Arch Linux Package Management) module using a pure Go implementation that parses Arch Linux sync databases.

**Key Files:**
- `/home/abuss/Work/devel/chisel2/pkg/alpm/db.go` (lines 126-207): Core resolution logic
- `/home/abuss/Work/devel/chisel2/pkg/alpm/alpm.go` (lines 174-182): Public API
- `/home/abuss/Work/devel/chisel2/internal/cli/install.go` (lines 463-524): CLI integration

### Resolution Flow

1. **Entry Point:** `ALPMClient.ResolveDependencies(packageName)` → `/pkg/alpm/alpm.go:176`
   - Routes to internal `Client.ResolveDependencies()`

2. **Algorithm:** `Client.ResolveDependencies()` → `/pkg/alpm/db.go:130`
   - Uses depth-first search with cycle detection
   - Calls `resolveDepsRecursive()` for each dependency

3. **Recursion:** `Client.resolveDepsRecursive()` → `/pkg/alpm/db.go:150`
   - Tracks visited packages (completed) and visiting packages (in-progress)
   - Detects circular dependencies by checking if package is currently "visiting"
   - Parses dependency strings: `ParseDependency()` → `/pkg/alpm/parse.go` (extracts name & version constraint)
   - Resolves constraints: `CheckVersionConstraint()` validates package version meets requirement

4. **Package Lookup:**
   - First tries exact match: `Cache.GetPackage(depName, arch)` → `/pkg/alpm/cache.go:52`
   - Falls back to virtual package: `Cache.GetProvidingPackages(depName)` → `/pkg/alpm/cache.go:82`

5. **Return Value:** Ordered list of package names (dependencies before dependents)

### Version Constraint Handling

**Location:** `/pkg/alpm/version.go` (lines 69-84)

Constraint types:
- `ConstraintNone` - Any version acceptable
- `ConstraintEqual` - Exact version match (e.g., `vim=8.2`)
- `ConstraintGreaterEqual` - `vim>=8.2`
- `ConstraintGreater` - `vim>8.2`
- `ConstraintLessEqual` - `vim<=8.2`
- `ConstraintLess` - `vim<8.2`

Validation done in `CheckVersionConstraint()` using RPM version comparison algorithm (`VerCmp()`).

### Version Comparison Algorithm

**Location:** `/pkg/alpm/version.go` (lines 9-34)

Implements exact RPM version scheme:
1. Parse epochs (prefix before `:`)
2. Split release and revision (suffix after last `-`)
3. Compare using `compareRPMVersions()` with segment-based tokenization
4. Handles "1.0" < "1.0.1" cases correctly

### Where Dependency Data Comes From

**Location:** `/pkg/alpm/db.go:252` - `LoadCachedDatabase()`
**Database Location:** `/kod/db/sync/{core,extra,community}.db` files

1. **Sync Process** → `/pkg/database/sync.go:34`
   - `Syncer.Sync()` downloads databases from Arch mirror
   - Format: `https://mirror.rackspace.com/archlinux/{repo}/os/{arch}/{repo}.db`
   - Databases are gzipped tar archives

2. **Parsing** → `/pkg/alpm/parse.go:18`
   - `parsePackageDatabase()` decompresses and reads tar format
   - Extracts per-package metadata files (desc, depends, optdepends, provides, conflicts, replaces)

3. **Storage** → `/pkg/alpm/types.go:9-31`
   - `Package` struct stores all metadata
   - `Database` struct holds per-repository index: `map[string]*Package` and `map[string][]*Package` (provides)
   - `DatabaseCache` merges multiple repos respecting precedence (core > extra > community)

### How the Resolver Handles Conflicts

**Currently:** No explicit conflict resolution implemented

**Data Available:** Conflicts stored in `Package.Conflicts` array
- Parsed from `.db` file `%CONFLICTS%` section → `/pkg/alpm/parse.go:183`
- Can detect conflicting packages but doesn't prevent installation

**Future Enhancement Point:** 
- Check `depPkg.Conflicts` against already-to-install packages
- Could raise `ResolutionError` if conflict detected → `/pkg/alpm/types.go:86`

---

## 2. DATABASE SCHEMA & PACKAGE METADATA

### What's Stored About Packages

**Location:** `/pkg/alpm/types.go:9-31` - `Package` struct

```
Package Fields:
├── Name (string) - Package name
├── Version (string) - Version string (e.g., "5.3.9-1")
├── Description (string) - Short description
├── Architecture (string) - x86_64, aarch64, any
├── URL (string) - Upstream project URL
├── Licenses ([]string) - License identifiers
├── Groups ([]string) - Package groups
├── Provides ([]string) - Virtual packages provided
├── DependsOn ([]string) - Required dependencies
├── OptDepends ([]string) - Optional dependencies
├── Conflicts ([]string) - Conflicting packages
├── Replaces ([]string) - Packages replaced by this one
├── CompressedSize (int64) - Download size
├── InstalledSize (int64) - Installed size
├── Repository (string) - Source repo (core, extra, community)
├── PackageBase (string) - Base package name
├── BuildDate (string) - Build timestamp
├── Maintainer (string) - Package maintainer
├── MD5Sum (string) - Checksum
└── SHA256Sum (string) - Checksum
```

### How Versions Are Tracked

**In Store** → `/pkg/store/store.go`
- Directory structure: `/kod/store/{package}/{version}/` → Lines 37-40
- Symlink: `/kod/store/{package}/current` → Latest version → Lines 42-46, 182-211

**In Registry** → `/pkg/registry/registry.go`
- JSON file: `/kod/registry.json`
- Per-installed-package: name, version, file list, executables, dependencies, install date → Lines 18-25

**Version Comparison:** Uses `VerCmp()` for sorting
- `ListVersions()` sorts descending (newest first) → `/pkg/store/store.go:76-104`

### Database Schema for Sync Databases

**Location:** `/pkg/alpm/types.go:34-41` - `Database` struct

```
Database (in-memory representation):
├── Name (string) - Repository name
├── Path (string) - Disk location (/kod/db/sync/core.db)
├── Packages (map[string]*Package) - Name → Package lookup
├── Provides (map[string][]*Package) - Virtual → Packages providing
└── Arch (string) - Target architecture
```

**Precedence:** Defined in `DefaultRepositoryPriority` → `/pkg/alpm/types.go:100-105`
```
core (0)      - Highest priority
extra (1)
community (2)
multilib (3)  - Lowest priority
```

When same package in multiple repos, lower number wins.

### How AUR Package Metadata Would Differ

**Current Arch Package Fields:**
- Repository: core/extra/community/multilib
- BuildDate: Arch build timestamp
- Packager: Arch maintainer

**AUR-Specific Needed:**
- AUR Build Status: git repo URL, build script location
- AUR Maintainer: Different from Arch packager
- PKGBUILD Location: Source for building
- Build Dependencies: makepkg-deps vs runtime-deps
- Out-of-date Status: AUR-specific flag
- Download Statistics: AUR metrics

**Recommended Storage:**
Option 1: Extend `Package` struct with optional AUR fields
```go
type Package struct {
    // ... existing fields ...
    
    // AUR-specific (optional)
    IsAUR bool
    AURPackageBase string
    AURMaintainer string
    PKGBUILDRef string // git commit/branch
    AURBuildDeps []string
}
```

Option 2: Create `AURPackage` wrapper type inheriting from `Package`

Option 3: Store AUR metadata separately in `/kod/db/aur/{package}.json`

---

## 3. PACKAGE INSTALLATION FLOW

### End-to-End Installation Flow

```
chisel install <package> [--no-deps] [--no-extract] [--no-symlink]
        ↓
main.go:handleInstall() → internal/cli/install.go:InstallCommand.Run()
        ↓
[1. DEPENDENCY RESOLUTION]
    ALPMClient.RegisterAllSyncDBs(repos) → pkg/alpm/db.go:38
            ↓ (load /kod/db/sync/*.db files)
    ALPMClient.ResolveDependencies(pkgName) → pkg/alpm/db.go:126
            ↓ (depth-first, cycle-detection)
    Returns: []string (package names in install order)
        ↓
    For each dependency:
        ALPMClient.GetPackageInfo(depName) → pkg/alpm/alpm.go:143
                ↓
        Skip if already installed (registry check)
                ↓
        Append to toInstall: []download.PackageInfo
        ↓
[2. DOWNLOAD]
    downloader := download.NewDownloader(mirrorURL, cachePath, arch, maxConcurrent)
        ↓
    For each package:
        Construct URL: {mirror}/{repo}/os/{arch}/{name}-{version}-{arch}.pkg.tar.zst
        ↓
        downloader.DownloadPackage() → pkg/download/download.go:45
            - Download to {cachePath}/{name}-{version}-{arch}.pkg.tar.zst.tmp
            - Atomic rename to final location (remove .tmp)
        ↓
[3. EXTRACT]
    storeManager := store.NewStore(storeRoot)
        ↓
    For each downloaded package:
        storeManager.ExtractPackage(cachePath, pkgName, version)
            → pkg/store/store.go:50
            ↓
        extract.NewExtractor(true).ExtractPackage(pkgPath, destDir)
            → pkg/extract/extract.go:39
            ↓
        Extract .pkg.tar.zst to /kod/store/{pkg}/{version}/
            - Decompress zstd
            - Read tar format
            - Handle files, dirs, symlinks, hardlinks
            - Preserve permissions
            ↓
        Store extracted file metadata for later use
        ↓
        storeManager.SetLatestVersion(pkgName, version)
            - Create/update /kod/store/{pkg}/current → {version} symlink
        ↓
[4. SYMLINK CREATION]
    If --no-symlink not set:
        For each package's files:
            If file is usr/bin/* or usr/sbin/*:
                → Will use wrapper script instead (see [6])
            Else:
                symlinkPath = {symlinkDir}/{file}
                targetPath = /kod/store/{pkg}/{version}/{file}
                os.Symlink(targetPath, symlinkPath)
        ↓
[5. WRAPPER GENERATION]
    wrapperGen := wrapper.NewGenerator(storeRoot, wrapperDir, symlinkRoot)
        ↓
    For each executable in usr/bin/ and usr/sbin/:
        wrapperGen.GenerateWrapperWithDeps(cmdName, pkgName, version, libDirs, dependencies)
            → pkg/wrapper/wrapper.go
            ↓
        Creates wrapper shell script in /kod/wrappers/{cmdName}
        Sets LD_LIBRARY_PATH for dependencies
        ↓
    Create symlink: /usr/bin/{cmdName} → /kod/wrappers/{cmdName}
        ↓
[6. REGISTRY UPDATE]
    reg := registry.NewRegistry(registryPath)
        → pkg/registry/registry.go:35
        ↓
    For each installed package:
        regPkg := &registry.Package{
            Name, Version, Files, Executables, Dependencies, InstallDate
        }
        ↓
        reg.AddPackage(regPkg)
        reg.Save() → /kod/registry.json
```

### Key Data Structures in Flow

**Location:** `/internal/cli/install.go` Lines 110-116

```go
type PackageFiles struct {
    AllExtractedFiles []extract.ExtractedFile
    AllFiles []string // All non-directory files
    Executables []string // Files in usr/bin/ or usr/sbin/
}
extractedFilesMap := map[string]map[string]PackageFiles
    // pkgName → version → PackageFiles
```

### Symlink Creation Details

**Location:** `/internal/cli/install.go` Lines 200-293

For each file in a package:
1. Skip metadata files: `.PKGINFO`, `.BUILDINFO`, `.MTREE`, `.INSTALL`
2. Check if file is a symlink in extracted package → use as-is pointing to store
3. If executable (usr/bin/* or usr/sbin/*) → point to wrapper script
4. Otherwise → point directly to store location
5. Check if symlink already exists:
   - If points to correct location: skip
   - If points elsewhere: skip with warning (unless --force)
   - If regular file: skip with warning
6. Create symlink with `os.Symlink(targetPath, symlinkPath)`

**Example Symlink Tree:**
```
/usr/bin/bash → /kod/wrappers/bash
/usr/share/doc/bash/README → /kod/store/bash/5.3.9-1/usr/share/doc/bash/README
/usr/lib/libc.so.6 → /kod/store/glibc/2.37-1/usr/lib/libc.so.6
```

### Database Entry Updates

**Location:** `/internal/cli/install.go` Lines 350-391

Writes to `/kod/registry.json`:
```json
{
  "bash": {
    "name": "bash",
    "version": "5.3.9-1",
    "files": ["usr/bin/bash", "usr/share/..."],
    "executables": ["usr/bin/bash"],
    "dependencies": ["glibc", "ncurses"],
    "install_date": "2024-01-15T10:30:00Z"
  }
}
```

### Where AUR Build Logic Would Inject

**Proposed Extension Points:**

1. **New CLI Command:** `/cmd/chisel/main.go` (lines 41-87)
   ```go
   case "aur-install":
       handleAURInstall(args[1:])
   case "aur-search":
       handleAURSearch(args[1:])
   ```

2. **New ALPM Variant:** `/pkg/alpm/aur.go` (new file)
   - `AURClient` similar to `ALPMClient`
   - Fetch metadata from AUR API
   - Clone PKGBUILD from git

3. **New Build Stage:** `/internal/cli/aur_install.go` (new file)
   ```
   Before [2. DOWNLOAD]:
   - Get PKGBUILD from AUR git
   - Run makepkg build process
   - Output: .pkg.tar.zst in cache directory
   - Continue with normal flow [2. onwards]
   ```

4. **Build Environment:** `/pkg/build/builder.go` (new file)
   - Run `makepkg` in sandboxed environment
   - Mount dependencies from store
   - Handle build-time library dependencies
   - Set up environment variables for cross-distro builds

---

## 4. DOWNLOAD & EXTRACT SYSTEM

### Download Implementation

**Location:** `/pkg/download/download.go`

**Main Entry Point:** `Downloader.DownloadPackages()` (line 103)

```go
type Downloader struct {
    mirrorURL string
    cachePath string
    arch string
    maxConcurrent int
    downloadTimeout time.Duration
    httpClient *http.Client
}

func NewDownloader(mirrorURL, cachePath, arch string, maxConcurrent int, timeout time.Duration)
```

**URL Construction** → Line 57:
```
{mirrorURL}/{repo}/os/{arch}/{name}-{version}-{arch}.pkg.tar.zst
Example: https://mirror.rackspace.com/archlinux/core/os/x86_64/bash-5.3.9-1-x86_64.pkg.tar.zst
```

**Download Process** → Line 45-99

For each package:
1. Create cache directory if missing
2. HTTP GET with configurable timeout
3. Check status code (must be 200 OK)
4. Write to temporary file: `{cache}/{filename}.tmp`
5. Atomic rename to final: `{cache}/{filename}`
6. Return local path

**Concurrency** → Line 109-110
- Semaphore pattern limits concurrent downloads
- Default: 5 concurrent → `/pkg/config/config.go:97`

**Cache Check** → Line 157-163
- `PackageExists()`: Check if already downloaded
- `GetLocalPath()`: Compute cache path

### Extraction Implementation

**Location:** `/pkg/extract/extract.go`

**Main Entry Point:** `Extractor.ExtractPackage()` (line 39)

```go
type Extractor struct {
    preservePerms bool // Preserve original file permissions
}

func NewExtractor(preservePerms bool) *Extractor
```

**Archive Format:** `.pkg.tar.zst`
- Zstd compression: `github.com/klauspost/compress/zstd`
- Tar format: `archive/tar`

**Extraction Process** → Lines 39-189

1. **Open & Decompress:**
   - Open `.pkg.tar.zst` file
   - Create zstd decoder
   - Create tar reader

2. **Directory Creation:**
   - `os.MkdirAll(destDir, 0755)`
   - Create `/kod/store/{pkg}/{version}` structure

3. **File Processing Loop:**
   For each tar entry:
   - **Regular File** (TypeReg):
     - Create parent directories
     - Create output file
     - Copy contents from tar to disk
     - Optionally set permissions
   - **Directory** (TypeDir):
     - `os.MkdirAll(targetPath, 0755)`
   - **Symlink** (TypeSymlink):
     - `os.Symlink(header.Linkname, targetPath)`
   - **Hard Link** (TypeLink):
     - `os.Link(linkTargetPath, targetPath)`

4. **Security:**
   - Path traversal prevention: `HasPrefix(filepath.Clean(targetPath), filepath.Clean(destDir))`
   - Prevents archive from writing outside destination

5. **Return Value:**
   ```go
   type ExtractedFile struct {
       Path string         // Relative in archive: "usr/bin/bash"
       AbsPath string      // Absolute: "/kod/store/bash/5.3.9-1/usr/bin/bash"
       IsDirectory bool
       IsSymlink bool      // True for symlinks and hardlinks
       LinkTarget string   // Target of symlink
       Size int64          // File size
       Mode os.FileMode    // Permissions
   }
   ```

### Integration Points

**Download → Extract Chain** → `/internal/cli/install.go:118-191`

1. Download packages using `Downloader.DownloadPackages(packages)`
2. Get results map: `{packageName: localPath}`
3. For each result:
   - Call `StoreManager.ExtractPackage(cachePath, pkgName, version)`
   - Store returned `[]ExtractedFile` for later symlink/wrapper use

**Store Manager** → `/pkg/store/store.go:50`

```go
func (s *Store) ExtractPackage(pkgPath, pkgName, version string) ([]ExtractedFile, error)
```

Wraps `Extractor.ExtractPackage()` with store-aware paths.

### Where AUR Artifacts Would Integrate

**Build Output:** AUR build produces same `.pkg.tar.zst` format
- `makepkg` creates package in build directory
- Move to cache: `/kod/cache/{name}-{version}-{arch}.pkg.tar.zst`
- Then continue with normal [Extract] flow

**No Changes Needed** to extract logic - same format!

**Build-Specific Caching:**
- Could cache PKGBUILD sources separately
- Cache failed builds for debugging
- Cache build logs in `/kod/build-logs/`

---

## 5. CLI COMMAND STRUCTURE

### Command Entry Points

**Main Router** → `/cmd/chisel/main.go:22-87`

```go
func main() {
    flag.Parse()
    command := args[0]
    
    switch command {
    case "sync": handleSync()
    case "search": handleSearch()
    case "info": handleInfo()
    case "download": handleDownload()
    case "extract": handleExtract()
    case "install": handleInstall()
    case "remove": handleRemove()
    case "list": handleList()
    case "upgrade": handleUpgrade()
    case "cleanup": handleCleanup()
    case "cache": handleCache()
    }
}
```

### Command Handler Pattern

**Location:** `/cmd/chisel/main.go` and `/internal/cli/`

Each handler:
1. Parses command-line flags
2. Calls `loadConfig()` to get configuration
3. Creates command object from `/internal/cli/`
4. Calls `Execute()` or `Run()` method
5. Handles errors and exit codes

**Example** → `/cmd/chisel/main.go:317-335` - handleInstall

```go
func handleInstall(args []string) {
    cfg := loadConfig()
    cmd := cli.NewInstallCommandWithSymlinkDir(cfg, symlinkDir)
    if err := cmd.Run(args); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

### Command Implementation Structure

**Internal CLI Package** → `/internal/cli/`

Each command is a struct + methods:
```
{Command}Command struct {
    config *config.Config
    [command-specific fields]
}

NewCommand(cfg *config.Config) *{Command}Command
(c *{Command}Command) Run(args []string) error
(c *{Command}Command) Execute(args...) error
(c *{Command}Command) Help() string
```

**Existing Commands** (with flow):

| Command | File | Entry | Processing |
|---------|------|-------|-----------|
| sync | sync.go | Execute() | Download .db files via database.Syncer |
| search | search.go | Execute(pattern) | Query ALPM cache, regex match |
| info | info.go | Execute(name) | Get PackageInfo from ALPM |
| download | download.go | Run(args) | Resolve packages, download via Downloader |
| extract | extract.go | Run(args) | Parse filenames, extract to store |
| install | install.go | Run(args) | Full flow: resolve → download → extract → symlink → wrapper → registry |
| remove | remove.go | Run(args) | Remove symlinks, registry entry, optionally store |
| list | list.go | Execute() | Read registry.json, display installed |
| upgrade | upgrade.go | Execute() | Compare versions, install updates |
| cleanup | cleanup.go | Execute() | Remove old versions from store |
| cache | cache.go | Execute() | Clean download cache |

### Where AUR-Specific Logic Would Live

**Proposed Structure:**

```
cmd/chisel/main.go
├── case "aur-install": handleAURInstall(args)
├── case "aur-search": handleAURSearch(args)
├── case "aur-info": handleAURInfo(args)
└── case "aur-upgrade": handleAURUpgrade(args)

internal/cli/
├── aur_install.go (AURInstallCommand)
│   ├── Search AUR
│   ├── Clone/build with makepkg
│   ├── Fall through to normal install flow
│   └── Update registry with AUR source flag
├── aur_search.go (AURSearchCommand)
│   ├── Query AUR API
│   ├── Format results
├── aur_info.go (AURInfoCommand)
│   ├── Get PKGBUILD details
│   ├── Show dependencies
├── aur_upgrade.go (AURUpgradeCommand)
│   ├── Check AUR for updates
│   └── Rebuild if needed

pkg/alpm/ (or pkg/aur/)
├── aur_client.go (AURClient)
│   ├── AUR API queries
│   ├── PKGBUILD fetching
│   └── Metadata parsing

pkg/build/
├── builder.go (PackageBuilder)
│   ├── Setup build environment
│   ├── Run makepkg
│   ├── Handle dependencies
│   └── Capture output
```

### Global Configuration

**Location:** `/cmd/chisel/main.go:23-29`

Global flags (applied before command):
```
-c, --config {path}        Path to config file
--base-dir {path}          Base directory (/kod default)
--mirror {url}             Arch mirror URL
--symlink-dir {path}       Symlink root directory
```

**Config Loading** → `/cmd/chisel/main.go:152-202`

Priority order:
1. Command-line flags
2. Environment variables (CHISEL_CONFIG, CHISEL_BASE_DIR, CHISEL_MIRROR)
3. Config file (/etc/chisel/config.json)
4. Built-in defaults → `/pkg/config/config.go:80-101`

**Config Structure** → `/pkg/config/config.go:22-78`

```go
type Config struct {
    BaseDir string                  // /kod
    SymlinkRoot string              // /
    StoreRoot string                // /kod/store
    RegistryPath string             // /kod/registry.json
    DBPath string                   // /kod/db/sync
    CachePath string                // /kod/cache
    WrapperDir string               // /kod/wrappers
    MirrorURL string                // https://mirror.rackspace.com/archlinux
    Architecture string             // x86_64
    Repositories []string           // [core, extra, community]
    VerifySignatures bool
    MaxConcurrentDownloads int      // 5
    DownloadTimeout int             // 300 seconds
    KeepVersions int                // 3
}
```

### Future Extension Points for AUR Commands

**New Command Registration** → `/cmd/chisel/main.go:41-87`
```go
case "aur-install":
    handleAURInstall(args[1:])
case "aur-search":
    handleAURSearch(args[1:])
```

**New CLI Command Classes** → `/internal/cli/{aur_*.go}`
```go
type AURInstallCommand struct {
    config *config.Config
    builder *build.PackageBuilder
}

func (c *AURInstallCommand) Run(args []string) error {
    // Parse package names from args
    // For each package:
    //   1. Search AUR
    //   2. Clone PKGBUILD
    //   3. Run build.Builder.Build()
    //   4. Add to cache
    //   5. Fall through to normal Install flow
}
```

**New Builder Integration** → `/pkg/build/builder.go`
```go
type PackageBuilder struct {
    aurDir string
    buildDir string
    storeRoot string
}

func (b *PackageBuilder) Build(pkgName, maintainer string) (string, error) {
    // Clone PKGBUILD from AUR
    // Setup build environment
    // Run makepkg
    // Return path to built .pkg.tar.zst
}
```

---

## Summary: Key Integration Points

### For Official Packages (Current Flow)
1. **DB Sync** → `/pkg/database/sync.go` downloads `.db` files
2. **ALPM Parse** → `/pkg/alpm/parse.go` extracts metadata
3. **Dependency Resolution** → `/pkg/alpm/db.go:resolveDepsRecursive()` builds install order
4. **Download** → `/pkg/download/download.go` fetches `.pkg.tar.zst`
5. **Extract** → `/pkg/extract/extract.go` uncompresses to store
6. **Symlinks** → `/internal/cli/install.go` creates filesystem hierarchy
7. **Registry** → `/pkg/registry/registry.go` tracks installation

### For AUR Packages (Proposed Extensions)
1. **New Commands** → `/cmd/chisel/main.go` - `aur-install`, `aur-search`, etc.
2. **AUR Client** → `/pkg/alpm/aur.go` (new) or `/pkg/aur/` - fetch from AUR API
3. **Build System** → `/pkg/build/builder.go` (new) - makepkg wrapper
4. **AUR Install Command** → `/internal/cli/aur_install.go` (new)
   - Stages: Search → Clone PKGBUILD → Build → Cache → Install
5. **Metadata Extension** → `/pkg/alpm/types.go` - add AUR fields to Package struct
6. **Registry Update** → Mark packages as AUR-sourced in registry

---

## File Reference Index

| Component | Files | Key Lines |
|-----------|-------|-----------|
| **CLI Router** | cmd/chisel/main.go | 22-87, 152-202 |
| **Dependency Resolution** | pkg/alpm/db.go | 126-207 |
| **Version Comparison** | pkg/alpm/version.go | 9-34 |
| **ALPM Public API** | pkg/alpm/alpm.go | 38-206 |
| **Database Parsing** | pkg/alpm/parse.go | 14-250 |
| **Database Caching** | pkg/alpm/cache.go | 3-95 |
| **Database Sync** | pkg/database/sync.go | 34-116 |
| **Download Manager** | pkg/download/download.go | 24-177 |
| **Extraction** | pkg/extract/extract.go | 39-272 |
| **Store Management** | pkg/store/store.go | 26-273 |
| **Symlink Management** | pkg/symlink/symlink.go | 12-206 |
| **Registry** | pkg/registry/registry.go | 35-116 |
| **Install Command** | internal/cli/install.go | 56-524 |
| **Download Command** | internal/cli/download.go | 26-108 |
| **Extract Command** | internal/cli/extract.go | 26-145 |
| **Sync Command** | internal/cli/sync.go | 27-91 |
| **Config** | pkg/config/config.go | 22-200+ |

