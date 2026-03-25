# Implementation Plan for Chisel (Cross-Distribution)

This document provides a detailed, step-by-step implementation plan for the chisel cross-distribution system. It breaks down the work into phases, defines priorities, and establishes clear milestones.

**Version:** 4.0 (Cross-Distribution Architecture)  
**Date:** 2026-03-21  
**Estimated Timeline:** 7-9 weeks (single developer)

## Table of Contents

1. [Overview](#overview)
2. [Current State](#current-state)
3. [Cross-Distribution Goals](#cross-distribution-goals)
4. [Phase 1: Foundation & ALPM](#phase-1-foundation--alpm-week-1-15)
5. [Phase 2: Storage & Extraction](#phase-2-storage--extraction-week-15-25)
6. [Phase 3: Wrapper & Symlink Management](#phase-3-wrapper--symlink-management-week-25-35)
7. [Phase 4: Package Installation](#phase-4-package-installation-week-35-5)
8. [Phase 5: Package Removal & Queries](#phase-5-package-removal--queries-week-5-6)
9. [Phase 6: Testing & Polish](#phase-6-testing--polish-week-6-7)
10. [Phase 7: Documentation](#phase-7-documentation-week-7)
11. [Testing Strategy](#testing-strategy)
12. [Success Criteria](#success-criteria)
13. [Timeline & Resources](#timeline--resources)

---

## 1. Overview

### Purpose
Build a cross-distribution package manager in Go that brings Arch Linux packages to ANY Linux distribution (Ubuntu, Fedora, Debian, etc.) using complete dependency isolation and wrapper scripts.

### Approach
- **Incremental**: Build feature by feature
- **Test-Driven**: Write tests alongside implementation
- **Multi-Distribution**: Test on Ubuntu, Fedora, Debian using Docker
- **Pragmatic**: Ship working software, not perfect software
- **Focused**: Core features only in v1.0

### Scope

**In Scope (v1.0):**
- **Cross-distribution support** (Ubuntu 22.04+, Fedora 39+, Debian 12+, Arch)
- **Database sync** from Arch mirrors
- **Wrapper script generation** for library isolation
- **Complete dependency isolation** (install ALL deps from Arch)
- Package installation with full dependency resolution
- Package removal
- Package upgrade
- Search and query
- Symlink management (to wrappers)
- Central package store (`/kod/`)
- Registry management

**Out of Scope (Future):**
- Generation management (v2.0)
- Boot integration (v2.0)
- Web UI (v2.0)
- Package script execution (v1.1)
- AUR support (v2.5)
- ARM64 architecture (v1.1)

---

## 2. Current State

**Starting Point:** Specification documents updated to v4.0, some foundation code exists.

**Available Resources:**
- ✅ Complete specification v4.0 (00-SPECIFICATION.md)
- ✅ Cross-distribution architecture diagrams (01-DIAGRAMS.md)
- ✅ Design decisions documented (02-CRITICAL-DECISIONS.md)
- ✅ Go module initialized (`go.mod` with go-alpm/v2)
- ✅ Project structure created
- ✅ Config package COMPLETE (85% test coverage)
- ✅ Registry package COMPLETE (75% test coverage)
- ✅ Basic CLI structure
- ❌ ALPM wrapper not implemented
- ❌ Database sync not implemented
- ❌ Wrapper generation not implemented
- ❌ Installation workflow not implemented

---

## 3. Cross-Distribution Goals

### Primary Goals
1. **Cross-Distribution Compatibility**: Work on Ubuntu, Fedora, Debian, Arch
2. **Complete Isolation**: ALL dependencies from Arch, not host system
3. **Library Safety**: Never mix host and Arch libraries
4. **Functionality**: Install, remove, and query packages
5. **Reliability**: Packages work correctly after installation
6. **Usability**: Simple CLI interface
7. **Maintainability**: Clean, documented code
8. **Testability**: Good test coverage + multi-distro Docker tests

### Success Metrics
- [ ] Can sync databases from Arch mirrors
- [ ] Can install packages with ALL dependencies (including glibc)
- [ ] Wrapper scripts correctly set LD_LIBRARY_PATH
- [ ] Can remove packages safely
- [ ] Can search and query packages
- [ ] Works on Ubuntu 22.04, Fedora 40, Debian 12
- [ ] Symlinks and wrappers work correctly
- [ ] Registry stays synchronized
- [ ] 80%+ code coverage
- [ ] Docker integration tests pass on all target distros

---

## 4. Phase 1: Foundation & ALPM (Week 1-1.5)

**Goal**: Project setup, configuration, ALPM integration with `/kod` root, and database sync

**Duration**: 7-10 days

**Status**: PARTIALLY COMPLETE
- ✅ Go module initialized
- ✅ Project structure created
- ✅ Config package COMPLETE (85% test coverage)
- ✅ Registry package COMPLETE (75% test coverage)  
- ✅ Basic CLI structure
- ❌ ALPM wrapper needs `/kod` root implementation
- ❌ Database sync needs implementation

### 1.1 Enhanced Configuration (COMPLETE)

**Status**: ✅ DONE

**Completed**:
- ✅ JSON configuration format
- ✅ Config loading and validation
- ✅ Tests (85% coverage)

**Files**: `pkg/config/config.go`, `pkg/config/config_test.go`

### 1.2 Database Sync System (NEW - Days 1-3)

**Tasks**:
- [ ] Implement database downloader
- [ ] Download core.db, extra.db, community.db from Arch mirrors
- [ ] Save to `/kod/db/`
- [ ] Handle download errors and retries
- [ ] Progress indication
- [ ] Write tests

**Files**:
```
pkg/database/
├── sync.go       # Database sync implementation
└── sync_test.go  # Tests
```

**Key Functions**:
```go
// sync.go
type DatabaseSync struct {
    mirrorURL    string
    dbPath       string
    httpClient   *http.Client
}

func NewDatabaseSync(mirrorURL, dbPath string) *DatabaseSync
func (ds *DatabaseSync) Sync(repos []string) error
func (ds *DatabaseSync) downloadDatabase(repo, arch string) error
func (ds *DatabaseSync) verifyDatabase(dbPath string) error
```

**Implementation Details**:
```go
// Download URL format:
// https://mirror.rackspace.com/archlinux/core/os/x86_64/core.db
// https://mirror.rackspace.com/archlinux/extra/os/x86_64/extra.db

// Save to:
// /kod/db/core.db
// /kod/db/extra.db
```

**Deliverables**:
- Can download databases from Arch mirrors
- Saves to `/kod/db/`
- Error handling for network failures
- Progress reporting

**Estimated Time**: 2-3 days

### 1.3 ALPM Wrapper with Custom Root (Days 4-7)

**Tasks**:
- [ ] Initialize ALPM with `/kod` root (NOT `/`)
- [ ] Register sync databases from `/kod/db/`
- [ ] Implement package queries
- [ ] Implement dependency resolution (ALL deps, including system libs)
- [ ] Write comprehensive tests

**Files**:
```
pkg/alpm/
├── alpm.go       # ALPM wrapper with /kod root
└── alpm_test.go  # Tests
```

**Key Functions**:
```go
// alpm.go
type ALPMWrapper struct {
    handle *alpm.Handle
}

// Initialize with /kod root instead of /
func NewALPMWrapper(kodRoot, dbPath string) (*ALPMWrapper, error) {
    handle, err := alpm.Initialize(kodRoot, dbPath)  // "/kod", "/kod/db"
    // ...
}

func (a *ALPMWrapper) RegisterSyncDB(name string) error
func (a *ALPMWrapper) SearchPackages(query string) ([]Package, error)
func (a *ALPMWrapper) GetPackage(name string) (*Package, error)
func (a *ALPMWrapper) ResolveDependencies(pkgName string) ([]string, error)
func (a *ALPMWrapper) Close() error
```

**Critical Implementation**:
```go
// Initialize ALPM with /kod root (works on ANY distribution!)
handle, err := alpm.Initialize("/kod", "/kod/db")

// Register sync databases (downloaded via database sync)
handle.RegisterSyncDB("core", 0)
handle.RegisterSyncDB("extra", 0)

// Now can query packages and resolve dependencies
pkg := handle.SyncDbByName("core").Pkg("vim")
```

**Deliverables**:
- ALPM initialized with `/kod` root
- Can query Arch packages on Ubuntu/Fedora/Debian
- Dependency resolution works
- 80%+ test coverage

**Estimated Time**: 3-4 days

### 1.4 CLI Commands (Days 7-10)

**Tasks**:
- [ ] Implement `chisel sync` command
- [ ] Implement `chisel search` command
- [ ] Implement `chisel info` command
- [ ] Test CLI parsing

**Files**:
```
internal/cli/
├── root.go
├── sync.go     # NEW: sync command
├── search.go
└── info.go
```

**Commands**:
```bash
chisel sync                    # Sync databases from Arch mirrors
chisel search <query>          # Search packages
chisel info <package>          # Show package info
chisel --help
chisel --version
```

**Deliverables**:
- `chisel sync` downloads databases
- `chisel search` works
- `chisel info` shows package details

**Estimated Time**: 2-3 days

**Phase 1 Total**: 7-10 days (1-1.5 weeks)

---

**Goal**: Basic Go project setup and ALPM integration

**Duration**: 5-7 days

### 1.1 Project Setup (Days 1-2)

**Tasks**:
- [x] Create Go module structure
- [ ] Set up directory structure
- [ ] Configure development tools
- [ ] Initialize git repository
- [ ] Create Makefile

**Commands**:
```bash
mkdir -p chisel-go/{cmd/chisel,internal/{alpm,package,storage,download,config,cli},pkg/registry}
cd chisel-go
go mod init github.com/yourusername/chisel-go
go get github.com/Jguer/go-alpm/v2
go get github.com/spf13/cobra
touch Makefile README.md
```

**Deliverables**:
- Clean project structure
- `go.mod` with dependencies
- Makefile with build targets
- README.md with basic info

**Estimated Time**: 1-2 days

### 1.2 ALPM Wrapper (Days 2-4)

**Tasks**:
- [ ] Implement ALPM handle initialization
- [ ] Add database operations (sync DBs)
- [ ] Add package queries
- [ ] Add dependency resolution
- [ ] Write unit tests

**Files**:
```
internal/alpm/
├── wrapper.go      # Handle management
├── database.go     # DB operations
├── package.go      # Package queries
└── wrapper_test.go # Tests
```

**Key Functions**:
```go
// wrapper.go
type ALPMWrapper struct {
    handle *alpm.Handle
}

func NewALPMWrapper(root, dbpath string) (*ALPMWrapper, error)
func (a *ALPMWrapper) SyncDatabases() error
func (a *ALPMWrapper) SearchPackages(query string) ([]Package, error)
func (a *ALPMWrapper) GetPackage(name string) (*Package, error)
func (a *ALPMWrapper) ResolveDependencies(pkgName string) ([]string, error)
func (a *ALPMWrapper) Close() error
```

**Deliverables**:
- Working ALPM integration
- Can query Arch repositories
- 80%+ test coverage

**Estimated Time**: 2-3 days

### 1.3 Configuration Management (Days 4-5)

**Tasks**:
- [ ] Define configuration struct
- [ ] Implement YAML parsing
- [ ] Add default configuration
- [ ] Add validation
- [ ] Write tests

**Files**:
```
internal/config/
├── config.go      # Config struct and loading
└── config_test.go # Tests
```

**Configuration Structure**:
```go
type Config struct {
    StorePath              string   `yaml:"store_path"`
    RegistryPath           string   `yaml:"registry_path"`
    CachePath              string   `yaml:"cache_path"`
    MaxConcurrentDownloads int      `yaml:"max_concurrent_downloads"`
    VerifySignatures       bool     `yaml:"verify_signatures"`
    LogLevel               string   `yaml:"log_level"`
}
```

**Deliverables**:
- Configuration loading works
- Default config provided
- Validation working

**Estimated Time**: 1-2 days

### 1.4 Basic CLI (Days 5-7)

**Tasks**:
- [ ] Set up Cobra CLI framework
- [ ] Add root command
- [ ] Add basic commands (search, info)
- [ ] Add global flags
- [ ] Test CLI parsing

**Files**:
```
cmd/chisel/main.go
internal/cli/
├── root.go
├── search.go
└── info.go
```

**Commands to Implement**:
```bash
chisel search <query>
chisel info <package>
chisel --help
chisel --version
```

**Deliverables**:
- Basic CLI works
- Can search packages
- Can show package info

**Estimated Time**: 2-3 days

**Phase 1 Total**: 5-7 days

---

## 5. Phase 2: Storage Management (Week 2)

**Goal**: Package extraction and storage management

**Duration**: 5-7 days

### 2.1 Package Store (Days 1-3)

**Tasks**:
- [ ] Implement store directory structure
- [ ] Add package extraction logic
- [ ] Add store manifest management
- [ ] Write tests

**Files**:
```
internal/storage/
├── store.go          # Store operations
├── extractor.go      # Tar.zst extraction
├── manifest.go       # Store manifest
└── store_test.go     # Tests
```

**Key Functions**:
```go
type Store struct {
    basePath string
    manifest *Manifest
}

func NewStore(basePath string) (*Store, error)
func (s *Store) ExtractPackage(pkgPath, name, version string) error
func (s *Store) GetPackagePath(name, version string) string
func (s *Store) PackageExists(name, version string) bool
func (s *Store) RemovePackage(name, version string) error
```

**Deliverables**:
- Can extract packages to store
- Store manifest tracking works
- Directory structure correct

**Estimated Time**: 2-3 days

### 2.2 Registry Management (Days 3-5)

**Tasks**:
- [ ] Define registry data structure
- [ ] Implement JSON serialization
- [ ] Add CRUD operations
- [ ] Add atomic file writes
- [ ] Write tests

**Files**:
```
internal/package/
├── registry.go      # Registry operations
└── registry_test.go # Tests

pkg/registry/
└── types.go         # Public types
```

**Registry Operations**:
```go
type Registry struct {
    Version      string     `json:"version"`
    Updated      time.Time  `json:"updated"`
    Packages     []Package  `json:"packages"`
    TotalPackages int       `json:"total_packages"`
    TotalSize    int64      `json:"total_size"`
}

func LoadRegistry(path string) (*Registry, error)
func (r *Registry) Save(path string) error
func (r *Registry) AddPackage(pkg Package) error
func (r *Registry) RemovePackage(name string) error
func (r *Registry) GetPackage(name string) (*Package, error)
func (r *Registry) ListPackages() []Package
```

**Deliverables**:
- Registry persists correctly
- Atomic file writes work
- CRUD operations tested

**Estimated Time**: 2-3 days

### 2.3 Download Manager (Days 5-7)

**Tasks**:
- [ ] Implement HTTP download
- [ ] Add progress tracking
- [ ] Add concurrent downloads
- [ ] Add checksum verification
- [ ] Write tests

**Files**:
```
internal/download/
├── fetcher.go       # HTTP download
├── progress.go      # Progress bars
├── verifier.go      # Verification
└── fetcher_test.go  # Tests
```

**Key Functions**:
```go
type Downloader struct {
    cachePath string
    maxConcurrent int
}

func NewDownloader(cachePath string, maxConcurrent int) *Downloader
func (d *Downloader) DownloadPackage(url, destPath string) error
func (d *Downloader) DownloadPackages(urls []string) error
func (d *Downloader) VerifyChecksum(filePath, expectedSum string) error
```

**Deliverables**:
- Can download packages
- Progress bars work
- Concurrent downloads work

**Estimated Time**: 2-3 days

**Phase 2 Total**: 5-7 days

---

## 6. Phase 3: Symlink Management (Week 2-3)

**Goal**: Create and manage symlinks from system to store

**Duration**: 5-7 days

### 3.1 Symlink Operations (Days 1-4)

**Tasks**:
- [ ] Implement symlink creation
- [ ] Add conflict detection
- [ ] Implement symlink removal
- [ ] Add verification
- [ ] Handle edge cases (existing files, broken symlinks)
- [ ] Write tests

**Files**:
```
internal/storage/
├── symlink.go       # Symlink operations
└── symlink_test.go  # Tests
```

**Key Functions**:
```go
type SymlinkManager struct {
    registry *Registry
}

func NewSymlinkManager(registry *Registry) *SymlinkManager
func (s *SymlinkManager) CreateSymlink(target, link string) error
func (s *SymlinkManager) RemoveSymlink(link string) error
func (s *SymlinkManager) CreateSymlinksForPackage(pkg Package) error
func (s *SymlinkManager) RemoveSymlinksForPackage(pkg Package) error
func (s *SymlinkManager) VerifySymlink(link string) error
func (s *SymlinkManager) VerifyAllSymlinks() error
```

**Deliverables**:
- Can create symlinks safely
- Conflict detection works
- Can remove symlinks cleanly

**Estimated Time**: 3-4 days

### 3.2 Verification System (Days 4-7)

**Tasks**:
- [ ] Implement symlink verification
- [ ] Add registry consistency checks
- [ ] Add store consistency checks
- [ ] Implement repair functionality
- [ ] Write tests

**Files**:
```
internal/package/
├── verifier.go      # Verification logic
└── verifier_test.go # Tests
```

**Verification Checks**:
```go
func VerifySymlinks() error                    // All symlinks valid
func VerifyRegistryConsistency() error         // Registry matches reality
func VerifyStoreConsistency() error            // Store matches manifest
func RepairSymlinks() error                    // Fix broken symlinks
```

**Deliverables**:
- Verification command works
- Can detect inconsistencies
- Can repair broken state

**Estimated Time**: 2-3 days

**Phase 3 Total**: 5-7 days

---

## 7. Phase 4: Package Installation (Week 3-4)

**Goal**: Complete end-to-end package installation

**Duration**: 7-10 days

### 4.1 Installation Logic (Days 1-5)

**Tasks**:
- [ ] Implement installation workflow
- [ ] Integrate dependency resolution
- [ ] Add transaction support
- [ ] Implement rollback on failure
- [ ] Write integration tests

**Files**:
```
internal/package/
├── installer.go      # Installation logic
├── transaction.go    # Transaction support
└── installer_test.go # Tests
```

**Installation Flow**:
```go
type Installer struct {
    alpm      *ALPMWrapper
    store     *Store
    symlink   *SymlinkManager
    registry  *Registry
    downloader *Downloader
}

func NewInstaller(...) *Installer
func (i *Installer) Install(packages []string) error
func (i *Installer) installPackage(pkg string) error
```

**Steps**:
1. Resolve dependencies via ALPM
2. Check for conflicts
3. Download packages
4. Verify signatures/checksums
5. Extract to store
6. Create symlinks
7. Update registry
8. Rollback on any error

**Deliverables**:
- Complete installation works
- Dependencies resolved
- Rollback works on failure

**Estimated Time**: 4-5 days

### 4.2 CLI Integration (Days 5-7)

**Tasks**:
- [ ] Add install command
- [ ] Add progress output
- [ ] Add confirmation prompts
- [ ] Add dry-run mode
- [ ] Test CLI thoroughly

**Files**:
```
internal/cli/
└── install.go       # Install command
```

**Command**:
```bash
chisel install <package>...
  --no-confirm       Skip confirmation
  --dry-run          Show what would be installed
  --as-dep           Mark as dependency
```

**Deliverables**:
- Install command works
- Good user feedback
- Dry-run mode helpful

**Estimated Time**: 2-3 days

### 4.3 Upgrade Support (Days 7-10)

**Tasks**:
- [ ] Implement upgrade logic
- [ ] Add upgrade command
- [ ] Handle version conflicts
- [ ] Test upgrades thoroughly

**Files**:
```
internal/package/
└── upgrader.go      # Upgrade logic

internal/cli/
└── upgrade.go       # Upgrade command
```

**Deliverables**:
- Can upgrade packages
- Handles dependencies correctly
- Works reliably

**Estimated Time**: 2-3 days

**Phase 4 Total**: 7-10 days

---

## 8. Phase 5: Package Removal & Queries (Week 4-5)

**Goal**: Package removal and query functionality

**Duration**: 7-10 days

### 5.1 Package Removal (Days 1-5)

**Tasks**:
- [ ] Implement removal logic
- [ ] Add reverse dependency checking
- [ ] Implement orphan detection
- [ ] Add remove command
- [ ] Write tests

**Files**:
```
internal/package/
├── remover.go       # Removal logic
└── remover_test.go  # Tests

internal/cli/
└── remove.go        # Remove command
```

**Removal Flow**:
```go
type Remover struct {
    alpm     *ALPMWrapper
    symlink  *SymlinkManager
    registry *Registry
    store    *Store
}

func (r *Remover) Remove(packages []string) error
func (r *Remover) CheckReverseDeps(pkg string) ([]string, error)
func (r *Remover) FindOrphans() ([]string, error)
```

**Deliverables**:
- Safe package removal
- Orphan detection works
- User prompted appropriately

**Estimated Time**: 3-5 days

### 5.2 Query Commands (Days 5-8)

**Tasks**:
- [ ] Implement list command
- [ ] Implement files command
- [ ] Add query filters
- [ ] Improve search output
- [ ] Write tests

**Files**:
```
internal/cli/
├── list.go          # List installed
├── files.go         # Show files
└── query.go         # Enhanced queries
```

**Commands**:
```bash
chisel list [--explicit|--deps]
chisel files <package>
chisel search <query>
chisel info <package>
```

**Deliverables**:
- All query commands work
- Output is clean and useful
- Filters work correctly

**Estimated Time**: 2-3 days

### 5.3 Cleanup Command (Days 8-10)

**Tasks**:
- [ ] Implement cleanup logic
- [ ] Add keep-versions support
- [ ] Add cleanup command
- [ ] Test cleanup scenarios

**Files**:
```
internal/package/
└── cleanup.go       # Cleanup logic

internal/cli/
└── cleanup.go       # Cleanup command
```

**Command**:
```bash
chisel cleanup [--keep N] [--dry-run] [--aggressive]
```

**Deliverables**:
- Cleanup removes unused packages
- Keep-versions works
- Safe and tested

**Estimated Time**: 2-3 days

**Phase 5 Total**: 7-10 days

---

## 9. Phase 6: Polish & Testing (Week 5-6)

**Goal**: Production readiness

**Duration**: 7-14 days

### 6.1 Error Handling (Days 1-3)

**Tasks**:
- [ ] Improve error messages
- [ ] Add context to errors
- [ ] Implement better logging
- [ ] Add debug mode
- [ ] Test error paths

**Improvements**:
- Clear error messages with suggestions
- Debug logging for troubleshooting
- Graceful handling of edge cases
- Good error recovery

**Deliverables**:
- Helpful error messages
- Debug mode works
- Error handling robust

**Estimated Time**: 2-3 days

### 6.2 Testing (Days 3-7)

**Tasks**:
- [ ] Increase unit test coverage to 70%+
- [ ] Write integration tests
- [ ] Add end-to-end tests
- [ ] Test on real Arch system
- [ ] Fix discovered bugs

**Test Types**:
- **Unit Tests**: Test individual functions
- **Integration Tests**: Test component interaction
- **E2E Tests**: Test full workflows on test system
- **Manual Tests**: Install/remove real packages

**Deliverables**:
- 70%+ code coverage
- Integration tests pass
- Works on real Arch system
- Known bugs documented

**Estimated Time**: 3-5 days

### 6.3 Documentation (Days 7-10)

**Tasks**:
- [ ] Write comprehensive README
- [ ] Add inline code documentation
- [ ] Create usage examples
- [ ] Write troubleshooting guide
- [ ] Add contributing guide

**Documentation**:
```
README.md              # Overview, installation, quick start
docs/
├── installation.md    # Detailed installation
├── usage.md           # Usage guide
├── troubleshooting.md # Common issues
└── contributing.md    # How to contribute
```

**Deliverables**:
- Complete README
- Good inline docs
- Usage examples
- Troubleshooting guide

**Estimated Time**: 2-3 days

### 6.4 Installation Script (Days 10-12)

**Tasks**:
- [ ] Create installation script
- [ ] Add system checks
- [ ] Create initial setup wizard
- [ ] Test installation process

**Script Features**:
```bash
# install.sh
- Check for Arch Linux
- Check for dependencies (libalpm)
- Create /kod directory structure
- Copy binary to /usr/bin
- Create default config
- Initialize registry
```

**Deliverables**:
- Working installation script
- Clean installation process
- Good error messages

**Estimated Time**: 1-2 days

### 6.5 Performance Optimization (Days 12-14)

**Tasks**:
- [ ] Profile critical paths
- [ ] Optimize bottlenecks
- [ ] Benchmark operations
- [ ] Document performance

**Focus Areas**:
- Symlink creation (batch operations)
- Registry loading (lazy loading if needed)
- Concurrent downloads (tune workers)
- File operations (buffering)

**Deliverables**:
- Performance benchmarks
- Optimized critical paths
- Meets performance targets

**Estimated Time**: 1-3 days

**Phase 6 Total**: 7-14 days

---

## 10. Testing Strategy

### 10.1 Unit Tests (Target: 70%+ coverage)

**What to Test**:
- Individual functions
- Edge cases
- Error conditions
- Input validation

**Tools**:
- Standard `go test`
- `testify` for assertions
- `gomock` for mocking (if needed)

**Example**:
```go
func TestRegistry_AddPackage(t *testing.T) {
    registry := NewRegistry()
    pkg := Package{Name: "bash", Version: "5.2.26-1"}
    
    err := registry.AddPackage(pkg)
    assert.NoError(t, err)
    
    retrieved, err := registry.GetPackage("bash")
    assert.NoError(t, err)
    assert.Equal(t, pkg.Version, retrieved.Version)
}
```

### 10.2 Integration Tests

**What to Test**:
- Component interactions
- ALPM integration
- File operations
- Complete workflows

**Example**:
```go
func TestInstallPackage_Integration(t *testing.T) {
    // Set up test environment
    tmpDir := t.TempDir()
    
    // Create installer
    installer := setupTestInstaller(tmpDir)
    
    // Install package
    err := installer.Install([]string{"coreutils"})
    assert.NoError(t, err)
    
    // Verify symlinks created
    _, err = os.Stat("/usr/bin/ls")
    assert.NoError(t, err)
    
    // Verify registry updated
    pkg, err := installer.registry.GetPackage("coreutils")
    assert.NoError(t, err)
    assert.NotNil(t, pkg)
}
```

### 10.3 End-to-End Tests

**What to Test**:
- Complete user workflows
- CLI commands
- Real package operations

**Test Scenarios**:
1. Install package with dependencies
2. Remove package with orphans
3. Upgrade package
4. Verify installation
5. Cleanup old versions

**Environment**:
- VM or container with Arch Linux
- Isolated /kod directory
- Test with real packages

---

## 11. Success Criteria

### Functional Criteria
- [ ] Can install packages from Arch repos
- [ ] Dependencies are resolved correctly
- [ ] Can remove packages safely
- [ ] Can upgrade packages
- [ ] Can search and query packages
- [ ] Symlinks work correctly
- [ ] Registry stays synchronized
- [ ] Cleanup removes unused packages

### Quality Criteria
- [ ] 70%+ code coverage
- [ ] Zero critical bugs
- [ ] No known data loss bugs
- [ ] Complete documentation
- [ ] Passes all tests
- [ ] Works on real Arch system

### Performance Criteria
- [ ] Install single package in <10s
- [ ] Install 50 packages in <5min
- [ ] Remove package in <5s
- [ ] Search returns in <2s
- [ ] List packages in <1s

### Usability Criteria
- [ ] CLI is intuitive
- [ ] Error messages are helpful
- [ ] Progress feedback is clear
- [ ] Documentation is complete

---

## 12. Timeline & Resources

### Phase Overview (v4.0 - Cross-Distribution)

| Phase | Duration | Cumulative | Key Deliverables |
|-------|----------|------------|------------------|
| Phase 1: Foundation & ALPM | 7-10 days | Week 1-1.5 | ALPM with `/kod` root, database sync, basic CLI |
| Phase 2: Storage & Extraction | 5-7 days | Week 1.5-2.5 | Package extraction, library discovery, download manager |
| Phase 3: Wrapper & Symlinks | 5-7 days | Week 2.5-3.5 | Wrapper generation, symlink management (to wrappers) |
| Phase 4: Installation | 7-10 days | Week 3.5-5 | Complete installation, full dependency isolation |
| Phase 5: Removal & Queries | 5-7 days | Week 5-6 | Removal, cleanup, search, queries |
| Phase 6: Testing & Polish | 7-10 days | Week 6-7 | Multi-distro Docker tests, bug fixes, optimization |
| Phase 7: Documentation | 3-5 days | Week 7 | User guide, FAQ, troubleshooting |
| **Total** | **44-60 days** | **7-9 weeks** | **Production-ready v1.0 (cross-distro)** |

**NEW in v4.0:**
- Added database sync system (Phase 1)
- Added wrapper generation (Phase 3)
- Added library path discovery (Phase 2)
- Added multi-distribution Docker testing (Phase 6)
- Extended timeline by 1-3 weeks for cross-distribution complexity

### Gantt Chart (v4.0)

```
Week:    1      2      3      4      5      6      7      8      9
Phase 1: [======]
Phase 2:       [=====]
Phase 3:             [=====]
Phase 4:                   [=======]
Phase 5:                           [=====]
Phase 6:                                 [======]
Phase 7:                                        [===]
Testing: ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
Docs:                                                [===]
Docker:                                         [======]
```

### Resource Requirements

**Developer Time**:
- 1 full-time developer: 7-9 weeks
- 2 developers: 5-6 weeks (some parallelization possible)

**Hardware**:
- Development machine (ANY Linux distribution: Ubuntu, Fedora, Debian, Arch)
- Docker or Podman for multi-distribution testing
- 30-50 GB free space for `/kod/` testing + Docker images

**Tools/Services**:
- Docker/Podman (for testing on Ubuntu, Fedora, Debian)
- Go 1.21+
- libalpm-dev (`apt install libalpm-dev` on Ubuntu/Debian)
- git, make
- Arch Linux mirror (https://mirror.rackspace.com/archlinux/)

**Testing Distributions**:
- Ubuntu 22.04 LTS (Docker)
- Ubuntu 24.04 LTS (Docker)
- Fedora 40 (Docker)
- Debian 12 (Docker)
- Arch Linux (optional, native development)

### Risk Factors

**Technical Risks**:
- Library compatibility issues on older distributions
- ALPM API changes
- Wrapper script edge cases
- SELinux/AppArmor policy conflicts

**Mitigation**:
- Early testing on all target distributions
- Comprehensive test suite
- Clear error messages with troubleshooting hints
- Docker-based CI/CD for continuous multi-distro testing

---
- Git (version control)
- Go 1.21+ compiler
- libalpm development files
- Optional: GitHub for hosting

### Milestones

**Week 1**: ✅ Foundation complete, can query packages  
**Week 2**: ✅ Can extract packages to store  
**Week 3**: ✅ Symlinks working, basic install works  
**Week 4**: ✅ Full installation with dependencies works  
**Week 5**: ✅ Removal and queries working  
**Week 6**: ✅ Polished, tested, documented  
**Week 7-8**: Buffer for unexpected issues

---

## 13. Risk Management

### High Risk Items

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| ALPM integration issues | Critical | Low | Test early, check go-alpm examples |
| Symlink conflicts | High | Medium | Good conflict detection, clear errors |
| Data loss in registry | Critical | Low | Atomic writes, backups, testing |
| Performance issues | Medium | Medium | Profile early, optimize as needed |

### Medium Risk Items

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Scope creep | Medium | High | Stick to plan, defer features |
| Bugs in edge cases | Medium | High | Comprehensive testing |
| User confusion | Medium | Medium | Good docs, clear errors |

---

## 14. Post-Launch (v1.1+)

### v1.1 Features (2-3 weeks)
- [ ] Package script execution
- [ ] Hook system integration
- [ ] Improved error recovery

### v2.0 Features (8-10 weeks)
- [ ] Generation management
- [ ] Boot integration
- [ ] Multiple system states

### v2.5 Features (4-5 weeks)
- [ ] AUR support
- [ ] Custom repositories
- [ ] Package building

---

## Document Control

**Revision History:**

| Version | Date | Changes |
|---------|------|---------|
| 3.0 | 2026-03-21 | Simplified 6-8 week plan for Go |
| 2.0 | 2026-03-21 | Complex 20-week plan |
| 1.0 | 2026-01-01 | Initial plan |

---

*End of Implementation Plan*
